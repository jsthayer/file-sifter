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
	"sort"
	"strings"
	"testing"
)

func Test_entrySorter(t *testing.T) {
	paths := []string{"baz", "foo", "foo", "bar"}
	sizes := []int64{10, 5, 4, 20}
	entries := []fileEntry{}
	for i, path := range paths {
		entry := newFileEntry()
		entry.setStringField(ColPath, path)
		entry.setNumericField(ColSize, sizes[i])
		entries = append(entries, entry)
	}
	// TODO: test nulls
	sorter := newEntrySorter(nil, entries, []Column{ColPath, ColSize})
	sort.Sort(sorter)
	sortedPaths := []string{"bar", "baz", "foo", "foo"}
	sortedSizes := []int64{20, 10, 4, 5}
	for i, entry := range entries {
		path, _ := entry.getStringField(ColPath)
		size, _ := entry.getNumericField(ColSize)
		if path != sortedPaths[i] || size != sortedSizes[i] {
			t.Error("Sort order did not match")
		}
	}
}

func Test_modeStrToFileType(t *testing.T) {
	var tests = []struct {
		input string
		want  string
	}{
		{"c---", "c"},
		{"D---", "b"},
		{"p---", "p"},
		{"L---", "L"},
		{"d---", "d"},
		{"Ld---", "L"},
		{"S---", "S"},
		{"----", "f"},
	}
	for _, test := range tests {
		checkVal(t, test.want, modeStrToFileType(test.input))
	}
}

func Test_fileEntry_setStringField(t *testing.T) {
	var tests = []struct {
		cols []Column
		vals []string
		want fileEntry
	}{
		{[]Column{}, []string{}, fileEntry{}},
		{[]Column{ColPath}, []string{"foo"}, fileEntry{ColPath: "foo"}},
		{[]Column{ColPath, ColSize}, []string{"foo", "bar"}, fileEntry{ColPath: "foo", ColSize: "bar"}},
	}
	for _, test := range tests {
		entry := newFileEntry()
		for i, col := range test.cols {
			entry.setStringField(col, test.vals[i])
		}
		checkVal(t, test.want, entry)
	}
}

func Test_fileEntry_getStringField(t *testing.T) {
	var tests = []struct {
		entry   fileEntry
		col     Column
		wantVal string
		wantOk  bool
	}{
		{fileEntry{ColPath: "foo"}, ColPath, "foo", true},
		{fileEntry{ColPath: "foo"}, ColMtime, "", false},
		{fileEntry{ColPath: ""}, ColPath, "", true},
		{fileEntry{ColPath: "foo/bar.x"}, ColBase, "bar.x", true},
		{fileEntry{ColPath: "foo/bar.x"}, ColDir, "foo", true},
		{fileEntry{ColPath: "foo/bar.x"}, ColExt, ".x", true},
		{fileEntry{ColBase: "bar.x"}, ColExt, ".x", true},
		{fileEntry{ColModestr: "L-----"}, ColFileType, "L", true},
		{fileEntry{ColMstamp: int64(1484707710)}, ColMtime, "2017-01-18T02:48:30Z", true},
		{fileEntry{ColSide: int64(1)}, ColMembership, "", false},
		{fileEntry{ColSide: int64(1), ColMatched: int64(1)}, ColMembership, ">=", true},
		{fileEntry{ColSide: int64(1), ColMatched: int64(0)}, ColMembership, ">!", true},
		{fileEntry{ColSide: int64(0), ColMatched: int64(1)}, ColMembership, "<=", true},
		{fileEntry{ColSide: int64(0), ColMatched: int64(0)}, ColMembership, "<!", true},
	}
	for _, test := range tests {
		got, ok := test.entry.getStringField(test.col)
		checkVal(t, test.wantOk, ok)
		checkVal(t, test.wantVal, got)
	}
}

func Test_fileEntry_setBoolField(t *testing.T) {
	var tests = []struct {
		cols []Column
		vals []bool
		want fileEntry
	}{
		{[]Column{ColMatched}, []bool{true}, fileEntry{ColMatched: int64(1)}},
		{[]Column{ColMatched}, []bool{false}, fileEntry{ColMatched: int64(0)}},
	}
	for _, test := range tests {
		entry := newFileEntry()
		for i, col := range test.cols {
			entry.setBoolField(col, test.vals[i])
		}
		checkVal(t, test.want, entry)
	}
}

func Test_fileEntry_getBoolField(t *testing.T) {
	var tests = []struct {
		entry fileEntry
		col   Column
		want  [2]bool
	}{
		{fileEntry{ColMatched: int64(0)}, ColMatched, [2]bool{false, true}},
		{fileEntry{ColMatched: int64(1)}, ColMatched, [2]bool{true, true}},
		{fileEntry{ColMatched: int64(42)}, ColMatched, [2]bool{true, true}},
		{fileEntry{}, ColMatched, [2]bool{false, false}},
	}
	for _, test := range tests {
		val, ok := test.entry.getBoolField(test.col)
		got := [2]bool{val, ok}
		checkVal(t, test.want, got)
	}
}

func Test_fileEntry_setNumericField(t *testing.T) {
	var tests = []struct {
		cols []Column
		vals []int64
		want fileEntry
	}{
		{[]Column{ColSize}, []int64{123}, fileEntry{ColSize: int64(123)}},
	}
	for _, test := range tests {
		entry := newFileEntry()
		for i, col := range test.cols {
			entry.setNumericField(col, test.vals[i])
		}
		checkVal(t, test.want, entry)
	}
}

func Test_fileEntry_getNumericField(t *testing.T) {
	type i64Bool struct {
		i int64
		b bool
	}
	var tests = []struct {
		entry     fileEntry
		col       Column
		want      i64Bool
		wantPanic bool
	}{
		{fileEntry{ColSize: int64(123)}, ColSize, i64Bool{123, true}, false},
		{fileEntry{}, ColSize, i64Bool{0, false}, false},
		{fileEntry{ColSize: "foo"}, ColSize, i64Bool{}, true},
		{fileEntry{ColPath: "foo/bar/baz"}, ColDepth, i64Bool{2, true}, false},
		{fileEntry{ColMtime: "2017-01-18T02:48:30Z"}, ColMstamp, i64Bool{1484707710, true}, false},
	}
	for _, test := range tests {
		panicked := false
		var val int64
		var ok bool
		func() {
			defer func() {
				panicked = recover() != nil
			}()
			val, ok = test.entry.getNumericField(test.col)
		}()
		got := i64Bool{val, ok}
		if panicked != test.wantPanic {
			t.Errorf("Wanted panic? %v, Got panic? %v", test.wantPanic, panicked)
		}
		checkVal(t, test.want, got)
	}
}

func Test_fileEntry_compare(t *testing.T) {
	f1 := fileEntry{ColSize: int64(100), ColPath: "a"}
	f2 := fileEntry{ColSize: int64(200), ColPath: "b"}
	f3 := fileEntry{ColSize: int64(100), ColPath: "b", ColMtime: "x"}
	var tests = []struct {
		in1    fileEntry
		in2    fileEntry
		cols   []Column
		want   int
		wantOk bool
	}{
		{f2, f1, []Column{ColPath}, 1, true},
		{f1, f2, []Column{ColPath}, -1, true},
		{f1, f1, []Column{ColPath}, 0, true},
		{f2, f1, []Column{ColSize}, 1, true},
		{f1, f2, []Column{ColSize}, -1, true},
		{f1, f3, []Column{ColSize}, 0, true},
		{f1, f3, []Column{ColSize, ColPath}, -1, true},
		{f1, f3, []Column{ColSize, ColSize}, 0, true},
		{f1, f3, []Column{ColMtime}, -1, false},
		{f3, f1, []Column{ColMtime}, 1, false},
		{f1, f1, []Column{ColMtime}, 0, false},
	}
	for _, test := range tests {
		got, ok := test.in1.compare(test.in2, test.cols)
		checkVal(t, test.want, got)
		checkVal(t, test.wantOk, ok)
	}
}

func Test_unescapeField(t *testing.T) {
	type stringBool struct {
		string
		bool
	}
	var tests = []struct {
		input    string
		want     stringBool
		wantLast stringBool
	}{
		{"abc", stringBool{"abc", true}, stringBool{"abc", true}},               // no escapes
		{`\~`, stringBool{"", false}, stringBool{"", false}},                    // null
		{`\-`, stringBool{"", true}, stringBool{"", true}},                      // empty
		{`a\ b`, stringBool{"a b", true}, stringBool{`a\ b`, true}},             // space
		{`a\\b`, stringBool{`a\b`, true}, stringBool{`a\\b`, true}},             // backslash
		{`\~x`, stringBool{`~x`, true}, stringBool{`\~x`, true}},                // tilde (not really used)
		{`\ab\c\d\\`, stringBool{`abcd\`, true}, stringBool{`\ab\c\d\\`, true}}, // multi
	}
	for _, test := range tests {
		val, notNull := unescapeField(test.input, false)
		got := stringBool{val, notNull}
		checkVal(t, test.want, got)
		val, notNull = unescapeField(test.input, true)
		got = stringBool{val, notNull}
		checkVal(t, test.wantLast, got)
	}
}

func Test_escapeField(t *testing.T) {
	var tests = []struct {
		input   string
		notNull bool
		lastCol bool
		want    string
	}{
		{"", true, false, `\-`},                // empty
		{"", true, true, `\-`},                 // empty last
		{"", false, false, `\~`},               // null
		{"", false, true, `\~`},                // null last
		{"abc", true, false, "abc"},            // normal
		{`a\b`, true, false, `a\\b`},           // backslash
		{`a\b`, true, true, `a\\b`},            // backslash last
		{`a b`, true, false, `a\ b`},           // space
		{`a b`, true, true, `a b`},             // space last
		{`a b\cd `, true, false, `a\ b\\cd\ `}, // multi
	}
	for _, test := range tests {
		got := escapeField(test.input, test.notNull, test.lastCol)
		checkVal(t, test.want, got)
	}
}

func Test_fileEntry_parseAndSetField(t *testing.T) {
	var tests = []struct {
		col         Column
		input       string
		want        interface{}
		wantErr     string
		wantNotNull bool
	}{
		{ColPath, "foo", "foo", "", true},
		{ColSize, "42", int64(42), "", true},
		{ColSize, "xx", nil, "strconv.ParseInt", false},
		{ColMatched, "1", nil, "", false},
	}
	ctx := NewContext()
	for _, test := range tests {
		entry := newFileEntry()
		err := entry.parseAndSetField(ctx, test.col, test.input)
		got, notNull := entry[test.col]
		checkValErr1(t, test.want, got, test.wantErr, err)
		checkVal(t, test.wantNotNull, notNull)
	}
}

func Test_fileEntry_formatField(t *testing.T) {
	var tests = []struct {
		col   Column
		width int
		flags string
		entry fileEntry
		want  string
	}{
		{ColPath, -1, "", fileEntry{}, `\~`},                                // null string
		{ColPath, -1, "L", fileEntry{}, `\~`},                               // null string
		{ColSize, -1, "", fileEntry{}, `\~`},                                // null numeric
		{ColPath, -1, "", fileEntry{ColPath: ``}, `\-`},                     // empty string
		{ColPath, -1, "L", fileEntry{ColPath: ``}, `\-`},                    // empty string
		{ColMatched, -1, "", fileEntry{}, `\~`},                             // null bool
		{ColPath, -1, "", fileEntry{ColPath: `a b`}, `a\ b`},                // normal
		{ColPath, -1, "L", fileEntry{ColPath: `a b`}, `a b`},                // normal, last col
		{ColPath, 6, "", fileEntry{ColPath: `a b`}, `a\ b  `},               // normal, padded
		{ColPath, 6, "L", fileEntry{ColPath: `a b`}, `a b   `},              // normal, padded, last col
		{ColMatched, -1, "", fileEntry{ColMatched: int64(1)}, "1"},          // bool
		{ColMatched, -1, "", fileEntry{ColMatched: int64(0)}, "0"},          // bool
		{ColSize, -1, "", fileEntry{ColSize: int64(42)}, `42`},              // numeric
		{ColSize, 6, "", fileEntry{ColSize: int64(42)}, `    42`},           // numeric, padded
		{ColSize, -1, "G", fileEntry{ColSize: int64(42)}, `42`},             // numeric, grouped
		{ColSize, -1, "G", fileEntry{ColSize: int64(123)}, `123`},           // numeric, grouped
		{ColSize, -1, "G", fileEntry{ColSize: int64(1234)}, `1,234`},        // numeric, grouped
		{ColSize, -1, "G", fileEntry{ColSize: int64(1234567)}, `1,234,567`}, // numeric, grouped
		{ColSize, -1, "", fileEntry{ColSize: int64(1234)}, `1234`},          // numeric, ungrouped
		{ColSize, -1, "", fileEntry{ColSize: int64(1234567)}, `1234567`},    // numeric, ungrouped
	}
	for _, test := range tests {
		last := strings.Contains(test.flags, "L")
		group := strings.Contains(test.flags, "G")
		ctx := Context{GroupNumerics: group}
		got := test.entry.formatField(&ctx, test.col, test.width, last)
		checkVal(t, test.want, got)
	}
}

func Test_fileEntry_toJson(t *testing.T) {
	fe := fileEntry{ColSize: int64(100), ColPath: "a"}
	cols := []Column{ColSize, ColPath, ColMd5, ColRedundancy}
	got, err := fe.toJson(cols)
	want := `{
        "md5": null,
        "path": "a",
        "redundancy": null,
        "size": 100
    }`
	checkValErr1(t, want, string(got), "", err)
}
