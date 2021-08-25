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
	expected := []db.DuplicateSet{{
		{Path: "/c/x", Size: 10248, ShortHash: nil, FullHash: nil},
		{Path: "/x", Size: 10248, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/.bar", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/.foo", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/c/.d/foo", Size: 100, ShortHash: nil, FullHash: nil},
	}}
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

func TestScanMinimumSize(t *testing.T) {
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
	err := ps.Scan([]string{"/"}, &ScanOptions{Minimum: 123})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{{
		{Path: "/c/x", Size: 10248, ShortHash: nil, FullHash: nil},
		{Path: "/x", Size: 10248, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanMaximumSize(t *testing.T) {
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
	err := ps.Scan([]string{"/"}, &ScanOptions{Maximum: 1000})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{{
		{Path: "/.bar", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/.foo", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/c/.d/foo", Size: 100, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanMinimumMaximumSize(t *testing.T) {
	fs := testfs.Read(`
/small1 [10 1]
/small2 [10 1]
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
/big1 [1000000 1]
/big2 [1000000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{Minimum: 50, Maximum: 20000})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	expected := []db.DuplicateSet{{
		{Path: "/c/x", Size: 10248, ShortHash: nil, FullHash: nil},
		{Path: "/x", Size: 10248, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/.bar", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/.foo", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/c/.d/foo", Size: 100, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanKeepOtherSizes(t *testing.T) {
	fs := testfs.Read(`
/small1 [10 1]
/small2 [10 1]
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
/big1 [1000000 1]
/big2 [1000000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{Maximum: 100000})
	check(t, err)
	err = ps.Scan([]string{"/"}, &ScanOptions{Minimum: 100000})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 4 {
		t.Fatalf("expected 4 duplicate sets, got %d", len(got))
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
	expected := []db.DuplicateSet{{
		{Path: filepath.Join(dir, "y"), Size: 1, ShortHash: nil, FullHash: nil},
		{Path: filepath.Join(dir, "z"), Size: 1, ShortHash: nil, FullHash: nil},
	}}
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
	expected := []db.DuplicateSet{{
		{Path: "/m/r/p/first.mp4", Size: 104857600, ShortHash: nil, FullHash: nil},
		{Path: "/m/r/u/x/0000.mp4", Size: 104857600, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestScanSameStart(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	s1 := append(bytes.Repeat([]byte{'x'}, 10485760), byte('1'))
	s2 := append(bytes.Repeat([]byte{'x'}, 10485760), byte('2'))
	afero.WriteFile(fs, "/a", s1, 0o644)
	afero.WriteFile(fs, "/b", s2, 0o644)
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
	afero.WriteFile(fs, "/a", s1, 0o644)
	afero.WriteFile(fs, "/b", s2, 0o644)
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected to find no duplicates, got %d", len(got))
	}
}

func TestScanPrefixSameSize(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	s1 := bytes.Repeat([]byte{'x'}, 10485760)
	s2 := bytes.Repeat([]byte{'x'}, 10485760)
	s2[123456] = 0
	afero.WriteFile(fs, "/a", s1, 0o644)
	afero.WriteFile(fs, "/b", s2, 0o644)
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
	expected = []db.DuplicateSet{{
		{Path: "/a/b/c/x", Size: 394820, ShortHash: nil, FullHash: nil},
		{Path: "/d/e/y", Size: 394820, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/a/z/e", Size: 1337, ShortHash: nil, FullHash: nil},
		{Path: "/d/e/f/g/h/z", Size: 1337, ShortHash: nil, FullHash: nil},
	}}
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
	expected := []db.DuplicateSet{{
		{Path: "/a/b/c/x", Size: 394820, ShortHash: nil, FullHash: nil},
		{Path: "/d/e/y", Size: 394820, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/a/b/c/q", Size: 10203, ShortHash: nil, FullHash: nil},
		{Path: "/y/q", Size: 10203, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/a/z/e", Size: 1337, ShortHash: nil, FullHash: nil},
		{Path: "/d/e/f/g/h/z", Size: 1337, ShortHash: nil, FullHash: nil},
	}, {
		{Path: "/a/b/c/z", Size: 1000, ShortHash: nil, FullHash: nil},
		{Path: "/z", Size: 1000, ShortHash: nil, FullHash: nil},
	}}
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
	expected := []db.DuplicateSet{{
		{Path: filepath.Join(dir, "x"), Size: 1, ShortHash: nil, FullHash: nil},
		{Path: filepath.Join(dir, "y"), Size: 1, ShortHash: nil, FullHash: nil},
	}}
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
	expected := []db.DuplicateSet{{
		{Path: filepath.Join(dir, "d1", "w"), Size: 1, ShortHash: nil, FullHash: nil},
		{Path: filepath.Join(dir, "d1", "x"), Size: 1, ShortHash: nil, FullHash: nil},
	}}
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

func TestScanIncrementalNoOverlap(t *testing.T) {
	fs := testfs.Read(`
/a/x1 [1000 1]
/a/x2 [1000 1]
/b/y1 [1234 2]
/b/y2 [1234 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	err = ps.Scan([]string{"/b"}, &ScanOptions{})
	check(t, err)
	got, err = ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 2 {
		t.Fatalf("expected 2 duplicate sets, got %d", len(got))
	}
}

func TestScanIncrementalOverlapAddition(t *testing.T) {
	fs := testfs.Read(`
/a/x1 [1000 1]
/a/x2 [1000 1]
/a/z [1337 3]
/b/y1 [1234 2]
/b/y2 [1234 2]
/b/z [1337 3]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	expected := []db.FileInfo{
		{Path: "/a/z", Size: 1337, ShortHash: nil, FullHash: nil},
		{Path: "/a/x1", Size: 1000, ShortHash: dummyHash, FullHash: dummyHash},
		{Path: "/a/x2", Size: 1000, ShortHash: dummyHash, FullHash: dummyHash},
	}
	got2, _ := ps.db.AllInfos()
	checkEquivalentInfos(t, expected, got2)
	err = ps.Scan([]string{"/b"}, &ScanOptions{})
	check(t, err)
	got, err = ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 3 {
		t.Fatalf("expected 3 duplicate sets, got %d", len(got))
	}
}

func TestScanIncrementalCommonPrefix(t *testing.T) {
	fs := testfs.Read(`
/a/x1 [9000 1]
/b/y1 [9999 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected no duplicate sets, got %d", len(got))
	}
	err = ps.Scan([]string{"/b"}, &ScanOptions{})
	check(t, err)
	got, err = ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected no duplicate sets, got %d", len(got))
	}
}

func TestScanIncrementalPartialToFull(t *testing.T) {
	fs := testfs.Read(`
/a/x [9000 1]
/b/y [9999 1]
/c/z [9000 1]
/d/x [9000 1]
	`).Mkfs()
	// fix up '/c/z' to make it differ
	z, _ := fs.OpenFile("/c/z", os.O_RDWR, 0755)
	z.WriteAt([]byte("this file is different"), 35)
	z.Close()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/a"}, &ScanOptions{})
	check(t, err)
	err = ps.Scan([]string{"/b"}, &ScanOptions{})
	check(t, err)
	err = ps.Scan([]string{"/c"}, &ScanOptions{})
	check(t, err)
	// at this point, we know about the files in dirs a, b, and c, and we
	// should have computed short hashes for /a/x and /c/z, but no full
	// hashes
	expected := []db.FileInfo{
		{Path: "/b/y", Size: 9999, ShortHash: nil, FullHash: nil},
		{Path: "/a/x", Size: 9000, ShortHash: dummyHash, FullHash: nil},
		{Path: "/c/z", Size: 9000, ShortHash: dummyHash, FullHash: nil},
	}
	got2, _ := ps.db.AllInfos()
	// checkEquivalentInfos(t, expected, got2)
	err = ps.Scan([]string{"/d"}, &ScanOptions{})
	check(t, err)
	got, err := ps.db.AllDuplicates()
	check(t, err)
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	if len(got[0]) != 2 {
		t.Fatalf("expected duplicate set to have 2 duplicates, got %d", len(got[0]))
	}
	expected = []db.FileInfo{
		{Path: "/b/y", Size: 9999, ShortHash: nil, FullHash: nil},
		{Path: "/a/x", Size: 9000, ShortHash: dummyHash, FullHash: dummyHash},
		{Path: "/c/z", Size: 9000, ShortHash: dummyHash, FullHash: nil},
		{Path: "/d/x", Size: 9000, ShortHash: dummyHash, FullHash: dummyHash},
	}
	got2, _ = ps.db.AllInfos()
	checkEquivalentInfos(t, expected, got2)
}

func TestScanIncrementalFindMore(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/b/x [1000 1]
/c/x [1000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/a"}, &ScanOptions{})
	ps.Scan([]string{"/b"}, &ScanOptions{})
	got, _ := ps.db.AllDuplicates()
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	if len(got[0]) != 2 {
		t.Fatalf("expected 2 duplicates in the set, got %d", len(got[0]))
	}
	ps.Scan([]string{"/c"}, &ScanOptions{})
	got, _ = ps.db.AllDuplicates()
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	if len(got[0]) != 3 {
		t.Fatalf("expected 3 duplicates in the set, got %d", len(got[0]))
	}
}

func TestScanIncrementalRepeatUnlink(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/a/y [1000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/a"}, &ScanOptions{})
	got, _ := ps.db.AllDuplicates()
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	fs.Remove("/a/y")
	ps.Scan([]string{"/a"}, &ScanOptions{})
	got, _ = ps.db.AllDuplicates()
	if len(got) != 0 {
		t.Fatalf("expected no duplicate sets, got %d", len(got))
	}
	infos, _ := ps.db.AllInfos()
	if len(infos) != 1 {
		t.Fatalf("expected 1 info, found %d", len(infos))
	}
}

func TestScanIncrementalRepeatNoticeChange(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/a/y [1000 1]
/a/z [1000 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/a"}, &ScanOptions{})
	got, _ := ps.db.AllDuplicates()
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	if len(got[0]) != 2 {
		t.Fatalf("expected 2 duplicates in the set, got %d", len(got[0]))
	}
	// now, we modify "/a/z" to match
	b, _ := afero.ReadFile(fs, "/a/x")
	afero.WriteFile(fs, "/a/z", b, 0755)
	ps.Scan([]string{"/a"}, &ScanOptions{})
	got, _ = ps.db.AllDuplicates()
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	if len(got[0]) != 3 {
		t.Fatalf("expected 3 duplicates in the set, got %d", len(got[0]))
	}
}
