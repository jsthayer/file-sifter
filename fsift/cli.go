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

// This is the main entry point for the file sifter CLI program.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jsthayer/file-sifter"
	"github.com/jsthayer/miniflags"
)

// The sifter context to use
var ctx *sifter.Context

// Version info
var (
	ProgName  = "sifter"
	Version   string // filled in at link time (see Makefile)
	BuildDate string // filled in at link time
	Copyright = "Copyright 2017  John Thayer"
)

// Factory function to create an option handler that parses columns.
// Handler operates on the referenced column selector.
func columnOption(colSel *sifter.ColSelector) func(string) {
	return func(val string) {
		ctx.UpdateColumnsCmdlineArg(colSel, -1, val)
	}
}

// Factory function to create an option handler that parses filters
// Handler operates on the referenced filter list.
func filterOption(filts *[]*sifter.Filter) func(string) error {
	return func(val string) error {
		filt, err := sifter.ParseFilter(val)
		*filts = append(*filts, filt)
		return err
	}
}

// Handler for --exclude option adds an exclude pattern.
func excludeAction(arg string) error {
	rex, err := sifter.GlobToRegex(arg)
	if err == nil {
		ctx.Excludes = append(ctx.Excludes, rex)
	}
	return err
}

// Show version info.
func showVersionAndExit() {
	fmt.Printf("%s version %s (build time: %s)\n", ProgName, Version, BuildDate)
	fmt.Println(Copyright)
	fmt.Println("License GPLv2+: GNU GPL version 2 or later.")
	fmt.Println("This is free software: you are free to change and redistribute it.")
	fmt.Println("There is NO WARRANTY, to the extent permitted by law.")
	os.Exit(0)
}

// Handler for --base option adds a custom prefilter.
func baseAction(arg string) error {
	filt, err := sifter.ParseFilter("base*=*" + arg + "*")
	ctx.PreFilterArgs = append(ctx.PreFilterArgs, filt)
	return err
}

// Handler for --out-zone supports both location names like "Local" or
// "America/Chicago", and fixed offsets like "+04:00".
func outZoneAction(arg string) (err error) {
	offsPat := regexp.MustCompile(`^[+-]\d\d:\d\d$`)
	if offsPat.MatchString(arg) {
		h, _ := strconv.Atoi(arg[1:3])
		m, _ := strconv.Atoi(arg[4:6])
		secs := h*60*60 + m*60
		if arg[0] == '-' {
			secs = -secs
		}
		ctx.OutputTimezone = time.FixedZone("fixed", secs)
	} else {
		ctx.OutputTimezone, err = time.LoadLocation(arg)
	}
	return
}

// Nonoption argument handler adds root to current side; ":" switches from
// left to right side.
func argAction(arg string) error {
	if arg == ":" {
		ctx.CurSide = true
	} else {
		ctx.Roots[ctx.CurSide] = append(ctx.Roots[ctx.CurSide], arg)
	}
	return nil
}

// Define and parse command line arguments
func parseArgs(args []string) error {

	miniflags.UsageHeader =
		fmt.Sprintf(`Usage: %s [ options | scan-roots | ":" ]...`, filepath.Base(os.Args[0])) +
			"\n\n  Scan roots before a \":\" argument belong to the \"left\" side," +
			"\n  those after the colon belong to the \"right\" side" +
			"\n\nCOLUMNS codes (example: 'size,time,path' can be shortened to 'stp'):\n  " +
			strings.Join(sifter.GetColumnHelp(), "\n  ") + "\n"

	_, err := miniflags.NewOptionSet().
		ArgAction(argAction).
		Section("Field selection, comparing and sorting:").
		Option("c columns     ", columnOption(&ctx.OutCols), "=COLUMNS; Output columns (default: ostp)").
		Option("s sort        ", columnOption(&ctx.SortCols), "=COLUMNS; Sort output using these fields (default: no sort)").
		Option("k key         ", columnOption(&ctx.KeyCols), "=COLUMNS;Set fields used in comparisons  (default: psto)").
		Option("5 md5         ", &ctx.AddMd5, "Add md5 column to compare key and output").
		Option("2 sha256      ", &ctx.AddSha256, "Add sha256 column to compare key and output").
		Option("A sha512      ", &ctx.AddSha512, "Add sha512 column to compare key and output").
		Option("1 sha1        ", &ctx.AddSha1, "Add sha1 column to compare key and output").
		Section("Pre-analysis filtering:").
		Option("e prefilter   ", filterOption(&ctx.PreFilterArgs), "=FILTER-EXP; Filter files before indexing").
		Option("b base-match  ", baseAction, "=GLOB-PAT; Shortcut for --prefilter 'base*=*GLOB-PAT*'").
		Option("x exclude     ", excludeAction, "=GLOB-PAT; Exclude file system files and/or dir trees by path glob").
		Option("R regular-only", &ctx.RegularOnly, "Only consider regular files while scanning file system").
		Option("L follow-links", &ctx.FollowLinks, "Follow symbolic links while scanning file system").
		Option("X xdev        ", &ctx.XDev, "Don't descend directories on different file systems").
		Section("Post-analysis filtering:").
		Option("f postfilter  ", filterOption(&ctx.PostFilterArgs), "=FILTER-EXP; Filter output after analysis").
		Option("m membership  ", &ctx.MembershipFilt, "=CHARS; Filter output by membership (one or more of lrLR)").
		Option("d diff        ", func() { ctx.MembershipFilt = "LR" }, "Show differing entries only; shortcut for -mLR").
		Option("  nodetect    ", &ctx.NoDetect, "Don't try to detect type of regular files specified as roots").
		Section("Output formatting").
		Option("o out         ", &ctx.OutputPath, "=PATH; Output to file instead of stdout").
		Option("Y verify      ", &ctx.Verify, "Checks that all left entries are matched on right (Analogous to 'md5sum -c'.)").
		Option("S summary     ", &ctx.SummaryOnly, "Only output summary info; no entry lines").
		Option("p plain       ", &ctx.Plain, "Only output entries, no header info").
		Option("0 plain0      ", &ctx.Plain0, "Like 'plain', but also separate all output fields with null chars").
		Option("G group-nums  ", &ctx.GroupNumerics, "Output ',' between groups of numeric digits").
		Option("N ignore-nulls", &ctx.IgnoreNullCmps, "No warnings for comparing nonexistent fields; match always false").
		Option("J json-out    ", &ctx.JsonOut, "Output in JSON format").
		Option("Z out-zone    ", outZoneAction, "Format output times for given location (default=UTC)").
		Option("v verbose     ", miniflags.IncOption(&ctx.Verbosity), "Increase verbosity").
		Option("q quiet       ", miniflags.DecOption(&ctx.Verbosity), "Decrease verbosity").
		Option("V version     ", showVersionAndExit, "Show program version and exit").
		ParseArgs(args)

	return err
}

// Run the context with the given args (nil for os.Args). Can also be called by unit tests.
func run(args []string) int {
	parseArgs(args)
	return ctx.Run()
}

// Main entry point; create a context and run it
func main() {
	ctx = sifter.NewContext()
	rc := run(nil)
	os.Exit(rc)
}
