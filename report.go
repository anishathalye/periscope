package periscope

import (
	"fmt"

	"github.com/anishathalye/periscope/herror"

	"github.com/dustin/go-humanize"
)

type ReportOptions struct {
}

func (ps *Periscope) Report(options *ReportOptions) herror.Interface {
	// We could stream duplicates with AllDuplicatesC, but then if someone
	// had `psc report | less` open in one window and tried to `psc rm` in
	// another, they'd get a "database is locked" error. This seems like
	// it's a common enough use case that it's worth avoiding it, even if
	// it increases the latency of a `psc report` to output the first
	// screen of duplicates.
	sets, err := ps.db.AllDuplicates()
	if err != nil {
		return err
	}
	for _, set := range sets {
		fmt.Fprintf(ps.outStream, "%s\n", humanize.Bytes(uint64(set.Size)))
		for _, info := range set.Paths {
			fmt.Fprintf(ps.outStream, "  %s\n", info)
		}
		fmt.Fprintf(ps.outStream, "\n")
	}
	return nil
}
