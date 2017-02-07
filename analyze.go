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
	"fmt"
	"sort"
	"strings"
)

// hold the statistics for a phase of a program run.
type stats struct {
	name       string // the name of this statistics type
	leftCount  int64  // left side total file count
	leftSize   int64  // left side total file size in bytes
	rightCount int64
	rightSize  int64
}

// update this stats object on the given side with one file's size
func (self *stats) update(side bool, size int64) {
	if side {
		self.rightSize += size
		self.rightCount++
	} else {
		self.leftSize += size
		self.leftCount++
	}
}

// after all command line options have been read, determine the entire set
// of columns that need to be calculated; the result goes in self.neededCols.
func (self *Context) calcNeededCols() {
	for _, col := range self.OutCols.cols {
		self.neededCols[col] = true // all output columns
	}
	for _, col := range self.SortCols.cols {
		self.neededCols[col] = true // all sort keys
	}
	for _, col := range self.KeyCols.cols {
		self.neededCols[col] = true // all compare keys
	}
	for _, filt := range self.PreFilterArgs {
		self.neededCols[filt.column] = true // all prefilter fields
	}
	for _, filt := range self.PostFilterArgs {
		self.neededCols[filt.column] = true // all postfilter fields
	}
}

// returns true if the field in the given column is needed for this program run
func (self *Context) needsCol(col Column) bool {
	return self.neededCols[col]
}

// Compare file entries on the right side and left side using the compare key columns,
// and determine which ones match. Update the entries with the relevant info, and update
// the context statistics objects.
func (self *Context) analyzeMatches() {
	entries := self.entries
	unmatchedLeft := false // any files on the left side were unmatched by a file on right

	if len(self.SortCols.cols) == 0 {
		// if not sorting later, use a copied list and leave original scan order in context
		entries = make([]fileEntry, len(self.entries))
		copy(entries, self.entries)
	}
	// sort the entries using the values in the compare key columns
	self.outTempf(0, "Analyzing... %d files", len(entries))
	sorter := newEntrySorter(self, entries, self.KeyCols.cols)
	sort.Sort(sorter)

	base := 0 // first entry in list still matching current file
	needRedun := self.needsCol(ColRedundancy)
	needRedunIdx := self.needsCol(ColRedunIdx)
	// scan list looking for matching groups of files
	for cur := 1; cur < len(entries)+1; cur++ {
		differs := true
		if cur < len(entries) {
			// compare this entry to the base entry of this match group
			d, notNull := entries[base].compare(entries[cur], self.KeyCols.cols)
			self.checkNullCompare(notNull)
			differs = d != 0
		}
		if differs {
			// file differs (or end of list); need to end the old group and start a new group
			leftRedun, rightRedun := 0, 0
			// compute redundancy counts for each side of this group
			for i := base; i < cur; i++ {
				pRedun := &leftRedun
				if entries[i].getBoolFieldOrFalse(ColSide) {
					pRedun = &rightRedun
				}
				*pRedun++
				// Update redundancy index col if needed
				if needRedunIdx {
					entries[i].setNumericField(ColRedunIdx, int64(*pRedun))
				}
			}
			// if there was at least one file on each side, file is considered to "match"
			matched := leftRedun > 0 && rightRedun > 0
			// update all of the files in the match group
			for ; base < cur; base++ {
				// set the matched field in the entry
				entries[base].setBoolField(ColMatched, matched)
				es := entries[base].getBoolFieldOrFalse(ColSide)
				if !es && !matched {
					unmatchedLeft = true // any unmatched left side files trigger an error with --verify
				}
				// update the matched and unmatched scan stats
				size := entries[base].getNumericFieldOrZero(ColSize)
				path, _ := entries[base].getStringField(ColPath)
				if strings.HasSuffix(path, "/") {
					size = 0
				}
				if matched {
					self.matchingStats.update(es, size)
				} else {
					self.unmatchedStats.update(es, size)
				}
				// update redundancy field if needed
				if needRedun {
					redun := leftRedun
					if es {
						redun = rightRedun
					}
					entries[base].setNumericField(ColRedundancy, int64(redun))
				}
			}
		}
	}
	if unmatchedLeft && self.Verify {
		self.onError("At least one entry on the left was unmatched (--verify was specified)")
	}
}

// If a comparison used a null value, flag a warning unless we're ignoring them
func (self *Context) checkNullCompare(notNull bool) {
	if !notNull && !self.IgnoreNullCmps {
		self.nullErrorCount++
	}
}

// Using the information in all of the stats objects, format the summary
// information into a 2D array of strings. The exact fields selected
// depend on the scan options, such as if there are roots on both sides.
func (self *Context) calcSummaryInfo() [][]string {
	hasLeft := len(self.Roots[false]) > 0
	hasRight := len(self.Roots[true]) > 0

	var allStats []*stats

	// more stats are relevant if we have scans on both sides
	if hasLeft && hasRight {
		allStats = []*stats{
			&self.scanStats, &self.indexStats, &self.unmatchedStats, &self.matchingStats, &self.outputStats,
		}
	} else {
		allStats = []*stats{
			&self.scanStats, &self.indexStats, &self.outputStats,
		}
	}

	// initialize header and stat names
	header := []string{"STATISTICS:"}

	out := [][]string{}
	for _, stat := range allStats {
		out = append(out, []string{stat.name})
	}

	if hasLeft {
		// output stat columns for left side stats
		if hasRight {
			header = append(header, "L:Count", "L:Size")
		} else {
			header = append(header, "Count", "Size")
		}
		for i, stat := range allStats {
			out[i] = append(out[i], self.formatNumber(stat.leftCount))
			out[i] = append(out[i], self.formatNumber(stat.leftSize))
		}
	}
	if hasRight {
		// output stat columns for right side stats
		header = append(header, "R:Count", "R:Size")
		for i, stat := range allStats {
			out[i] = append(out[i], self.formatNumber(stat.rightCount))
			out[i] = append(out[i], self.formatNumber(stat.rightSize))
		}
	}
	out = append([][]string{header}, out...) // insert header at top of output

	// calculate max width of each stats column
	widths := make([]int, len(header))
	for _, line := range out {
		for i, s := range line {
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}
	// pad each item to max width in its column
	for _, line := range out {
		for i, s := range line {
			line[i] = fmt.Sprintf("%*s", widths[i], s)
		}
	}
	return out
}
