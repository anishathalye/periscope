package periscope

import (
	"strings"
	"testing"

	"github.com/anishathalye/periscope/testfs"
)

func TestExportBasic(t *testing.T) {
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
	err := ps.Export(&ExportOptions{Format: JsonFormat})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
{
  "duplicates": [
    {
      "paths": [
        "/d1/b",
        "/d2/b"
      ],
      "size": 13923446
    },
    {
      "paths": [
        "/d1/a",
        "/d2/a",
        "/d3/a"
      ],
      "size": 10000
    },
    {
      "paths": [
        "/.x",
        "/.y"
      ],
      "size": 123
    }
  ]
}
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestExportEmpty(t *testing.T) {
	fs := testfs.New(nil).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Export(&ExportOptions{Format: JsonFormat})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
{
  "duplicates": []
}
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}
