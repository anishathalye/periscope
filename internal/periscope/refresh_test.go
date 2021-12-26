package periscope

import (
	"github.com/anishathalye/periscope/internal/db"
	"github.com/anishathalye/periscope/internal/testfs"

	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestRefreshBasic(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/b")
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: "/a", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshNoChange(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: "/a", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshModifyFile(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	afero.WriteFile(fs, "/a", []byte{'a', 'b', 'c'}, 0o644)
	// double check that they're different
	hashA, _ := ps.hashFile("/a")
	hashB, _ := ps.hashFile("/b")
	if bytes.Equal(hashA, hashB) {
		t.Fatal("files are still equal")
	}
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	// refresh only checks that the files are still there, so we should
	// still see it as a duplicate
	expected := []db.DuplicateSet{{
		{Path: "/a", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshMove(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Rename("/a", "/f")
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)

	ps.Scan([]string{"/"}, &ScanOptions{})
	got, _ = ps.db.AllDuplicates("")
	expected = []db.DuplicateSet{{
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/f", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshReplaceFileWithDirectory(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/a")
	fs.Mkdir("/a", 0o755)
	afero.WriteFile(fs, "/a/x", []byte{'x'}, 0o644)
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshRemoveSingletons(t *testing.T) {
	fs := testfs.Read(`
/a/x/1 [10000 1]
/b/2 [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/a/x/1")
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	if len(got) != 0 {
		t.Fatalf("expected no duplicate sets, got %d", len(got))
	}
}

func TestRefreshPreserveNonSingletons(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c/d/e [10000 1]
/f [1337 2]
/g [1337 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/a")
	fs.Remove("/f")
	err := ps.Refresh(&RefreshOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: "/b", Size: 10000, ShortHash: nil, FullHash: nil},
		{Path: "/c/d/e", Size: 10000, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshPermissionError(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d1"), 0o755)
	os.Mkdir(filepath.Join(dir, "d2"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d1", "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d1", "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d2", "y"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d2", "z"), []byte{'b'}, 0o644)
	ps, _, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.Chmod(filepath.Join(dir, "d1"), 0o000)
	ps.Refresh(&RefreshOptions{})
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: filepath.Join(dir, "d2", "y"), Size: 1, ShortHash: nil, FullHash: nil},
		{Path: filepath.Join(dir, "d2", "z"), Size: 1, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshNonRegularFile(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "z"), []byte{'b'}, 0o644)
	ps, _, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.Remove(filepath.Join(dir, "w"))
	os.Symlink(filepath.Join(dir, "x"), filepath.Join(dir, "w"))
	ps.Refresh(&RefreshOptions{})
	got, _ := ps.db.AllDuplicates("")
	expected := []db.DuplicateSet{{
		{Path: filepath.Join(dir, "y"), Size: 1, ShortHash: nil, FullHash: nil},
		{Path: filepath.Join(dir, "z"), Size: 1, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRefreshSymlinkDir(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	os.Mkdir(filepath.Join(dir, "d2"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d", "x"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d2", "y"), []byte{'b'}, 0o644)
	ps, _, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.RemoveAll(filepath.Join(dir, "d2"))
	os.Symlink(filepath.Join(dir, "d"), filepath.Join(dir, "d2"))
	ps.Refresh(&RefreshOptions{})
	got, _ := ps.db.AllDuplicates("")
	if len(got) != 0 {
		t.Fatalf("expected no duplicates, got %d", len(got))
	}
}
