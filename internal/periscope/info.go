package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"encoding/hex"
	"fmt"
	"path/filepath"
	"text/tabwriter"
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
	fmt.Fprintf(ps.outStream, "%s\n", path)
	w := tabwriter.NewWriter(ps.outStream, 0, 0, 0, ' ', tabwriter.DiscardEmptyColumns|tabwriter.AlignRight)
	if len(dupeSet) > 0 {
		info := dupeSet[0]
		if info.ShortHash != nil {
			fmt.Fprintf(w, "  short hash:\v %s\n", hex.EncodeToString(info.ShortHash))
		}
		if info.FullHash != nil {
			fmt.Fprintf(w, "  full hash:\v %s\n", hex.EncodeToString(info.FullHash))
		}
	}
	if nDupes > 0 {
		fmt.Fprintf(w, "  duplicates:\v %d\n", nDupes)
	}
	w.Flush()
	if nDupes > 0 {
		dirPath := filepath.Dir(absPath)
		for _, info := range dupeSet {
			if info.Path != absPath {
				showPath := info.Path
				if options.Relative {
					showPath = relPath(dirPath, info.Path)
				}
				fmt.Fprintf(ps.outStream, "    %s\n", showPath)
			}
		}
	}
	return nil
}
