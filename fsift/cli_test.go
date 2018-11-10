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

package main

// TODO --out
// TODO all columns

// This test checks the overall output against various simulated runs
// using different command line parameters. It creates a directory
// tree in temporary storage, and it creates a temporary FSIFT file
// to scan.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jsthayer/file-sifter"
)

// Check that values using DeepEqual, report source location if doesn't match.
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

// Description of a file in the temporary test tree
type testFile struct {
	path string // relative path
	data string // contents
}

// Define the temporary file tree to create. Dirs must preceed their contents.
var tree1 = []testFile{
	{"1/", ""},
	{"1/x/", ""},
	{"1/x/a", "A"},
	{"1/x/c", "CCC"},
	{"1/y/", ""},
	{"1/y/b", "BB"},
	{"1/y/c", "CCC"},
	{"1/y/d/", ""},
	{"2/", ""},
	{"2/e", "EE"},
}

// FSIFT magic ID line
const sifterFileHeader = "| File Sifter output file - V1 |"

//---------
// Expected test outputs
//---------

var fsfile1 = `| File Sifter output file - V1 |
|
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
|
| Columns: modestr,size,mtime,path
|
  -rw-rw-r--  1  2016-11-24T15:06:42Z  x/a
  -rw-rw-r--  3  2016-11-24T15:06:43Z  x/c
  drwxr-xr-x  4  2016-11-24T15:06:41Z  x/
  -rw-rw-r--  2  2016-11-24T15:06:45Z  y/b
  -rw-rw-r--  3  2016-11-24T15:06:46Z  y/c
  drwxr-xr-x  0  2016-11-24T15:06:47Z  y/d/
  drwxr-xr-x  5  2016-11-24T15:06:44Z  y/
  drwxr-xr-x  9  2016-11-24T15:06:40Z  ./
|
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      8     9`

var diff1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,side,matched,membership,modestr
|
| Columns: membership,modestr,size,mtime,path
|
| STATISTICS:  L:Count  L:Size  R:Count  R:Size
|    Scanned:        8       9        8       9
|    Indexed:        8       9        8       9
|  Unmatched:        0       0        0       0
|   Matching:        8       9        8       9
|     Output:        0       0        0       0`

var cols1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
| Columns: size,path
  9  ./
  4  x/
  1  x/a
  3  x/c
  5  y/
  2  y/b
  3  y/c
  0  y/d/
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      8     9`

var key1 = `| File Sifter output file - V1 |
| Compare keys: base
| Evaluated columns: path,base,size,mtime,side,matched,membership,redundancy,modestr
|
| Columns: membership,modestr,size,mtime,redundancy,path
  <=  drwxr-xr-x  3  2016-11-24T15:06:40Z  1  ./
  >=  drwxr-xr-x  9  2016-11-24T15:06:40Z  1  ./
  <=  drwxr-xr-x  1  2016-11-24T15:06:41Z  1  x/
  >=  drwxr-xr-x  4  2016-11-24T15:06:41Z  1  x/
  <=  -rw-rw-r--  1  2016-11-24T15:06:42Z  1  x/a
  >=  -rw-rw-r--  1  2016-11-24T15:06:42Z  1  x/a
  >!  -rw-rw-r--  3  2016-11-24T15:06:43Z  2  x/c
  <=  drwxr-xr-x  2  2016-11-24T15:06:44Z  1  y/
  >=  drwxr-xr-x  5  2016-11-24T15:06:44Z  1  y/
  <=  -rw-rw-r--  2  2016-11-24T15:06:45Z  1  y/b
  >=  -rw-rw-r--  2  2016-11-24T15:06:45Z  1  y/b
  >!  -rw-rw-r--  3  2016-11-24T15:06:46Z  2  y/c
  <=  drwxr-xr-x  0  2016-11-24T15:06:47Z  1  y/d/
  >=  drwxr-xr-x  0  2016-11-24T15:06:47Z  1  y/d/
|
| STATISTICS:  L:Count  L:Size  R:Count  R:Size
|    Scanned:        8       9        8       9
|    Indexed:        6       3        8       9
|  Unmatched:        0       0        2       6
|   Matching:        6       3        6       3
|     Output:        6       3        8       9`

var digest1 = `| File Sifter output file - V1 |
| Compare keys: sha512,sha256,sha1,md5,path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr,sha1,sha256,sha512,md5
| Columns: modestr,size,mtime,md5,sha1,sha256,sha512,path
  -rw-rw-r--  1  2016-11-24T15:06:42Z  7fc56270e7a70fa81a5935b72eacbe29  6dcd4ce23d88e2ee9568ba546c007c63d9131c1b  559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd  21b4f4bd9e64ed355c3eb676a28ebedaf6d8f17bdc365995b319097153044080516bd083bfcce66121a3072646994c8430cc382b8dc543e84880183bf856cff5  x/a
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      1     1
|     Output:      1     1`

var exclude1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,base,size,mtime,modestr
| Columns: modestr,size,mtime,path
  -rw-rw-r--  2  2016-11-24T15:06:45Z  y/b
  drwxr-xr-x  0  2016-11-24T15:06:47Z  y/d/
| STATISTICS:  Count  Size
|    Scanned:      5     5
|    Indexed:      2     2
|     Output:      2     2`

var regular1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
| Columns: modestr,size,mtime,path
  -rw-rw-r--  3  2016-11-24T15:06:46Z  y/c
  -rw-rw-r--  2  2016-11-24T15:06:45Z  y/b
  -rw-rw-r--  3  2016-11-24T15:06:43Z  x/c
  -rw-rw-r--  1  2016-11-24T15:06:42Z  x/a
| STATISTICS:  Count  Size
|    Scanned:      4     9
|    Indexed:      4     9
|     Output:      4     9`

var symlink1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr,nlinks
| Columns: modestr,size,nlinks,path
  drwxr-xr-x  4  2  ./
  -rw-rw-r--  2  2  e
  -rw-rw-r--  2  2  ee
  Lrwxrwxrwx  0  1  xx
| STATISTICS:  Count  Size
|    Scanned:      4     4
|    Indexed:      4     4
|     Output:      4     4`

var symlink2 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr,nlinks
| Columns: modestr,size,nlinks,path
  drwxr-xr-x  8  2  ./
  -rw-rw-r--  2  2  e
  -rw-rw-r--  2  2  ee
  drwxr-xr-x  4  2  xx/
  -rw-rw-r--  1  1  xx/a
  -rw-rw-r--  3  1  xx/c
| STATISTICS:  Count  Size
|    Scanned:      6     8
|    Indexed:      6     8
|     Output:      6     8`

var prefilter1 = `| File Sifter output file - V1 |
|
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
|
| Columns: modestr,size,mtime,path
|
  -rw-rw-r--  1  2016-11-24T15:06:42Z  x/a
  -rw-rw-r--  3  2016-11-24T15:06:43Z  x/c
  drwxr-xr-x  4  2016-11-24T15:06:41Z  x/
|
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      3     4
|     Output:      3     4`

var prunefilter1 = `| File Sifter output file - V1 |
|
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
|
| Columns: modestr,size,mtime,path
|
  -rw-rw-r--  1  2016-11-24T15:06:42Z  x/a
  -rw-rw-r--  3  2016-11-24T15:06:43Z  x/c
  drwxr-xr-x  4  2016-11-24T15:06:41Z  x/
  drwxr-xr-x  4  2016-11-24T15:06:40Z  ./
|
| STATISTICS:  Count  Size
|    Scanned:      4     4
|    Indexed:      4     4
|     Output:      4     4`

var postfilter1 = `| File Sifter output file - V1 |
|
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
|
| Columns: modestr,size,mtime,path
|
  drwxr-xr-x  5  2016-11-24T15:06:44Z  y/
  drwxr-xr-x  9  2016-11-24T15:06:40Z  ./
|
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      2     0`

var membership1 = `| File Sifter output file - V1 |
| Compare keys: base
| Evaluated columns: path,base,size,mtime,side,matched,membership,redundancy,modestr
| Columns: membership,modestr,size,mtime,redundancy,path
  <=  drwxr-xr-x  3  2016-11-24T15:06:40Z  1  ./
  <=  drwxr-xr-x  1  2016-11-24T15:06:41Z  1  x/
  <=  -rw-rw-r--  1  2016-11-24T15:06:42Z  1  x/a
  >!  -rw-rw-r--  3  2016-11-24T15:06:43Z  2  x/c
  <=  drwxr-xr-x  2  2016-11-24T15:06:44Z  1  y/
  <=  -rw-rw-r--  2  2016-11-24T15:06:45Z  1  y/b
  >!  -rw-rw-r--  3  2016-11-24T15:06:46Z  2  y/c
  <=  drwxr-xr-x  0  2016-11-24T15:06:47Z  1  y/d/
| STATISTICS:  Count  Size
|    Scanned:        8       9        8       9
|    Indexed:        6       3        8       9
|  Unmatched:        0       0        2       6
|   Matching:        6       3        6       3
|     Output:        6       3        2       6`

var nodetect1 = `| File Sifter output file - V1 |
| Compare keys: modestr,size,path
| Evaluated columns: path,size,modestr
| Columns: modestr,size,path
  -rw-------  610  .
| STATISTICS:  Count  Size
|    Scanned:      1   610
|    Indexed:      1   610
|     Output:      1   610`

var summary1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,modestr
| Columns: modestr,size,mtime,path
|
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      0     0`

var plain1 = `  -rw-rw-r--  1  2016-11-24T15:06:42Z  x/a
  -rw-rw-r--  3  2016-11-24T15:06:43Z  x/c
  drwxr-xr-x  4  2016-11-24T15:06:41Z  x/
  -rw-rw-r--  2  2016-11-24T15:06:45Z  y/b
  drwxr-xr-x  0  2016-11-24T15:06:47Z  y/d/
  -rw-rw-r--  3  2016-11-24T15:06:46Z  y/c
  drwxr-xr-x  5  2016-11-24T15:06:44Z  y/
  drwxr-xr-x  9  2016-11-24T15:06:40Z  ./`

var plain0 = "drwxr-xr-x\x009\x002016-11-24T15:06:40Z\x00./\x00" +
	"drwxr-xr-x\x004\x002016-11-24T15:06:41Z\x00x/\x00" +
	"-rw-rw-r--\x001\x002016-11-24T15:06:42Z\x00x/a\x00" +
	"-rw-rw-r--\x003\x002016-11-24T15:06:43Z\x00x/c\x00" +
	"drwxr-xr-x\x005\x002016-11-24T15:06:44Z\x00y/\x00" +
	"-rw-rw-r--\x002\x002016-11-24T15:06:45Z\x00y/b\x00" +
	"-rw-rw-r--\x003\x002016-11-24T15:06:46Z\x00y/c\x00" +
	"drwxr-xr-x\x000\x002016-11-24T15:06:47Z\x00y/d/\x00"

var group1 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,mstamp,modestr
| Columns: mstamp,path
  1480000002  x/a
  1480000003  x/c
  1480000001  x/
  1480000005  y/b
  1480000007  y/d/
  1480000006  y/c
  1480000004  y/
  1480000000  ./
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      8     9`

var group2 = `| File Sifter output file - V1 |
| Compare keys: path,size,mtime,modestr
| Evaluated columns: path,size,mtime,mstamp,modestr
| Columns: mstamp,path
  1,480,000,002  x/a
  1,480,000,003  x/c
  1,480,000,001  x/
  1,480,000,005  y/b
  1,480,000,007  y/d/
  1,480,000,006  y/c
  1,480,000,004  y/
  1,480,000,000  ./
| STATISTICS:  Count  Size
|    Scanned:      8     9
|    Indexed:      8     9
|     Output:      8     9`

// Defines a test case
type test struct {
	name     string   // name for diag output
	args     []string // command line args
	postSort bool     // should unit test sort the output for predictability?
	wantOut  string   // expected output text
	wantRc   int      // expected result code
}

var tests = []test{
	{
		// scan dir tree
		"base tree", []string{"$T/1"}, true, fsfile1, 0,
	},
	{
		// scan FSIFT file
		"base FSIFT", []string{"$F"}, true, fsfile1, 0,
	},
	{
		// diff between dir tree and equivalent FSIFT should match all entries
		"NOWIN diff tree FSIFT", []string{"$T/1", ":", "$F", "-d"}, true, diff1, 0,
	},
	{
		// define custom output columns
		"cols", []string{"$F", "-sp", "-csp"}, false, cols1, 0,
	},
	{
		// define custom output key, use prefilter to cause mismatches, also test redundancy column
		"key", []string{"$T/1", ":", "$F", "-kb", "-spS", "-c+r", "-eor", "-eside=1", "-ebase!*=c"}, false, key1, 0,
	},
	{
		// test digest algorithms
		"digest", []string{"$T/1", "-ep=x/a", "-512A"}, true, digest1, 0,
	},
	{
		// test --exclude and --base option
		"exclude b", []string{"$T/1", "-xx", "-b[bd]"}, true, exclude1, 0,
	},
	{
		// test regular-only option and inverse sort
		"regular only", []string{"$T/1", "-R", "-s/p"}, false, regular1, 0,
	},
	{
		// check with symlink, not following links
		"NOWIN symlink1", []string{"$T/2", "-sp", "-cosLp"}, false, symlink1, 0,
	},
	{
		// check with symlink, while following links
		"NOWIN symlink2", []string{"$T/2", "-sp", "-L", "-cosLp"}, false, symlink2, 0,
	},
	{
		// check a prefilter expression
		"prefilter", []string{"$T/1", "-ep*=x/**"}, true, prefilter1, 0,
	},
	{
		// check a pruning prefilter expression
		"prunefilter", []string{"$T/1", "-Pp*=x/**"}, true, prunefilter1, 0,
	},
	{
		// check a postfilter expression
		"postfilter", []string{"$T/1", "-fsize>4"}, true, postfilter1, 0,
	},
	{
		// check the --membership option, use prefilters to create some mismatches
		"membership", []string{"$T/1", ":", "$F", "-kb", "-spS", "-c+r", "-eor", "-eside=1", "-ebase!*=c", "-mlR"}, false, membership1, 0,
	},
	{
		// check --nodetect option on an FSIFT file
		"nodetect", []string{"$F", "-cosp", "-kosp", "--nodetect"}, true, nodetect1, 0,
	},
	{
		// check --verify option with matching sets of files
		"verify1", []string{"$F", ":", "$T/1", "-ktsp", "--verify"}, false, "", 0,
	},
	{
		// check --verify option with mismatched sets of files
		"verify2", []string{"$T/1/x", ":", "$T/1/y", "-ktsp", "--verify"}, false, "", 1,
	},
	{
		// check --summary option
		"summary", []string{"$T/1", "--summary"}, false, summary1, 0,
	},
	{
		// check --plain option
		"plain", []string{"$T/1", "--plain"}, true, plain1, 0,
	},
	{
		// check --plain0 option
		"plain0", []string{"$T/1", "-sp", "--plain0"}, false, plain0, 0,
	},
	{
		// check numeric output without -G option
		"group1", []string{"$T/1", "-cTp"}, true, group1, 0,
	},
	{
		// check numeric output with -G option
		"group2", []string{"$T/1", "-cTp", "-G"}, true, group2, 0,
	},
	{
		// check that null compares create an error
		"nulls1", []string{"$F", ":", "$F", "-kp5"}, false, "", 1,
	},
	{
		// check that null compares with --ignore-nulls don't create an error
		"nulls2", []string{"$F", ":", "$F", "-N", "-kp5"}, false, "", 0,
	},
}

// An object to help verify that the run output matches expectations
type analyzer struct {
	outCols  []sifter.Column // the reported column lists
	evalCols []sifter.Column // "
	keyCols  []sifter.Column // "
	files    []string        // the file entry lines
	stats    []string        // the stats lines
}

// Create an analyzer from test run output or expected text string
func newAnalyzer(t *testing.T, text io.Reader) *analyzer {
	self := analyzer{}
	scanner := bufio.NewScanner(text)
	stats := false // once stats line is found, all remaining lines are added to stats field
	for scanner.Scan() {
		line := scanner.Text()
		if stats {
			self.stats = append(self.stats, line)
			continue
		}
		if !strings.HasPrefix(line, "|") {
			// any non-directive line before stats is assumed to be a file entry
			self.files = append(self.files, fudge(line))
			continue
		}
		// directive line; check for ones we care about
		parts := strings.SplitN(line, ":", 2)
		var err error
		if len(parts) > 1 {
			switch parts[0] {
			case "| STATISTICS":
				stats = true
			case "| Columns":
				self.outCols, err = sifter.ParseColumnsList(strings.Trim(parts[1], " "), false)
			case "| Evaluated columns":
				self.evalCols, err = sifter.ParseColumnsList(strings.Trim(parts[1], " "), false)
			case "| Compare keys":
				self.keyCols, err = sifter.ParseColumnsList(strings.Trim(parts[1], " "), false)
			}
		}
		if err != nil {
			t.Error("Failed to parser columns in output: ", err)
		}

	}
	return &self
}

var permsField = regexp.MustCompile("[dwrx-]{10}")

// In windows, ignore contents of permissions fields since they won't match what we set
func fudge(s string) string {
	if runtime.GOOS == "windows" {
		return permsField.ReplaceAllString(s, "----------")
	}
	return s
}

// Support sorting column lists for predictable comparisons
type colSlice []sifter.Column

func (p colSlice) Len() int           { return len(p) }
func (p colSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p colSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p colSlice) sorted() colSlice {
	out := append(colSlice{}, p...)
	sort.Sort(out)
	return out
}

// Check if this analyzer matches the other analyzer
func (self *analyzer) check(t *testing.T, got *analyzer) {
	checkVal(t, colSlice(self.outCols).sorted(), colSlice(got.outCols).sorted())
	checkVal(t, colSlice(self.evalCols).sorted(), colSlice(got.evalCols).sorted())
	checkVal(t, colSlice(self.keyCols).sorted(), colSlice(got.keyCols).sorted())

	checkVal(t, len(self.files), len(got.files))
	for i, line := range self.files {
		if i < len(got.files) {
			checkVal(t, line, got.files[i])
		}
	}
	checkVal(t, len(self.stats), len(got.stats))
	for i, line := range self.stats {
		if i < len(got.stats) {
			checkVal(t, line, got.stats[i])
		}
	}

}

// Create a test file or directory
func makePath(tempDir, path, data string) error {
	fpath := filepath.Join(tempDir, path)
	if !strings.HasSuffix(path, "/") {
		err := ioutil.WriteFile(fpath, []byte(data), os.FileMode(0664))
		return err
	} else {
		return os.Mkdir(fpath, 0755)
	}
}

// Run the tests
func Test_main(t *testing.T) {
	// Create test directory tree
	dirPath, err := ioutil.TempDir("", "sifter_unittest_")
	if err != nil {
		t.Error("Couln't create temp dir for unit test", err)
		return
	}
	defer func() { os.RemoveAll(dirPath) }()
	for _, tf := range tree1 {
		err = makePath(dirPath, tf.path, tf.data)
		if err != nil {
			t.Error("Couln't create test file for unit test", err)
			return
		}
	}

	// Create some links to test in the tree
	err = os.Symlink(filepath.Join(dirPath, "1/x"), filepath.Join(dirPath, "2/xx"))
	if err != nil {
		t.Error("Couln't create temp symlink for unit test", err)
		return
	}
	err = os.Link(filepath.Join(dirPath, "2/e"), filepath.Join(dirPath, "2/ee"))
	if err != nil {
		t.Error("Couln't create temp hard link for unit test", err)
		return
	}

	// Set all of the temp files to have predictable dates
	i := int64(1480000000) // arbitrary recent date
	for _, tf := range tree1 {
		fpath := filepath.Join(dirPath, tf.path)
		tm := time.Unix(i, 0)
		i++
		os.Chtimes(fpath, tm, tm)
	}

	// Create the test FSIFT file
	f, err := ioutil.TempFile("", "sifter_unittest_")
	defer func() { os.Remove(f.Name()) }()
	if err == nil {
		_, err = f.WriteString(fsfile1)
	}
	fspath := f.Name()
	f.Close()
	if err != nil {
		t.Error("Couln't create test file for unit test", err)
		return
	}

	out := ""
	eOut := ""
	for _, test := range tests {
		// Skip tests that won't work on windows
		if runtime.GOOS == "windows" && strings.HasPrefix(test.name, "NOWIN") {
			continue
		}
		fmt.Println("Running test case ", test.name)
		args := []string{}
		// substitute the temp file paths in the args as needed
		for _, arg := range test.args {
			arg = strings.Replace(arg, "$T", dirPath, -1)
			arg = strings.Replace(arg, "$F", fspath, -1)
			args = append(args, arg)
		}
		// create a context and set streams to capture its output
		output := bytes.Buffer{}
		errOut := bytes.Buffer{}
		ctx = sifter.NewContext()
		ctx.SetOutputStreams(&output, &errOut)
		// run the scan and verify the output
		rc := run(args)
		checkVal(t, test.wantRc, rc)
		out = output.String()
		eOut = errOut.String()
		_ = out
		_ = eOut
		// fmt.Println("@@@\n", out)
		// fmt.Println("!!!\n", eOut)
		if test.wantOut != "" {
			got := newAnalyzer(t, &output)
			want := newAnalyzer(t, strings.NewReader(strings.Trim(test.wantOut, "\n")))
			if test.postSort {
				sort.Strings(got.files)
				sort.Strings(want.files)
			}
			want.check(t, got)
		}
	}
}
