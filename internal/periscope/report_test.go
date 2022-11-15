package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestReportBasic(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [13923446 2]
/d1/c [1191 3]
/d2/a [10000 1]
/d2/b [13923446 2]
/d2/c [1002 5]
/d3/a [10000 1]
/.x [123 5]
/.y [123 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Report("", &ReportOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
14 MB
  /d1/b
  /d2/b

10 kB
  /d1/a
  /d2/a
  /d3/a

123 B
  /.x
  /.y
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestReportSubdirectory(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [13923446 2]
/d1/c [1191 3]
/d2/a [10000 1]
/d2/b [13923446 2]
/d2/c [1002 5]
/d3/a [10000 1]
/.x [123 5]
/.y [123 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Report("/d3", &ReportOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
10 kB
  /d1/a
  /d2/a
  /d3/a
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestReportEmpty(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Report("", &ReportOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
}

func TestReportRelative(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a', 'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a', 'a'}, 0o644)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d", "a"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d", "b"), []byte{'b'}, 0o644)
	ps, out, _ := newTest(fs)
	oldWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	os.Chdir(dir)
	defer os.Chdir(oldWd)
	ps.Scan([]string{"."}, &ScanOptions{})
	err = ps.Report("", &ReportOptions{Relative: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
2 B
  x
  y

1 B
  d/a
  d/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestReportRelativeArgument(t *testing.T) {
	fs := afero.NewOsFs()
	dir := tempDir()
	defer os.RemoveAll(dir)
	ioutil.WriteFile(filepath.Join(dir, "x"), []byte{'a', 'a'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "y"), []byte{'a', 'a'}, 0o644)
	os.Mkdir(filepath.Join(dir, "d"), 0o755)
	ioutil.WriteFile(filepath.Join(dir, "d", "a"), []byte{'b'}, 0o644)
	ioutil.WriteFile(filepath.Join(dir, "d", "b"), []byte{'b'}, 0o644)
	ps, out, _ := newTest(fs)
	ps.Scan([]string{dir}, &ScanOptions{})
	err := ps.Report(filepath.Join(dir, "d"), &ReportOptions{Relative: true})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 B
  a
  b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}
