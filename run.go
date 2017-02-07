/*
	Copyright (C) 2017  John Thayer

	This program is free software; you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation; either version 2 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License along
	with this program; if not, write to the Free Software Foundation, Inc.,
	51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
*/

// Package sifter contains the core functionality of the
// file sifter application.
package sifter

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// The max number of error messages to save for the summary output
const maxErrorMessages = 50

// When running unit tests, don't start IO thread
var unitTest = false

// Holds a list of columns and the default list
type ColSelector struct {
	cols    []Column // the list of columns to use for processing
	defauls []Column // the default list to use when no command line option given
}

// Context is the main object that holds the state of a running instance of
// the file sifter program.
type Context struct {
	// The following fields are set by the command line program before starting
	OutCols        ColSelector       // columns to show in output
	SortCols       ColSelector       // columns to sort by, in order of precedence
	KeyCols        ColSelector       // columns to use for compare key
	PreFilterArgs  []*Filter         // filter objects as parsed from command line --prefilter args
	PostFilterArgs []*Filter         // filter objects as parsed from command line --postfilter args
	Roots          map[bool][]string // lists of root paths, [false] = left side, [true] = right side
	CurSide        bool              // current side to add roots to (true after ":" command line arg)
	Verbosity      int               // verbosity level, default=0
	SummaryOnly    bool              // true to suppress file entry output
	GroupNumerics  bool              // true to group numbers with commas, like 1,234
	Plain          bool              // true to suppress header and footer output
	Plain0         bool              // like Plain, but also use '0x00' to separate fields
	FollowLinks    bool              // true to follow/use targets of symbolic links
	RegularOnly    bool              // true to only index regular files
	AddMd5         bool              // true to add digest columns to output and compare key...
	AddSha1        bool              // "
	AddSha256      bool              // "
	AddSha512      bool              // "
	JsonOut        bool              // true to output data in JSON format
	MembershipFilt string            // add a postfilter based on membership codes [lrLR]
	IgnoreNullCmps bool              // suppress warnings about null comparisons
	Excludes       []*regexp.Regexp  // pattern to exclude files, dir trees by path match
	OutputPath     string            // output file, if any (default=stdout)
	NoDetect       bool              // true to suppress autodetect of FSIFT files for roots
	XDev           bool              // true to prevent descending into directories on different file systems
	Verify         bool              // true to check that all files on left are matched on right
	OutputTimezone *time.Location    // if set, translate output dates to given timezone

	// Internal fields
	entries         []fileEntry     // all of the loaded file entries
	preFilter       *Filter         // the compiled prefilter tree, if any
	postFilter      *Filter         // the compiled postfilter tree, if any
	neededCols      map[Column]bool // set of all columns relevant to this run
	curFileCount    int64           // files/bytes read so far while calculating digest(s)
	curByteCount    int64           // "
	scanStats       stats           // stats for files scanned in directory trees
	indexStats      stats           // stats for files loaded into self.entries
	unmatchedStats  stats           // stats for files that did not match
	matchingStats   stats           // stats for files that did match
	outputStats     stats           // stats for files that were output
	startTime       time.Time       // run start time
	warningCount    int             // total warnings
	warningMessages []string        // warning messages up to limit
	errorCount      int             // total errors
	errorMessages   []string        // error messages up to limit
	nullErrorCount  int             // number of null comparisons made during run
	outputFile      *os.File        // if writing to a file, the handle so it can be closed
	outputState                     // output thread management object
}

// Convert a time.Time object to a string in RFC3339 format in the given
// location. If location is nil, use UTC.
func timeToMtime(tm time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return tm.In(loc).Format(time.RFC3339)
}

// Convert a date/time string in RFC3339 format to a time.Time object.
// Return an error if unable to parse.
func mtimeToTime(mtime string) (time.Time, error) {
	return time.Parse(time.RFC3339, mtime)
}

// NewContext creates a newly initialized context object with default settings.
// Note that unless the global flag unitTest is set, the shutDown method
// must eventually be called to avoid leaking a goroutine.
func NewContext() *Context {
	ctx := Context{
		Roots: map[bool][]string{},
	}
	ctx.OutCols.defauls = []Column{ColModestr, ColSize, ColMtime, ColPath}
	ctx.KeyCols.defauls = []Column{ColPath, ColSize, ColMtime, ColModestr}
	ctx.neededCols = map[Column]bool{}
	ctx.outputState.stream = os.Stdout
	ctx.outputState.errStream = os.Stderr
	ctx.outputState.lineSeparator = "\n"
	ctx.scanStats.name = "Scanned:"
	ctx.indexStats.name = "Indexed:"
	ctx.unmatchedStats.name = "Unmatched:"
	ctx.matchingStats.name = "Matching:"
	ctx.outputStats.name = "Output:"
	if !unitTest {
		// start the output thread
		ctx.outputState.msgChan = make(chan message, 50)
		ctx.outputState.exitChan = make(chan int)
		go ctx.outputState.outputThread()
	}
	return &ctx
}

// Dates are always internally represented in UTC. If the user has specified
// a custom output timezone, convert a date string to that timezone.
// If there is an error, return the string unmodified.
func (self *Context) adjustOutputTimezone(mtime string) (out string) {
	out = mtime
	if self.OutputTimezone != nil {
		tm, e := mtimeToTime(mtime)
		if e == nil {
			out = timeToMtime(tm, self.OutputTimezone)
		}
	}
	return
}

// Set the output and error streams. Used for unit testing.
func (self *Context) SetOutputStreams(out, err io.Writer) {
	if out != nil {
		self.outputState.stream = out
	}
	if err != nil {
		self.outputState.errStream = err
	}
}

// For unrecoverable errors: show an error message and exit immediately.
func (self *Context) fatal(v ...interface{}) {
	self.onError(v...)
	self.shutDown()
	os.Exit(2)
}

// Show an error message, and save it for output with the summary info.
func (self *Context) onError(v ...interface{}) {
	msg := "Error: " + fmt.Sprint(v...)
	self.outputState.message(msgError, msg)
	self.errorCount++
	if len(self.errorMessages) < maxErrorMessages {
		self.errorMessages = append(self.errorMessages, msg)
	}
}

// Show a warning message, and save it for output with the summary info.
func (self *Context) onWarning(v ...interface{}) {
	msg := "Warning: " + fmt.Sprint(v...)
	self.outputState.message(msgError, msg)
	self.warningCount++
	if len(self.warningMessages) < maxErrorMessages {
		self.warningMessages = append(self.warningMessages, msg)
	}
}

// Update a column-based option by adding columns at the indicated position.
// An argument prefixed with '+' causes the columns to be appended to any current
// columns. The posn index may be negative to insert near the end. The
// string in arg is parsed into a list of columns.
func (self *Context) UpdateColumnsCmdlineArg(colSel *ColSelector, posn int, arg string) {
	// check if user wants to append columns
	appending := strings.HasPrefix(arg, "+")
	if appending {
		arg = arg[1:]
	}
	// parse the list of columns
	cols, err := ParseColumnsList(arg)
	if err == nil {
		if appending {
			if colSel.cols == nil {
				// no columns were specified previously; start with defaults
				colSel.cols = append(colSel.cols, colSel.defauls...)
			}
			// add all of the new columns to the list
			for _, col := range cols {
				if !containsCol(colSel.cols, col) {
					insertCol(&colSel.cols, posn, col)
				}
			}
		} else {
			// not appending; just replace any list with the new one
			colSel.cols = cols
		}
	} else {
		self.fatal("Error parsing column specification:", err)
	}
}

// After all command line parameters have been processed, make any further
// required setting changes based on the options that were given.
func (self *Context) adjustCmdlineOptions() {
	var err error
	self.outputState.verbosity = self.Verbosity // copy verbosity to output object
	// --plain0 implies --plain, use null chars for line separator
	if self.Plain0 {
		self.Plain = true
		self.outputState.lineSeparator = "\x00"
	}

	// check for roots on each side
	hasLeft := len(self.Roots[false]) != 0
	hasRight := len(self.Roots[true]) != 0

	if hasLeft && hasRight {
		// if both sides are present, we add membership column by default
		self.UpdateColumnsCmdlineArg(&self.OutCols, 0, "+membership")
	} else {
		// make sure output defaults are set if no options were given
		self.UpdateColumnsCmdlineArg(&self.OutCols, 0, "+")
	}
	// make sure output defaults are set if no options were given
	self.UpdateColumnsCmdlineArg(&self.KeyCols, 0, "+")

	// if digest shortcuts were specified, add the relevant columns
	if self.AddMd5 {
		self.UpdateColumnsCmdlineArg(&self.OutCols, -1, "+md5")
		self.UpdateColumnsCmdlineArg(&self.KeyCols, 0, "+md5")
	}
	if self.AddSha1 {
		self.UpdateColumnsCmdlineArg(&self.OutCols, -1, "+sha1")
		self.UpdateColumnsCmdlineArg(&self.KeyCols, 0, "+sha1")
	}
	if self.AddSha256 {
		self.UpdateColumnsCmdlineArg(&self.OutCols, -1, "+sha256")
		self.UpdateColumnsCmdlineArg(&self.KeyCols, 0, "+sha256")
	}
	if self.AddSha512 {
		self.UpdateColumnsCmdlineArg(&self.OutCols, -1, "+sha512")
		self.UpdateColumnsCmdlineArg(&self.KeyCols, 0, "+sha512")
	}

	// add postfilters to implement any --membership codes
	var filts []*Filter
	for _, c := range self.MembershipFilt {
		switch c {
		case 'L':
			filts = append(filts, &Filter{op: opEq, column: ColMembership, value: "<!"})
		case 'R':
			filts = append(filts, &Filter{op: opEq, column: ColMembership, value: ">!"})
		case 'l':
			filts = append(filts, &Filter{op: opEq, column: ColMembership, value: "<="})
		case 'r':
			filts = append(filts, &Filter{op: opEq, column: ColMembership, value: ">="})
		default:
			self.fatal("--membership filter codes must be one or more of [lrLR]")
		}
	}
	// all of the membership filters get ORed together
	for i := 0; i < len(filts)-1; i++ {
		self.PostFilterArgs = append(self.PostFilterArgs, &Filter{op: opOr})
	}
	self.PostFilterArgs = append(self.PostFilterArgs, filts...)

	// determine the set of all columns to calculate
	self.calcNeededCols()
	if hasRight {
		self.neededCols[ColMatched] = true
		self.neededCols[ColSide] = true
	}

	// compile filter lists into trees
	self.postFilter, err = compileFilter(self.PostFilterArgs)
	if err != nil {
		self.fatal("Error compiling post filter args:", err)
	}
	self.preFilter, err = compileFilter(self.PreFilterArgs)
	if err != nil {
		self.fatal("Error compiling pre filter args:", err)
	}

	// reset current side flag in preparation for run
	self.CurSide = false

	// if no roots given, default to just "."
	if !hasLeft && !hasRight {
		self.Roots[false] = append(self.Roots[false], ".")
	}
	// create the output file if specified
	if self.OutputPath != "" {
		self.outputFile, err = os.Create(self.OutputPath)
		self.outputState.stream = self.outputFile
		if err != nil {
			self.fatal("Error opening output file:", err)
		}
	}
}

// Run the scan defined for this context. Assumes that the exported option
// fields have been set as necessary before this method is called.
// Returns zero on success, nonzero if any errors occurred during the run.
func (self *Context) Run() int {
	// finalize settings and output header
	self.adjustCmdlineOptions()
	self.showHeader()

	// scan roots on left then right side
	for i := 0; i < 2; i++ {
		for _, path := range self.Roots[i != 0] {
			self.outTempf(0, "Processing root... '%s'", path)
			self.CurSide = i != 0
			self.processRoot(path)
		}
	}

	// if calculating matches, go do file matching
	if self.needsCol(ColMatched) || self.needsCol(ColRedundancy) || self.needsCol(ColRedunIdx) {
		self.analyzeMatches()
	}

	// do postfiltereing, and also dummy output pass to calc column widths
	self.outTempf(0, "Filtering and formatting... %d", len(self.entries))
	nCols := len(self.OutCols.cols)
	widths := make([]int, nCols)
	if len(widths) > 0 {
		// last column doesn't get padded
		widths[nCols-1] = -1
	}
	filtered := []fileEntry{} // entries that pass the postfilter
	for _, e := range self.entries {
		// check against postfilter
		match, notNull := self.postFilter.filter(e)
		self.checkNullCompare(notNull)
		if !match {
			continue
		}
		filtered = append(filtered, e)
		// calculate output for all but last column to get max widths
		for i := 0; i < nCols-1; i++ {
			// plain0 and json don't get padded, also skip if only summary to speed things up
			if !self.Plain0 && !self.JsonOut && !self.SummaryOnly {
				field := e.formatField(self, self.OutCols.cols[i], -1, false)
				width := utf8.RuneCountInString(field)
				if width > widths[i] {
					widths[i] = width
				}
			} else {
				widths[i] = -1
			}

		}
	}
	// sort entries if necessary
	if len(self.SortCols.cols) > 0 {
		self.outTempf(0, "Sorting... (%d entries)", len(filtered))
		sorter := newEntrySorter(self, filtered, self.SortCols.cols)
		sort.Sort(sorter)
	}
	// report any null comparisons during run
	if self.nullErrorCount > 0 {
		self.onError("Comparison of a NULL value attempted ", self.nullErrorCount, " time(s)")
	}

	separator := "  "
	indent := "  "
	if self.Plain0 {
		separator = "\x00"
		indent = ""
	}
	var fields []string
	if !self.SummaryOnly {
		// go through filtered entries output them
		for j, e := range filtered {
			if !self.JsonOut {
				// update output stats
				filePath, _ := e.getStringField(ColPath)
				if !strings.HasSuffix(filePath, "/") {
					self.outputStats.update(e.getBoolFieldOrFalse(ColSide), e.getNumericFieldOrZero(ColSize))
				} else {
					self.outputStats.update(e.getBoolFieldOrFalse(ColSide), 0)
				}
				// format the output fields in this entry, padded to the max column width and output the line
				fields = fields[:0]
				for i, col := range self.OutCols.cols {
					fields = append(fields, e.formatField(self, col, widths[i], i >= nCols-1 || self.Plain0))
				}
				self.outf(-1, indent+"%s", strings.Join(fields, separator))
			} else {
				// JSON mode; figure out if we need a comma after this entry
				sep := ","
				if j == len(filtered)-1 {
					sep = ""
				}
				// convert entry fields to JSON
				json, err := e.toJson(self.OutCols.cols)
				// output each line in the JSON text
				if err == nil {
					for _, line := range strings.Split("    "+string(json)+sep, "\n") {
						self.outf(-1, "%s", line)
					}
				} else {
					self.onError("Error encoding JSON output: ", err)
				}
			}
		}
	}
	// show summary info
	self.showFooter()

	// flush the output thread and shut it down, then return error code
	self.shutDown()
	rc := 0
	if self.errorCount > 0 {
		rc = 1
	}
	return rc
}

// Notify output thread we're done, wait for output to flush, also close any
// output file. This method *must* be called on each context or the output
// goroutine will leak.
func (self *Context) shutDown() {
	close(self.outputState.msgChan)
	<-self.outputState.exitChan

	if self.outputFile != nil {
		self.outputFile.Close()
	}
}
