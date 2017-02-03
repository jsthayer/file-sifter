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
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Sorting interface for file entries
type entrySorter struct {
	ctx     *Context    // the context the entry belongs to
	entries []fileEntry // the entries to sort
	columns []Column    // the columns to sort by from highest to lowest precedence
}

// Create an entry sorter for the given entries and key columns
func newEntrySorter(ctx *Context, entries []fileEntry, columns []Column) *entrySorter {
	sorter := entrySorter{
		ctx:     ctx,
		entries: entries,
		columns: append([]Column{}, columns...), // make a copy
	}
	return &sorter
}

// Implement sort.Interface
func (self *entrySorter) Len() int {
	return len(self.entries)
}

func (self *entrySorter) Swap(i, j int) {
	self.entries[i], self.entries[j] = self.entries[j], self.entries[i]
}

func (self *entrySorter) Less(i, j int) bool {
	diff, notNull := self.entries[i].compare(self.entries[j], self.columns)
	self.ctx.checkNullCompare(notNull) // warn about any null compares if applicable
	return diff < 0
}

// A file entry object is a map from column ID to value (either string or int64)
type fileEntry map[Column]interface{}

// Create a new empty file entry object
func newFileEntry() fileEntry {
	return make(fileEntry, 5) // file entries usually have several fields
}

// Set a field with a string value
func (self fileEntry) setStringField(col Column, value string) {
	self[col] = value
}

// Compute a file type code from a mode string field. We change
// 'D' to 'b' (block device) and '-' to 'f' (regular file).
func modeStrToFileType(modeStr string) string {
	switch {
	case strings.IndexByte(modeStr, 'c') >= 0:
		return "c"
	case strings.IndexByte(modeStr, 'D') >= 0:
		return "b"
	case strings.IndexByte(modeStr, 'p') >= 0:
		return "p"
	case strings.IndexByte(modeStr, 'L') >= 0:
		return "L"
	case strings.IndexByte(modeStr, 'd') >= 0:
		return "d"
	case strings.IndexByte(modeStr, 'S') >= 0:
		return "S"
	default:
		return "f"
	}
}

// Get a string-valued field from the entry. If the field doesn't
// exist and it can be derived from another field that does exist,
// compute the derived value. If no value found, return ("", false).
// Will panic if field exists and is numeric.
func (self fileEntry) getStringField(col Column) (string, bool) {
	val, ok := self[col]
	if !ok {
		// Not found; try to derive
		switch col {
		case ColBase:
			if val, ok := self[ColPath]; ok {
				return path.Base(val.(string)), true
			}
		case ColDir:
			if val, ok := self[ColPath]; ok {
				return path.Dir(val.(string)), true
			}
		case ColExt:
			if val, ok := self[ColPath]; ok {
				return path.Ext(val.(string)), true
			} else {
				if val, ok := self[ColBase]; ok {
					return path.Ext(val.(string)), true
				}
			}
		case ColMembership:
			if side, ok := self.getBoolField(ColSide); ok {
				if match, ok := self.getBoolField(ColMatched); ok {
					switch {
					case side && match:
						return ">=", true
					case !side && match:
						return "<=", true
					case side && !match:
						return ">!", true
					case !side && !match:
						return "<!", true
					}
				}
			}
		case ColFileType:
			if val, ok := self[ColModestr]; ok {
				return modeStrToFileType(val.(string)), true
			}
		case ColMtime:
			if val, ok := self.getNumericField(ColMstamp); ok {
				// mtime is always stored internally in UTC
				tm := timeToMtime(time.Unix(val, 0), nil)
				self.setStringField(col, tm)
				return tm, true
			}
		}
	}
	if ok {
		return val.(string), true
	} else {
		return "", false
	}
}

// Set a numeric field from a boolean. true -> 1, false -> 0.
func (self fileEntry) setBoolField(col Column, value bool) {
	if value {
		self[col] = int64(1)
	} else {
		self[col] = int64(0)
	}
}

// Get boolean from a numeric field. If field doesn't exist,
// return (false, false).
func (self fileEntry) getBoolField(col Column) (bool, bool) {
	val, ok := self[col]
	if ok {
		return val.(int64) != 0, true
	} else {
		return false, false
	}
}

// Get a boolean field; if it doesn't exist, return false.
func (self fileEntry) getBoolFieldOrFalse(col Column) bool {
	b, _ := self.getBoolField(col)
	return b
}

// Set a numeric field
func (self fileEntry) setNumericField(col Column, value int64) {
	self[col] = value
}

// Get a numeric-valued field from the entry. If the field doesn't
// exist and it can be derived from another field that does exist,
// compute the derived value. If no value found, return (0, false).
// Will panic if field exists and is not numeric.
func (self fileEntry) getNumericField(col Column) (int64, bool) {
	ival, ok := self[col]
	if !ok {
		switch col {
		case ColMstamp:
			if val, ok := self[ColMtime]; ok {
				tm, err := mtimeToTime(val.(string))
				if err == nil {
					ts := tm.Unix()
					self.setNumericField(col, ts)
					return ts, true
				}
			}
		case ColDepth:
			if sval, ok := self[ColPath]; ok {
				d := int64(strings.Count(sval.(string), "/"))
				self.setNumericField(col, d)
				return d, true
			}
		}
		return 0, false
	} else {
		return ival.(int64), true
	}
}

// Get a numeric field; if it doesn't exist, return zero.
func (self fileEntry) getNumericFieldOrZero(col Column) int64 {
	n, _ := self.getNumericField(col)
	return n
}

// Compare this entry to another entry according to the given key columns in
// order of precedence.  The return integer is less than zero if this entry is
// less than that, zero if the entries are equal, otherwise greater than zero.
// The returned boolean is false if either entry tried to compare a null value.
func (self fileEntry) compare(that fileEntry, columns []Column) (int, bool) {
	gotNull := false
	for _, col := range columns {
		diff := 0
		var ok1, ok2 bool
		if col.isNumeric() {
			var v1, v2 int64
			v1, ok1 = self.getNumericField(col)
			v2, ok2 = that.getNumericField(col)
			switch {
			case v1 > v2:
				diff = 1
			case v1 < v2:
				diff = -1
			}
		} else {
			var v1, v2 string
			v1, ok1 = self.getStringField(col)
			v2, ok2 = that.getStringField(col)
			diff = strings.Compare(v1, v2)
		}
		switch {
		case ok1 && ok2 && diff != 0:
			return diff, true // there was a difference; return it now
		case !ok1 && ok2:
			return -1, false // this entry had a null; less
		case ok1 && !ok2:
			return 1, false // that entry had a null; greater
		case !ok1 || !ok2:
			gotNull = true // both were nulls; remember and continue
		}
	}
	return 0, !gotNull // all cols equal
}

// Matches a backslash followed by another char
var unescapePat = regexp.MustCompile(`\\(.)`)

// Remove escapes from a FSIFT file field for parsing. The boolean
// returned is true unless the field was an encoded null value.
func unescapeField(val string, lastCol bool) (out string, notNull bool) {
	notNull = true
	switch {
	case strings.IndexByte(val, '\\') < 0:
		// quick check for no escapes
		out = val
	case val == `\-`:
		// empty value
	case val == `\~`:
		// NULL value
		notNull = false
	case lastCol:
		// last column gets no escapes other than above two
		out = val
	default:
		// remove remaining escape backslashes
		out = unescapePat.ReplaceAllString(val, "$1")
	}
	return
}

// Escapes a field for writing to a FSIFT file. If notNull is false, encode
// a null escape. For only the last column, spaces are not escaped.
func escapeField(text string, notNull, lastCol bool) string {
	switch {
	case !notNull:
		return `\~` // null value
	case text == "":
		return `\-` // empty value
	case lastCol:
		return strings.Replace(text, `\`, `\\`, -1) // just escape backslashes
	default:
		text = strings.Replace(text, `\`, `\\`, -1) // escape backslashes and spaces
		return strings.Replace(text, ` `, `\ `, -1)
	}
}

// Set a field from an already-unescaped string value. If the field is numeric,
// convert to an integer. If the field is dynamic, ignore and do nothing.
func (self fileEntry) parseAndSetField(ctx *Context, col Column, value string) (err error) {
	switch {
	case col.isDynamic():
		return
	case col.isNumeric():
		var ival int64
		value = strings.Replace(value, ",", "", -1) // ignore any grouping commas
		ival, err = strconv.ParseInt(value, 10, 64)
		if err == nil {
			self.setNumericField(col, ival)
		}
	default:
		self.setStringField(col, value)
	}
	return
}

// Format a field for output. If it's numeric, convert to decimal string. If it's
// an mtime date string and the user specified an output timezone, change the timezone.
// Escape the field (using the lastCol flag). If width >= zero, pad the result to the
// given width.
func (self fileEntry) formatField(ctx *Context, col Column, width int, lastCol bool) string {
	var text string
	var ok bool
	var ival int64

	switch {
	case col.isNumeric():
		ival, ok = self.getNumericField(col)
		text = ctx.formatNumber(ival)
	case col == ColMtime:
		if text, ok = self.getStringField(col); ok {
			text = ctx.adjustOutputTimezone(text)
		}
	default:
		text, ok = self.getStringField(col)
	}
	text = escapeField(text, ok, lastCol)
	if width < 0 {
		return text
	} else if col.isNumeric() {
		return fmt.Sprintf("%*s", width, text)
	} else {
		return fmt.Sprintf("%-*s", width, text)
	}
}

// Convert the given columns in this file entry to JSON text.
func (self fileEntry) toJson(columns []Column) ([]byte, error) {
	// a map to hold the JSON-ready values
	m := make(map[string]interface{})

	for _, col := range columns {
		var val interface{}
		switch {
		case col.isNumeric():
			if n, ok := self.getNumericField(col); ok {
				val = n
			}
		default:
			if s, ok := self.getStringField(col); ok {
				val = s
			}
		}
		m[col.String()] = val
	}
	// convert the map to JSON
	return json.MarshalIndent(m, "    ", "    ")
}
