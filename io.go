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

package sifter

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Magic header identifying file sifter files
const sifterFileHeader = "| File Sifter output file - V1 |"

// Parse a sifter file and load its entries into the current context.
func (self *Context) loadSifterFile(r io.Reader) error {
	columns := []Column{}          // columns detected in the file from header directive
	scanner := bufio.NewScanner(r) // help read file by lines
	var err error
	// regex to select delimiters in entry lines
	pat := regexp.MustCompile(`[^\\] `)

	// process each line
	for scanner.Scan() {
		line := scanner.Bytes()
		// if directive, handle it
		if bytes.HasPrefix(line, []byte("|")) {
			// check if it's a 'Columns' directive and set column list if so
			cols, err := parseColumnsDirective(line)
			if err != nil {
				return err
			} else {
				if cols != nil {
					columns = cols
				}
				// It's some other directive; ignore
				continue
			}
		}

		// Must be a file entry line; columns must be defined by now
		if len(columns) < 1 {
			return fmt.Errorf("No column names were defined before data entries")
		}

		// create a new file entry object and fill in its fields
		entry := newFileEntry()
		for i := 0; i < len(columns); i++ {
			// Fields are separated by spaces; remove previous delimiter
			line = bytes.TrimLeft(line, " ")

			// Look for next delimiter (or end-of-line in last column)
			end := len(line)
			if i < len(columns)-1 {
				ends := pat.FindIndex(line)
				if ends == nil {
					self.onError("Could not find delimiter in FSIFT file")
					break
				}
				end = ends[0] + 1
			}
			// get field value and add it to the file entry
			field, notNull := unescapeField(string(line[:end]))
			if notNull {
				err = entry.parseAndSetField(self, columns[i], field)
				if err != nil {
					self.onError("Parse error in FSIFT file: ", err)
					break
				}
			}
			line = line[end:]
		}

		if err == nil {
			// add "side" field if needed
			if self.needsCol(ColSide) {
				entry.setBoolField(ColSide, self.CurSide)
			}
			// check any prefilter conditions against the entry
			match, notNull := self.preFilter.filter(entry)
			self.checkNullCompare(notNull)
			// get size field for stats computation; directory sizes assumed zero for stats
			size := entry.getNumericFieldOrZero(ColSize)
			path, _ := entry.getStringField(ColPath)
			if strings.HasSuffix(path, "/") {
				size = 0
			}
			self.scanStats.update(self.CurSide, size)
			// if prefilter passes, add the entry to the current context
			if match {
				self.indexStats.update(self.CurSide, size)
				self.entries = append(self.entries, entry)
			}
		}
	}
	return scanner.Err()
}

// Compute a string representation of the given number using the current format settings in the context.
func (self *Context) formatNumber(n int64) string {
	// compute the basic decimal number, abs value and sign
	minus := ""
	if n < 0 {
		minus = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)

	// if group digits option is on, add commas every 3rd digit
	if self.GroupNumerics {
		buf := make([]byte, 0, 16)
		remain := len(s) - 1
		for _, c := range []byte(s) {
			buf = append(buf, c)
			if remain > 0 && remain%3 == 0 {
				buf = append(buf, ',')
			}
			remain--
		}
		s = string(buf)
	}
	return minus + s
}

// Return true if this looks like a FSIFT file, report any errors reading the file.
func detectSifterFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, len(sifterFileHeader))
	_, err = io.ReadAtLeast(f, buf, len(sifterFileHeader))
	switch {
	case err == nil && string(buf) == sifterFileHeader:
		// First line matches magic header, return true
		return true, nil
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		// Short file, return false
		return false, nil
	default:
		// Otherwise false, report any error
		return false, err
	}
}

// Internal path representation: join paths like filepath package, but always convert to forward slashes.
func myJoin(paths ...string) string {
	return filepath.ToSlash(filepath.Join(paths...))
}

// Look at a file in the file system under "root/relPath", and create a new
// file entry object with the relevant info. Also returns the size of the file
// (which is zero for nonregular files).  Returns nil if the file info can't be
// accessed or if it was rejected by the prefilter.  If pruneCheck is false, the
// entry is also added to the current context and stats are updated if not
// filtered.
func (self *Context) processFile(root, relPath string, pruneCheck bool) (fileEntry, int64) {
	// create entry, compute paths, and get stat info
	entry := newFileEntry()
	relPath = path.Clean(relPath)
	filePath := myJoin(root, relPath)
	finfo, err := self.statFile(filePath)
	if err != nil {
		self.onError("Can't get info about file: ", err)
		return nil, 0
	}
	xinfo := statExtended(finfo)
	if finfo.Mode().IsDir() {
		relPath += "/"
	}

	// always add the path and size fields
	entry.setStringField(ColPath, relPath)
	size := finfo.Size()
	if !finfo.Mode().IsRegular() {
		size = 0
	}
	entry.setNumericField(ColSize, size)

	// add additional fields as required
	for col, _ := range self.neededCols {
		switch col {
		case ColPath:
			// already set
		case ColSize:
			// already set
		case ColMtime:
			entry.setStringField(col, timeToMtime(finfo.ModTime(), nil)) // always UTC
		case ColMstamp:
			entry.setNumericField(col, finfo.ModTime().Unix())
		case ColSide:
			entry.setBoolField(col, self.CurSide)
		case ColDevice:
			entry.setNumericField(col, int64(xinfo.device))
		case ColNlinks:
			entry.setNumericField(col, int64(xinfo.nlinks))
		case ColUid:
			if xinfo.uidGidValid {
				entry.setNumericField(col, int64(xinfo.uid))
			}
		case ColGid:
			if xinfo.uidGidValid {
				entry.setNumericField(col, int64(xinfo.gid))
			}
		case ColUser:
			if xinfo.uidGidValid {
				user, err := user.LookupId(fmt.Sprintf("%v", xinfo.uid))
				if err == nil {
					entry.setStringField(col, user.Username)
				} else {
					self.onError("Could not get user name for UID ", xinfo.uid, " :", err)
				}
			}
		case ColGroup:
			if xinfo.uidGidValid {
				group, err := user.LookupGroupId(fmt.Sprintf("%v", xinfo.gid))
				if err == nil {
					entry.setStringField(col, group.Name)
				} else {
					self.onError("Could not get group name for GID ", xinfo.gid, " :", err)
				}
			}
		case ColModestr:
			entry.setStringField(col, finfo.Mode().String())
		case ColFileType:
			entry.setStringField(col, modeStrToFileType(finfo.Mode().String()))
		}
	}
	match := false
	notNull := false
	if !pruneCheck {
		// update the "scan" stats and the interactive progress message
		self.scanStats.update(self.CurSide, size)
		allBytes := self.scanStats.leftSize + self.scanStats.rightSize
		allFiles := self.scanStats.leftCount + self.scanStats.rightCount
		self.outTempf(0, "Scan(%dMB in %d) %s", allBytes/1000000, allFiles, filePath)

		// apply any prefilters; if not filtered, update index stats and add entry to context
		match, notNull = self.preFilter.filter(entry)
		if match {
			self.indexStats.update(self.CurSide, size)
			self.entries = append(self.entries, entry)
		}
	} else {
		match, notNull = self.pruneFilter.filter(entry)
	}
	self.checkNullCompare(notNull)
	if match {
		return entry, size
	} else {
		return nil, 0
	}
}

// Scan a directory tree in the file system, adding file entries to the context.
// The tree is at root/relPath. dirInfos contains a list of the directory
// nodes that have been visited so far in the recursive scan; it is used to
// detect cyclic symlinks. The last entry in dirInfos must be the directory
// specified by root/relPath. The return value is the cumulative size of the
// files in the directory tree.
func (self *Context) scanDirTree(root, relPath string, dirInfos []os.FileInfo) int64 {
	size := int64(0)

	dirInf := dirInfos[len(dirInfos)-1]
	device := statExtended(dirInf).device

	// read the entries in this directory
	dir := myJoin(root, relPath)
	if dir == "/" {
		dir = "/." // For some reason, a bare "/" doesn't work on Windows
	}
	f, err := os.Open(dir)
	if err != nil {
		self.onError("Could not open directory: ", err)
		return size
	}
	defer f.Close()

	list, err := f.Readdir(0)
	if err != nil {
		self.onError("Could not read directory: ", err)
		return size
	}
	// process each file in this directory
DirLoop:
	for _, fi := range list {
		// skip if file matches an exclude pattern
		for _, regex := range self.Excludes {
			if regex.MatchString(fi.Name()) {
				continue DirLoop
			}
		}
		// get file info
		newRelPath := myJoin(relPath, fi.Name())
		newAbsPath := myJoin(root, newRelPath)
		fi, err = self.statFile(newAbsPath)
		if err != nil {
			self.onError("Can't get info about file: ", err)
			continue
		}
		if fi.IsDir() && (fi.Mode()&os.ModeSymlink == 0) && (!self.XDev || statExtended(fi).device == device) {
			// file is a subdirectory to descend into; check for circular links
			for _, pfi := range dirInfos {
				if os.SameFile(fi, pfi) {
					self.onWarning("Found circular symlink reference at: ", newAbsPath)
					continue DirLoop
				}
			}
			entry, _ := self.processFile(root, newRelPath, true)
			// recursively scan the subdirectory unless pruned by prefilter
			if entry != nil {
				dirInfos = append(dirInfos, fi)
				s := self.scanDirTree(root, newRelPath, dirInfos)
				dirInfos = dirInfos[:len(dirInfos)-1]
				size += s
			}
		} else if !self.RegularOnly || fi.Mode().IsRegular() {
			// other type of file; add it to the context
			_, s := self.processFile(root, newRelPath, false)
			size += s
		}
	}
	if !self.RegularOnly {
		// add an entry for this directory to the context
		entry, _ := self.processFile(root, relPath, false)
		if entry != nil {
			entry.setNumericField(ColSize, size)
		}
	}
	return size
}

// Extra file info not returned by standard Stat or Lstat
type statEx struct {
	device      uint64 // device ID that file resides on
	nlinks      uint64 // number of hard links
	uid         uint32
	gid         uint32
	uidGidValid bool // true if uid and gid are supported on this platform
}

// Do the appropriate type of stat call depending on whether the the "follow-links"
// option was specified.
func (self *Context) statFile(path string) (os.FileInfo, error) {
	if self.FollowLinks {
		return os.Stat(path)
	} else {
		return os.Lstat(path)
	}
}

// Map from digest column IDs to algorithm factories
var hashes = map[Column]func() hash.Hash{
	ColMd5:    md5.New,
	ColSha256: sha256.New,
	ColSha512: sha512.New,
	ColSha1:   sha1.New,
	ColCrc32:  func() hash.Hash { return crc32.NewIEEE() },
}

// Compute the value of a digest field for a file by reading the file.
// col specifies the type of digest. The field is added to the given entry.
func (self *Context) calcDigestFile(col Column, root string, entry fileEntry) {
	// get the file name and open it
	relPath, ok := entry.getStringField(ColPath)
	if !ok {
		self.onError("Missing path in file entry: ")
		return
	}
	filePath := myJoin(root, relPath)
	fi, err := self.statFile(filePath)
	if err != nil {
		self.onError("Can't get file information: ", err)
		return
	}
	if !fi.Mode().IsRegular() {
		// nonregular files get empty digests (not null, so we don't get null compare warnings)
		entry.setStringField(col, "")
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		self.onError("Can't open file for reading: ", err)
		return
	}
	defer file.Close()

	// create the hash algorithm and feed the file data to it, then add the result to the entry
	// TODO: for huge files, read in chunks, update info message periodically
	hashFactory, _ := hashes[col]
	hash := hashFactory()
	_, err = io.Copy(hash, file)
	if err != nil {
		self.onError("Can't read file for digest calculation: ", err)
		return
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	entry.setStringField(col, sum)

	// update the interactive info message with the scan progress
	self.curFileCount++
	self.curByteCount += entry.getNumericFieldOrZero(ColSize)
	// TODO: if multiple digest cols specified, displayed counts will be off
	allBytes := self.scanStats.leftSize + self.scanStats.rightSize
	allFiles := self.scanStats.leftCount + self.scanStats.rightCount
	self.outTempf(0, "%s(%dMB/%dMB in %d/%d) %s", col,
		self.curByteCount/1000000, allBytes/1000000,
		self.curFileCount, allFiles, filePath)
}

// Calculate any needed digest fields for the file entries in the given list.
func (self *Context) calcDigestList(root string, entries []fileEntry) {
	for _, col := range []Column{ColMd5, ColSha1, ColSha256, ColSha512, ColCrc32} {
		if self.neededCols[col] {
			for _, entry := range entries {
				self.calcDigestFile(col, root, entry)
			}
		}
	}
}

// Scan a given "root" specified on the command line, adding entries
// to the context as appropriate.
func (self *Context) processRoot(path string) {
	if path == "-" {
		// special case: '-' means stdin
		err := self.loadSifterFile(os.Stdin)
		if err != nil {
			self.fatal("Can't parse FSIFT content from stdin:", err)
		}
		return
	}
	finfo, err := self.statFile(path)
	if err != nil {
		self.fatal("Can't get file information:", err)
	}
	if !finfo.IsDir() {
		// not a dir; check to see if it's a FSIFT file
		isFS := false
		if !self.NoDetect && finfo.Mode().IsRegular() {
			isFS, _ = detectSifterFile(path)
		}
		if isFS {
			// it's a FSIFT file; parse it and load its entries
			f, err := os.Open(path)
			if err != nil {
				self.fatal("Can't open file:", err)
			}
			err = self.loadSifterFile(f)
			if err != nil {
				self.fatal("Can't parse FSIFT file:", err)
			}
		} else {
			// not a FSIFT file; just add an entry for it
			base := len(self.entries)
			self.processFile(path, "", false)
			self.calcDigestList("", self.entries[base:])
		}
	} else {
		// root is a directory; go scan it
		base := len(self.entries)
		self.scanDirTree(path, ".", []os.FileInfo{finfo})
		// calc any digests for the newly added entries
		self.calcDigestList(path, self.entries[base:])
	}
}

// Output a header or footer line, which is always prefixed with "| "
func (self *Context) headerOut(format string, a ...interface{}) {
	self.outf(-1, "| "+format, a...)
}

// Output the header info before processing roots
func (self *Context) showHeader() {
	if self.Plain {
		// skip header in 'plain' mode
		return
	}
	if self.JsonOut {
		// in json mode, just start the entry array
		self.outf(-1, "[")
		return
	}
	// output magic header ID and command line parameters
	self.outf(-1, "%s", sifterFileHeader)
	cmdLine := strings.Join(os.Args[1:], " ")
	if len(cmdLine) > 500 {
		cmdLine = cmdLine[:500] + " ..."
	}
	self.headerOut("Command line: %s", cmdLine)
	// output CWD, compare key columns, and all computed columns
	cwd, _ := os.Getwd()
	self.headerOut("Current working directory: %s", cwd)
	self.headerOut("Compare keys: %s", formatColumnNames(self.KeyCols.cols))
	if len(self.SortCols.cols) > 0 {
		self.headerOut("Sort keys: %s", formatColumnNames(self.SortCols.cols))
	}
	var needed []Column
	for col := Column(0); col < ColLAST; col++ {
		if self.neededCols[col] {
			needed = append(needed, col)
		}
	}
	self.headerOut("Evaluated columns: %s", formatColumnNames(needed))
	// output start time and the main entry column header
	self.startTime = time.Now()
	self.headerOut("Run start time: %v", timeToMtime(self.startTime, self.OutputTimezone))
	self.headerOut("")
	self.headerOut("Columns: %s", formatColumnNames(self.OutCols.cols))
	self.headerOut("")
}

// Show a summary of any saved error or warning messages after a program run.
// The list has the messages, count is the total number found (which may
// be greater than len(list) if limit hit), prefix describes the list.
func (self *Context) showErrors(list []string, count int, prefix string) {
	if count > 0 {
		self.headerOut("")
		self.headerOut("*** %s ENCOUNTERED DURING RUN:", prefix)
		self.headerOut("")
		for _, msg := range list {
			self.headerOut(msg)
		}
		if len(list) < count {
			self.headerOut("Limit reached; %d more error(s) not printed", count-len(list))
		}

	}
}

// Show final summary info after processing roots.
func (self *Context) showFooter() {
	if self.Plain {
		return
	}
	if self.JsonOut {
		// in json mode, just terminate the entry array
		self.outf(-1, "]")
		return
	}
	// show run times
	self.headerOut("")
	now := time.Now()
	self.headerOut("Run end time: %v", timeToMtime(now, self.OutputTimezone))
	self.headerOut("Elapsed time: %v", now.Sub(self.startTime))
	self.headerOut("")
	// calculate the summary stats and show them
	for _, line := range self.calcSummaryInfo() {
		self.headerOut(strings.Join(line, "  "))
	}

	// show any warnings or errors
	self.showErrors(self.warningMessages, self.warningCount, "WARNINGS")
	self.showErrors(self.errorMessages, self.errorCount, "ERRORS")
}
