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
	"regexp"
	"strconv"
	"strings"
)

// The possible filter operations
const (
	opEq     = iota // ==
	opLessEq        // <=
	opLess          // <
	opRegex         // matches regular expression
	opGlob          // matches file glob pattern
	opAnd           // logical AND of child filters
	opOr            // logical OR of child filters
	opIsNull        // field is NULL
)

type filtOp int

// Filter holds the definition of a prefilter or postfilter.
type Filter struct {
	op     filtOp         // the operation this filter performs
	value  interface{}    // the value this filter uses for comparisons, if any
	column Column         // the column this filter operates on
	not    bool           // true to invert results
	left   *Filter        // left child filter
	right  *Filter        // right child filter
	regex  *regexp.Regexp // for glob or regex filters, the pattern
}

// Pattern to parse glob expressions for conversion to regular expressions.
var globPat = regexp.MustCompile(
	`\\\*|\\\?|\\\[` + // \*, \?, \[
		`|\*\*|\*|\?` + //  **, *, ?
		`|\[\^?\][^]]*\]` + // bracketed char class starting with a ']'
		`|\[\^?[^]]+\]`) // normal bracketed char class

// GlobToRegex parses a file glob pattern and return a regular expression object
// that implements the glob pattern. Return error if expression can't
// be parsed.
func GlobToRegex(pat string) (*regexp.Regexp, error) {
	coords := globPat.FindAllStringIndex(pat, -1) // look for relevant glob pieces
	base := 0                                     // current leftmost position in string
	parts := []string{}                           // built up parts of regex string
	for _, coord := range coords {
		// escape and copy unmatched parts of glob pattern
		parts = append(parts, regexp.QuoteMeta(pat[base:coord[0]]))
		op := pat[coord[0]:coord[1]] // extract matched operator

		switch op {
		case "**":
			// match anything including '/'
			parts = append(parts, `.*`)
		case "*":
			// match anything except '/'
			parts = append(parts, `[^/]*`)
		case "?":
			// match one char unless '/'
			parts = append(parts, `[^/]`)
		default:
			// includes [...] along with \*, \? and \[; copy verbatim
			parts = append(parts, op)
		}
		base = coord[1]
	}
	// add last part of string and compile regexp
	parts = append(parts, regexp.QuoteMeta(pat[base:]))
	return regexp.Compile("^" + strings.Join(parts, "") + "$")
}

// Pattern to parse filter arguments (<fieldname> <op> <value>)
var filterArgPat = regexp.MustCompile(`\s*(\w+)\s*(!?~=|!?\*=|>=|<=|>|<|!?=|!?\.isnull)(.*)`)

// ParseFilter parses a filter specification from a command line argument and
// returns a new filter object implementing it, or an error if it couldn't be
// parsed.
func ParseFilter(arg string) (*Filter, error) {
	// AND and OR filters have no other settings, children are added when compiled
	if arg == "and" {
		return &Filter{op: opAnd}, nil
	}
	if arg == "or" {
		return &Filter{op: opOr}, nil
	}

	// Parse the filter expression into its components, look up column name
	parts := filterArgPat.FindStringSubmatch(arg)
	if parts == nil {
		return nil, fmt.Errorf("Bad filter argument: '%s'", arg)
	}

	col, ok := colIndex[parts[1]]
	if !ok {
		return nil, fmt.Errorf("Bad column name in filter: '%s'", parts[1])
	}

	var op filtOp
	var regex *regexp.Regexp
	var err error
	not := strings.HasPrefix(parts[2], "!") // invert sense of filter

	// take action based on filter op
	switch parts[2] {
	case "!~=", "~=":
		// a regular expression filter
		op = opRegex
		regex, err = regexp.Compile(parts[3])
		if err != nil {
			return nil, err
		}
	case "!*=", "*=":
		// a glob expression filter
		op = opGlob
		regex, err = GlobToRegex(parts[3])
		if err != nil {
			return nil, err
		}
	case "!=", "=":
		op = opEq
	case "<":
		op = opLess
	case "<=":
		op = opLessEq
	case ">":
		op = opLessEq
		not = true
	case ">=":
		op = opLess
		not = true
	case "!.isnull", ".isnull":
		op = opIsNull
	default:
		panic("Unexpected filter op") // shouldn't be possible due to regex match
	}

	var value interface{} = parts[3]
	if col.isNumeric() {
		// if numeric comparison, convert the value to a number
		switch op {
		case opEq, opLess, opLessEq:
			ival, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				return nil, err
			}
			value = ival
		}
	}

	// create the filter object and return it
	filt := Filter{
		op:     op,
		value:  value,
		column: col,
		not:    not,
		regex:  regex,
	}
	return &filt, nil
}

// Given a list of filter arguments, "compile" them into a single filter tree.
// Any AND or OR filters are used as-is. AND/OR filters must appear in prefix
// order before their operands (forward Polish notation).  Any remaining
// filters in the list which are not children of AND/OR filters get joined
// together with new auto-generated AND filters (they are implicitly ANDed).
func compileFilter(args []*Filter) (*Filter, error) {
	if len(args) == 0 {
		// nothing to do
		return nil, nil
	}
	// scan list backwards, looking for AND/OR filters
	for i := len(args) - 1; i >= 0; i-- {
		filt := args[i]
		switch filt.op {
		case opAnd, opOr:
			// found one; pull the following two list items as its children
			if i >= len(args)-2 {
				return nil, fmt.Errorf("Filter expression AND/OR op: not enough arguments provided")
			}
			filt.left = args[i+1]
			filt.right = args[i+2]
			// snip the children from the list
			args = append(args[:i+1], args[i+3:]...)
		}
	}
	// for each remaining nonparented item in the list after the first,
	// create a new AND filter between the first list item and the second
	for len(args) > 1 {
		and := Filter{op: opAnd, left: args[0], right: args[1]}
		args[0] = &and
		args = append(args[0:1], args[2:]...)
	}
	// now the list has only one filter left; return it
	return args[0], nil
}

// Perform the filtering operation specified in this object on the given file
// entry. Return (true, true) if the filter passes, (false, true) if it's
// rejected. If the operation involved comparing a null value, both returned
// booleans will be false.
func (self *Filter) filter(entry fileEntry) (bool, bool) {
	if self == nil {
		// filter is nil if none were specified; pass by default
		return true, true
	}

	// AND/OR filters evaluate children
	switch self.op {
	case opAnd:
		match, ok := self.left.filter(entry)
		if !match || !ok {
			return match && ok, ok
		}
		return self.right.filter(entry)
	case opOr:
		match, ok := self.left.filter(entry)
		if match || !ok {
			return match && ok, ok
		}
		return self.right.filter(entry)
	}

	diff := 0  // result of comparison +/0/-
	sval := "" // string version of value from file entry
	if self.column.isNumeric() {
		// do numeric evaluation
		eival, ok := entry.getNumericField(self.column)
		if self.op == opIsNull {
			return self.not && ok || !self.not && !ok, true
		} else if !ok {
			return false, false
		}
		if fival, ok := self.value.(int64); ok {
			// filter value is numeric; calculate diff
			switch {
			case fival > eival:
				diff = 1
			case fival < eival:
				diff = -1
			}
		} else {
			// filter value is not numeric; may be regex, for example
			// convert entry value to string
			sval = strconv.FormatInt(eival, 10)
		}
	} else {
		// do string evaluation
		ok := false
		sval, ok = entry.getStringField(self.column)
		if self.op == opIsNull {
			return self.not && ok || !self.not && !ok, true
		} else if !ok {
			return false, false
		}
		diff = strings.Compare(self.value.(string), sval)
	}

	var match bool
	// compute match status
	switch self.op {
	case opEq:
		match = diff == 0
	case opLessEq:
		match = diff >= 0
	case opLess:
		match = diff > 0
	case opRegex, opGlob:
		match = self.regex.MatchString(sval)
	default:
		panic("Bad filter operation code")
	}
	if self.not {
		match = !match
	}
	return match, true
}
