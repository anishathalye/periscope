package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/afero"
)

type LsOptions struct {
	All       bool
	Verbose   bool
	Duplicate bool
	Unique    bool
	Relative  bool
	Recursive bool
}

func (ps *Periscope) Ls(paths []string, options *LsOptions) herror.Interface {
	var herr herror.Interface
	multi := len(paths) > 1
	firstToShow := true
	for _, p := range paths {
		didShow, err := ps.ls1(p, options, multi, true, firstToShow)
		if didShow {
			firstToShow = false
		}
		if err != nil {
			herr = err
		}
		if herr != nil && !herror.IsSilent(herr) {
			return herr
		}
	}
	return herr
}

func (ps *Periscope) ls1(path string, options *LsOptions, multi, top, firstToShow bool) (bool, herror.Interface) {
	absPath, _, herr := ps.checkFile(path, false, true, "list", false, false)
	if herr != nil {
		return false, herr
	}
	files, err := afero.ReadDir(ps.fs, absPath)
	if os.IsPermission(err) {
		fmt.Fprintf(ps.errStream, "cannot access '%s': permission denied\n", path)
		return false, herror.Silent()
	}
	if err != nil {
		return false, herror.Internal(err, "")
	}
	w := tabwriter.NewWriter(ps.outStream, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	var recurseDirs []string
	showAny := false
	for _, file := range files {
		if file.Name()[0] != '.' || options.All {
			didShow, isDirectory, err := ps.list1(w, file, absPath, options)
			if err != nil {
				return false, err
			}
			showAny = showAny || didShow
			if isDirectory && options.Recursive {
				recurseDirs = append(recurseDirs, file.Name())
			}
		}
	}
	withFilter := options.Duplicate || options.Unique
	// only show directory name if any of the following are true:
	// - multiple directories are listed in the command invocation, and
	//   this is one of those directories (this is the top-level invocation)
	// - we're in recursive mode, and any of:
	//     - there are no filters
	//     - we are going to list a non-zero number of files
	echoDir := multi && top || options.Recursive && (!withFilter || showAny)
	if echoDir {
		if !firstToShow {
			fmt.Fprint(ps.outStream, "\n")
		}
		fmt.Fprintf(ps.outStream, "%s:\n", path)
	}
	w.Flush()

	didShow := echoDir || showAny
	for _, dir := range recurseDirs {
		recDidShow, err := ps.ls1(filepath.Join(path, dir), options, multi, false, firstToShow && !didShow)
		if err != nil {
			if herr == nil {
				herr = err
			}
			if !herror.IsSilent(herr) {
				return didShow, herr
			}
			continue
		}
		didShow = didShow || recDidShow
	}
	return didShow, herr
}

func (ps *Periscope) list1(out io.Writer, file os.FileInfo, dirPath string, options *LsOptions) (bool, bool, herror.Interface) {
	mode := file.Mode()
	isDirectory := false
	var desc string
	var fullPath string
	var dupeSet db.DuplicateSet
	if mode&os.ModeDir == os.ModeDir {
		desc = "d"
		isDirectory = true
	} else if mode&os.ModeSymlink == os.ModeSymlink {
		desc = "L"
	} else if mode&os.ModeNamedPipe == os.ModeNamedPipe {
		desc = "p"
	} else if mode&os.ModeSocket == os.ModeSocket {
		desc = "S"
	} else if mode&os.ModeDevice == os.ModeDevice {
		desc = "D"
	} else if mode&os.ModeCharDevice == os.ModeCharDevice {
		desc = "c"
	} else if mode.IsRegular() {
		fullPath = filepath.Join(dirPath, file.Name())
		var err herror.Interface
		dupeSet, err = ps.db.Lookup(fullPath)
		if err != nil {
			return false, false, err
		}
		nDupes := len(dupeSet) - 1
		if nDupes > 0 {
			desc = strconv.Itoa(nDupes)
		}
	} else {
		desc = "?"
	}
	show := true
	if options.Unique && len(dupeSet) > 1 {
		show = false
	}
	if options.Duplicate && len(dupeSet) < 2 {
		show = false
	}
	if show {
		fmt.Fprintf(out, "%s\v%s\n", desc, file.Name())
		if options.Verbose && len(dupeSet) > 1 {
			for _, info := range dupeSet {
				if info.Path != fullPath {
					showPath := info.Path
					if options.Relative {
						showPath = relPath(dirPath, info.Path)
					}
					fmt.Fprintf(out, "\v  %s\n", showPath)
				}
			}
		}
	}
	return show, isDirectory, nil
}
