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
	Minimum int64
	Maximum int64
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
	dupes, done := ps.findDuplicates(absPaths, options)
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	// remove previously scanned files in paths we are now searching
	for _, path := range absPaths {
		tx.RemoveDir(path, options.Minimum, options.Maximum)
	}
	// add all the new things we've found
	for info := range dupes {
		tx.Add(info.(db.FileInfo))
	}
	// create indexes if they don't exist already
	err = tx.CreateIndexes()
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	done()
	return nil
}

// we use this to avoid database writes; findFilesBySize finds files in the
// directory to be scanned, and it also looks up relevant files to consider
// from the database; we want to add newly scanned files to the database
// regardless, but for infos that were already there, we only want to write to
// the database if they info has been updated (we've computed a hash that
// wasn't there before)
type searchResult struct {
	info db.FileInfo
	old  bool
}

// return value also includes the relevant stuff in the DB
//
// we do this here so that there are no db reads in the rest of findDuplicates,
// so we can do a streaming write into the db without concurrent reads
func (ps *Periscope) findFilesBySize(paths []string, options *ScanOptions) (map[int64][]searchResult, int) {
	sizeToInfos := make(map[int64][]searchResult)
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
			if size > int64(options.Minimum) && (options.Maximum == 0 || size <= int64(options.Maximum)) {
				if len(sizeToInfos[size]) == 0 {
					// find all relevant files from the database, skipping the
					// ones that are included in paths; we only do this once per
					// size (after we've seen a particular size, the [size] key
					// will contain at least one element, so we won't re-do this
					if known, err := ps.db.InfosBySize(size); err == nil {
						for _, k := range known {
							if !containedInAny(k.Path, paths) {
								sizeToInfos[size] = append(sizeToInfos[size], searchResult{info: k, old: true})
								files++
							}
						}
					}
				}
				sizeToInfos[size] = append(sizeToInfos[size], searchResult{
					info: db.FileInfo{
						Path:      path,
						Size:      size,
						ShortHash: nil,
						FullHash:  nil,
					},
					old: false,
				})
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
	return sizeToInfos, files
}

// paths consists of absolute paths with no symlinks
func (ps *Periscope) findDuplicates(searchPaths []string, options *ScanOptions) (<-chan interface{}, func()) {
	sizeToInfos, files := ps.findFilesBySize(searchPaths, options)

	bar := ps.progressBar(files, `analyzing: {{ counters . }} {{ bar . "[" "=" ">" " " "]" }} {{ etime . }} {{ rtime . "ETA %s" "%.0s" " " }} `)
	done := func() {
		bar.Finish()
	}

	return par.MapN(sizeToInfos, scanThreads, func(k, v interface{}, emit func(x interface{})) {
		size := k.(int64)
		searchResults := v.([]searchResult)
		defer bar.Add(len(searchResults))

		// files may appear multiple times, if the same directory is repeated in the
		// arguments to scan; skip those
		seen := make(map[string]struct{})
		var infos []db.FileInfo
		// have we updated the data for this path (computed a new hash)? if so, we will
		// write the relevant info to the database
		var updated []bool // has infos[i] been updated?
		for _, result := range searchResults {
			path := result.info.Path
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			infos = append(infos, result.info)
			if !result.old {
				updated = append(updated, true)
			} else {
				updated = append(updated, false)
			}
		}

		// if there's only one file with this size, we don't need to do any hashing
		if len(infos) == 1 {
			// the following check should always be true
			if updated[0] {
				emit(infos[0])
			}
			return
		}

		// compute short hashes for all files (skipping the ones where
		// we already have short hashes), bucketing results by short hash
		szBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(szBuf, uint64(size))
		byShortHash := make(map[[ShortHashSize]byte][]int) // indices into infos array
		for i := range infos {
			info := &infos[i]
			// compute short hash if necessary
			if info.ShortHash == nil {
				// key by size to have unique short hashes, so we can use them as global identifiers
				hash, err := ps.hashPartial(info.Path, szBuf)
				if err != nil {
					log.Printf("hashPartial() returned error: %s", err)
					continue // ignore this file
				}
				info.ShortHash = hash
				updated[i] = true
			}
			hashArr := shortHashToArray(info.ShortHash)
			byShortHash[hashArr] = append(byShortHash[hashArr], i)
		}

		// wherever there is > 1 file in a bucket, compute the full
		// hashes (skipping the ones where we already have full hashes)
		for _, indices := range byShortHash {
			if len(indices) <= 1 {
				// no need to compute full hash
				continue
			}
			// collide on short hash; hash full file
			for _, index := range indices {
				info := &infos[index]
				if info.FullHash == nil {
					hash, err := ps.hashFile(info.Path)
					if err != nil {
						log.Printf("hashPartial() returned error: %s", err)
						continue // ignore this file
					}
					info.FullHash = hash
					updated[index] = true
				}
			}
		}

		// emit all files for which we've done some work, where there is new info to save to
		// the database
		for i, info := range infos {
			if updated[i] {
				emit(info)
			}
		}
	}), done
}
