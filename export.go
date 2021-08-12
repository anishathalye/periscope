package periscope

import (
	"encoding/json"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/herror"
)

type ExportFormat int

const (
	JsonFormat ExportFormat = iota
)

type ExportOptions struct {
	Format ExportFormat
}

func (ps *Periscope) Export(options *ExportOptions) herror.Interface {
	c, err := ps.db.AllDuplicatesC()
	if err != nil {
		return err
	}
	return ps.jsonExport(c)
}

type exportDuplicateInfo struct {
	Paths []string `json:"paths"`
	Size  int64    `json:"size"`
}

type exportResult struct {
	Duplicates []exportDuplicateInfo `json:"duplicates"`
}

func (ps *Periscope) jsonExport(c <-chan db.DuplicateSet) herror.Interface {
	duplicates := make([]exportDuplicateInfo, 0)
	for set := range c {
		size := set[0].Size // all files within a set are the same size
		var paths []string
		for _, info := range set {
			paths = append(paths, info.Path)
		}
		duplicates = append(duplicates, exportDuplicateInfo{
			Paths: paths,
			Size:  size,
		})
	}
	res := exportResult{
		Duplicates: duplicates,
	}
	enc := json.NewEncoder(ps.outStream)
	enc.SetIndent("", "  ")
	err := enc.Encode(res)
	if err != nil {
		return herror.Internal(err, "")
	}
	return nil
}
