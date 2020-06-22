package periscope

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/anishathalye/periscope/herror"

	"golang.org/x/crypto/blake2b"
)

const initialChunkSize = 4 * 1024
const readChunkSize = 1024 * 1024

const HashSize = blake2b.Size256

func (ps *Periscope) hashPartial(path string, key []byte, short bool) ([HashSize]byte, error) {
	var buf []byte
	if short {
		buf = make([]byte, initialChunkSize)
	} else {
		buf = make([]byte, readChunkSize)
	}
	var ret [HashSize]byte
	h, err := blake2b.New256(key)
	if err != nil {
		return ret, err
	}
	f, err := ps.fs.Open(path)
	if err != nil {
		return ret, err
	}
	defer f.Close()

	if !short {
		if _, err := f.Seek(initialChunkSize, os.SEEK_SET); err != nil {
			return ret, err
		}
	}

	if short {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return ret, err
		}
		h.Write(buf[:n])
	} else {
		if _, err := io.CopyBuffer(h, f, buf); err != nil {
			return ret, err
		}
	}

	h.Sum(ret[:0])
	return ret, nil
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
