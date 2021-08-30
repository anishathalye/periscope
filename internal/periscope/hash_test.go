package periscope

import (
	"github.com/anishathalye/periscope/internal/testfs"

	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestHashBasic(t *testing.T) {
	fs := testfs.Read(`
/a [1234 1]
	`).Mkfs()
	ps, out, _ := newTest(fs)
	err := ps.Hash([]string{"/a"}, &HashOptions{})
	check(t, err)
	infos, _ := ps.db.Lookup("/a")
	if len(infos) != 1 {
		t.Fatal("expected 1 info")
	}
	ref, _ := ps.hashFile("/a")
	if infos[0].ShortHash == nil || !bytes.Equal(infos[0].FullHash, ref) {
		t.Fatal("expected hashes to be populated and correct")
	}
	got := strings.TrimSpace(out.String())
	expected := fmt.Sprintf("%s  /a", hex.EncodeToString(ref))
	if got != expected {
		t.Fatalf("expected '%s', got '%s'", expected, got)
	}
}

func TestHashDir(t *testing.T) {
	fs := testfs.Read(`
/a [1234 1]
/d/y [1337 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	err := ps.Hash([]string{"/d", "/a"}, &HashOptions{})
	checkErr(t, err)
	infos, _ := ps.db.Lookup("/a")
	if len(infos) != 1 {
		t.Fatal("expected 1 info")
	}
	if infos[0].ShortHash == nil || infos[0].FullHash == nil {
		t.Fatal("expected hashes to be populated")
	}
}

func TestHashComplete(t *testing.T) {
	fs := testfs.Read(`
/a [1234 1]
/b [1234 2]
	`).Mkfs()
	ps, _, _ := newTest(fs)
	ps.Scan([]string{"/"}, &ScanOptions{})
	infos, _ := ps.db.Lookup("/a")
	if len(infos) != 1 {
		t.Fatal("expected 1 info")
	}
	if infos[0].ShortHash == nil {
		t.Fatal("expected short hash to be computed")
	}
	if infos[0].FullHash != nil {
		t.Fatal("expected full hash to be omitted")
	}
	err := ps.Hash([]string{"/a"}, &HashOptions{})
	check(t, err)
	infos, _ = ps.db.Lookup("/a")
	if len(infos) != 1 {
		t.Fatal("expected 1 info")
	}
	if infos[0].ShortHash == nil || infos[0].FullHash == nil {
		t.Fatal("expected hashes to be populated")
	}
}
