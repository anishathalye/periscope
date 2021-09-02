package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"testing"
)

func TestForgetBasic(t *testing.T) {
	fs := testfs.Read(`
/d1/a [1024 1]
/d1/b [1234 2]
/d2/a [1024 1]
/d2/b [1234 2]
/d3/a [1024 1]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Scan([]string{"/"}, &ScanOptions{})
	check(t, err)
	infos, _ := ps.db.Lookup("/d1/a")
	if len(infos) != 3 {
		t.Fatal("expected 3 infos")
	}
	err = ps.Forget([]string{"/d2"}, &ForgetOptions{})
	check(t, err)
	infos, _ = ps.db.Lookup("/d1/a")
	if len(infos) != 2 {
		t.Fatal("expected 2 infos")
	}
	infos, _ = ps.db.Lookup("/d2/a")
	if len(infos) != 0 {
		t.Fatal("expected 0 infos")
	}
	infos, _ = ps.db.Lookup("/d1/b")
	if len(infos) != 1 {
		t.Fatal("expected 1 info")
	}
}
