# Periscope [![Build Status](https://github.com/anishathalye/periscope/actions/workflows/ci.yml/badge.svg)](https://github.com/anishathalye/periscope/actions/workflows/ci.yml)
<!--
Other useful stuff:

https://goreportcard.com/report/github.com/anishathalye/periscope
-->

Periscope gives you "duplicate vision" to help you organize and de-duplicate your files without losing data.

<p align="center">
<img src="https://raw.githubusercontent.com/anishathalye/assets/master/periscope/demo.gif" width="636" alt="Periscope demo">
</p>

Periscope (`psc`) works differently from most other duplicate file finders. It
is designed to be used _interactively_: Periscope will help you explore the
filesystem, understand which files are duplicated, determine where duplicate
copies live, and safely delete duplicates without losing data.

Following a `psc scan`, Periscope lets you navigate and explore your filesystem
with the workflow you're already used to &mdash; using your shell and commands
like `cd`, `ls`, `tree`, and so on &mdash; while providing additional
_duplicate-aware commands_ that mirror core filesystem utilities. For example,
`psc ls` gives a directory listing that highlights duplicates, and `psc rm`
deletes files only if a duplicate exists elsewhere. This makes it easy to
understand how data is organized (and duplicated), reorganize files, and delete
duplicates without worrying about accidentally losing data.

<p align="center">
<a href="#workflow">Workflow</a> &middot; <a href="#commands">Commands</a> &middot; <a href="#installation">Installation</a> &middot; <a href="#contributing">Contributing</a>
</p>

## Workflow

**Find duplicates**

Start with `psc scan` to scan folders for duplicates. Once you run this, you
shouldn't need to run it again while looking at and deleting duplicates, unless
you move files around. If you delete files manually (rather than with `psc rm`),
you can make Periscope detect deletions with `psc refresh`, which runs much
faster than a full scan. `psc scan` is incremental, so if you want to scan a new
directory or re-analyze one that was already scanned, you can always run the
command again.

**Understand duplicates**

You can get a high-level understanding of how many duplicates you have and
where they are located:

- `psc summary` gives statistics on duplicate files
- `psc report` shows a full list of duplicates, sorted by file size

After identifying areas to explore with `psc report`, you can navigate to those
directories in your shell with `cd`, and then you can use Periscope commands to
understand duplicates:

- `psc ls` gives a duplicate-aware directory listing (optionally recursively,
  with the `-R` flag)
- `psc info` shows information on a specific file (and its duplicates)

**Delete duplicates**

You can use the `psc rm` command to delete duplicates. You can think of it like
a safe version of `rm`: it will not let you delete files unless there are
duplicate copies elsewhere. A `psc rm -r` will recursively delete duplicates
but not unique files. A `psc rm --contained <path>` will delete duplicates only
if a copy is contained in the given folder.

**Remove duplicate database**

When you're done with a Periscope session, you can delete the duplicate
database with `psc finish`.

## Commands

Run `psc help` to see the full list of commands and `psc help [command]` to see
help on a specific command.

**`psc scan` scans for duplicates**

Scans paths for duplicates and populates the database with information about
duplicates. Scans the current directory if given no argument. Scanning is
incremental; if you want to start from scratch, run `psc finish` first.

**`psc refresh` removes deleted files from the database**

Removes deleted files from the duplicate database. `psc rm` does this
automatically, so this command only needs to be used if you use some other
program (e.g. coreutils `rm`) and want to remove missing files from the
database. This command does not re-analyze files, so if you've made substantial
changes to the filesystem, like moving files around or adding new files, it's
best to do a `psc scan` of the relevant directories.

**`psc finish` deletes the duplicate database**

Deletes the duplicate database. Once you're done using Periscope, it's good to
use this command to delete the duplicate database, so it doesn't waste space on
disk.

**`psc summary` reports statistics**

Prints statistics about the duplicate database, such as number of duplicate
files and the amount of space duplicates consume.

**`psc report` reports scan results**

Lists all duplicates in the duplicate database, sorted by file size. Because
this list is usually large, it's helpful to pipe the output to a pager, e.g.
`psc report | less`.

**`psc export` exports scan results**

Exports information about duplicates in a machine-readable format (default
JSON). **This is the only output from Periscope that other programs should
consume.** Future versions of Periscope may add to the information that's
included in the dump, but the layout of existing data will not change.

**`psc ls` lists a directory**

Lists files and folders in the given directory (or the current directory, if
none is given). This command shows the number of duplicates that each file has:
1 means that there is a single duplicate elsewhere in the filesystem; if a file
has no duplicates, the number is omitted. Directories are tagged with a 'd',
and special files are tagged with a character describing their type, e.g. 'p'
for named pipes. `-a` shows hidden files. `-d` lists only duplicates, while
`-u` lists only unique files. `-v` lists all duplicates of every file, and `-r`
shows the path to the duplicate as a relative path instead of an absolute path.
`-R` lists files recursively; this flag combines well with the `-d` flag, to
list only duplicate files recursively contained in a given directory. `-f`
shows only files in the listing; this flag combines well with the `-R` flag
along with `-u` or `-d`, to recursively list all the unique or duplicated files
while omitting names of irrelevant directories in the output.

**`psc tree` lists all duplicates in a given directory**

Lists all files recursively contained in the given directory (or the current
directory, if none is given) that have a duplicate file elsewhere. This command
hides hidden files and folders by default; the `-a` flag includes hidden files.

This command shows a "flattened" representation; in most cases, a `psc ls -Rd`
is more useful.

**`psc info` inspects a file**

Shows information about a single file's duplicates. Like with `psc ls`, the
`-r` flag shows the path to the duplicate as a path relative to the given file.

**`psc rm` deletes duplicates**

Deletes duplicates but not unique files; no way of invoking this command will
delete unique files. This command makes use of the database, but it
double-checks files and their copies before it deletes anything, so a stale
duplicate database will not result in data loss. The `-n` flag will perform a
dry run, listing files that would be deleted but not actually deleting
anything. `-r` deletes duplicates recursively. The `--contained <path>`
argument gives more fine-grained control over deletion: files are only deleted
if they have a duplicate _in the given location_ (or locations, you can specify
multiple by specifying additional `--contained` options). This is useful, for
example, for deleting files from a "to organize" directory only if they are
also contained in the "organized" directory, as in the demo video above. By
default, `psc rm` does not delete any files when it's given a set where there
are no duplicates outside the set: for example, if files "/a/x1" and "/a/x2"
are duplicates, recursively removing "/a" will leave both files untouched.
Passing the `--arbitrary` flag will result in such duplicates being handled by
arbitrarily choosing one file to save and deleting the rest.

## Installation

**Install with [Homebrew](https://brew.sh/) (on macOS):**

```bash
brew install periscope
```

**Download a binary release:**
[Periscope releases](https://github.com/anishathalye/periscope/releases).

Periscope has binary releases for macOS and Linux. It has not been tested on
Windows.

**Install from source with `go install`:**

```bash
go install -v github.com/anishathalye/periscope/cmd/psc@latest
```

Periscope depends on go-sqlite3, which uses cgo, so you need a C compiler
present in your path. You might also need to set `CGO_ENABLED=1` if you have it
disabled otherwise.

## Contributing

Bug reports, feature requests, feedback on the tool or documentation, and pull
requests are all appreciated. If you are planning on making substantial changes
that you hope to have merged, it is highly recommended that you first open an
issue to discuss your proposed change.

## License

Copyright (c) Anish Athalye (me@anishathalye.com). Released under GPLv3.
See [LICENSE.txt](LICENSE.txt) for details.
