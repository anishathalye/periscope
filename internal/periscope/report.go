package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/herror"

	"container/list"
	"fmt"
	"sync"

	"github.com/dustin/go-humanize"
)

type ReportOptions struct {
}

func (ps *Periscope) Report(options *ReportOptions) herror.Interface {
	// We stream duplicates with AllDuplicatesC, but we don't read directly
	// from it and write results in the straightforward way. Writing to
	// output may block (e.g. if the user is using a pager), so if a user
	// had `psc report | less` open in one window and tried to `psc rm` in
	// another, they'd get a "database is locked" error. This seems like
	// it's a common enough use case that it's worth avoiding it. We
	// achieve this by buffering the results in memory.
	sets, err := ps.db.AllDuplicatesC()
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
			fmt.Fprintf(ps.outStream, "  %s\n", info.Path)
		}
		first = false
	}

	return nil
}
