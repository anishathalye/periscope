package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"os"
	"path/filepath"
)

type ForgetOptions struct {
}

func (ps *Periscope) Forget(paths []string, options *ForgetOptions) herror.Interface {
	// we don't need paths to exist before doing work, but we do need to be able to determine absolute paths
	var herr herror.Interface
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(ps.errStream, "cannot forget '%s': cannot determine absolute path\n", path)
			herr = herror.Silent()
		} else {
			err := tx.RemoveDir(abs, 0, 0)
			if err != nil {
				tx.Rollback()
				return err
			}
			// format path for nicer printing
			if path[len(path)-1] != os.PathSeparator {
				path = path + string(os.PathSeparator)
			}
			fmt.Fprintf(ps.outStream, "forgot %s*\n", path)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return herr
}
