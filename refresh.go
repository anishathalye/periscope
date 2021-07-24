package periscope

import (
	"fmt"
	"log"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/herror"
	"github.com/anishathalye/periscope/par"
)

type RefreshOptions struct {
}

func (ps *Periscope) Refresh(options *RefreshOptions) herror.Interface {
	summary, err := ps.db.Summary()
	if err != nil {
		return err
	}
	dupes, err := ps.db.AllDuplicatesC()
	if err != nil {
		return err
	}

	bar := ps.progressBar(int(summary.Files), `scanning: {{ counters . }} {{ bar . "[" "=" ">" " " "]" }} {{ etime . }} {{ rtime . "ETA %s" "%.0s" " " }} `)

	var gone []string
	for path := range par.MapN(dupes, scanThreads, func(_, v interface{}, emit func(x interface{})) {
		for _, path := range v.(db.DuplicateSet).Paths {
			_, _, err := ps.checkFile(path, true, false, "", true, false)
			bar.Increment()
			if err != nil {
				log.Printf("removing '%s' from database", path)
				emit(path)
			}
		}
	}) {
		gone = append(gone, path.(string))
	}
	// note: we can't actually delete the files while scanning because
	// we're doing a streaming read from the database
	err = ps.db.RemoveAll(gone)
	if err != nil {
		return err
	}
	err = ps.db.PruneSingletons()
	if err != nil {
		return err
	}
	bar.Finish()
	fmt.Fprintf(ps.outStream, "removed %d files from the database\n", len(gone))
	return nil
}
