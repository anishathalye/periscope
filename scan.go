package periscope

import (
	"encoding/binary"
	"log"
	"os"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/herror"
	"github.com/anishathalye/periscope/par"

	"github.com/spf13/afero"
)

type ScanOptions struct {
}

func (ps *Periscope) Scan(paths []string, options *ScanOptions) herror.Interface {
	// check that paths exist before starting any work
	absPaths := make([]string, len(paths))
	for i, path := range paths {
		abs, _, err := ps.checkFile(path, false, true, "scan", false, true)
		if err != nil {
			return err
		}
		absPaths[i] = abs
	}
	err := ps.db.Initialize()
	if err != nil {
		return err
	}
	tagged := make(chan db.DuplicateSet)
	var tag int64 = 1
	go func() {
		dupes, done := ps.findDuplicates(absPaths)
		for result := range dupes {
			ds := result.(db.DuplicateSet)
			ds.Tag = tag
			tag++
			tagged <- ds
		}
		done()
		close(tagged)
	}()
	err = ps.db.AddAllC(tagged)
	if err != nil {
		return err
	}
	err = ps.db.CreateIndexes()
	if err != nil {
		return err
	}
	return nil
}

func (ps *Periscope) findFilesBySize(paths []string) (map[int64][]string, int) {
	sizeToFiles := make(map[int64][]string)
	files := 0

	bar := ps.progressBar(0, `searching: {{ counters . }} files {{ etime . }} `)

	for _, root := range paths {
		err := afero.Walk(ps.fs, root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("%s", err)
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			size := info.Size()
			if size > 0 {
				sizeToFiles[size] = append(sizeToFiles[size], path)
				files++
				bar.Increment()
			}
			return nil
		})
		if err != nil {
			log.Printf("Walk() returned error: %s", err)
		}
	}
	bar.Finish()
	return sizeToFiles, files
}

func (ps *Periscope) findDuplicates(paths []string) (<-chan interface{}, func()) {
	sizeToFiles, files := ps.findFilesBySize(paths)
	_ = files

	bar := ps.progressBar(files, `analyzing: {{ counters . }} {{ bar . "[" "=" ">" " " "]" }} {{ etime . }} {{ rtime . "ETA %s" "%.0s" " " }} `)
	done := func() {
		bar.Finish()
	}

	return par.MapN(sizeToFiles, scanThreads, func(k, v interface{}, emit func(x interface{})) {
		size := k.(int64)
		infos := v.([]string)
		defer bar.Add(len(infos))
		if len(infos) <= 1 {
			return
		}

		szBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(szBuf, uint64(size))

		seen := make(map[string]struct{})
		byShortHash := make(map[[HashSize]byte][]string)
		for _, info := range infos {
			if _, ok := seen[info]; ok {
				continue
			}
			seen[info] = struct{}{}
			// key by size to have unique short hashes, so we could use them as global identifiers
			// even though we don't currently make use of this
			hash, err := ps.hashPartial(info, szBuf, true)
			if err != nil {
				log.Printf("hashPartial() returned error: %s", err)
				continue // ignore this file
			}
			byShortHash[hash] = append(byShortHash[hash], info)
		}

		// second pass
		var byFullHash map[[HashSize]byte][]string
		if size <= initialChunkSize {
			byFullHash = byShortHash
		} else {
			byFullHash = make(map[[HashSize]byte][]string)
			for shortHash, infos := range byShortHash {
				if len(infos) <= 1 {
					continue
				}
				// collide, hash whole file
				for _, info := range infos {
					// key by short hash to get unique result for overall file
					hash, err := ps.hashPartial(info, shortHash[:], false)
					if err != nil {
						log.Printf("hashPartial() returned error: %s", err)
						continue // ignore this file
					}
					byFullHash[hash] = append(byFullHash[hash], info)
				}
			}
		}
		for _, entries := range byFullHash {
			// note: this copy is necessary; just passing hash[:]
			// directly will just give a pointer to the same
			// variable whose contents are changing on every
			// iteration of the for loop, which would be a
			// concurrency bug
			if len(entries) > 1 {
				emit(db.DuplicateSet{
					Paths: entries,
					Size:  size,
					// note: no Tag provided here; that is
					// filled in by db just before writing,
					// because it's a little harder to get
					// a globally unique tag here
				})
			}
		}
	}), done
}
