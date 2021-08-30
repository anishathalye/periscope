package periscope

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/blake2b"
)

const initialChunkSize = 4 * 1024
const readChunkSize = 1024 * 1024
const ShortHashSize = 8
const HashSize = blake2b.Size256

func hashToArray(hash []byte) [HashSize]byte {
	var res [HashSize]byte
	copy(res[:], hash)
	return res
}

func shortHashToArray(hash []byte) [ShortHashSize]byte {
	var res [ShortHashSize]byte
	copy(res[:], hash)
	return res
}

func (ps *Periscope) hashPartial(path string, key []byte) ([]byte, error) {
	buf := make([]byte, initialChunkSize)
	h, err := blake2b.New256(key)
	if err != nil {
		return nil, err
	}
	f, err := ps.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	h.Write(buf[:n])
	return h.Sum(nil)[:ShortHashSize], nil
}

// a simpler hashFile that hashes the full file
// purposefully avoiding code reuse with the above
func (ps *Periscope) hashFile(path string) ([]byte, error) {
	f, err := ps.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, readChunkSize)
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func relPath(absDirectory, absPath string) string {
	relPath, err := filepath.Rel(absDirectory, absPath)
	if err != nil || len(relPath) > len(absPath) {
		return absPath
	}
	return relPath
}

func relFrom(relDirectory, absPath string) string {
	absDirectory, err := filepath.Abs(relDirectory)
	if err != nil {
		return absPath
	}
	relPath, err := filepath.Rel(absDirectory, absPath)
	if err != nil {
		return absPath
	}
	path := filepath.Join(relDirectory, relPath)
	if len(path) <= len(absPath) {
		return path
	}
	return absPath
}

func (ps *Periscope) checkSymlinks(cleanAbsPath string) (hasSymlinks bool, realAbsPath string, err error) {
	if !ps.realFs {
		// filepath.EvalSymlinks() won't be a sensible thing to do when
		// testing with afero's in-memory filesystem
		return false, cleanAbsPath, nil
	}
	resolved, err := filepath.EvalSymlinks(cleanAbsPath)
	if err != nil {
		return false, "", err
	}
	return (resolved != cleanAbsPath), resolved, nil
}

func (ps *Periscope) checkFile(path string, mustBeRegularFile, mustBeDirectory bool, action string, quiet, fatal bool) (realAbsPath string, info os.FileInfo, herr herror.Interface) {
	checkFileError := func(format string, a ...interface{}) herror.Interface {
		if !fatal {
			if !quiet {
				fmt.Fprintf(ps.errStream, format, a...)
				fmt.Fprintf(ps.errStream, "\n")
			}
			return herror.Silent()
		}
		return herror.UserF(nil, format, a...)
	}
	// check that it exists
	info, err := ps.fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, checkFileError("cannot %s '%s': no such file or directory", action, path)
		}
		if os.IsPermission(err) {
			return "", nil, checkFileError("cannot %s '%s': permission denied", action, path)
		}
		// what else can go wrong here? one example is too many levels of symbolic links
		return "", nil, checkFileError("cannot %s '%s': %s", action, path, err.Error())
	}
	// get an absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// when can this happen?
		return "", nil, herror.Internal(err, "")
	}
	// check whether there are any symbolic links involved
	hasLinks, resolved, err := ps.checkSymlinks(absPath)
	if err != nil {
		// when can this happen? we've already checked that the file or
		// directory exists
		return "", nil, herror.Internal(err, "")
	}
	if hasLinks {
		return "", nil, checkFileError("cannot %s '%s': path has symbolic links (use '%s' instead)", action, path, resolved)
	}
	// check whether it's a regular file or directory
	if !info.Mode().IsRegular() && !info.Mode().IsDir() {
		return "", nil, checkFileError("cannot %s '%s': not a regular file or directory", action, path)
	}
	// check whether it's a regular file, if required
	if mustBeRegularFile && !info.Mode().IsRegular() {
		return "", nil, checkFileError("cannot %s '%s': not a regular file", action, path)
	}
	// check whether it's a directory, if required
	if mustBeDirectory && !info.Mode().IsDir() {
		return "", nil, checkFileError("cannot %s '%s': not a directory", action, path)
	}
	// all okay
	return resolved, info, nil
}

func containedInAny(path string, dirs []string) bool {
	for _, dir := range dirs {
		if dir[len(dir)-1] != os.PathSeparator {
			dir = dir + string(os.PathSeparator)
		}
		if strings.HasPrefix(path, dir) {
			return true
		}
	}
	return false
}
