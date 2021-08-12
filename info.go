package periscope

import (
	"fmt"
	"path/filepath"

	"github.com/anishathalye/periscope/herror"
)

type InfoOptions struct {
	Relative bool
}

func (ps *Periscope) Info(paths []string, options *InfoOptions) herror.Interface {
	var herr herror.Interface
	for i, p := range paths {
		if i > 0 {
			fmt.Fprintf(ps.outStream, "\n")
		}
		err := ps.info1(p, options)
		if err != nil {
			herr = err
		}
		if herr != nil && !herror.IsSilent(herr) {
			return herr
		}
	}
	return herr
}

func (ps *Periscope) info1(path string, options *InfoOptions) herror.Interface {
	absPath, _, err := ps.checkFile(path, true, false, "show", false, false)
	if err != nil {
		return err
	}
	dupeSet, herr := ps.db.Lookup(absPath)
	if herr != nil {
		return herr
	}
	nCopies := len(dupeSet)
	nDupes := 0
	if nCopies > 1 {
		nDupes = nCopies - 1
	}
	fmt.Fprintf(ps.outStream, "%d %s\n", nDupes, path)
	dirPath := filepath.Dir(absPath)
	for _, info := range dupeSet {
		if info.Path != absPath {
			showPath := info.Path
			if options.Relative {
				showPath = relPath(dirPath, info.Path)
			}
			fmt.Fprintf(ps.outStream, "  %s\n", showPath)
		}
	}
	return nil
}
