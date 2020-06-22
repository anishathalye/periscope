package testfs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

type FileDesc struct {
	Path string
	Size int64
	Seed int64
}

type Fs struct {
	files []FileDesc
}

func New(files []FileDesc) *Fs {
	sort.Sort(byPath(files))
	return &Fs{files}
}

type byPath []FileDesc

func (a byPath) Len() int {
	return len(a)
}

func (a byPath) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byPath) Less(i, j int) bool {
	return a[i].Path < a[j].Path
}

func genFile(w io.Writer, size int64, seed int64) {
	if size == 0 {
		return
	}
	if size < 8 {
		panic("testfs: doesn't support 0 < size < 8")
	}
	binary.Write(w, binary.LittleEndian, seed)
	r := rand.New(rand.NewSource(seed))
	io.CopyN(w, r, size-8)
}

func allDirs(path string) []string {
	var dirs []string
	for path != "/" {
		path = filepath.Dir(path)
		dirs = append(dirs, path)
	}
	rdirs := make([]string, len(dirs))
	for i := range dirs {
		rdirs[i] = dirs[len(dirs)-1-i]
	}
	return rdirs
}

func (fs *Fs) Mkfs() afero.Fs {
	afs := afero.NewMemMapFs()
	// work around a bug in afero where "/" has empty permissions, so it's
	// not marked as a directory
	afs.Chmod("/", 0o755|os.ModeDir)
	for _, spec := range fs.files {
		// work around a bug in afero where MkdirAll doesn't properly
		// create intermediate directories: they end up with empty
		// permissions, and so they are not marked as directories
		//
		// so instead we manually create all the intermediate
		// directories
		for _, dir := range allDirs(spec.Path) {
			if ex, _ := afero.DirExists(afs, dir); !ex {
				afs.Mkdir(dir, 0o755)
			}
		}
		f, _ := afs.Create(spec.Path)
		genFile(f, spec.Size, spec.Seed)
	}
	return afs
}

func From(fs afero.Fs) *Fs {
	var files []FileDesc
	afero.Walk(fs, "/", func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			f, _ := fs.Open(path)
			size := info.Size()
			var seed int64
			if size >= 8 {
				binary.Read(f, binary.LittleEndian, &seed)
			}
			files = append(files, FileDesc{path, size, seed})
		}
		return nil
	})
	sort.Sort(byPath(files))
	return &Fs{files}
}

func Equal(afs afero.Fs, reference *Fs) bool {
	fs := From(afs)
	return fs.Equal(reference)
}

func (fs *Fs) Equal(other *Fs) bool {
	if len(fs.files) != len(other.files) {
		return false
	}
	for i := range fs.files {
		if fs.files[i] != other.files[i] {
			return false
		}
	}
	return true
}

func ShowIndent(afs afero.Fs, n int) string {
	return From(afs).ShowIndent(n)
}

func Show(afs afero.Fs) string {
	return ShowIndent(afs, 0)
}

func (fs *Fs) ShowIndent(n int) string {
	out := new(bytes.Buffer)
	for _, file := range fs.files {
		fmt.Fprintf(out, strings.Repeat(" ", n))
		fmt.Fprintf(out, "%s [%d %d]\n", file.Path, file.Size, file.Seed)
	}
	return out.String()
}

func (fs *Fs) Show() string {
	return fs.ShowIndent(0)
}

func Read(s string) *Fs {
	re := regexp.MustCompile(`(.*) \[(\d+) (\d+)\]\n`)
	var files []FileDesc
	for _, match := range re.FindAllStringSubmatch(s, -1) {
		size, _ := strconv.ParseInt(match[2], 10, 64)
		seed, _ := strconv.ParseInt(match[3], 10, 64)
		files = append(files, FileDesc{
			Path: match[1],
			Size: size,
			Seed: seed,
		})
	}
	return New(files)
}
