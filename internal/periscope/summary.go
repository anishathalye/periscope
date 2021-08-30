package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
)

type SummaryOptions struct {
}

func (ps *Periscope) Summary(options *SummaryOptions) herror.Interface {
	summary, err := ps.db.Summary()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ps.outStream, 0, 0, 0, ' ', tabwriter.DiscardEmptyColumns|tabwriter.AlignRight)
	fmt.Fprintf(w, "tracked\v %s\v\n", humanize.Comma(summary.Files))
	fmt.Fprintf(w, "unique\v %s\v\n", humanize.Comma(summary.Unique))
	fmt.Fprintf(w, "duplicate\v %s\v\n", humanize.Comma(summary.Duplicate))
	fmt.Fprintf(w, "overhead\v %s\v\n", humanize.Bytes(uint64(summary.Overhead)))
	w.Flush()
	return nil
}
