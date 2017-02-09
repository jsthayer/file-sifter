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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Context_loadSifterFile(t *testing.T) {
	var tests = []struct {
		input        string
		needsSideCol bool
		wantEntries  []fileEntry
		wantErr      string
		wantErrMsgs  []string
	}{
		{``, false, nil, "", nil},
		{`foo`, false, nil, "No column names", nil},
		{`| Columns: w`, false, nil, "Bad column name", nil},
		{
			`| foo
| Columns: size,path
  123 path
  456
  \~  path3
  789 \-
  bad path2`, false,
			[]fileEntry{
				{ColSize: int64(123), ColPath: "path"},
				{},
				{ColPath: `path3`},
				{ColSize: int64(789), ColPath: ""},
			},
			"",
			[]string{`Error: Could not find delimiter in FSIFT file`,
				`Error: Parse error in FSIFT file: strconv.ParseInt: parsing "bad": invalid syntax`}},
		{
			`| Columns: path,size
  pa\ t\\h 123
  path2    bad`, true,
			[]fileEntry{
				{ColSize: int64(123), ColPath: `pa t\h`, ColSide: int64(0)},
			},
			"",
			[]string{`Error: Parse error in FSIFT file: strconv.ParseInt: parsing "bad": invalid syntax`}},
	}
	for _, test := range tests {
		ctx := NewContext()
		if test.needsSideCol {
			ctx.neededCols[ColSide] = true
		}
		got := ctx.loadSifterFile(strings.NewReader(test.input))
		checkValErr1(t, test.wantEntries, ctx.entries, test.wantErr, got)
		checkVal(t, test.wantErrMsgs, ctx.errorMessages)
	}
}

func Test_Context_formatNumber(t *testing.T) {
	var tests = []struct {
		input int64
		want  string
		wantG string
	}{
		{0, "0", "0"},
		{42, "42", "42"},
		{-42, "-42", "-42"},
		{1234, "1234", "1,234"},
		{-1234, "-1234", "-1,234"},
		{231234567890123, "231234567890123", "231,234,567,890,123"},
		{-231234567890123, "-231234567890123", "-231,234,567,890,123"},
	}
	for _, test := range tests {
		ctx := NewContext()
		got := ctx.formatNumber(test.input)
		checkVal(t, test.want, got)
		ctx.GroupNumerics = true
		got = ctx.formatNumber(test.input)
		checkVal(t, test.wantG, got)
	}
}

func Test_detectSifterFile(t *testing.T) {
	got, err := detectSifterFile("noexist/9161ffff-b5ed-41f8-8205-5aa6f9e6e05a")
	checkValErr1(t, false, got, "open noexist/9161ffff-b5ed-41f8-8205-5aa6f9e6e05a", err)

	var tests = []struct {
		input   string
		want    bool
		wantErr string
	}{
		{"", false, ""},                                       // empty
		{"foo", false, ""},                                    // short header
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false, ""}, // mismatched header
		{sifterFileHeader, true, ""},                          // matches
	}
	for _, test := range tests {
		f, err := ioutil.TempFile("", "sifter_unittest_")
		if err != nil {
			t.Error("Couln't create temp file for unit test")
			continue
		}
		defer func() { os.Remove(f.Name()) }()
		_, err = f.WriteString(test.input)
		if err != nil {
			t.Error("Couln't write to temp file for unit test")
			continue
		}
		got, err := detectSifterFile(f.Name())
		checkValErr1(t, test.want, got, test.wantErr, err)
	}
}

func Test_myJoin(t *testing.T) {
	var tests = []struct {
		input  []string
		want   string
		volume bool
	}{
		{[]string{""}, "", false},
		{[]string{"/"}, "/", false},
		{[]string{"a", "b"}, "a/b", false},
		{[]string{"a", fmt.Sprintf("b%cc", os.PathSeparator)}, "a/b/c", false},
		{[]string{"c:a", "b"}, "c:a/b", true},
		{[]string{`c:\`, "b"}, "c:/b", true},
		{[]string{`c:\a`, "b"}, "c:/a/b", true},
		{[]string{`\\foo\bar\a`, "b"}, "//foo/bar/a/b", true},
	}
	for _, test := range tests {
		if test.volume && filepath.VolumeName(test.input[0]) == "" {
			continue // this test only works on Windows
		}
		checkVal(t, test.want, myJoin(test.input...))
	}
}

func Test_Context_processFile(t *testing.T) {
	ctx := NewContext()
	ctx.CurSide = true
	// nonexistent file
	got, gotSize := ctx.processFile("", "noexist-9161ffff-b5ed-41f8-8205-5aa6f9e6e05a")
	checkVal(t, fileEntry(nil), got)
	checkVal(t, int64(0), gotSize)
	if !strings.HasPrefix(ctx.errorMessages[0], "Error: Can't get info about file: ") {
		t.Error("Expected file error")
	}

	// set context to get all file info
	cols := []Column{ColPath, ColSize, ColMtime, ColMstamp, ColSide, ColDevice, ColNlinks, ColUid, ColGid, ColModestr, ColFileType}
	var err error
	ctx.preFilter, err = ParseFilter("path!*=**nomatch*")
	if err != nil {
		panic("Bad filter")
	}
	for _, col := range cols {
		ctx.neededCols[col] = true
	}

	// process temp directory
	dirPath, err := ioutil.TempDir("", "sifter_unittest_")
	if err != nil {
		t.Error("Couln't create temp dir for unit test")
		return
	}
	defer func() { os.Remove(dirPath) }()
	got, gotSize = ctx.processFile("", dirPath)
	for _, col := range cols {
		_, notNull := got[col]
		checkVal(t, true, notNull)
	}
	checkVal(t, int64(0), got[ColSize])
	checkVal(t, int64(0), gotSize)
	checkVal(t, "d", got[ColFileType])
	checkVal(t, true, strings.HasSuffix(got[ColPath].(string), "/"))

	// create temp file and process it
	f1, err := ioutil.TempFile("", "sifter_unittest_")
	if err != nil {
		t.Error("Couln't create temp file for unit test")
		return
	}
	defer func() { os.Remove(f1.Name()) }()
	_, err = f1.WriteString("foo")
	if err != nil {
		t.Error("Couln't write to temp file for unit test")
		return
	}
	f1.Close()
	got, gotSize = ctx.processFile("", f1.Name())
	for _, col := range cols {
		_, notNull := got[col]
		checkVal(t, true, notNull)
	}
	checkVal(t, int64(3), got[ColSize])
	checkVal(t, int64(3), gotSize)
	checkVal(t, "f", got[ColFileType])

	// create temp file that gets rejected by prefilter
	f2, err := ioutil.TempFile("", "sifter_unittest_nomatch_")
	if err != nil {
		t.Error("Couln't create temp file for unit test")
		return
	}
	defer func() { os.Remove(f2.Name()) }()
	_, err = f2.WriteString("foo")
	if err != nil {
		t.Error("Couln't write to temp file for unit test")
		return
	}
	f2.Close()
	got, gotSize = ctx.processFile("", f2.Name())
	checkVal(t, fileEntry(nil), got)
	checkVal(t, int64(0), gotSize)

	// check final stats
	checkVal(t, int64(3), ctx.scanStats.rightCount)
	checkVal(t, int64(6), ctx.scanStats.rightSize)
	checkVal(t, int64(2), ctx.indexStats.rightCount)
	checkVal(t, int64(3), ctx.indexStats.rightSize)
}

func Test_Context_calcDigestList(t *testing.T) {
	ctx := NewContext()
	cols := []Column{ColCrc32, ColMd5, ColSha1, ColSha256, ColSha512}
	for _, col := range cols {
		ctx.neededCols[col] = true
	}
	ed1 := fileEntry{ColPath: os.TempDir()}

	// create a test file to digest
	f1, err := ioutil.TempFile("", "sifter_unittest_")
	if err != nil {
		t.Error("Couln't create temp file for unit test")
		return
	}
	defer func() { os.Remove(f1.Name()) }()
	_, err = f1.WriteString("foo")
	if err != nil {
		t.Error("Couln't write to temp file for unit test")
		return
	}
	f1.Close()
	ef1 := fileEntry{ColPath: f1.Name()}

	entries := []fileEntry{ed1, ef1}

	ctx.calcDigestList("", entries)
	wantD1 := fileEntry{
		ColPath:   ed1[ColPath],
		ColCrc32:  "",
		ColMd5:    "",
		ColSha1:   "",
		ColSha256: "",
		ColSha512: "",
	}
	checkVal(t, wantD1, ed1)
	wantF1 := fileEntry{
		ColPath:   ef1[ColPath],
		ColCrc32:  "8c736521",
		ColMd5:    "acbd18db4cc2f85cedef654fccc4a4d8",
		ColSha1:   "0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33",
		ColSha256: "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
		ColSha512: "f7fbba6e0636f890e56fbbf3283e524c6fa3204ae298382d624741d0dc6638326e282c41be5e4254d8820772c5518a2c5a8c0c7f7eda19594a7eb539453e1ed7",
	}
	checkVal(t, wantF1, ef1)
}
