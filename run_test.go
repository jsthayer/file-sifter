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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// checkVal function checks values match using a deep comparison of the
// expected and actual values. Reports a unit test error at the caller's
// location if failed.
func checkVal(t *testing.T, want, got interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		file = filepath.Base(file)
	} else {
		file = "???"
		line = 1
	}
	if !reflect.DeepEqual(want, got) {
		fmt.Fprintf(os.Stderr, "    %s:%d Got '%v', expected '%v'\n", file, line, got, want)
		t.Fail()
	}
}

// checkValErr function checks that both value and error return from a function
// are as expected.  If errPrefix is not empty, check that the error returned
// from the function is not nil and has a message starting with the prefix.
// Otherwise, make sure the error is nil and do a deep comparison of the
// expected and actual returned values. Reports a unit test error at the
// caller's location if failed.
func checkValErr1(t *testing.T, want, got interface{}, errPrefix string, gotErr error) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		file = filepath.Base(file)
	} else {
		file = "???"
		line = 1
	}
	switch {
	case errPrefix == "" && gotErr != nil:
		fmt.Fprintf(os.Stderr, "Got unexpected error '%v'", gotErr)

	case errPrefix != "" && gotErr == nil:
		fmt.Fprintf(os.Stderr, "Expected error starting with '%s', got no error", errPrefix)

	case errPrefix != "" && !strings.HasPrefix(gotErr.Error(), errPrefix):
		fmt.Fprintf(os.Stderr, "Expected error starting with '%s', got error '%v'", errPrefix, gotErr)

	case !reflect.DeepEqual(want, got):
		fmt.Fprintf(os.Stderr, "    %s:%d Got '%v', expected '%v'\n", file, line, got, want)

	default:
		return
	}
	t.Fail()
}

// checkValErr function checks that both value and error return from a function
// are as expected.  If errPrefix is not empty, check that the error returned
// from the function is not nil and has a message starting with the prefix.
// Otherwise, make sure the error is nil and do a deep comparison of the
// expected and actual returned values. Returns the test error message to log
// on failure, empty string on success.
func checkValErr(t *testing.T, want, got interface{}, errPrefix string, gotErr error) string {
	switch {
	case errPrefix == "" && gotErr != nil:
		return fmt.Sprintf("Got unexpected error '%v'", gotErr)

	case errPrefix != "" && gotErr == nil:
		return fmt.Sprintf("Expected error starting with '%s', got no error", errPrefix)

	case errPrefix != "" && !strings.HasPrefix(gotErr.Error(), errPrefix):
		return fmt.Sprintf("Expected error starting with '%s', got error '%v'", errPrefix, gotErr)

	case !reflect.DeepEqual(want, got):
		return fmt.Sprintf("Got '%v', expected '%v'", got, want)

	default:
		return ""
	}
}

func Test_timeToMtime(t *testing.T) {
	// UTC
	got := timeToMtime(time.Unix(0, 0), nil)
	checkVal(t, "1970-01-01T00:00:00Z", got)
	// custom timezone
	got = timeToMtime(time.Unix(0, 0), time.FixedZone("foo", 3600))
	checkVal(t, "1970-01-01T01:00:00+01:00", got)
}

func Test_mtimeToTime(t *testing.T) {
	// UTC
	got, err := mtimeToTime("1970-01-01T00:00:00Z")
	checkValErr1(t, int64(0), got.Unix(), "", err)
	// custom timezone
	got, err = mtimeToTime("1970-01-01T01:00:00+01:00")
	checkValErr1(t, int64(0), got.Unix(), "", err)
	// bad date
	got, err = mtimeToTime("bad")
	checkValErr1(t, nil, nil, "parsing time", err)
}

func TestMain(m *testing.M) {
	flag.Parse()
	// set unitTest flag to prevent starting output thread
	unitTest = true
	os.Exit(m.Run())
}
