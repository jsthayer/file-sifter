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

import "testing"

func Test_GetColumnHelp(t *testing.T) {
	got := GetColumnHelp()
	checkVal(t, "p path         The path of this file relative to the given root", got[0])
	checkVal(t, ColLAST, len(got))
}

func Test_isNumeric_isDynamic(t *testing.T) {
	var tests = []struct {
		col         Column
		wantNumeric bool
		wantDynamic bool
	}{
		{ColSize, true, false},
		{ColPath, false, false},
		{ColMatched, true, true},
		{99999, false, false}, // invalid col
	}
	for _, test := range tests {
		gotNumeric := test.col.isNumeric()
		gotDynamic := test.col.isDynamic()
		switch {
		case test.wantNumeric != gotNumeric:
			t.Errorf("isNumeric: Got '%v', expected '%v'", gotNumeric, test.wantNumeric)
		case test.wantDynamic != gotDynamic:
			t.Errorf("isDynamic: Got '%v', expected '%v'", gotDynamic, test.wantDynamic)
		}
	}
}

func Test_parseColumnList(t *testing.T) {
	var tests = []struct {
		arg          string
		expect       []Column
		expectErr    string
		allowInverse bool
	}{
		{"", []Column{}, "", false},                                                    // empty
		{"p", []Column{ColPath}, "", false},                                            // single short
		{"stp", []Column{ColSize, ColMtime, ColPath}, "", false},                       // multi short
		{"path", []Column{ColPath}, "", false},                                         // single long
		{"size,mtime,path", []Column{ColSize, ColMtime, ColPath}, "", false},           // multi long
		{"z", nil, "Bad column name", false},                                           // bad short
		{"fooz", nil, "Bad column name", false},                                        // bad long
		{"path,foo", nil, "Bad column name", false},                                    // bad long multi
		{"/path,foo", nil, "This columns list may not contain inverse", false},         // bad long multi
		{"/path,size", []Column{ColPath | ColInvertFlag, ColSize}, "", true},           // inverse
		{"/p/s", []Column{ColPath | ColInvertFlag, ColSize | ColInvertFlag}, "", true}, // inverse
	}
	for _, test := range tests {
		cols, err := ParseColumnsList(test.arg, test.allowInverse)
		checkValErr1(t, test.expect, cols, test.expectErr, err)
	}
}

func Test_parseColumnsDirective(t *testing.T) {
	var tests = []struct {
		input     string
		expect    []Column
		expectErr string
	}{
		{"| foo", nil, ""},                       // different directive
		{"|", nil, ""},                           // empty directive
		{"", nil, ""},                            // empty line
		{" foo ", nil, ""},                       // not a directive
		{"| Columns: z", nil, "Bad column name"}, // bad column
		{"| Columns: p", []Column{ColPath}, ""},  // ok
	}
	for _, test := range tests {
		cols, err := parseColumnsDirective([]byte(test.input))
		checkValErr(t, test.expect, cols, test.expectErr, err)
	}
}

func Test_formatColumnNames(t *testing.T) {
	var tests = []struct {
		input  []Column
		expect string
	}{
		{[]Column{}, ""},
		{[]Column{ColPath}, "path"},
		{[]Column{ColPath, ColSize}, "path,size"},
	}
	for _, test := range tests {
		got := formatColumnNames(test.input)
		checkVal(t, test.expect, got)
	}
}

func Test_containsCol(t *testing.T) {
	cols := []Column{ColPath, ColSize}
	checkVal(t, true, containsCol(cols, ColPath))
	checkVal(t, false, containsCol(cols, ColMtime))
}

func Test_insertCol(t *testing.T) {
	var tests = []struct {
		input  []Column
		index  int
		expect []Column
	}{
		{[]Column{}, 0, []Column{ColMd5}},
		{[]Column{}, -3, []Column{ColMd5}},
		{[]Column{ColPath, ColSize, ColMtime}, 0, []Column{ColMd5, ColPath, ColSize, ColMtime}},
		{[]Column{ColPath, ColSize, ColMtime}, 1, []Column{ColPath, ColMd5, ColSize, ColMtime}},
		{[]Column{ColPath, ColSize, ColMtime}, -1, []Column{ColPath, ColSize, ColMd5, ColMtime}},
		{[]Column{ColPath, ColSize, ColMtime}, -3, []Column{ColMd5, ColPath, ColSize, ColMtime}},
		{[]Column{ColPath, ColSize, ColMtime}, -8, []Column{ColMd5, ColPath, ColSize, ColMtime}},
	}
	for _, test := range tests {
		insertCol(&test.input, test.index, ColMd5)
		checkVal(t, test.expect, test.input)
	}
}
