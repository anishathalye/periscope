package periscope

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/testfs"

	"github.com/spf13/afero"
)

func TestScanBasic(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{
		{[]string{"/c/x", "/x"}, 10248, 0},
		{[]string{"/.bar", "/.foo", "/c/.d/foo"}, 100, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanZeroLength(t *testing.T) {
	fs := testfs.Read(`
/a [0 4]
/b [0 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected no duplicate sets, got %d", len(got))
	}
}

func TestScanNoAccess(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o000)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "z"), []byte{'b'}, 0o644)
	fs.Mkdir(filepath.Join(dir, "d"), 0o644)
	fs.Mkdir(filepath.Join(dir, "e"), 0o000)
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{dir}, &ScanOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates()
	expected := []db.DuplicateSet{
		{[]string{filepath.Join(dir, "y"), filepath.Join(dir, "z")}, 1, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanLargeFiles(t *testing.T) {
	fs := testfs.Read(`
/m/r/p/first.mp4 [104857600 12]
/m/r/u/x/0000.mp4 [104857600 12]
/m/r/p/second.mp4 [104857600 0]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{
		{[]string{"/m/r/p/first.mp4", "/m/r/u/x/0000.mp4"}, 104857600, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanSameStart(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	s1 := append(bytes.Repeat([]byte{'x'}, 10485760), byte('1'))
	s2 := append(bytes.Repeat([]byte{'x'}, 10485760), byte('2'))
	ioutil.WriteFile("/a", s1, 0o644)
	ioutil.WriteFile("/b", s2, 0o644)
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected to find no duplicates, got %d", len(got))
	}
}

func TestScanPrefix(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	s1 := bytes.Repeat([]byte{'x'}, 10485760)
	s2 := bytes.Repeat([]byte{'x'}, 10485761)
	ioutil.WriteFile("/a", s1, 0o644)
	ioutil.WriteFile("/b", s2, 0o644)
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected to find no duplicates, got %d", len(got))
	}
}

func TestScanMultiple(t *testing.T) {
	fs := testfs.Read(`
/a/b/c/x [394820 33]
/d/e/y [394820 33]
/a/b/c/q [10203 2]
/a/z/e [1337 3]
/d/e/f/g/h/z [1337 3]
/y/q [10203 2]
/a/b/c/z [1000 1]
/z [1000 1]
/x/z [1000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)

	err := ps.Scan([]string{"/a/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{}
	checkEquivalentDuplicateSet(t, expected, got)

	err = ps.Scan([]string{"/d", "/a"}, &ScanOptions{})
	check(t, err)
	got, err = ps.db.AllDuplicates()
	check(t, err)
	expected = []db.DuplicateSet{
		{[]string{"/a/b/c/x", "/d/e/y"}, 394820, 0},
		{[]string{"/a/z/e", "/d/e/f/g/h/z"}, 1337, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanOverlap(t *testing.T) {
	fs := testfs.Read(`
/a/b/c/x [394820 33]
/d/e/y [394820 33]
/a/b/c/q [10203 2]
/a/z/e [1337 3]
/d/e/f/g/h/z [1337 3]
/y/q [10203 2]
/a/b/c/z [1000 1]
/z [1000 1]
/x/z [1000 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/", "/a", "/d"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{
		{[]string{"/a/b/c/x", "/d/e/y"}, 394820, 0},
		{[]string{"/a/b/c/q", "/y/q"}, 10203, 0},
		{[]string{"/a/z/e", "/d/e/f/g/h/z"}, 1337, 0},
		{[]string{"/a/b/c/z", "/z"}, 1000, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanNonexistent(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/b/x [1000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a", "/b", "/c"}, &ScanOptions{})
	checkErr(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected to find no duplicates, got %d", len(got))
	}
}

func TestScanFile(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/b/x [1000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a", "/b/x"}, &ScanOptions{})
	checkErr(t, err)
}

func TestScanNoReadSymlinks(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Symlink(filepath.Join(dir, "x"), filepath.Join(dir, "z"))
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{dir}, &ScanOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates()
	expected := []db.DuplicateSet{
		{[]string{filepath.Join(dir, "x"), filepath.Join(dir, "y")}, 1, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanNoDescendSymlinks(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d1"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d1", "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d1", "x"), []byte{'a'}, 0o644)
	os.Symlink(filepath.Join(dir, "d1"), filepath.Join(dir, "d2"))
	ps, _, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	got, _ := ps.db.AllDuplicates()
	expected := []db.DuplicateSet{
		{[]string{filepath.Join(dir, "d1", "w"), filepath.Join(dir, "d1", "x")}, 1, 0},
	}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanSymlink(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d1"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d1", "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d1", "x"), []byte{'a'}, 0o644)
	os.Symlink(filepath.Join(dir, "d1"), filepath.Join(dir, "d2"))
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{filepath.Join(dir, "d2")}, &ScanOptions{})
	checkErr(t, err)
}

func TestScanInsideSymlink(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d1"), 0o755)
	os.Mkdir(filepath.Join(dir, "d1", "d2"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d1", "d2", "w"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d1", "d2", "x"), []byte{'a'}, 0o644)
	os.Symlink(filepath.Join(dir, "d1"), filepath.Join(dir, "d3"))
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{filepath.Join(dir, "d2")}, &ScanOptions{})
	checkErr(t, err)
}
