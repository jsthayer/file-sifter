% FSIFT(1)
% John Thayer
% January 2017

<!-- This Markdown file uses pandoc extensions -->

# NAME

fsift - File Sifter: Powerful tool to scan, digest, compare and report on files
in directory trees (or in previously saved reports) 

# SYNOPSIS

**fsift** [ *options* ] [ *left-roots*... ] [ **:** *right-roots*... ]

# DESRIPTION

File Sifter is a utility that has features inspired by the \*NIX utilites
*find* and *diff*, as well as file digest programs such as *sha256sum*. It can
be used for a variety of tasks involving the management of large numbers of
files, such as determining which files may be out-of-date in offsite backups.

# OVERVIEW

File Sifter able to scan filesystems like *find*, but instead of immediately
printing entries, it loads the information into an internal index.

Once loaded, File Sifter performs analysis on the index. It then optionally
sorts the entries and prints them out in tabular format, followed by summary
statistics. This output may be saved to an *FSIFT* file, and entries from that
file may be loaded during a later run for comparisons or similar analysis.

The non-option arguments given to File Sifter are called *roots*. Roots
are generally either the top of a directory tree to scan, or a previously
saved *FSIFT* file. Each root is loaded into one of two *sides* of the index,
which are called *left* and *right*. During analysis, File Sifter can
compare the contents of the left and right sides of the index based on
user-specified criteria, and it can generate output and reports about the
comparisons.

File entries may be subjected to user-defined filters at two points: before
loading into the index, or before the final output. These filters allow
tailoring the analysis to specific requirements.

Each piece of information about a file processed by the program is called a
*column*.  Columns are defined for the usual file system attributes such as
path, size and permissions. There are also columns for digests of file data.
Finally, some columns can by added by the File Sifter program as the result of
analysis, such as whether the file *matches* at least one other file.

Columns are only computed and stored if they are made necessary by the
user-specified command line options. For example, the contents of files are
only read if the user specifies operations on a digest column.

# EXAMPLES

* Scan and print information about the current directory tree:

>    **fsift**

> Example output (can also be saved into an *FSIFT* file):

        | File Sifter output file - V1 |
        | Command line:
        | Current working directory: /home/user/projects/file-sifter
        | Compare keys: path,size,mtime,modestr
        | Evaluated columns: path,size,mtime,modestr
        | Run start time: 2017-02-06T15:36:04Z
        | 
        | Columns: modestr,size,mtime,path
        | 
          -rw-rw-r--  15028  2017-02-06T03:41:10Z  fsift_manpage.html
          -rw-rw-r--  10476  2017-02-06T03:41:03Z  manpage.md
          -rw-rw-r--  12383  2017-02-06T03:41:10Z  fsift.1
          -rw-r--r--  24576  2017-02-06T03:41:07Z  .manpage.md.swp
          drwxrwxr-x  62463  2017-02-06T03:41:03Z  ./
        | 
        | Run end time: 2017-02-06T15:36:04Z
        | Elapsed time: 254.186µs
        | 
        | STATISTICS:  Count   Size
        |    Scanned:      5  62463
        |    Indexed:      5  62463
        |     Output:      5  62463

* Save the information about a directory tree to an *FSIFT* file; add md5sum digests:

>   **fsift /path/to/mydir --md5 --out mydir.FSIFT**

>>   *or*

>   **fsift /path/to/mydir -5omydir.FSIFT**

* Compare the contents of a directory tree to a previously saved *FSIFT* file, showing any differences:

>   **fsift mydir.FSIFT : /path/to/mydir --diff**

* Find which direct child subdirectories are using the most storage:

>   **fsift top/dir --sort size --postfilter filetype=d --postfilter 'depth<2'**

>>   *or*

>   **fsift top/dir -ss -ff=d -fd\\<2**

* Find which git repositories are most in need of garbage collection:

>   **fsift my/projects --prefilter 'path \*=\*\*/.git/objects/' --sort nlinks --columns +nlinks**

>>   *or*

>   **fsift my/projects '-ep\*=\*\*/.git/objects/' -sL -c+L**

* List what kinds of files are in a directory tree:

>   **fsift top/dir --columns extension --key extension --postfilter redunidx=1**

>>   *or*

>   **fsift top/dir -cx -kx -fI=1**

* Assume lists of previously archived files have been saved in a set of *FSIFT*
files. Before decomissioning a disk, scan it to check against the archives to
find any files that may need to be added to archives:

>   **fsift archive-index/\*.FSIFT : path/to/disk --membership R --key base,size,mtime,md5**

>>   *or*

>   **fsift archive-index/\*.FSIFT : path/to/disk -mR -kbst5**

* Find all files that have redundant data content:

>   **fsift top/dir --postfilter 'redundancy >1' --key md5 --columns +redundancy --sort size --regular-only**

>>   *or*

>   **fsift top/dir -fr\\>1 -k5 -c+r -Rss**


# OPTIONS

Options may be freely intermixed with roots. Most options have both a long
version and a short version. Short boolean options may be concatenated
(such as -qS), and short options which take an argument may have the
argument concatenated to the option. Long options can have the argument
joined with an "**=**" character. The following are all equivalent:
**-st**, **-s t**, **-smtime**, **--sort mtime**, **--sort=mtime**, **--sort t**.

**:**
 ~ A single colon on the command line is a special marker that divides the
   left-side roots from the right-side roots. If no colon is present, all roots
   are assigned to the left side. Otherwise, any roots on the command line
   *after* the colon are assigned to the right side.

# Field selection, comparing and sorting:
**-c**, **--columns=COLUMNS**
 ~ Select output columns. The default is "modestr,size,mtime,path"
   (alternatively, "ostp"). If
   there are roots on both sides, then "membership" is also added to the
   default column set.

**-s**, **--sort=COLUMNS**
 ~ Sort output using these fields, in order of precedence.
   By default, the output is not sorted.

**-k**, **--key=COLUMNS**
 ~ Specify which fields used to compare files on each side for equivalence.
   The default is "modestr,size,mtime,path".

**-5**, **--md5**
 ~ Shortcut to add md5 column to compare key and output.

**-2**, **--sha256**
 ~ Shortcut to add sha256 column to compare key and output.

**-A**, **--sha512**
 ~ Shortcut to add sha512 column to compare key and output.

**-1**, **--sha1**
 ~ Shortcut to add sha1 column to compare key and output.

# Pre-analysis filtering:
**-e**, **--prefilter=FILTER-EXP**
 ~ Filter to screen files before they are loaded into the index.

**-b**, **--base-match=GLOB-PAT**
 ~ Filter files by basename glob pattern. Shortcut for **--prefilter 'base\*=\*GLOB-PAT\*'**.

**-x**, **--exclude=GLOB-PAT**
 ~ Exclude file system files and/or directory trees if their path matches the given glob pattern.
   If a directory name matches an exclude pattern, do not descend into it. This option only
   applies to scans of file systems; it does not filter entries loaded from FSIFT files.

**-R**, **--regular-only**
 ~ Only load regular files into the index while scanning. This option only
   applies to scans of file systems; it does not filter entries loaded from FSIFT files.

**-L**, **--follow-links**
 ~ Follow symbolic links while scanning the file system. By default, when a
   symbolic link is found, an entry is created about the symbolic link itself.
   When this option is in effect, an entry is created with information about the
   link *target*. If a symbolic link points to a directory, the program will
   not descend into that directory unless this option is in effect.

**-X**, **--xdev**
 ~ Stay within one file system. If a subdirectory is mount point, don't descend into it.

# Post-analysis filtering:
**-f**, **--postfilter=FILTER-EXP**
 ~ This filter does not prevent entries from being loaded into the index or analyzed,
   but it does prevent any filtered entries from being printed to the output.

**-m**, **--membership=CHARS**
 ~ Filter output by membership code. The code must be a string containing only a subset
   of the characters **l**,**r**,**L** or **R**. The **L** and **l** codes only allow
   entries from left-side roots, and the other two codes allow entries from the right-side
   roots. The lower case codes only allow files that were *matched* by files on the
   other side, and the upper case codes only allow files that were *unmatched*
   by files on the other side.  For example, the option **--membership=Lr**
   only prints files from the left that were unmatched, as well as files from
   the right that were matched.

**-d**, **--diff**
 ~ Show unmatched entries only. This is a shortcut for **--membership=LR**.
   (Which in turn is a shortcut for **-f OR -f m=<! -f m=>!**.)

**--nodetect**
 ~ Don't try to detect whether regular files specified as roots are FSIFT files.
   By default, if a file looks like it is an FSIFT file, entries are parsed
   and loaded from the contents of the file instead of loading information about
   the file itself. If this option is in effect, the detection step is not
   done, no FSIFT parsing is attempted, and for any regular file given as a
   root, an entry is created about the file itself.

# Output formatting
**-o**, **--out=PATH**
 ~ Output to the given file path instead of the default stdout.

**-Y**, **--verify**
 ~ Checks that all left entries under left-sided roots are matched by at least
   one entry under a right-sided root.  If the left side root is a FSIFT file,
   this is somewhat similar to running a program like **sha1sum -c**.
   If a mismatched left-side file is found, the program exits with a nonzero
   status code.

**-S**, **--summary**
 ~ Only the header and footer summary info. No file entries are output.

**-p**, **--plain**
 ~ Only output the file entries. No header or footer summary info is printed.

**-0**, **--plain0**
 ~ Like 'plain', but also separate all output fields with ASCII **NUL** characters.
   Newlines betweeen file entries are also replaced with **NUL** characters.
   The usual FSIFT format escaping is *not* done.

**-G**, **--group-nums**
 ~ For numeric output, separate groups of three decimal digits with commas.

**-N**, **--ignore-nulls**
 ~ File entry fields get compared during filtering, membership analysis and sorting.
   By default, if any comparison uses a field that is missing (in other words,
   *null*), then an error message will be generated and the program will have a
   nonzero exit status. If this option is in effect, then no error is
   generated. In any case, all *null* comparisons are considered to have a false result.

**-J**, **--json-out**
 ~ Output file entries in JSON format. The output will be an array of JSON
   objects, each with a key-value pair for each field defined in the
   output. Numeric fields are output as JSON integers, string fields
   are output as JSON strings, and missing fields are JSON **null** values.
   No header or footer information is output.

**-Z**, **--out-zone**
 ~ Format output times for given location. The default is UTC. Locations
   can be specified as fixed offsets like "**+06:00**", or as locations
   recognized by the Go language *time* package, such as "**Local**" or
   "**America/Chicago**".  Times are always output in RFC3339 format, such as
   "**2017-02-03T04:52:13Z**".

**-v**, **--verbose**
 ~ Increase verbosity.

**-q**, **--quiet**
 ~ Decrease verbosity.

**-V**, **--version**
 ~ Show program version and exit.

**-h**, **--help**
 ~ Print a help message and exit.

# COLUMN CODES

Each column has a full name and a single-character short name alias.

For options that take a list of columns as an argument, the columns can be
specified with a comma-separated list of names, long or short. If no commas are
in the argument and it does not match a long name, then the program tries
assuming that the argument is a concatenation of short name characters. The
argument must not contain whitespace.

For example, **size,mtime,path** can be shortened to **stp**.

**p    path**
 ~ The path of this file relative to the given root.

**b    base**
 ~ The base name of this file.

**x    ext**
 ~ The extention of this filename, if any.

**D    dir**
 ~ The directory part of the 'path' field.

**d    depth**
 ~ How many subdirectories this file is below its root.

**s    size**
 ~ Regular files: size in bytes. Dirs: cumulative size; Other: 0.
   Note that the cumulative sizes of directories do not figure
   into analysis or statistics roundups.

**t    mtime**
 ~ Modification time as a string in RFC3339 format.

**T    mstamp**
 ~ Modification time as seconds since the Unix epoch.

**V    device**
 ~ The ID of the device this file resides on.

**S    side**
 ~ The *side* of this file's root: **0**=left **1**=right.

**M    matched**
 ~ True if this file matches any file from the *other* side, according to the fields
   in the **--key** option.

**m    membership**
 ~ Visual representation of 'side' and 'matched' columns:  

        side  matched  membership
        0     0        "<!"
        0     1        "<="
        1     0        ">!"
        1     1        ">="


**r    redundancy**
 ~ Count of all files on *this* side matching this file.

**I    redunidx**
 ~ Ordinal of this file amongst equivalents on *this* side. If a postfilter
   is set to only select entries where the **redunidx** value is **1**,
   then only one entry per group of equivalent files will be output
   on each side, so such a filter can be used to enumerate unique values.

**o    modestr**
 ~ Mode and permission bits as a human readable string, in
   Unix-like format. See the documentation of the Go
   language *os.FileMode* object for exact details on
   the format.

**f    filetype**
 ~ A single-character code indicating the type of this file. This
   is usually the same as the first character of **modestr**, except
   that for regular files '**-**' is changed to '**f**', and for
   block devices '**D**' is changed to '**b**'.

**U    uid**
 ~ The user ID of this file's owner.

**G    gid**
 ~ The group ID of this file's group.

**u    user**
 ~ The name of this file's owner.

**g    group**
 ~ The name of this file's group.

**L    nlinks**
 ~ The number of hard links to this file.

**3    crc32**
 ~ The CRC32 checksum of this file. **Note**: for all checksum and digest
   fields, directories and other nonregular files get an empty string for a value
   instead of *null*. This avoids excessive *null* comparison errors when these
   fields are compared. **Important:** The crc32 checksum should not be used for
   security purposes.

**1    sha1**
 ~ The SHA1 digest of this file. **Important:** The sha1 digest probably
   should not be used for security purposes.

**2    sha256**
 ~ The SHA256 digest of this file.

**A    sha512**
 ~ The SHA512 digest of this file.

**5    md5**
 ~ The MD5 digest of this file. **Important:** The md5 digest
   should not be used for security purposes.

# FILTER SPECIFICATIONS

*Filters* allow the rejection of file entries based on user-defined criteria.

Any column can be used in a filter specification. The general filter syntax is as follows:

<**column-name**><**operator**><**value**>

The column name may be any long or short column identifier. The value is an
arbitrary string for comparison to an entry's value. There should be no
whitespace between the operator and the value.

The supported operators are:

**=**
 ~ Equals

**< <= > >=**
 ~ Ordered comparisons (numeric or lexicographic, depending on the column type)

**\*=**
 ~ Glob pattern match (see below).

**~=**
 ~ Regular expression match. For detailed syntax, see the Go language
   *regexp* package documentation.

**.isnull**
 ~ Matches if the value is missing in this file entry

**!= !\*= !~= !.isnull**
 ~ Negated versions of the above operators

## Combining Filters

Multiple filters of the same type may be specified. In addition to the above filters,
there are two special combining filters:

**or**
    Matches if either child filter matches.

**and**
    Matches if both child filters match.

Combining filters act as binary operators between other filters. They
are applied in *prefix* order (also known as *normal* Polish notation).
In other words, a combining filter appears directly in front of its two child filters.

For example, the following set of filters selects all the files larger
than one megabyte that belong to Jack or Jill:

**-f and -f 'size>1000000' -f or -f user=jack -f user=jill**

If there are any filters of a given type remaining after combining filters have
been assigned children, then the remaining filters are assigned to implicit
**and** filters. So for the common case of multiple filters defined
with no combining filters, they are *and*ed together.


## Glob Patterns

Glob patterns are similar to glob expansion in shell interpreters: "__?__" matches any
single character except "__/__". "__\*__" matches any number of non-"__/__" characters.
In this implementation, the contents of brackets **\[**...**\]** are fed directly
to the underlying regular expression evaluator; the result is similar to many
glob implementations, but there are some differences. See the Go langauge *regexp*
package documentation for details.

As a special case, a __\*\*__ matches any number of characters, *including*
"__/__".  This can be used to match entire segments of file paths. For example,
the filter: __path\*=\*\*/.config/\*\*__ will select all files under *.config*
directories below the top level.

When using the __\*\*__ operator, note that directory paths are always stored
with a trailing "__/__" character.  Also note that files directly under the
root will not have a "__/__" preceding them. If this creates problems with glob
matching entire paths, a regular expression pattern may be a more flexible alternative.

# OTHER FEATURES

## FSIFT Files

The output of File Sifter may be saved to a file (an extention of **.FSIFT** is
recommended).  If this file is later specified as a root during another run of
File Sifter, then by default entries will parsed and loaded from that file.

Note that if the header information or the file entries are suppressed using
command line options (such as **--summary** or **--plain**), then the output
will not be useful for loading later.

### Syntax

*FSIFT* files are text files with two kinds of lines: *directive* lines
and *entry* lines. Directive lines start with a **|** character. 

The file starts with a *directive* line identifying it as an *FSIFT* file. After that
is a group of informational header directives. The only other directive that
is relevant to parsing *FSIFT* files is the **Columns** directive. This
specifies which fields are present in each *entry* line. The **Columns**
directive must appear before the first *entry* line. All other *directive*
lines are ignored by the parser.

*Entry* lines follow the header, with one *entry* line per output file entry.
These lines start with a space, and are followed by one or more fields.
The number of fields matches the number of names in the **Columns** directive,
and each item is the field information from the corresponding column name.

After the entries, a footer is output. This contains a set of *directive*
lines with summary information such as run time, file and byte counts.

In *entry* lines, fields are separated by two space characters.  Certain
characters within a field are escaped with backslashes: spaces, newlines,
carraige returns and backslashes.  In addition, there are two special escape
sequences: **\\-** indicates a zero-length string, and **\\~** indicates a
missing value (called a *null*).  These escapes are removed when *FSIFT* files
are parsed.

As a special exception, space characters are *not* escaped in the last column.
This is possible because the parser knows that no other fields will follow this
one before the next newline. In the common case where *path* is the last field
on each line, this makes the output look cleaner when there are spaces in file
names. (However, even in the last column, any unlikely spaces at the begining
or end of a field are still escaped.)

When the **--plain0** option is specified, there is no escaping performed, and
all data is separated by ASCII **NUL** characters. Then the **--json** option
is specified, the output is escaped according to JSON rules. File Sifter
does not support later loading from either of these formats.

## Summary Statistics

At the end of the run, a footer is printed by default which summarizes
the analysis of the files. If both left and right roots were specified,
it breaks out the statistics by left and right files. It shows
file count and total size for the files processed.

The *entry* lines for directories show the cumulative size of all the files
indexed under the directory. These cumulative sizes are not inlcuded
in the summary statistics because they would cause double-counting.

The *Scanned* line shows all of the files considered (which does not
include those files rejected by **--exclude** or **--regular-only**).

The *Indexed* line shows all of the files that pass the *prefilter* stage
and get loaded into the index. The *Unmatched* line shows all files
that did not have a match on the other side, and the *Matching* line
shows the files that did have a match. The *Output* line shows all of
the files that passed the *postfilter* stage and were printed to the
output (or would be if **--summary** is specified).

    | Run end time: 2017-02-10T02:58:56Z
    | Elapsed time: 732.146µs
    |
    | STATISTICS:  L:Count  L:Size  R:Count  R:Size
    |    Scanned:       26  241927       21  177969
    |    Indexed:       26  241927       21  177969
    |  Unmatched:        7   92064        2   45056
    |   Matching:       19  149863       19  132913
    |     Output:       26  241927       21  177969

## Interactive Status Output

While scanning the file system, File Sifter can print temporary interactive
messages that show the current status of the scan. This includes the
initial scan phase, as well as any required digest scan phases. This
output can be suppressed with the **--quiet** option.

## Character Encodings

All characters are processed assuming UTF-8 encoding. File names with
characters that are not decodable as valid Unicode may produce unexpected
results. Such characters are likely to pass through to the output unchanged,
but comparisons and analysis might have problems.  Note that in some cases,
file systems can be mounted with options that automatically translate
characters which cannot be converted to Unicode to "safe" substitute sequences.

## Platform Specific Differences

On all platforms, path separators are internally represented and output as
"**/**", regardless of what the OS uses.

On all platforms, FSIFT files always use \*NIX-style line endings.

On Windows, the following columns do not currently get populated with meaningful
values: *uid*, *user*, *gid*, *group*, *nlinks* and *device*.

On windows, the *modestr* column contains a simplified approximation of permissions.

On Windows, the program is not currently able to detect the console width and
assumes a fixed value of *80*. This may affect the appearance of interactive
status messages.

# HISTORY

File Sifter is the result of a long evolution of personal utilities that I
wrote over the last 20-odd years to help keep track of files from various
computer systems.

The first utilities were simple Perl scripts that did a simple scan/sort/diff
on directories. Eventually, I wrote an implementation in C++ that used Sqlite
for an internal engine and was largely a superset of the current
implementation. However, it was hard to use the advanced features of that
version, and although I found it very useful and used it for many years, I
was never very happy with it.

I recently decided to pare down the program to its most useful features, clean
up the user interface, drop the embedded database, and port it to Go. The
result of that effort is this rendition of File Sifter.

