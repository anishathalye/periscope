package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/herror"

	"container/list"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/dustin/go-humanize"
)

type ReportOptions struct {
	Relative bool
}

func (ps *Periscope) Report(dir string, options *ReportOptions) herror.Interface {
	var absDir string
	if dir != "" {
		var err herror.Interface
		absDir, _, err = ps.checkFile(dir, false, true, "filter for", false, true)
		if err != nil {
			return err
		}
	}
	// We stream duplicates with AllDuplicatesC, but we don't read directly
	// from it and write results in the straightforward way. Writing to
	// output may block (e.g. if the user is using a pager), so if a user
	// had `psc report | less` open in one window and tried to `psc rm` in
	// another, they'd get a "database is locked" error. This seems like
	// it's a common enough use case that it's worth avoiding it. We
	// achieve this by buffering the results in memory.
	sets, err := ps.db.AllDuplicatesC(absDir)
	if err != nil {
		return err
	}

	buf := list.New()
	done := false
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	go func() {
		for set := range sets {
			mu.Lock()
			buf.PushBack(set)
			cond.Signal()
			mu.Unlock()
		}
		mu.Lock()
		done = true
		cond.Signal()
		mu.Unlock()
	}()

	var refDir string
	if options.Relative {
		var err error
		refDir, err = filepath.Abs(dir) // if dir == "", this will treat it like dir = "."
		if err != nil {
			return herror.Internal(err, "")
		}
	}
	first := true
	for {
		mu.Lock()
		for !done && buf.Len() == 0 {
			cond.Wait()
		}
		if done && buf.Len() == 0 {
			mu.Unlock()
			break
		}
		front := buf.Front()
		set := front.Value.(db.DuplicateSet)
		buf.Remove(front)
		mu.Unlock()

		if !first {
			fmt.Fprintf(ps.outStream, "\n")
		}
		fmt.Fprintf(ps.outStream, "%s\n", humanize.Bytes(uint64(set[0].Size))) // all files within a set have the same size
		for _, info := range set {
			path := info.Path
			if options.Relative {
				path = relPath(refDir, path)
			}
			fmt.Fprintf(ps.outStream, "  %s\n", path)
		}
		first = false
	}

	return nil
}
