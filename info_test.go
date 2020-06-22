package periscope

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anishathalye/periscope/testfs"

	"github.com/spf13/afero"
)

func TestInfoBasic(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	err = ps.Info([]string{"/b"}, &InfoOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
2 /b
  /a
  /c
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestInfoMultiple(t *testing.T) {
	fs := testfs.Read(`
/a [10000 1]
/b [10000 1]
/c [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	err = ps.Info([]string{"/b", "/a"}, &InfoOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
2 /b
  /a
  /c

2 /a
  /b
  /c
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestInfoRelative(t *testing.T) {
	fs := testfs.Read(`
/long/directory/path/a [10000 1]
/long/directory/path/b [10000 1]
/long/directory/other/a [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Scan([]string{"/long/directory"}, &ScanOptions{})
	check(t, err)
	err = ps.Info([]string{"/long/directory/path/a"}, &InfoOptions{Relative: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
2 /long/directory/path/a
  ../other/a
  b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestInfoRelativeTooLong(t *testing.T) {
	fs := testfs.Read(`
/long/directory/path/a [10000 1]
/long/directory/path/b [10000 1]
/s/a [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	err = ps.Info([]string{"/long/directory/path/a"}, &InfoOptions{Relative: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
2 /long/directory/path/a
  b
  /s/a
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestInfoDirectory(t *testing.T) {
	fs := testfs.Read(`
/long/directory/path/a [10000 1]
/s/a [10000 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	err = ps.Info([]string{"/long/directory/path"}, &InfoOptions{Relative: true})
	checkErr(t, err)
}

func TestInfoBadSymlinks(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "b"))
	os.Symlink(filepath.Join(dir, "b"), filepath.Join(dir, "a"))
	ps, _, _ := newTest(fs)
	err := ps.Info([]string{filepath.Join(dir, "a")}, &InfoOptions{})
	checkErr(t, err)
}
