package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestLsBasic(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/c [1191 3]
/d2/a [10000 1]
/d2/b [1392 2]
/d2/c [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
1 a
1 b
  c
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsOutsideScan(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/a2 [10000 1]
/d1/b [1002 5]
/d2/b [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/d2"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
a
a2
b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsCountDuplicates(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/c [1191 3]
/d2/a [10000 1]
/d2/b [1392 2]
/d2/c [1002 5]
/d3/a [10000 1]
/d4/a [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
3 a
1 b
  c
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsCountOver10(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1234 2]
/d2/a1 [10000 1]
/d2/a2 [10000 1]
/d2/a3 [10000 1]
/d2/a4 [10000 1]
/d2/a5 [10000 1]
/d2/a6 [10000 1]
/d2/a7 [10000 1]
/d2/a8 [10000 1]
/d2/a9 [10000 1]
/d2/a10 [10000 1]
/d2/a11 [10000 1]
/d2/a12 [10000 1]
/d2/b [1234 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
12 a
1  b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsDuplicateSameDir(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/a2 [10000 1]
/d1/a3 [10000 1]
/d1/b [1392 2]
/d2/a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
3 a
3 a2
3 a3
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsNoCountUnique(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1234 2]
/d1/c [1111 3]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
a
b
c
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsShowOnlyUnique(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/c [1191 3]
/d1/d [1337 3]
/d2/a [10000 1]
/d2/b [1392 2]
/d2/c [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{Unique: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
c
d
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsShowOnlyDuplicate(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/c [1191 3]
/d1/d [1337 3]
/d2/a [10000 1]
/d2/b [1392 2]
/d2/c [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{Duplicate: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
1 a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsDirectories(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/dir/x [1191 3]
/d2/a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
1 a
1 b
d dir
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsSpecialModes(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	os.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Symlink(filepath.Join(dir, "x"), filepath.Join(dir, "z"))
	ps, out, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	err := ps.Ls([]string{dir}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
1 x
1 y
L z
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsVerbose(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/a2 [10000 1]
/d1/a3 [10000 1]
/d1/b [1392 2]
/d2/a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{Verbose: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
3 a
    /d1/a2
    /d1/a3
    /d2/a
3 a2
    /d1/a
    /d1/a3
    /d2/a
3 a3
    /d1/a
    /d1/a2
    /d2/a
1 b
    /d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsVerboseRelative(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/a2 [10000 1]
/d1/a3 [10000 1]
/d1/b [1392 2]
/d2/a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{Verbose: true, Relative: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
3 a
    a2
    a3
    /d2/a
3 a2
    a
    a3
    /d2/a
3 a3
    a
    a2
    /d2/a
1 b
    /d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsVerboseRelativeLong(t *testing.T) {
	fs := testfs.Read(`
/long/directory/a [10 1]
/long/directory/b [10 1]
/long/x/c [10 1]
/other/directory/d [10 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/long/directory"}, &LsOptions{Verbose: true, Relative: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
3 a
    b
    ../x/c
    /other/directory/d
3 b
    a
    ../x/c
    /other/directory/d
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsHidden(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/.b [1392 2]
/d2/.a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
1 a
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
	out.Reset()
	err = ps.Ls([]string{"/d1"}, &LsOptions{All: true})
	check(t, err)
	got = strings.TrimSpace(out.String())
	expected = strings.TrimSpace(`
1 .b
1 a
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsMultiple(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/.b [1392 2]
/d2/.a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1", "/d2"}, &LsOptions{All: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/d1:
1 .b
1 a

/d2:
1 .a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsInaccessible(t *testing.T) {
	dir := tempDir()
	defer os.RemoveAll(dir)
	inner := filepath.Join(dir, "d")
	os.Mkdir(inner, 0111)
	fs := afero.NewOsFs()
	ps, out, stderr := newTest(fs)
	err := ps.Ls([]string{inner}, &LsOptions{})
	checkErr(t, err)
	got := strings.TrimRight(out.String(), "\n")
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "permission denied"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestLsNonexistent(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/.b [1392 2]
	`).Mkfs()
	ps, out, stderr := newTest(fs)
	err := ps.Ls([]string{"/d2"}, &LsOptions{})
	checkErr(t, err)
	got := strings.TrimRight(out.String(), "\n")
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "no such file or directory"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestLsFile(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
	`).Mkfs()
	ps, out, stderr := newTest(fs)
	err := ps.Ls([]string{"/d1/a"}, &LsOptions{})
	checkErr(t, err)
	got := strings.TrimRight(out.String(), "\n")
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "not a directory"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestLsSomeInaccessible(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d3/a [10000 1]
/d3/b [1337 2]
	`).Mkfs()
	ps, out, stderr := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1", "/d2", "/d3"}, &LsOptions{})
	checkErr(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/d1:
1 a

/d3:
1 a
  b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
	expected = "cannot list '/d2': no such file or directory"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestLsRejectSymlink(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	os.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Symlink(dir, filepath.Join(dir, "rec"))
	ps, out, stderr := newTest(fs)
	err := ps.Ls([]string{filepath.Join(dir, "rec")}, &LsOptions{})
	checkErr(t, err)
	got := strings.TrimRight(out.String(), "\n")
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
	expected := "path has symbolic links"
	got = stderr.String()
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stderr to contain '%s', was '%s'", expected, got)
	}
}

func TestLsRecursive(t *testing.T) {
	fs := testfs.Read(`
/foo.txt [100 1]
/Pictures/2020/January/IMG_1234.JPG [1000 3]
/Pictures/2021/August/DSC1337.HEIC [1000 2]
/Pictures/2021/August/DSC1337.copy.HEIC [1000 2]
/Temporary/recovered.heic [1000 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/"}, &LsOptions{Recursive: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/:
d Pictures
d Temporary
  foo.txt

/Pictures:
d 2020
d 2021

/Pictures/2020:
d January

/Pictures/2020/January:
IMG_1234.JPG

/Pictures/2021:
d August

/Pictures/2021/August:
2 DSC1337.HEIC
2 DSC1337.copy.HEIC

/Temporary:
2 recovered.heic
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsRecursiveDuplicateOnly(t *testing.T) {
	fs := testfs.Read(`
/foo.txt [100 1]
/Pictures/2020/January/IMG_1234.JPG [1000 3]
/Pictures/2021/August/DSC1337.HEIC [1000 2]
/Pictures/2021/August/DSC1337.copy.HEIC [1000 2]
/Temporary/recovered.heic [1000 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/"}, &LsOptions{Recursive: true, Duplicate: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/Pictures/2021/August:
2 DSC1337.HEIC
2 DSC1337.copy.HEIC

/Temporary:
2 recovered.heic
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsRecursiveHidden(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/.b [1392 2]
/d2/.a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/"}, &LsOptions{Recursive: true, Duplicate: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
/d1:
1 a

/d2:
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
	out.Reset()
	err = ps.Ls([]string{"/"}, &LsOptions{Recursive: true, Duplicate: true, All: true})
	check(t, err)
	got = strings.TrimSpace(out.String())
	expected = strings.TrimSpace(`
/d1:
1 .b
1 a

/d2:
1 .a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsRelative(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	os.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	os.WriteFile(filepath.Join(dir, "d", "a"), []byte{'b'}, 0o644)
	os.WriteFile(filepath.Join(dir, "d", "b"), []byte{'b'}, 0o644)
	ps, out, _ := newTest(fs)
	oldWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	ps.Scan([]string{"."}, &ScanOptions{})
	err = ps.Ls([]string{".", "d"}, &LsOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
.:
d d
1 x
1 y

d:
1 a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsRelativeRecursive(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	os.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	os.WriteFile(filepath.Join(dir, "d", "a"), []byte{'b'}, 0o644)
	os.WriteFile(filepath.Join(dir, "d", "b"), []byte{'b'}, 0o644)
	ps, out, _ := newTest(fs)
	oldWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	ps.Scan([]string{"."}, &ScanOptions{})
	err = ps.Ls([]string{"."}, &LsOptions{Recursive: true, Duplicate: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
.:
1 x
1 y

d:
1 a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsRelativeDir(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "x"), []byte{'a'}, 0o644)
	os.WriteFile(filepath.Join(dir, "y"), []byte{'a'}, 0o644)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	os.WriteFile(filepath.Join(dir, "d", "a"), []byte{'b'}, 0o644)
	os.WriteFile(filepath.Join(dir, "d", "b"), []byte{'b'}, 0o644)
	ps, out, _ := newTest(fs)
	oldWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	ps.Scan([]string{"."}, &ScanOptions{})
	err = ps.Ls([]string{"d"}, &LsOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 a
1 b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsFilesOnly(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/subdir/c [1191 3]
/d1/subdir/d [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Ls([]string{"/d1"}, &LsOptions{Files: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
a
b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsFilesOnlyRecursive(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/subdir/c [1191 3]
/d1/subdir/d [1002 5]
/d1/subdir2/e [1234 6]
	`).Mkfs()
	err := fs.Mkdir("/d1/empty", 0o755)
	check(t, err)
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err = ps.Ls([]string{"/d1"}, &LsOptions{Files: true, Recursive: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/d1:
a
b

/d1/subdir:
c
d

/d1/subdir2:
e
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestLsFilesOnlyRecursiveUnique(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [1392 2]
/d1/c [1191 3]
/d1/d [1002 5]
/d1/subdir/c [1191 3]
/d1/subdir/d [1002 5]
/d1/subdir2/e [1234 6]
	`).Mkfs()
	err := fs.Mkdir("/d1/empty", 0o755)
	check(t, err)
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err = ps.Ls([]string{"/d1"}, &LsOptions{Files: true, Recursive: true, Unique: true})
	check(t, err)
	got := strings.TrimRight(out.String(), "\n")
	expected := strings.TrimSpace(`
/d1:
a
b

/d1/subdir2:
e
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}
