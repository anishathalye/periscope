package periscope

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/anishathalye/periscope/herror"
)

type RmOptions struct {
	Recursive    bool
	Verbose      bool
	DryRun       bool
	HasContained bool
	Contained    string
}

func (ps *Periscope) Rm(paths []string, options *RmOptions) herror.Interface {
	var herr herror.Interface

	// validate arguments
	var absContained string
	if options.HasContained {
		absContained, _, herr = ps.checkFile(options.Contained, false, true, "access", false, true)
		if herr != nil {
			return herr
		}
	}

	for _, path := range paths {
		absPath, info, err := ps.checkFile(path, false, false, "remove", false, false)
		if err != nil {
			if !herror.IsSilent(err) {
				return err
			}
			herr = err
			continue
		}
		if info.Mode().IsRegular() {
			err = ps.removeFile(path, options, absContained)
		} else {
			err = ps.removeDirectory(path, absPath, options, absContained)
		}
		if err != nil {
			if !herror.IsSilent(err) {
				return err
			}
			herr = err
		}
	}
	err := ps.db.PruneSingletons()
	if err != nil {
		return err
	}
	return herr
}

func (ps *Periscope) removeFile(path string, options *RmOptions, absContained string) herror.Interface {
	return ps.remove1(map[string]struct{}{path: {}}, options, true, "", absContained)
}

func (ps *Periscope) removeDirectory(path string, absPath string, options *RmOptions, absContained string) herror.Interface {
	if !options.Recursive {
		fmt.Fprintf(ps.errStream, "cannot remove '%s': must specify -r, --recursive to delete directories\n", path)
		return herror.Silent()
	}
	c, herr := ps.db.LookupAll(absPath, true)
	if herr != nil {
		return herr
	}
	byTag := make(map[int64]map[string]struct{})
	for _, dupInfo := range c {
		if byTag[dupInfo.Tag] == nil {
			byTag[dupInfo.Tag] = make(map[string]struct{})
		}
		byTag[dupInfo.Tag][dupInfo.Path] = struct{}{}
	}
	for _, candidates := range byTag {
		err := ps.remove1(candidates, options, false, path, absContained)
		if err != nil {
			herr = err
		}
		if herr != nil && !herror.IsSilent(herr) {
			// interrupts execution
			return herr
		}
	}
	return herr
}

func (ps *Periscope) remove1(candidates map[string]struct{}, options *RmOptions, singleFile bool, directory, absContained string) herror.Interface {
	// take a conservative approach to deleting files:
	// - compute a full hash of all files in the set; if they're not all the same, abort
	// - go through all other files in the same duplicate set that are
	//   outside the candidate set (and if options.HasContained is set,
	//   then only count the ones that are present in the options.Contained
	//   directory); if none matches the hash, abort
	// - delete all candidates in the set

	// early sanity checks
	var path0, absPath0 string
	absPaths := make(map[string]struct{})
	infos := make(map[string]os.FileInfo)
	for path := range candidates {
		absPath, info, err := ps.checkFile(path, true, false, "remove", !singleFile, false)
		if err != nil {
			if singleFile {
				return err
			}
			delete(candidates, path) // this is safe to do while iterating over the map
			continue
		}
		infos[absPath] = info
		absPaths[absPath] = struct{}{}
		path0 = path // some arbitrary path
		absPath0 = absPath
	}
	set, _ := ps.db.Lookup(absPath0)
	// ensure all candidates contained in set
	duplicateSet := make(map[string]struct{})
	for _, path := range set.Paths {
		duplicateSet[path] = struct{}{}
	}
	allContained := true
	for path := range absPaths {
		if _, ok := duplicateSet[path]; !ok {
			allContained = false
			break
		}
	}
	if !allContained {
		if singleFile {
			// use path0 to use the non-absolute path that was passed in
			fmt.Fprintf(ps.errStream, "cannot remove '%s': no duplicates\n", path0)
			return herror.Silent()
		}
		return nil
	}

	// the rest of this function is the most critical code in this entire
	// program: it should only delete files if there's a duplicate
	// elsewhere in the filesystem

	// compute hash of files we are deleting, and ensure that hashes of all
	// candidates match each other
	var hash []byte
	for path := range absPaths {
		currHash, err := ps.hashFile(path)
		if err != nil {
			log.Printf("hashFile('%s') returned error: %s", path, err)
			if singleFile {
				if os.IsPermission(err) {
					fmt.Fprintf(ps.errStream, "cannot remove '%s': permission denied\n", path0)
				} else {
					fmt.Fprintf(ps.errStream, "cannot remove '%s': %s\n", path0, err)
				}
				return herror.Silent()
			}
			// skip/remove the candidate but keep going
			delete(absPaths, path) // note: this is safe to do while iterating over the map
			continue
		}
		if hash == nil {
			hash = currHash
		} else {
			if bytes.Compare(hash, currHash) != 0 {
				// files within set don't agree; give up
				return nil
			}
		}
	}

	// because we might delete paths in the above loop, check to see that
	// there are still paths to remove
	if len(absPaths) == 0 {
		return nil
	}

	// ensure that a copy exists elsewhere
	otherMatch := false
	for path := range duplicateSet {
		if _, ok := absPaths[path]; ok {
			// this is one of the paths we are considering deleting
			continue // bad candidate
		}
		if options.HasContained && !strings.HasPrefix(path, absContained+string(os.PathSeparator)) {
			// outside set we are considering deleting, but not in
			// contained directory
			continue // bad candidate
		}
		// check that the hash still matches, that the file still
		// exists and hasn't changed
		otherHash, err := ps.hashFile(path)
		if err != nil {
			log.Printf("hashFile('%s') returned error: %s", path, err.Error())
			continue // bad candidate
		}
		if bytes.Equal(hash, otherHash) {
			_, otherInfo, err := ps.checkFile(path, true, false, "", true, false)
			if err != nil {
				log.Printf("checkFile('%s') returned error: %s", path, err.Error())
				continue // bad candidate
			}
			// be extra sure that they aren't the same file
			bad := false
			if ps.realFs {
				for delPath := range absPaths {
					if os.SameFile(infos[delPath], otherInfo) {
						bad = true
						break
					}
				}
			}
			if !bad {
				otherMatch = true
				break
			}
		}
		// keep trying to find a duplicate ...
	}
	if !otherMatch {
		if singleFile {
			if options.HasContained {
				fmt.Fprintf(ps.errStream, "cannot remove '%s': no duplicates in '%s'\n", path0, options.Contained)
			} else {
				fmt.Fprintf(ps.errStream, "cannot remove '%s': no duplicates\n", path0)
			}
			return herror.Silent()
		}
		return nil
	}

	// okay, we can delete all candidates in the set
	if singleFile {
		// path that is passed in, path0, is what the user typed, so we
		// use that for printing purposes
		if !options.DryRun {
			err := ps.fs.Remove(absPath0)
			if os.IsNotExist(err) {
				fmt.Fprintf(ps.errStream, "cannot remove '%s': no such file\n", path0)
				return herror.Silent()
			} else if os.IsPermission(err) {
				fmt.Fprintf(ps.errStream, "cannot remove '%s': permission denied\n", path0)
				return herror.Silent()
			} else if err != nil {
				return herror.Internal(err, "")
			}
			herr := ps.db.Remove(absPath0)
			if herr != nil {
				return herr
			}
		}
		if options.Verbose {
			fmt.Fprintf(ps.outStream, "rm %s\n", path0)
		}
	} else {
		// delete in sorted order
		for absPath := range absPaths {
			// calculate a nicer version to print to the user
			rel := relFrom(directory, absPath)
			if options.Verbose {
				fmt.Fprintf(ps.outStream, "rm %s\n", rel)
			}
			if !options.DryRun {
				err := ps.fs.Remove(absPath)
				if err != nil && !(os.IsNotExist(err) || os.IsPermission(err)) {
					log.Printf("Remove('%s') returned an error: %s", absPath, err)
				}
				if err == nil {
					herr := ps.db.Remove(absPath)
					if herr != nil {
						return herr
					}
				}
			}
		}
	}
	return nil
}
