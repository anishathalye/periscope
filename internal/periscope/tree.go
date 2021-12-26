package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"text/tabwriter"
)

type TreeOptions struct {
	All bool
}

func (ps *Periscope) Tree(root string, options *TreeOptions) herror.Interface {
	absRoot, _, err := ps.checkFile(root, false, true, "show", false, true)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ps.outStream, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns)
	r, herr := ps.db.LookupAll(absRoot, options.All)
	if herr != nil {
		return herr
	}
	for _, dupe := range r {
		_, _, err := ps.checkFile(dupe.Path, true, false, "", true, false)
		if err != nil {
			// something has changed
			continue
		}
		showPath := relPath(absRoot, dupe.Path)
		fmt.Fprintf(w, "%d\v%s\n", dupe.Count-1, showPath)
	}
	w.Flush()
	return nil
}
