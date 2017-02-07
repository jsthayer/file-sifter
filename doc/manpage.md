% FSIFT(1)
% John Thayer
% January 2017

# NAME

fsift - file sifter:

# SYNOPSIS

**fsift** [ *options* ] [ *left-roots*... ] [ **:** *right-roots*... ]

# DESRIPTION

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


# OVERVIEW

## Roots

# OPTIONS

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

# FILTER SPECIFICATIONS

## Glob Patterns

# COLUMN CODES

COLUMNS codes (example: 'size,time,path' can be shortened to 'stp'):

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
 ~ The CRC32 digest of this file. **Note**: for all digest fields,
   directories and other nonregular files get an empty string for
   a digest value (not *null*). This avoids excessive *null* comparison
   errors when digest fields are compared.

**1    sha1**
 ~ The SHA1 digest of this file.

**2    sha256**
 ~ The SHA256 digest of this file.

**A    sha512**
 ~ The SHA512 digest of this file.

**5    md5**
 ~ The MD5 digest of this file.

# OTHER FEATURES

## FSIFT Files

## Character Encodings

## Summary Statistics


## Windows
