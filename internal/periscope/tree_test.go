package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"strings"
	"testing"
)

func TestTreeBasic(t *testing.T) {
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
	err := ps.Tree("/", &TreeOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 d1/a
1 d1/b
1 d2/a
1 d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestTreeHidden(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/.b [1392 2]
/d2/.a [10000 1]
/d2/b [1392 2]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Tree("/", &TreeOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 d1/a
1 d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
	out.Reset()
	err = ps.Tree("/", &TreeOptions{All: true})
	check(t, err)
	got = strings.TrimSpace(out.String())
	expected = strings.TrimSpace(`
1 d1/.b
1 d1/a
1 d2/.a
1 d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestTreeRelative(t *testing.T) {
	fs := testfs.Read(`
/d1/a [10000 1]
/d1/x/b [1392 2]
/d1/x/c [1191 3]
/d2/a [10000 1]
/d2/b [1392 2]
/d2/x/c [1002 5]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	err := ps.Tree("/d1", &TreeOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 a
1 x/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestTreeNoDeleted(t *testing.T) {
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
	fs.Remove("/d1/a")
	err := ps.Tree("/", &TreeOptions{})
	check(t, err)
	got := strings.TrimSpace(out.String())
	expected := strings.TrimSpace(`
1 d1/b
1 d2/a
1 d2/b
	`)
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}
