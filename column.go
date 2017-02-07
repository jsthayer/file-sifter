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
	"strings"
)

// IDs for column types
const (
	ColPath       = iota // path relative to scan root
	ColBase              // basename of file
	ColExt               // extention of file
	ColDir               // directory part of path
	ColDepth             // depth below scan root
	ColSize              // size in bytes
	ColMtime             // file mod time, RFC3339 format in UTC
	ColMstamp            // mtime, unix timestamp
	ColDevice            // ID of device file resides in
	ColSide              // "side" of the scan, false=left true=right
	ColMatched           // true if this file matches any on the other side
	ColMembership        // matching flags in string format
	ColRedundancy        // number of matches of this file on this side
	ColRedunIdx          // the index of this entry within equivalent entries on this side
	ColModestr           // file type and permissions in unix format
	ColFileType          // 1-char file type code
	ColUid               // user ID of file owner
	ColGid               // group ID of file
	ColUser              // user name of file owner
	ColGroup             // group name of file
	ColNlinks            // number of hard links to file
	ColCrc32             // crc32 digest
	ColSha1              // sha1 digest
	ColSha256            // sha256 digest
	ColSha512            // sha512 digest
	ColMd5               // md5 digest
	ColLAST              // dummy end marker; must be last
)

type Column int

// Struct to hold a column definition
type colDef struct {
	shortName string // single-char shortcut name
	longName  string // full name
	help      string // descriptive text
}

// Indexes from ID to column name and vice versa
var colNames = map[Column]colDef{}
var colIndex = map[string]Column{}

// Add a column to the indices. names is in format "short-space-long-space*"
func defineColumn(names string, col Column, help string) {
	subs := strings.Split(names, " ")
	shortName, longName := subs[0], subs[1]
	colNames[col] = colDef{shortName, longName, help}
	colIndex[shortName] = col
	colIndex[longName] = col
}

// Create the column definitions
func init() {
	defineColumn("p path      ", ColPath, "The path of this file relative to the given root")
	defineColumn("b base      ", ColBase, "The base name of this file")
	defineColumn("x ext       ", ColExt, "The extention of this filename, if any")
	defineColumn("D dir       ", ColDir, "The directory part of the 'path' field")
	defineColumn("d depth     ", ColDepth, "How many subdirectories this file is below its root")
	defineColumn("s size      ", ColSize, "Regular files: size in bytes. Dirs: cumulative size; Other: 0")
	defineColumn("t mtime     ", ColMtime, "Modification time as a string")
	defineColumn("T mstamp    ", ColMstamp, "Modification time as seconds since the Unix epoch")
	defineColumn("o modestr   ", ColModestr, "Mode and permission bits as a human readable string")
	defineColumn("f filetype  ", ColFileType, "The type of this file: f=regular, d=dir, etc.")
	defineColumn("U uid       ", ColUid, "The user ID of this file's owner")
	defineColumn("G gid       ", ColGid, "The group ID of this file's group")
	defineColumn("u user      ", ColUser, "The name of this file's owner")
	defineColumn("g group     ", ColGroup, "The name of this file's group")
	defineColumn("L nlinks    ", ColNlinks, "The number of hard links to this file")
	defineColumn("V device    ", ColDevice, "The ID of the device this file resides on")
	defineColumn("S side      ", ColSide, "The 'side' of this file's root: '0'=left '1'=right")
	defineColumn("M matched   ", ColMatched, "True if this file matches any file from the *other* side")
	defineColumn("m membership", ColMembership, "Visual representation of 'side' and 'matched' columns")
	defineColumn("r redundancy", ColRedundancy, "Count of files matching this file on *this* side")
	defineColumn("I redunidx  ", ColRedunIdx, "Ordinal of this file amongst equivalents on *this* side")
	defineColumn("3 crc32     ", ColCrc32, "The CRC32 digest of this file")
	defineColumn("1 sha1      ", ColSha1, "The SHA1 digest of this file")
	defineColumn("2 sha256    ", ColSha256, "The SHA256 digest of this file")
	defineColumn("A sha512    ", ColSha512, "The SHA512 digest of this file")
	defineColumn("5 md5       ", ColMd5, "The MD5 digest of this file")
}

// Return a list of strings holding help text describing all columns
func GetColumnHelp() []string {
	out := []string{}
	for col := Column(0); col != ColLAST; col++ {
		def := colNames[col]
		out = append(out, fmt.Sprintf("%s %-12s %s", def.shortName, def.longName, def.help))
	}
	return out
}

// Return the long name of a column
func (col Column) String() string {
	return colNames[col].longName
}

// Return true if this column holds a numeric (int64) value
func (col Column) isNumeric() bool {
	switch col {
	case ColDepth, ColSize, ColMstamp, ColDevice, ColRedundancy, ColRedunIdx, ColUid, ColGid, ColNlinks, ColSide, ColMatched:
		return true
	default:
		return false
	}
}

// Return true if this column is always computed at analyze time and never parsed or scanned
func (col Column) isDynamic() bool {
	switch col {
	case ColSide, ColMatched, ColRedundancy, ColRedunIdx, ColMembership:
		return true
	default:
		return false
	}
}

// Parse an argument that has a comma-separated list of long and/or short
// column names, return a corresponding slice of column IDs.  If the argument
// has no commas and it doesn't match a long name, also try assuming each char
// is an individual short name. If an error occurs, return nil and the error.
func ParseColumnsList(list string) ([]Column, error) {
	columns := []Column{}
	if len(list) == 0 {
		return columns, nil
	}
	if strings.IndexByte(list, ',') < 0 {
		_, ok := colIndex[list]
		if !ok {
			// no commas and name doesn't match; insert commas between each char
			list = strings.Replace(list, "", ",", -1)
			list = list[1 : len(list)-1]
		}
	}
	for _, name := range strings.Split(list, ",") {
		col, ok := colIndex[name]
		if !ok {
			return nil, fmt.Errorf("Bad column name '%s'", name)
		}
		columns = append(columns, col)
	}
	return columns, nil
}

// Pattern to match column directive line in FSIFT file.
var columnsDirectivePat = regexp.MustCompile(`^\|\s*Columns:\s+([\w,]+)\s*$`)

// Try to parse a line of text as a FSIFT file columns directive.  If it
// doesn't look like a columns directive, return nil. Otherwise, return the
// list of any columns and any error parsing the column names.
func parseColumnsDirective(line []byte) ([]Column, error) {
	match := columnsDirectivePat.FindSubmatch(line)
	if match != nil {
		return ParseColumnsList(string(match[1]))
	} else {
		return nil, nil
	}

}

// Create a comma-separated list of names of the given list of column IDs.
func formatColumnNames(cols []Column) string {
	var a []string
	for _, col := range cols {
		a = append(a, col.String())
	}
	return strings.Join(a, ",")
}

// Returns true if the given list of column IDs contains the specified column.
func containsCol(cols []Column, col Column) bool {
	for _, c := range cols {
		if c == col {
			return true
		}
	}
	return false
}

// Insert a column ID into a list of columns at the given index. If
// the index is less than zero, insert that far from end (-1 = append).
func insertCol(cols *[]Column, index int, col Column) {
	if index < 0 {
		index = len(*cols) + index
	}
	if index < 0 {
		index = 0
	}
	*cols = append(*cols, ColLAST)
	copy((*cols)[index+1:], (*cols)[index:])
	(*cols)[index] = col
}
