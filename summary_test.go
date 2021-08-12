package periscope

import (
	"strings"
	"testing"

	"github.com/anishathalye/periscope/testfs"
)

func TestSummaryBasic(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/b [13923446 2]
/d1/c [1191 3]
/d2/a [10000 1]
/d2/b [13923446 2]
/d2/c [1002 5]
/d3/a [10000 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Summary(&SummaryOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
  tracked     7
   unique     4
duplicate     3
 overhead 14 MB
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestSummaryEmpty(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Summary(&SummaryOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
  tracked   0
   unique   0
duplicate   0
 overhead 0 B
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}
