package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/herror"
	"github.com/anishathalye/periscope/internal/par"

	"fmt"
	"log"
)

type RefreshOptions struct {
}

func (ps *Periscope) Refresh(options *RefreshOptions) herror.Interface {
	summary, err := ps.db.Summary()
	if err != nil {
		return err
	}
	infos, err := ps.db.AllInfosC()
	if err != nil {
		return err
	}

	bar := ps.progressBar(int(summary.Files), `scanning: {{ counters . }} {{ bar . "[" "=" ">" " " "]" }} {{ etime . }} {{ rtime . "ETA %s" "%.0s" " " }} `)

	var gone []string
	for path := range par.MapN(infos, scanThreads, func(_, v interface{}, emit func(x interface{})) {
		path := v.(db.FileInfo).Path
		_, _, err := ps.checkFile(path, true, false, "", true, false)
		bar.Increment()
		if err != nil {
			log.Printf("removing '%s' from database", path)
			emit(path)
		}
	}) {
		gone = append(gone, path.(string))
	}
	// note: we can't actually delete the files while scanning because
	// we're doing a streaming read from the database
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	for _, path := range gone {
		if err = tx.Remove(path); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	bar.Finish()
	fmt.Fprintf(ps.outStream, "removed %d files from the database\n", len(gone))
	return nil
}
