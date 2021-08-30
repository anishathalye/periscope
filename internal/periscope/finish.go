package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"os"
)

type FinishOptions struct {
}

func Finish(options *FinishOptions) herror.Interface {
	path, herr := dbPath()
	if herr != nil {
		return herr
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if os.IsPermission(err) {
		return herror.Unlikely(err, fmt.Sprintf("cannot access '%s': permission denied", path), `
Ensure that the cache directory is accessible.
		`)
	}
	if err != nil {
		return herror.Unlikely(err, fmt.Sprintf("could not stat '%s'", path), `
Ensure that the cache directory is readable.
		`)
	}
	if !info.Mode().IsRegular() {
		return herror.Unlikely(err, fmt.Sprintf("database is not a regular file: '%s'", path), `
This should not happen under regular circumstances. If you are done using the database, you can safely delete it manually with 'rm -f'.
		`)
	}
	err = os.Remove(path)
	if err != nil {
		return herror.Unlikely(err, fmt.Sprintf("cannot delete database file: '%s'", path), `
Ensure that the cache directory is writable or manually delete the database file.
		`)
	}
	fmt.Printf("rm %s\n", path)
	return nil
}
