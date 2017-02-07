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
	"io/ioutil"
	"testing"
)

func Test_stats_update(t *testing.T) {
	var tests = []struct {
		size int64
		side bool
		want stats
	}{
		{3, false, stats{"", 1, 3, 0, 0}},
		{5, true, stats{"", 1, 3, 1, 5}},
		{4, false, stats{"", 2, 7, 1, 5}},
	}
	st := stats{}
	for _, test := range tests {
		st.update(test.side, test.size)
		checkVal(t, test.want, st)
	}
}

func Test_Context_calcNeededCols(t *testing.T) {
	ctx := NewContext()
	ctx.OutCols = ColSelector{cols: []Column{ColPath}}
	ctx.SortCols = ColSelector{cols: []Column{ColSize}}
	ctx.KeyCols = ColSelector{cols: []Column{ColMtime}}
	ctx.PreFilterArgs = []*Filter{{column: ColMd5}}
	ctx.PostFilterArgs = []*Filter{{column: ColSha256}, {column: ColSha1}}
	ctx.calcNeededCols()
	want := map[Column]bool{ColPath: true, ColSize: true, ColMtime: true, ColMd5: true, ColSha256: true, ColSha1: true}
	checkVal(t, want, ctx.neededCols)
}

func Test_Context_needsCol(t *testing.T) {
	ctx := NewContext()
	ctx.neededCols = map[Column]bool{ColPath: true}
	checkVal(t, true, ctx.needsCol(ColPath))
	checkVal(t, false, ctx.needsCol(ColSize))
}

func Test_Context_analyzeMatches(t *testing.T) {
	ctx := NewContext()
	ctx.outputState.errStream = ioutil.Discard
	ctx.neededCols = map[Column]bool{ColRedundancy: true, ColRedunIdx: true}
	ctx.Verify = true
	ctx.KeyCols = ColSelector{cols: []Column{ColPath}}
	ctx.entries = []fileEntry{
		{ColPath: "c", ColSize: int64(8), ColSide: int64(0)},
		{ColPath: "d/", ColSize: int64(15), ColSide: int64(0)},
		{ColPath: "b", ColSize: int64(5), ColSide: int64(1)},
		{ColPath: "a", ColSize: int64(2), ColSide: int64(1)},
		{ColPath: "a", ColSize: int64(2), ColSide: int64(0)},
		{ColPath: "a", ColSize: int64(9), ColSide: int64(0)},
	}
	ctx.analyzeMatches()
	checkVal(t, int64(0), ctx.entries[0][ColMatched])
	checkVal(t, int64(0), ctx.entries[1][ColMatched])
	checkVal(t, int64(0), ctx.entries[2][ColMatched])
	checkVal(t, int64(1), ctx.entries[3][ColMatched])
	checkVal(t, int64(1), ctx.entries[4][ColMatched])
	checkVal(t, int64(1), ctx.entries[5][ColMatched])

	checkVal(t, int64(1), ctx.entries[0][ColRedundancy])
	checkVal(t, int64(1), ctx.entries[1][ColRedundancy])
	checkVal(t, int64(1), ctx.entries[2][ColRedundancy])
	checkVal(t, int64(1), ctx.entries[3][ColRedundancy])
	checkVal(t, int64(2), ctx.entries[4][ColRedundancy])
	checkVal(t, int64(2), ctx.entries[5][ColRedundancy])

	checkVal(t, int64(1), ctx.entries[0][ColRedunIdx])
	checkVal(t, int64(1), ctx.entries[1][ColRedunIdx])
	checkVal(t, int64(1), ctx.entries[2][ColRedunIdx])
	checkVal(t, int64(1), ctx.entries[3][ColRedunIdx])
	checkVal(t, int64(1), ctx.entries[4][ColRedunIdx])
	checkVal(t, int64(2), ctx.entries[5][ColRedunIdx])

	checkVal(t, ctx.errorMessages[0], "Error: At least one entry on the left was unmatched (--verify was specified)")
}

func Test_Context_checkNullCompare(t *testing.T) {
	ctx := NewContext()
	checkVal(t, 0, ctx.nullErrorCount)
	ctx.checkNullCompare(true)
	checkVal(t, 0, ctx.nullErrorCount)
	ctx.checkNullCompare(false)
	checkVal(t, 1, ctx.nullErrorCount)
	ctx.IgnoreNullCmps = true
	ctx.checkNullCompare(false)
	checkVal(t, 1, ctx.nullErrorCount)
}

func Test_Context_calcSummaryInfo(t *testing.T) {
	ctx := NewContext()
	ctx.GroupNumerics = true
	ctx.Roots[false] = []string{"r1"}
	ctx.scanStats = stats{"scan", 1, 2, 3, 4}
	ctx.indexStats = stats{"index", 10, 20, 30, 40}
	ctx.unmatchedStats = stats{"unmatched", 100, 200, 300, 400}
	ctx.matchingStats = stats{"matching", 1000, 2000, 3000, 4000}
	ctx.outputStats = stats{"output", 10000, 200000000, 30000, 40000}
	want := [][]string{
		{"STATISTICS:", " Count", "       Size"},
		{"       scan", "     1", "          2"},
		{"      index", "    10", "         20"},
		{"     output", "10,000", "200,000,000"},
	}
	got := ctx.calcSummaryInfo()
	checkVal(t, want, got)

	ctx.Roots[true] = []string{"r2"}
	want = [][]string{
		{"STATISTICS:", "L:Count", "     L:Size", "R:Count", "R:Size"},
		{"       scan", "      1", "          2", "      3", "     4"},
		{"      index", "     10", "         20", "     30", "    40"},
		{"  unmatched", "    100", "        200", "    300", "   400"},
		{"   matching", "  1,000", "      2,000", "  3,000", " 4,000"},
		{"     output", " 10,000", "200,000,000", " 30,000", "40,000"},
	}
	got = ctx.calcSummaryInfo()
	checkVal(t, want, got)
}
