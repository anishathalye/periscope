package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"strings"
	"testing"
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
	err := ps.Report(&ReportOptions{})
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

func TestReportEmpty(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Report(&ReportOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	if got != "" {
		t.Fatalf("expected no output, got '%s'", got)
	}
}
