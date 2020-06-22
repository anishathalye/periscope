package db

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func newInMemoryDb(t *testing.T) *Session {
	db, err := New(":memory:")
	check(t, err)
	return db
}

func TestInitialize(t *testing.T) {
	db := newInMemoryDb(t)
	err := db.Add(DuplicateSet{[]string{"/a", "/b"}, 3, 1})
	check(t, err)
	dupes, err := db.AllDuplicates()
	check(t, err)
	if len(dupes) != 1 {
		t.Fatal("failed to add to db")
	}
	err = db.Initialize()
	check(t, err)
	dupes, err = db.AllDuplicates()
	check(t, err)
	if len(dupes) != 0 {
		t.Fatal("Initialize() failed to clear db")
	}
}

func TestVersionMismatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "db.sqlite")
	db, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// corrupt version
	_, err = db.db.Exec(`UPDATE meta SET value = "2" WHERE key = "version"`)
	if err != nil {
		t.Fatal(err)
	}
	// check that we fail when creating another db
	_, err = New(dbPath)
	if err == nil {
		t.Fatal("expected error")
	}
	expected := "database version mismatch"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error to contain '%s', was '%s'", expected, err.Error())
	}
}

func TestAddAllC(t *testing.T) {
	db := newInMemoryDb(t)
	expected := []DuplicateSet{
		{[]string{"/a/c", "/x/c", "/y/c"}, 1000, 2},
		{[]string{"/a/b", "/x/b"}, 12, 1},
	}
	c := make(chan DuplicateSet)
	go func() {
		c <- expected[1]
		c <- expected[0]
		close(c)
	}()
	check(t, db.AddAllC(c))
	check(t, db.CreateIndexes())
	got, err := db.AllDuplicates()
	check(t, err)
	if len(got) != 2 {
		t.Fatal("expected 2 duplicate sets")
	}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestSummary(t *testing.T) {
	db := newInMemoryDb(t)
	err := db.AddAll([]DuplicateSet{
		{[]string{"/a/c", "/x/c", "/y/c"}, 1000, 2},
		{[]string{"/a/b", "/x/b"}, 2000, 1},
	})
	check(t, err)
	expected := DuplicateSummary{
		Files:     5,
		Unique:    2,
		Duplicate: 3,
		Overhead:  1000*2 + 2000,
	}
	got, err := db.Summary()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestLookup(t *testing.T) {
	db := newInMemoryDb(t)
	expected := DuplicateSet{[]string{"/a", "/b"}, 133, 17}
	check(t, db.Add(expected))
	got, err := db.Lookup("/a")
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	got, err = db.Lookup("/b")
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	// non-existent
	expected = DuplicateSet{}
	got, err = db.Lookup("/c")
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestLookupAll(t *testing.T) {
	db := newInMemoryDb(t)
	err := db.AddAll([]DuplicateSet{
		{[]string{"/x/y/a", "/x/z/a"}, 1000, 1},
		{[]string{"/x/y/b", "/x/z/b"}, 1000, 2},
		{[]string{"/z/.c", "/y/.c"}, 1000, 3},
		{[]string{"/z/.d/e", "/y/.d/e"}, 1000, 4},
		{[]string{"/w/x/.a", "/w/x/.b"}, 1000, 5},
	})
	check(t, err)

	expected := []DuplicateInfo{
		{"/x/y/a", 1, 2},
		{"/x/y/b", 2, 2},
		{"/x/z/a", 1, 2},
		{"/x/z/b", 2, 2},
	}
	got, err := db.LookupAll("/x", false)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/x/y/a", 1, 2},
		{"/x/y/b", 2, 2},
	}
	got, err = db.LookupAll("/x/y", false)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	got, err = db.LookupAll("/z", false)
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected [], got %v", got)
	}

	expected = []DuplicateInfo{
		{"/z/.c", 3, 2},
		{"/z/.d/e", 4, 2},
	}
	got, err = db.LookupAll("/z", true)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/z/.d/e", 4, 2},
	}
	got, err = db.LookupAll("/z/.d", false)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/w/x/.a", 5, 2},
		{"/w/x/.b", 5, 2},
	}
	got, err = db.LookupAll("/w/", true)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestRemove(t *testing.T) {
	db := newInMemoryDb(t)
	err := db.AddAll([]DuplicateSet{
		{[]string{"/a/c", "/y/c", "/x/c"}, 1000, 2},
		{[]string{"/a/b", "/x/b"}, 12, 1},
	})
	check(t, err)
	check(t, db.Remove("/a/c"))
	expected := []DuplicateSet{
		{[]string{"/x/c", "/y/c"}, 1000, 2},
		{[]string{"/a/b", "/x/b"}, 12, 1},
	}
	got, err := db.AllDuplicates()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestPruneSingletons(t *testing.T) {
	db := newInMemoryDb(t)
	err := db.AddAll([]DuplicateSet{
		{[]string{"/a/c", "/y/c", "/x/c"}, 1000, 2},
		{[]string{"/a/b", "/x/b"}, 12, 1},
	})
	check(t, err)
	check(t, db.Remove("/a/b"))
	check(t, db.PruneSingletons())
	expected := []DuplicateSet{
		{[]string{"/a/c", "/x/c", "/y/c"}, 1000, 2},
	}
	got, err := db.AllDuplicates()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	check(t, db.Remove("/a/c"))
	check(t, db.PruneSingletons())
	expected = []DuplicateSet{
		{[]string{"/x/c", "/y/c"}, 1000, 2},
	}
	got, err = db.AllDuplicates()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	check(t, db.Remove("/nonexistent"))
	check(t, db.PruneSingletons())
	got, err = db.AllDuplicates()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	check(t, db.Remove("/y/c"))
	check(t, db.PruneSingletons())
	got, err = db.AllDuplicates()
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected [], got %v", got)
	}
}
