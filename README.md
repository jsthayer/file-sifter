
<h1 id="name">NAME</h1>
<p>fsift - File Sifter: Powerful tool to scan, compute digests and compare files. Files can be scanned in directory trees or loaded from previously saved results.</p>
<h1 id="synopsis">SYNOPSIS</h1>
<p><strong>fsift</strong> [ <em>options</em> ] [ <em>left-roots</em>... ] [ <strong>:</strong> <em>right-roots</em>... ]</p>
<h1 id="description">DESCRIPTION</h1>
<p>File Sifter is a utility that has features inspired by the *NIX utilities <em>find</em> and <em>diff</em>, as well as file digest programs such as <em>sha256sum</em>. It can be used for a variety of tasks involving the management of large numbers of files, such as determining which files may be out-of-date in off-site backups.</p>
<h1 id="overview">OVERVIEW</h1>
<p>File Sifter able to scan file systems like <em>find</em>, but instead of immediately printing entries, it loads the information into an internal index.</p>
<p>Once loaded, File Sifter performs analysis on the index. It then optionally sorts the entries and prints them out in tabular format, followed by summary statistics. This output may be saved to an <em>FSIFT</em> file, and entries from that file may be loaded during a later run for comparisons or similar analysis.</p>
<p>The non-option arguments given to File Sifter are called <em>roots</em>. Roots are generally either the top of a directory tree to scan, or a previously saved <em>FSIFT</em> file. Each root is loaded into one of two <em>sides</em> of the index, which are called <em>left</em> and <em>right</em>. During analysis, File Sifter can compare the contents of the left and right sides of the index based on user-specified criteria, and it can generate output and reports about the comparisons. Multiple roots may be specified for each side.</p>
<p>File entries may be subjected to user-defined filters at two points: before loading into the index, or before the final output. These filters allow tailoring the analysis to specific requirements.</p>
<p>Each class of information about a file processed by the program is called a <em>column</em>. Columns are defined for the usual file system attributes such as path, size and permissions. There are also columns for digests of file data. Finally, some columns can by added by the File Sifter program as the result of analysis, such as whether the file <em>matches</em> at least one other file belonging to the opposite side of the index.</p>
<p>Columns are only computed and stored if they are made necessary by the user-specified command line options. For example, the contents of files are only read if the user specifies operations on a digest column.</p>
<h1 id="examples">EXAMPLES</h1>
<ul>
<li>Scan and print information about the current directory tree:</li>
</ul>
<blockquote>
<p><strong>fsift</strong></p>
</blockquote>
<blockquote>
<p>Example output (can also be saved into an <em>FSIFT</em> file):</p>
</blockquote>
<pre><code>    | File Sifter output file - V1 |
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
    |     Output:      5  62463</code></pre>
<ul>
<li>Save the information about a directory tree to an <em>FSIFT</em> file; add md5sum digests:</li>
</ul>
<blockquote>
<p><strong>fsift /path/to/mydir --md5 --out mydir.FSIFT</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift /path/to/mydir -5omydir.FSIFT</strong></p>
</blockquote>
<ul>
<li>Compare the contents of a directory tree to a previously saved <em>FSIFT</em> file:</li>
</ul>
<blockquote>
<p><strong>fsift t.FSIFT : t/ --diff</strong> --md5 --sort path</p>
</blockquote>
<blockquote>
<p>Example output:</p>
</blockquote>
<pre><code>    | File Sifter output file - V1 |
    | Command line: t.FSIFT : t -5 -sp
    | Current working directory: /home/user/temp
    | Compare keys: md5,path,size,mtime,modestr
    | Sort keys: path
    | Evaluated columns: path,size,mtime,side,matched,membership,modestr,md5
    | Run start time: 2017-02-10T19:39:18Z
    | 
    | Columns: membership,modestr,size,mtime,md5,path
    | 
      &gt;!  drwxrwxr-x  3  2017-02-10T19:38:00Z  \-                                ./
      &lt;!  drwxrwxr-x  5  2017-02-10T19:36:24Z  \-                                ./
      &gt;!  -rw-rw-r--  0  2017-02-10T19:38:00Z  d41d8cd98f00b204e9800998ecf8427e  added.txt
      &lt;=  drwxrwxr-x  0  2017-02-10T19:35:39Z  \-                                dir/
      &gt;=  drwxrwxr-x  0  2017-02-10T19:35:39Z  \-                                dir/
      &lt;=  -rw-rw-r--  0  2017-02-10T19:35:39Z  d41d8cd98f00b204e9800998ecf8427e  dir/no-change.txt
      &gt;=  -rw-rw-r--  0  2017-02-10T19:35:39Z  d41d8cd98f00b204e9800998ecf8427e  dir/no-change.txt
      &lt;=  -rw-rw-r--  0  2017-02-10T19:34:53Z  d41d8cd98f00b204e9800998ecf8427e  no-change.txt
      &gt;=  -rw-rw-r--  0  2017-02-10T19:34:53Z  d41d8cd98f00b204e9800998ecf8427e  no-change.txt
      &lt;!  -rw-rw-r--  0  2017-02-10T19:36:24Z  d41d8cd98f00b204e9800998ecf8427e  remove-me.txt
      &lt;!  -rw-rw-r--  5  2017-02-10T19:36:06Z  9387a56cd6f2b6bd4a42f07329b93fca  update-me.txt
      &gt;!  -rw-rw-r--  3  2017-02-10T19:37:52Z  a2141a5ddbd5295d9ec96788790cf1bf  update-me.txt
    | 
    | Run end time: 2017-02-10T19:39:18Z
    | Elapsed time: 1.03466ms
    | 
    | STATISTICS:  L:Count  L:Size  R:Count  R:Size
    |    Scanned:        6       5        6       3
    |    Indexed:        6       5        6       3
    |  Unmatched:        3       5        3       3
    |   Matching:        3       0        3       0
    |     Output:        6       5        6       3</code></pre>
<ul>
<li>Compare the contents of a directory tree to a previously saved <em>FSIFT</em> file, showing just differences:</li>
</ul>
<blockquote>
<p><strong>fsift t.FSIFT : t/ --diff</strong></p>
</blockquote>
<ul>
<li>Find which direct child subdirectories are using the most storage:</li>
</ul>
<blockquote>
<p><strong>fsift top/dir --sort size --postfilter filetype=d --postfilter 'depth&lt;2'</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift top/dir -ss -ff=d -fd\&lt;2</strong></p>
</blockquote>
<ul>
<li>Find which git repositories are most in need of garbage collection:</li>
</ul>
<blockquote>
<p><strong>fsift my/projects --prefilter 'path *=**/.git/objects/' --sort nlinks --columns +nlinks</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift my/projects '-ep*=**/.git/objects/' -sL -c+L</strong></p>
</blockquote>
<ul>
<li>List what kinds of files are in a directory tree by unique extension:</li>
</ul>
<blockquote>
<p><strong>fsift top/dir --columns extension --key extension --postfilter redunidx=1</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift top/dir -cx -kx -fI=1</strong></p>
</blockquote>
<ul>
<li>Assume lists of previously archived files have been saved in a set of <em>FSIFT</em> files. Before decommissioning a disk, scan it to check against the archives to find any files that may need to be added to archives:</li>
</ul>
<blockquote>
<p><strong>fsift archive-index/*.FSIFT : path/to/disk --membership R --key base,md5</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift archive-index/*.FSIFT : path/to/disk -mR -kb5</strong></p>
</blockquote>
<ul>
<li>Find all files that have redundant data content:</li>
</ul>
<blockquote>
<p><strong>fsift top/dir --postfilter 'redundancy &gt;1' --key md5 --columns +redundancy --sort size --regular-only</strong></p>
</blockquote>
<blockquote>
<blockquote>
<p><em>or</em></p>
</blockquote>
</blockquote>
<blockquote>
<p><strong>fsift top/dir -fr\&gt;1 -k5 -c+r -Rss</strong></p>
</blockquote>
<h1 id="options">OPTIONS</h1>
<p>Options may be freely intermixed with roots. Most options have both a long version and a short version. Short boolean options may be concatenated (such as -qS), and short options which take an argument may have the argument concatenated to the option. Long options can have the argument joined with an &quot;<strong>=</strong>&quot; character. The following are all equivalent: <strong>-st</strong>, <strong>-s t</strong>, <strong>-smtime</strong>, <strong>--sort mtime</strong>, <strong>--sort=mtime</strong>, <strong>--sort t</strong>.</p>
<dl>
<dt><strong>:</strong></dt>
<dd>A single colon on the command line is a special marker that divides the left-side roots from the right-side roots. If no colon is present, all roots are assigned to the left side. Otherwise, any roots on the command line <em>after</em> the colon are assigned to the right side.
</dd>
</dl>
<h1 id="field-selection-comparing-and-sorting">Field selection, comparing and sorting:</h1>
<dl>
<dt><strong>-c</strong>, <strong>--columns=COLUMNS</strong></dt>
<dd>Select output columns. The default is &quot;modestr,size,mtime,path&quot; (alternatively, &quot;ostp&quot;). If there are roots on both sides, then &quot;membership&quot; is also added to the default column set.
</dd>
<dt><strong>-s</strong>, <strong>--sort=COLUMNS</strong></dt>
<dd>Sort output using these fields, in order of precedence. By default, the output is not sorted. For this option, any column name may be precede by a &quot;<strong>/</strong>&quot; character to sort in inverse order.
</dd>
<dt><strong>-k</strong>, <strong>--key=COLUMNS</strong></dt>
<dd>Specify which fields used to compare files on each side for equivalence. The default is &quot;modestr,size,mtime,path&quot;.
</dd>
<dt><strong>-5</strong>, <strong>--md5</strong></dt>
<dd>Shortcut to add md5 column to compare key and output.
</dd>
<dt><strong>-2</strong>, <strong>--sha256</strong></dt>
<dd>Shortcut to add sha256 column to compare key and output.
</dd>
<dt><strong>-A</strong>, <strong>--sha512</strong></dt>
<dd>Shortcut to add sha512 column to compare key and output.
</dd>
<dt><strong>-1</strong>, <strong>--sha1</strong></dt>
<dd>Shortcut to add sha1 column to compare key and output.
</dd>
</dl>
<h1 id="pre-analysis-filtering">Pre-analysis filtering:</h1>
<dl>
<dt><strong>-e</strong>, <strong>--prefilter=FILTER-EXP</strong></dt>
<dd>Filter to screen files before they are loaded into the index. Multiple filters may be specified.
</dd>
<dt><strong>-b</strong>, <strong>--base-match=GLOB-PAT</strong></dt>
<dd>Filter files by base name glob pattern. Shortcut for <strong>--prefilter 'base*=*GLOB-PAT*'</strong>.
</dd>
<dt><strong>-x</strong>, <strong>--exclude=GLOB-PAT</strong></dt>
<dd>Exclude file system files and/or directory trees if their path matches the given glob pattern. If a directory name matches an exclude pattern, do not descend into it. This option only applies to scans of file systems; it does not filter entries loaded from FSIFT files.
</dd>
<dt><strong>-R</strong>, <strong>--regular-only</strong></dt>
<dd>Only load regular files into the index while scanning. This option only applies to scans of file systems; it does not filter entries loaded from FSIFT files.
</dd>
<dt><strong>-L</strong>, <strong>--follow-links</strong></dt>
<dd>Follow symbolic links while scanning the file system. By default, when a symbolic link is found, an entry is created about the symbolic link itself. When this option is in effect, an entry is created with information about the link <em>target</em>. If a symbolic link points to a directory, the program will descend into that directory when this option is in effect.
</dd>
<dt><strong>-X</strong>, <strong>--xdev</strong></dt>
<dd>Stay within one file system. If a subdirectory is mount point, don't descend into it. (This option currently does not work on Windows systems.)
</dd>
</dl>
<h1 id="post-analysis-filtering">Post-analysis filtering:</h1>
<dl>
<dt><strong>-f</strong>, <strong>--postfilter=FILTER-EXP</strong></dt>
<dd>After analysis, any entries rejected by this filter are not output. Multiple filters may be specified.
</dd>
<dt><strong>-m</strong>, <strong>--membership=CHARS</strong></dt>
<dd>Filter output by membership code. The code must be a string containing only a subset of the characters <strong>l</strong>,<strong>r</strong>,<strong>L</strong> or <strong>R</strong>. The <strong>L</strong> and <strong>l</strong> codes only allow entries from left-side roots, and the other two codes allow entries from the right-side roots. The lower case codes only allow files that were <em>matched</em> by files on the other side, and the upper case codes only allow files that were <em>unmatched</em> by files on the other side. For example, the option <strong>--membership=Lr</strong> only prints files from the left that were unmatched, as well as files from the right that were matched.
</dd>
<dt><strong>-d</strong>, <strong>--diff</strong></dt>
<dd>Show unmatched entries only. This is a shortcut for <strong>--membership=LR</strong>. (Which in turn is a shortcut for <strong>-f OR -f 'm=&lt;!' -f 'm=&gt;!</strong>'.)
</dd>
<dt><strong>--nodetect</strong></dt>
<dd>Don't try to detect whether regular files specified as roots are FSIFT files. By default, if a file looks like it is an FSIFT file, entries are parsed and loaded from the contents of the file instead of loading information about the file itself. If this option is in effect, the detection step is not done, no FSIFT parsing is attempted, and for any regular file given as a root, an entry is created about the file itself.
</dd>
</dl>
<h1 id="output-formatting">Output formatting</h1>
<dl>
<dt><strong>-o</strong>, <strong>--out=PATH</strong></dt>
<dd>Output to the given file path instead of the default stdout.
</dd>
<dt><strong>-Y</strong>, <strong>--verify</strong></dt>
<dd>Checks that all left entries under left-sided roots are matched by at least one entry under a right-sided root. If the left side root is a FSIFT file, this is somewhat similar to running a program like <strong>sha1sum -c</strong>. If a mismatched left-side file is found, the program exits with a nonzero status code.
</dd>
<dt><strong>-S</strong>, <strong>--summary</strong></dt>
<dd>Only the header and footer summary info. No file entries are output.
</dd>
<dt><strong>-p</strong>, <strong>--plain</strong></dt>
<dd>Only output the file entries. No header or footer summary info is printed.
</dd>
<dt><strong>-0</strong>, <strong>--plain0</strong></dt>
<dd>Like 'plain', but also separate all output fields with ASCII <strong>NUL</strong> characters. Newlines betweeen file entries are also replaced with <strong>NUL</strong> characters. The usual FSIFT format escaping is <em>not</em> done.
</dd>
<dt><strong>-G</strong>, <strong>--group-nums</strong></dt>
<dd>For numeric output, separate groups of three decimal digits with commas.
</dd>
<dt><strong>-N</strong>, <strong>--ignore-nulls</strong></dt>
<dd>File entry fields get compared during filtering, membership analysis and sorting. By default, if any comparison uses a field that is missing (in other words, <em>null</em>), then an error message will be generated and the program will have a nonzero exit status. If this option is in effect, then no error is generated. In any case, all <em>null</em> comparisons are considered to have a false result.
</dd>
<dt><strong>-J</strong>, <strong>--json-out</strong></dt>
<dd>Output file entries in JSON format. The output will be an array of JSON objects, each with a key-value pair for each column defined in the output. Numeric fields are output as JSON integers, string fields are output as JSON strings, and missing fields are JSON <strong>null</strong> values. No header or footer information is output.
</dd>
<dt><strong>-Z</strong>, <strong>--out-zone</strong></dt>
<dd>Format output times for given location. The default is UTC. Locations can be specified as fixed offsets like &quot;<strong>+06:00</strong>&quot;, or as locations recognized by the Go language <em>time</em> package, such as &quot;<strong>Local</strong>&quot; or &quot;<strong>America/Chicago</strong>&quot;. Times are always output in RFC3339 format, such as &quot;<strong>2017-02-03T04:52:13Z</strong>&quot;.
</dd>
<dt><strong>-v</strong>, <strong>--verbose</strong></dt>
<dd>Increase verbosity.
</dd>
<dt><strong>-q</strong>, <strong>--quiet</strong></dt>
<dd>Decrease verbosity.
</dd>
<dt><strong>-V</strong>, <strong>--version</strong></dt>
<dd>Show program version and exit.
</dd>
<dt><strong>-h</strong>, <strong>--help</strong></dt>
<dd>Print a help message and exit.
</dd>
</dl>
<h1 id="column-codes">COLUMN CODES</h1>
<p>Each column has a full name and a single-character short name alias.</p>
<p>For options that take a list of columns as an argument, the columns can be specified with a comma-separated list of names, long or short. If no commas are in the argument and it does not match a long name, then the program tries assuming that the argument is a concatenation of short name characters. The argument must not contain whitespace.</p>
<p>For example, <strong>size,mtime,path</strong> can be shortened to <strong>stp</strong>.</p>
<dl>
<dt><strong>p path</strong></dt>
<dd>The path of this file relative to the given root.
</dd>
<dt><strong>b base</strong></dt>
<dd>The base name of this file.
</dd>
<dt><strong>x ext</strong></dt>
<dd>The extension of this filename, if any.
</dd>
<dt><strong>D dir</strong></dt>
<dd>The directory part of the 'path' field.
</dd>
<dt><strong>d depth</strong></dt>
<dd>How many subdirectories this file is below its root.
</dd>
<dt><strong>s size</strong></dt>
<dd>Regular files: size in bytes. Dirs: cumulative size. Other: 0. Note that the cumulative sizes of directories do not figure into statistics roundups.
</dd>
<dt><strong>t mtime</strong></dt>
<dd>Modification time as a string in RFC3339 format.
</dd>
<dt><strong>T mstamp</strong></dt>
<dd>Modification time as seconds since Jan 1, 1970.
</dd>
<dt><strong>V device</strong></dt>
<dd>The ID of the device this file resides on.
</dd>
<dt><strong>S side</strong></dt>
<dd>The <em>side</em> of this file's root: <strong>0</strong>=left <strong>1</strong>=right.
</dd>
<dt><strong>M matched</strong></dt>
<dd>True if this file matches any file from the <em>other</em> side, according to the fields in the <strong>--key</strong> option.
</dd>
<dt><strong>m membership</strong></dt>
<dd><p>Visual representation of 'side' and 'matched' columns:</p>
<pre><code>side  matched  membership
0     0        &quot;&lt;!&quot;
0     1        &quot;&lt;=&quot;
1     0        &quot;&gt;!&quot;
1     1        &quot;&gt;=&quot;</code></pre>
</dd>
<dt><strong>r redundancy</strong></dt>
<dd>Count of all files on <em>this</em> side matching this file.
</dd>
<dt><strong>I redunidx</strong></dt>
<dd>Ordinal of this file amongst equivalents on <em>this</em> side. If a postfilter is set to only select entries where the <strong>redunidx</strong> value is <strong>1</strong>, then only one entry per group of equivalent files will be output on each side, so such a filter can be used to enumerate unique values.
</dd>
<dt><strong>o modestr</strong></dt>
<dd>Mode and permission bits as a human readable string, in *NIX-like format. See the documentation of the Go language <em>os.FileMode</em> object for exact details on the format.
</dd>
<dt><strong>f filetype</strong></dt>
<dd>A single-character code indicating the type of this file. This is usually the same as the first character of <strong>modestr</strong>, except that for regular files '<strong>-</strong>' is changed to '<strong>f</strong>', and for block devices '<strong>D</strong>' is changed to '<strong>b</strong>'.
</dd>
<dt><strong>U uid</strong></dt>
<dd>The user ID of this file's owner.
</dd>
<dt><strong>G gid</strong></dt>
<dd>The group ID of this file's group.
</dd>
<dt><strong>u user</strong></dt>
<dd>The name of this file's owner.
</dd>
<dt><strong>g group</strong></dt>
<dd>The name of this file's group.
</dd>
<dt><strong>L nlinks</strong></dt>
<dd>The number of hard links to this file.
</dd>
<dt><strong>3 crc32</strong></dt>
<dd>The CRC32 checksum of this file. <strong>Note</strong>: for all checksum and digest fields, directories and other nonregular files get an empty string for a value instead of <em>null</em>. This avoids excessive <em>null</em> comparison errors when these fields are compared. <strong>Important:</strong> The crc32 checksum should not be used for security purposes.
</dd>
<dt><strong>1 sha1</strong></dt>
<dd>The SHA1 digest of this file. <strong>Important:</strong> The sha1 digest probably should not be used for security purposes.
</dd>
<dt><strong>2 sha256</strong></dt>
<dd>The SHA256 digest of this file.
</dd>
<dt><strong>A sha512</strong></dt>
<dd>The SHA512 digest of this file.
</dd>
<dt><strong>5 md5</strong></dt>
<dd>The MD5 digest of this file. <strong>Important:</strong> The md5 digest should not be used for security purposes.
</dd>
</dl>
<h1 id="filter-specifications">FILTER SPECIFICATIONS</h1>
<p><em>Filters</em> allow the rejection of file entries based on user-defined criteria.</p>
<p>Any column can be used in a filter specification. The general filter syntax is as follows:</p>
<p>&lt;<strong>column-name</strong>&gt;&lt;<strong>operator</strong>&gt;&lt;<strong>value</strong>&gt;</p>
<p>The column name may be any long or short column identifier. The value is an arbitrary string for comparison to an entry's value. There should be no whitespace between the operator and the value.</p>
<p>The supported operators are:</p>
<dl>
<dt><strong>=</strong></dt>
<dd>Equals
</dd>
<dt><strong>&lt; &lt;= &gt; &gt;=</strong></dt>
<dd>Ordered comparisons (numeric or lexicographic, depending on the column type)
</dd>
<dt><strong>*=</strong></dt>
<dd>Glob pattern match (see below).
</dd>
<dt><strong>~=</strong></dt>
<dd>Regular expression match. For detailed syntax, see the Go language <em>regexp</em> package documentation.
</dd>
<dt><strong>.isnull</strong></dt>
<dd>Matches if the value is missing in this file entry
</dd>
<dt><strong>!= !*= !~= !.isnull</strong></dt>
<dd>Negated versions of the above operators
</dd>
</dl>
<h2 id="combining-filters">Combining Filters</h2>
<p>Multiple filters of the same type may be specified. In addition to the above filters, there are two special combining filters:</p>
<p><strong>or</strong> Matches if either child filter matches.</p>
<p><strong>and</strong> Matches if both child filters match.</p>
<p>Combining filters act as binary operators between other filters. They are applied in <em>prefix</em> order (also known as <em>normal</em> Polish notation). In other words, a combining filter appears directly in front of its two child filters.</p>
<p>For example, the following set of filters selects all the files larger than one megabyte that belong to Jack or Jill:</p>
<p><strong>-f and -f 'size&gt;1000000' -f or -f user=jack -f user=jill</strong></p>
<p>If there are any filters of a given type remaining after combining filters have been assigned children, then the remaining filters are assigned to implicit <strong>and</strong> filters. So for the common case of multiple filters defined with no combining filters, they are <em>and</em>ed together.</p>
<h2 id="glob-patterns">Glob Patterns</h2>
<p>Glob patterns are similar to glob expansion in shell interpreters: &quot;<strong>?</strong>&quot; matches any single character except &quot;<strong>/</strong>&quot;. &quot;<strong>*</strong>&quot; matches any number of non-&quot;<strong>/</strong>&quot; characters. In this implementation, the contents of brackets <strong>[</strong>...<strong>]</strong> are fed directly to the underlying regular expression evaluator; the result is similar to many glob implementations, but there are some differences. See the Go language <em>regexp</em> package documentation for details.</p>
<p>As a special case, a <strong>**</strong> matches any number of characters, <em>including</em> &quot;<strong>/</strong>&quot;. This can be used to match entire segments of file paths. For example, the filter: <strong>path*=**/.config/**</strong> will select all files under <em>.config</em> directories below the top level.</p>
<p>When using the <strong>**</strong> operator, note that directory paths are always stored with a trailing &quot;<strong>/</strong>&quot; character. Also note that files directly under the root will not have a &quot;<strong>/</strong>&quot; preceding them. If this creates problems with glob matching entire paths, a regular expression pattern may be a more flexible alternative.</p>
<h2 id="pruning-filters">Pruning Filters</h2>
<p>If an entire filter specification is prefixed with a &quot;<strong>/</strong>&quot; character, that filter becomes a <em>pruning</em> filter. This only affects the <strong>--prefilter</strong> option, and then only when scanning file systems (not loading FSIFT files). The combining filters <strong>and</strong> and <strong>or</strong> cannot be prefixed in this way.</p>
<p>When a pruning filter is used, if the filter rejects a directory, then File Sifter will not descend into that directory to scan its contents. (By default, directories are scanned even when rejected by a prefilter because prefilters are often looking for certain files without regard to the properties of their parent directories.)</p>
<p>Example: only look in the &quot;data&quot; subdirectory, skipping any other directories below the current root:</p>
<p><strong>fsift . --prefilter '/path*=data/**'</strong></p>
<p>By contrast, the non-pruning version of the same filter would scan any other subdirectories below the current root, but not load any of those files into the index, and it could possibly take substantially more time. It should output the same entries, but with different statistics info:</p>
<p><strong>fsift . --prefilter 'path*=data/**'</strong></p>
<h1 id="other-features">OTHER FEATURES</h1>
<h2 id="fsift-files">FSIFT Files</h2>
<p>The output of File Sifter may be saved to a file (an extension of <strong>.FSIFT</strong> is recommended). If this file is later specified as a root during another run of File Sifter, then by default entries will parsed and loaded from that file.</p>
<p>Note that if the header information or the file entries are suppressed using command line options (such as <strong>--summary</strong> or <strong>--plain</strong>), then the output will not be useful for loading later.</p>
<h3 id="syntax">Syntax</h3>
<p><em>FSIFT</em> files are text files with two kinds of lines: <em>directive</em> lines and <em>entry</em> lines. Directive lines start with a <strong>|</strong> character.</p>
<p>The file starts with a <em>directive</em> line identifying it as an <em>FSIFT</em> file. After that is a group of informational header directives. The only other directive that is relevant to parsing <em>FSIFT</em> files is the <strong>Columns</strong> directive. This specifies which fields are present in each <em>entry</em> line. The <strong>Columns</strong> directive must appear before the first <em>entry</em> line. All other <em>directive</em> lines are ignored by the parser.</p>
<p><em>Entry</em> lines follow the header, with one <em>entry</em> line per output file entry. These lines start with a space, and are followed by one or more fields. The number of fields matches the number of names in the <strong>Columns</strong> directive, and each item is the field information from the corresponding column name.</p>
<p>After the entries, a footer is output. This contains a set of <em>directive</em> lines with summary information such as run time, file and byte counts.</p>
<p>In <em>entry</em> lines, fields are separated by space characters. Certain characters within a field are escaped with backslashes: spaces, newlines, carriage returns and backslashes. In addition, there are two special escape sequences: <strong>\-</strong> indicates a zero-length string, and <strong>\~</strong> indicates a missing value (called a <em>null</em>). These escapes are removed when <em>FSIFT</em> files are parsed.</p>
<p>As a special exception, space characters are <em>not</em> escaped in the last column. This is possible because the parser knows that no other fields will follow this one before the next newline. In the common case where <em>path</em> is the last field on each line, this makes the output look cleaner when there are spaces in file names. (However, even in the last column, any spaces at the beginning or end of a field are still escaped.)</p>
<p>When the <strong>--plain0</strong> option is specified, there is no escaping performed, and all data is separated by ASCII <strong>NUL</strong> characters. Then the <strong>--json</strong> option is specified, the output is escaped according to JSON rules. File Sifter does not support later loading from either of these formats.</p>
<h2 id="summary-statistics">Summary Statistics</h2>
<p>At the end of the run, a footer is printed by default which summarizes the analysis of the files. If both left and right roots were specified, it breaks out the statistics by left and right files. It shows file count and total size for the files processed.</p>
<p>The <em>entry</em> lines for directories show the cumulative size of all the files indexed under the directory. These cumulative sizes are not included in the summary statistics because they would cause double-counting.</p>
<p>The <em>Scanned</em> line shows all of the files considered (which does not include those files rejected by <strong>--exclude</strong> or <strong>--regular-only</strong>). The <em>Indexed</em> line shows all of the files that pass the <em>prefilter</em> stage and get loaded into the index.</p>
<p>The <em>Unmatched</em> line shows all files that did not have a match on the other side, and the <em>Matching</em> line shows the files that did have a match. The previous two lines are only output if there were roots on both sides. The <em>Output</em> line shows all of the files that passed the <em>postfilter</em> stage and were printed to the output (or if <strong>--summary</strong> is specified, would have been output).</p>
<pre><code>| Run end time: 2017-02-10T02:58:56Z
| Elapsed time: 732.146µs
|
| STATISTICS:  L:Count  L:Size  R:Count  R:Size
|    Scanned:       26  241927       21  177969
|    Indexed:       26  241927       21  177969
|  Unmatched:        7   92064        2   45056
|   Matching:       19  149863       19  132913
|     Output:       26  241927       21  177969</code></pre>
<h2 id="interactive-status-output">Interactive Status Output</h2>
<p>While scanning the file system, File Sifter can print temporary interactive messages that show the current status of the scan. This includes the initial scan phase, as well as any required digest scan phases. This output can be suppressed with the <strong>--quiet</strong> option.</p>
<h2 id="character-encodings">Character Encodings</h2>
<p>All characters are processed assuming UTF-8 encoding. File names with characters that are not decodable as valid Unicode may produce unexpected results. Such characters are likely to pass through to the output unchanged, but comparisons and analysis might have problems. Note that in some cases, file systems can be mounted with options that automatically translate characters which cannot be converted to Unicode to &quot;safe&quot; substitute sequences.</p>
<h2 id="platform-specific-differences">Platform Specific Differences</h2>
<p>On all platforms, path separators are internally represented and output as &quot;<strong>/</strong>&quot;, regardless of what the OS uses.</p>
<p>On all platforms, FSIFT files always use *NIX-style line endings.</p>
<p>On Windows, the following columns do not currently get populated with meaningful values: <em>uid</em>, <em>user</em>, <em>gid</em>, <em>group</em>, <em>nlinks</em> and <em>device</em>.</p>
<p>On windows, the <em>modestr</em> column contains a simplified approximation of permissions.</p>
<p>On Windows, the program is not currently able to detect the console width and assumes a fixed value of <em>80</em>. This may affect the appearance of interactive status messages.</p>
<h1 id="history">HISTORY</h1>
<p>File Sifter is the result of a long evolution of personal utilities that I wrote over the years to help keep track of files from various computer systems.</p>
<p>The first utilities were simple Perl scripts that did a simple scan/sort/diff on directories. Eventually, I wrote an program in C++ that used SQLite for an internal engine that had features somewhat similar to this implementation. However, it was hard to use the SQL-oriented features of that version, and although I found it very useful and used it for many years, I was never very happy with it.</p>
<p>I recently decided to pare down the program to its most useful features, clean up the user interface, drop the embedded database, and port it to Go. The result of that effort is this rendition of File Sifter.</p>

