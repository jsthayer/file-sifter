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

func Test_GlobToRegex(t *testing.T) {
	var tests = []struct {
		input     string
		want      string
		errPrefix string
	}{
		{``, `^$`, ``},                                           // empty
		{`a`, `^a$`, ``},                                         // one char
		{`**`, `^.*$`, ``},                                       // any including slash
		{`*`, `^[^/]*$`, ``},                                     // any excluding slash
		{`?`, `^[^/]$`, ``},                                      // any char excluding slash
		{`\?`, `^\?$`, ``},                                       // escaped ?
		{`\*`, `^\*$`, ``},                                       // escaped star
		{`\[`, `^\[$`, ``},                                       // escaped [
		{`[a*b]`, `^[a*b]$`, ``},                                 // char class
		{`[^a-c]`, `^[^a-c]$`, ``},                               // "
		{`[]^a-c]`, `^[]^a-c]$`, ``},                             // "
		{`[]]`, `^[]]$`, ``},                                     // "
		{`[^]]`, `^[^]]$`, ``},                                   // "
		{`.`, `^\.$`, ``},                                        // escaped .
		{`**/*ab?x.de[^1-2]`, `^.*/[^/]*ab[^/]x\.de[^1-2]$`, ``}, // complex
		{`[c-a]`, ``, `error parsing regexp`},                    // parse error
	}
	for _, test := range tests {
		re, err := GlobToRegex(test.input)
		got := ""
		if re != nil {
			got = re.String()
		}
		checkValErr1(t, test.want, got, test.errPrefix, err)
	}
}

func Test_ParseFilter(t *testing.T) {
	var tests = []struct {
		input     string
		want      *Filter
		wantRegex string
		errPrefix string
	}{
		{"and", &Filter{op: opAnd}, "", ""},                                            // and
		{"or", &Filter{op: opOr}, "", ""},                                              // or
		{"xx", nil, "", "Bad filter argument"},                                         // parse err
		{"z=a", nil, "", "Bad column name in filter"},                                  // bad col
		{"p=a", &Filter{op: opEq, value: "a", column: ColPath}, "", ""},                // equal string
		{"s=3", &Filter{op: opEq, value: int64(3), column: ColSize}, "", ""},           // equal num
		{"s=w", nil, "", "strconv.ParseInt"},                                           // equal bad num
		{"p!=a", &Filter{op: opEq, value: "a", column: ColPath, not: true}, "", ""},    // not equal
		{"p<a", &Filter{op: opLess, value: "a", column: ColPath}, "", ""},              // less
		{"p<=a", &Filter{op: opLessEq, value: "a", column: ColPath}, "", ""},           // less-equal
		{"p>a", &Filter{op: opLessEq, value: "a", column: ColPath, not: true}, "", ""}, // greater
		{"p>=a", &Filter{op: opLess, value: "a", column: ColPath, not: true}, "", ""},  // greater-equal
		{"p~=a", &Filter{op: opRegex, value: "a", column: ColPath}, "a", ""},           // regex match
		{"p~=a****", nil, "", "error parsing regexp"},                                  // bad regex
		{"p*=a", &Filter{op: opGlob, value: "a", column: ColPath}, "^a$", ""},          // glob match
		{"p.isnull", &Filter{op: opIsNull, value: "", column: ColPath}, "", ""},        // is null op
	}
	for _, test := range tests {
		got, err := ParseFilter(test.input)
		regex := ""
		if test.want != nil && got != nil {
			if got.regex != nil {
				regex = got.regex.String()
				got.regex = nil
			}
			checkValErr1(t, *test.want, *got, test.errPrefix, err)
			checkValErr1(t, test.wantRegex, regex, "", nil)
		} else {
			checkValErr1(t, test.want == nil, got == nil, test.errPrefix, err)
		}
	}
}

func Test_compileFilter(t *testing.T) {
	var tests = []struct {
		args      []string
		want      *Filter
		errPrefix string
	}{
		// empty
		{nil, nil, ""},
		// one filter
		{[]string{"p=a"}, &Filter{op: opEq, value: "a", column: ColPath}, ""},
		// two filters; implicit AND
		{[]string{"p=a", "p=b"}, &Filter{op: opAnd,
			left:  &Filter{op: opEq, value: "a", column: ColPath},
			right: &Filter{op: opEq, value: "b", column: ColPath}}, ""},
		// OR filter
		{[]string{"or", "p=a", "p=b"}, &Filter{op: opOr,
			left:  &Filter{op: opEq, value: "a", column: ColPath},
			right: &Filter{op: opEq, value: "b", column: ColPath}}, ""},
		// OR plus implicit AND
		{[]string{"or", "p=a", "p=b", "p=c"}, &Filter{op: opAnd, left: &Filter{op: opOr,
			left:  &Filter{op: opEq, value: "a", column: ColPath},
			right: &Filter{op: opEq, value: "b", column: ColPath}},
			right: &Filter{op: opEq, value: "c", column: ColPath}}, ""},
		// OR, insufficient args
		{[]string{"or", "p=a"}, nil, "Filter expression"},
	}
	for _, test := range tests {
		filts := []*Filter{}
		for _, a := range test.args {
			filt, _ := ParseFilter(a)
			filts = append(filts, filt)
		}
		got, err := compileFilter(filts)
		checkValErr1(t, test.want, got, test.errPrefix, err)
	}
}

func Test_Filter_filter(t *testing.T) {
	dropErr := func(f *Filter, e error) *Filter { return f }
	yes, _ := ParseFilter("p=a")
	no, _ := ParseFilter("p=b")
	var tests = []struct {
		filt  *Filter
		entry fileEntry
		want  [2]bool
	}{
		// string tests
		{nil, fileEntry{ColPath: "a"}, [2]bool{true, true}},                               // empty filter
		{dropErr(ParseFilter("p=a")), fileEntry{ColPath: "a"}, [2]bool{true, true}},       // = match
		{dropErr(ParseFilter("p!=a")), fileEntry{ColPath: "a"}, [2]bool{false, true}},     // != nonmatch
		{dropErr(ParseFilter("p=a")), fileEntry{}, [2]bool{false, false}},                 // = null
		{dropErr(ParseFilter("p=a")), fileEntry{ColPath: "b"}, [2]bool{false, true}},      // = nonmatch
		{dropErr(ParseFilter("p<a")), fileEntry{ColPath: "b"}, [2]bool{false, true}},      // < nonmatch
		{dropErr(ParseFilter("p<a")), fileEntry{ColPath: "a"}, [2]bool{false, true}},      // < nonmatch
		{dropErr(ParseFilter("p<b")), fileEntry{ColPath: "a"}, [2]bool{true, true}},       // < match
		{dropErr(ParseFilter("p<=a")), fileEntry{ColPath: "b"}, [2]bool{false, true}},     // <= nonmatch
		{dropErr(ParseFilter("p<=a")), fileEntry{ColPath: "a"}, [2]bool{true, true}},      // <= match
		{dropErr(ParseFilter("p<=b")), fileEntry{ColPath: "a"}, [2]bool{true, true}},      // <= match
		{dropErr(ParseFilter("p*=*b")), fileEntry{ColPath: "ab"}, [2]bool{true, true}},    // glob match
		{dropErr(ParseFilter("p!*=*b")), fileEntry{ColPath: "ab"}, [2]bool{false, true}},  // !glob nonmatch
		{dropErr(ParseFilter("p*=*b")), fileEntry{ColPath: "ac"}, [2]bool{false, true}},   // glob nonmatch
		{dropErr(ParseFilter("p~=.b")), fileEntry{ColPath: "ab"}, [2]bool{true, true}},    // regex match
		{dropErr(ParseFilter("p!~=.b")), fileEntry{ColPath: "ab"}, [2]bool{false, true}},  // !regex nonmatch
		{dropErr(ParseFilter("p~=.b")), fileEntry{ColPath: "ac"}, [2]bool{false, true}},   // regex nonmatch
		{dropErr(ParseFilter("p~=.b")), fileEntry{}, [2]bool{false, false}},               // regex null
		{dropErr(ParseFilter("p.isnull")), fileEntry{}, [2]bool{true, true}},              // isnull null
		{dropErr(ParseFilter("p.isnull")), fileEntry{ColPath: "a"}, [2]bool{false, true}}, // isnull notnull
		{dropErr(ParseFilter("p!.isnull")), fileEntry{}, [2]bool{false, true}},            // !isnull null
		{dropErr(ParseFilter("p!.isnull")), fileEntry{ColPath: "a"}, [2]bool{true, true}}, // !isnull notnull

		// numeric tests
		{dropErr(ParseFilter("s=5")), fileEntry{ColSize: int64(5)}, [2]bool{true, true}},       // = match
		{dropErr(ParseFilter("s!=5")), fileEntry{ColSize: int64(5)}, [2]bool{false, true}},     // != nonmatch
		{dropErr(ParseFilter("s=5")), fileEntry{}, [2]bool{false, false}},                      // = null
		{dropErr(ParseFilter("s=5")), fileEntry{ColSize: int64(8)}, [2]bool{false, true}},      // = nonmatch
		{dropErr(ParseFilter("s<5")), fileEntry{ColSize: int64(8)}, [2]bool{false, true}},      // < nonmatch
		{dropErr(ParseFilter("s<5")), fileEntry{ColSize: int64(5)}, [2]bool{false, true}},      // < nonmatch
		{dropErr(ParseFilter("s<8")), fileEntry{ColSize: int64(5)}, [2]bool{true, true}},       // < match
		{dropErr(ParseFilter("s<=5")), fileEntry{ColSize: int64(8)}, [2]bool{false, true}},     // <= nonmatch
		{dropErr(ParseFilter("s<=5")), fileEntry{ColSize: int64(5)}, [2]bool{true, true}},      // <= match
		{dropErr(ParseFilter("s<=8")), fileEntry{ColSize: int64(5)}, [2]bool{true, true}},      // <= match
		{dropErr(ParseFilter("s*=*8")), fileEntry{ColSize: int64(18)}, [2]bool{true, true}},    // glob match
		{dropErr(ParseFilter("s*=*8")), fileEntry{ColSize: int64(15)}, [2]bool{false, true}},   // glob nonmatch
		{dropErr(ParseFilter("s~=.8")), fileEntry{ColSize: int64(18)}, [2]bool{true, true}},    // regex match
		{dropErr(ParseFilter("s~=.8")), fileEntry{ColSize: int64(15)}, [2]bool{false, true}},   // regex nonmatch
		{dropErr(ParseFilter("s~=.8")), fileEntry{}, [2]bool{false, false}},                    // regex null
		{dropErr(ParseFilter("s.isnull")), fileEntry{}, [2]bool{true, true}},                   // isnull null
		{dropErr(ParseFilter("s.isnull")), fileEntry{ColSize: int64(3)}, [2]bool{false, true}}, // isnull notnull
		{dropErr(ParseFilter("s!.isnull")), fileEntry{}, [2]bool{false, true}},                 // !isnull null
		{dropErr(ParseFilter("s!.isnull")), fileEntry{ColSize: int64(3)}, [2]bool{true, true}}, // !isnull notnull

		// and/or filters
		{&Filter{op: opAnd, left: yes, right: yes}, fileEntry{ColPath: "a"}, [2]bool{true, true}}, // and 1 1
		{&Filter{op: opAnd, left: yes, right: no}, fileEntry{ColPath: "a"}, [2]bool{false, true}}, // and 1 0
		{&Filter{op: opAnd, left: no, right: no}, fileEntry{ColPath: "a"}, [2]bool{false, true}},  // and 0 0
		{&Filter{op: opOr, left: yes, right: yes}, fileEntry{ColPath: "a"}, [2]bool{true, true}},  // or 1 1
		{&Filter{op: opOr, left: yes, right: no}, fileEntry{ColPath: "a"}, [2]bool{true, true}},   // or 1 0
		{&Filter{op: opOr, left: no, right: no}, fileEntry{ColPath: "a"}, [2]bool{false, true}},   // or 0 0
	}
	for _, test := range tests {
		var got [2]bool
		got[0], got[1] = test.filt.filter(test.entry)
		checkValErr1(t, test.want, got, "", nil)
	}
}
