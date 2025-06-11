package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/herror"

	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type HashOptions struct {
}

func (ps *Periscope) Hash(paths []string, options *HashOptions) herror.Interface {
	szBuf := make([]byte, 8)
	tx, herr := ps.db.Begin()
	if herr != nil {
		return herr
	}
	for _, path := range paths {
		abs, statInfo, checkErr := ps.checkFile(path, true, false, "hash", false, false)
		if checkErr != nil {
			if herr == nil {
				herr = checkErr
			}
			continue
		}
		size := statInfo.Size()
		binary.LittleEndian.PutUint64(szBuf, uint64(size))
		shortHash, err := ps.hashPartial(abs, szBuf)
		if err != nil {
			tx.Rollback()
			return herror.Internal(err, "")
		}
		fullHash, err := ps.hashFile(abs)
		if err != nil {
			tx.Rollback()
			return herror.Internal(err, "")
		}
		info := db.FileInfo{
			Path:      abs,
			Size:      size,
			ShortHash: shortHash,
			FullHash:  fullHash,
		}
		if err := tx.Add(info); err != nil {
			tx.Rollback()
			return err
		}
		fmt.Fprintf(ps.outStream, "%s  %s\n", hex.EncodeToString(fullHash), path)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return herr
}
