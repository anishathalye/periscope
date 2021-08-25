package periscope

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anishathalye/periscope/db"
	"github.com/anishathalye/periscope/testfs"

	"github.com/spf13/afero"
)

func TestRmBasic(t *testing.T) {
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
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/c/x", "/.foo"}, &RmOptions{})
	check(t, err)
	expected := testfs.Read(`
/.bar [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/x [10248 3]
/c/.d/foo [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmRecursive(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/c"}, &RmOptions{Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/x [10248 3]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmNoRecursive(t *testing.T) {
	fs := testfs.Read(`
/d/x [100 1]
/x [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/d"}, &RmOptions{})
	checkErr(t, err)
	expected := testfs.Read(`
/d/x [100 1]
/x [100 1]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmRemovesFromDB(t *testing.T) {
	fs := testfs.Read(`
/a [100 1]
/b [100 1]
/c [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/a"}, &RmOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates()
	expected := []db.DuplicateSet{{
		{Path: "/b", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/c", Size: 100, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRmPrunesSingletons(t *testing.T) {
	fs := testfs.Read(`
/a [100 1]
/b [100 1]
/c [100 1]
/d [200 2]
/e [200 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/d"}, &RmOptions{})
	check(t, err)
	got, _ := ps.db.AllDuplicates()
	expected := []db.DuplicateSet{{
		{Path: "/a", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/b", Size: 100, ShortHash: nil, FullHash: nil},
		{Path: "/c", Size: 100, ShortHash: nil, FullHash: nil},
	}}
	checkEquivalentDuplicateSet(t, expected, got)
}

func TestRmMultiple(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/c", "/.bar", "/a"}, &RmOptions{Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/.foo [100 4]
/b [1024 0]
/x [10248 3]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmNoDuplicates(t *testing.T) {
	fs := testfs.Read(`
/a [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Rm([]string{"/a"}, &RmOptions{})
	checkErr(t, err)
	expected := testfs.Read(`
/a [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmRecursiveDisappeared(t *testing.T) {
	fs := testfs.Read(`
/d/a [100 4]
/d/b [100 4]
/c [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/d/b")
	err := ps.Rm([]string{"/d"}, &RmOptions{Recursive: true})
	if err != nil {
		t.Fatal(err.Herror(true))
	}
	check(t, err)
	expected := testfs.Read(`
/c [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmMultiplePartialFail(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, stderr := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/c", "/.bar", "/.foo", "/a", "/nonexistent"}, &RmOptions{Recursive: true})
	checkErr(t, err)
	expected := testfs.Read(`
/.foo [100 4]
/b [1024 0]
/x [10248 3]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
	expectedErr := "no duplicates"
	got := stderr.String()
	if !strings.Contains(got, expectedErr) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expectedErr, got)
	}
}

func TestRmContainedFile(t *testing.T) {
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
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/x"}, &RmOptions{HasContained: true, Contained: "/c"})
	check(t, err)
	expected := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/c/x [10248 3]
/c/.d/foo [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmNoContainsFile(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/a"}, &RmOptions{HasContained: true, Contained: "/c"})
	checkErr(t, err)
	expected := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmContainedRecursive(t *testing.T) {
	fs := testfs.Read(`
/u/a [100 1]
/u/b [100 2]
/u/c [100 3]
/u/d [100 4]
/a [100 1]
/o/b [100 2]
/o/c [100 3]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/u"}, &RmOptions{HasContained: true, Contained: "/o", Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/u/a [100 1]
/u/d [100 4]
/a [100 1]
/o/b [100 2]
/o/c [100 3]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmContainedTryDelete(t *testing.T) {
	fs := testfs.Read(`
/u/a [100 1]
/u/b [100 2]
/u/c [100 3]
/u/d [100 4]
/a [100 1]
/o/b [100 2]
/o/c [100 3]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/o"}, &RmOptions{HasContained: true, Contained: "/o", Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/u/a [100 1]
/u/b [100 2]
/u/c [100 3]
/u/d [100 4]
/a [100 1]
/o/b [100 2]
/o/c [100 3]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmRecursiveEntireSet(t *testing.T) {
	fs := testfs.Read(`
/d/a [100 1]
/d/a2 [100 1]
/d/b [100 2]
/b [100 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/d"}, &RmOptions{Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/d/a [100 1]
/d/a2 [100 1]
/b [100 2]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmRecursiveSingletonsAndUniques(t *testing.T) {
	fs := testfs.Read(`
/d/a [100 1]
/d/b [100 2]
/a [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/a")
	// "/a" will still be in DB
	err := ps.Rm([]string{"/d"}, &RmOptions{Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/d/a [100 1]
/d/b [100 2]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmSomeDuplicatesDisappeared(t *testing.T) {
	fs := testfs.Read(`
/a [100 1]
/b [100 1]
/c [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	fs.Remove("/b")
	// "/b" will still be in DB
	err := ps.Rm([]string{"/a"}, &RmOptions{})
	check(t, err)
	expected := testfs.Read(`
/c [100 1]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmChanged(t *testing.T) {
	fs := testfs.Read(`
/a [100 1]
/b [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	afero.WriteFile(fs, "/b", []byte{'x'}, 0o644)
	// "/b" will still be in DB, though it's no longer a dupe
	err := ps.Rm([]string{"/a"}, &RmOptions{})
	checkErr(t, err)
	ex, xerr := afero.Exists(fs, "/a")
	check(t, xerr)
	if !ex {
		t.Fatalf("'/a' was deleted")
	}
}

func TestRmDiverged(t *testing.T) {
	fs := testfs.Read(`
/c [200 2]
/d/a [100 1]
/d/b [100 1]
/x [100 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	f1, _ := fs.Create("/d/a")
	f2, _ := fs.Open("/c")
	io.Copy(f1, f2)
	f1.Close()
	f2.Close()
	// "/d/a" and "/d/b" have diverged, even though both have duplicates
	err := ps.Rm([]string{"/d"}, &RmOptions{Recursive: true})
	check(t, err)
	expected := testfs.Read(`
/c [200 2]
/d/a [200 2]
/d/b [100 1]
/x [100 1]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
	// if we re-scan and try again, it will work
	ps.Scan([]string{"/"}, &ScanOptions{})
	err = ps.Rm([]string{"/d"}, &RmOptions{Recursive: true})
	check(t, err)
	expected = testfs.Read(`
/c [200 2]
/x [100 1]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmNoSymlinks(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Symlink(dir, filepath.Join(dir, "rec"))
	ps, out, stderr := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	err := ps.Rm([]string{filepath.Join(dir, "rec", "x")}, &RmOptions{})
	checkErr(t, err)
	got := strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "path has symbolic links"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestRmReplacedBySymlink(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	ps, out, stderr := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.Remove(filepath.Join(dir, "y"))
	os.Symlink(filepath.Join(dir, "x"), filepath.Join(dir, "y"))
	err := ps.Rm([]string{filepath.Join(dir, "y")}, &RmOptions{})
	checkErr(t, err)
	got := strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "path has symbolic links"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
	stderr.Reset()

	err = ps.Rm([]string{filepath.Join(dir, "x")}, &RmOptions{})
	checkErr(t, err)
	got = strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected = "no duplicates"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestRmDryRun(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/c", "/.bar", "/a"}, &RmOptions{Recursive: true, DryRun: true})
	check(t, err)
	expected := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 0]
/c/x [10248 3]
/x [10248 3]
/c/.d/foo [100 4]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmLostPermission(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	ps, out, stderr := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.Chmod(filepath.Join(dir, "y"), 0o000)
	err := ps.Rm([]string{filepath.Join(dir, "y")}, &RmOptions{})
	checkErr(t, err)
	got := strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "permission denied"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
	stderr.Reset()

	err = ps.Rm([]string{filepath.Join(dir, "x")}, &RmOptions{})
	checkErr(t, err)
	got = strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected = "no duplicates"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestRmPartialPermission(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d", "x"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d", "y"), []byte{'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "z"), []byte{'a'}, 0o644)
	ps, _, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	os.Chmod(filepath.Join(dir, "d", "y"), 0o000)
	err := ps.Rm([]string{filepath.Join(dir, "d")}, &RmOptions{Recursive: true})
	check(t, err)
	_, xerr := os.Stat(filepath.Join(dir, "d", "x"))
	if !os.IsNotExist(xerr) {
		t.Fatalf("expected d/x to be deleted")
	}
	info, xerr := os.Stat(filepath.Join(dir, "d", "y"))
	if xerr != nil || !info.Mode().IsRegular() {
		t.Fatal("expected d/y to be preserved")
	}
}

func TestRmVerbose(t *testing.T) {
	fs := testfs.Read(`
/.bar [100 4]
/.foo [100 4]
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
/c/x [10248 3]
/x [10248 3]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/.foo", "/c"}, &RmOptions{Recursive: true, Verbose: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	// have to be a bit careful when setting up this test, because in
	// general, rm's output is nondeterministic, because it uses maps, so
	// it can delete in different orders; in this particular test, there's
	// only one legal output.
	expected := strings.TrimSpace(`
rm /.foo
rm /c/x
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestRmContainedCommonPrefix(t *testing.T) {
	fs := testfs.Read(`
/a/y [100 1]
/aa/x [200 2]
/b/x [200 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/b/x"}, &RmOptions{HasContained: true, Contained: "/a"})
	checkErr(t, err)
	expected := testfs.Read(`
/a/y [100 1]
/aa/x [200 2]
/b/x [200 2]
	`)
	if !testfs.Equal(fs, expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), testfs.ShowIndent(fs, 2))
	}
}

func TestRmArbitrary(t *testing.T) {
	fs := testfs.Read(`
/a/x [1000 1]
/a/x2 [1000 1]
/a/x3 [1000 1]
/a/y [2000 2]
/a/y2 [2000 2]
/a/z [2000 3]
/b/y [2000 2]
/b/z [2000 3]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Rm([]string{"/a"}, &RmOptions{Recursive: true, Arbitrary: true})
	check(t, err)
	if ex, _ := afero.Exists(fs, "/a/y"); ex {
		t.Fatal("expected '/a/y' to be deleted")
	}
	if ex, _ := afero.Exists(fs, "/a/y2"); ex {
		t.Fatal("expected '/a/y2' to be deleted")
	}
	if ex, _ := afero.Exists(fs, "/b/y"); !ex {
		t.Fatal("expected '/b/y' to exist")
	}
	if ex, _ := afero.Exists(fs, "/a/z"); ex {
		t.Fatal("expected '/a/z' to be deleted")
	}
	if ex, _ := afero.Exists(fs, "/b/z"); !ex {
		t.Fatal("expected '/b/z' to exist")
	}
	x := 0
	if ex, _ := afero.Exists(fs, "/a/x"); ex {
		x++
	}
	if ex, _ := afero.Exists(fs, "/a/x2"); ex {
		x++
	}
	if ex, _ := afero.Exists(fs, "/a/x3"); ex {
		x++
	}
	if x != 1 {
		t.Fatal("expected exactly one of {'/a/x', '/a/x2', '/a/x3'} to exist")
	}
}
