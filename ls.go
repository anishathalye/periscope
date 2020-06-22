package periscope

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/herror"

	"github.com/spf13/afero"
)

type LsOptions struct {
	All       bool
	Verbose   bool
	Duplicate bool
	Unique    bool
	Relative  bool
}

func (ps *Periscope) Ls(paths []string, options *LsOptions) herror.Interface {
	var herr herror.Interface
	echoDir := len(paths) > 1
	var err herror.Interface
	for i, p := range paths {
		if i > 0 && err == nil {
			fmt.Fprintf(ps.outStream, "\n")
		}
		err = ps.ls1(p, options, echoDir)
		if err != nil {
			herr = err
		}
		if herr != nil && !herror.IsSilent(herr) {
			return herr
		}
	}
	return herr
}

func (ps *Periscope) ls1(path string, options *LsOptions, echoDir bool) herror.Interface {
	absPath, _, herr := ps.checkFile(path, false, true, "list", false, false)
	if herr != nil {
		return herr
	}
	files, err := afero.ReadDir(ps.fs, absPath)
	if os.IsPermission(err) {
		fmt.Fprintf(ps.errStream, "cannot access '%s': permission denied\n", path)
		return herror.Silent()
	}
	if err != nil {
		return herror.Internal(err, "")
	}
	w := tabwriter.NewWriter(ps.outStream, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	if echoDir {
		fmt.Fprintf(ps.outStream, "%s:\n", path)
	}
	for _, file := range files {
		if file.Name()[0] != '.' || options.All {
			err := ps.list1(w, file, absPath, options)
			if err != nil {
				return err
			}
		}
	}
	w.Flush()
	return nil
}

func (ps *Periscope) list1(out io.Writer, file os.FileInfo, dirPath string, options *LsOptions) herror.Interface {
	mode := file.Mode()
	var desc string
	var fullPath string
	var dupeSet db.DuplicateSet
	if mode&os.ModeDir == os.ModeDir {
		desc = "d"
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
			return err
		}
		nDupes := len(dupeSet.Paths) - 1
		if nDupes > 0 {
			desc = strconv.Itoa(nDupes)
		}
	} else {
		desc = "?"
	}
	show := true
	if options.Unique && len(dupeSet.Paths) > 1 {
		show = false
	}
	if options.Duplicate && len(dupeSet.Paths) < 2 {
		show = false
	}
	if show {
		fmt.Fprintf(out, "%s\v%s\n", desc, file.Name())
		if options.Verbose && len(dupeSet.Paths) > 1 {
			for _, dupe := range dupeSet.Paths {
				if dupe != fullPath {
					showPath := dupe
					if options.Relative {
						showPath = relPath(dirPath, dupe)
					}
					fmt.Fprintf(out, "\v  %s\n", showPath)
				}
			}
		}
	}
	return nil
}
