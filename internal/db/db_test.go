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
	db, err := NewInMemory()
	check(t, err)
	return db
}

func addAll(db *Session, infos []FileInfo) error {
	for _, i := range infos {
		err := db.Add(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestVersionOk(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "db.sqlite")
	_, err = New(dbPath, true)
	if err != nil {
		t.Fatal(err)
	}
	// check that opening db is ok
	_, err = New(dbPath, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVersionMismatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "db.sqlite")
	db, err := New(dbPath, true)
	if err != nil {
		t.Fatal(err)
	}
	// corrupt version
	_, err = db.db.Exec(`UPDATE meta SET value = ? WHERE key = "version"`, version+1)
	if err != nil {
		t.Fatal(err)
	}
	// check that we fail when creating another db
	_, err = New(dbPath, true)
	if err == nil {
		t.Fatal("expected error")
	}
	expected := "database version mismatch"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error to contain '%s', was '%s'", expected, err.Error())
	}
}

func TestAdd(t *testing.T) {
	db := newInMemoryDb(t)
	expected := []FileInfo{
		{"/a/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/b/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/c/y", 33, []byte("xxxx"), nil},
		{"/d/z", 2, nil, nil},
	}
	db.Add(expected[0])
	db.Add(expected[1])
	db.Add(expected[2])
	db.Add(expected[3])
	check(t, db.CreateIndexes())
	got, err := db.AllInfos()
	check(t, err)
	if len(got) != 4 {
		t.Fatal("expected 4 infos")
	}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestAddOverwrite(t *testing.T) {
	db := newInMemoryDb(t)
	db.Add(FileInfo{"/a", 1000, nil, nil})
	db.Add(FileInfo{"/a", 1234, nil, nil})
	got, _ := db.AllInfos()
	if len(got) != 1 {
		t.Fatal("expected 1 infos")
	}
	expected := FileInfo{"/a", 1234, []byte("asdf"), []byte("asdfasdf")}
	db.Add(expected)
	got, _ = db.AllInfos()
	if len(got) != 1 {
		t.Fatal("expected 1 infos")
	}
	if !reflect.DeepEqual(expected, got[0]) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestAddTransaction(t *testing.T) {
	db := newInMemoryDb(t)
	expected := []FileInfo{
		{"/a/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/b/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/c/y", 33, []byte("xxxx"), nil},
		{"/d/z", 2, nil, nil},
	}
	tx, _ := db.Begin()
	tx.Add(expected[0])
	tx.Add(expected[1])
	tx.Add(expected[2])
	tx.Add(expected[3])
	check(t, tx.CreateIndexes())
	check(t, tx.Commit())
	got, err := db.AllInfos()
	check(t, err)
	if len(got) != 4 {
		t.Fatal("expected 4 infos")
	}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestSummary(t *testing.T) {
	db := newInMemoryDb(t)
	err := addAll(db, []FileInfo{
		{"/a/c", 1000, []byte("a"), []byte("aa")},
		{"/x/c", 1000, []byte("a"), []byte("aa")},
		{"/y/c", 1000, []byte("a"), []byte("aa")},
		{"/a/b", 2000, []byte("b"), []byte("bb")},
		{"/x/b", 2000, []byte("b"), []byte("bb")},
	})
	check(t, err)
	expected := InfoSummary{
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

func TestSummaryNonDuplicate(t *testing.T) {
	db := newInMemoryDb(t)
	err := addAll(db, []FileInfo{
		{"/a/c", 1000, []byte("a"), []byte("aa")},
		{"/x/c", 1000, []byte("a"), []byte("aa")},
		{"/y/c", 1000, []byte("a"), []byte("aa")},
		{"/a/b", 2000, []byte("b"), []byte("bb")}, // has full hash, but no duplicate
	})
	check(t, err)
	expected := InfoSummary{
		Files:     4,
		Unique:    2,
		Duplicate: 2,
		Overhead:  1000 * 2,
	}
	got, err := db.Summary()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestSummaryMissingFullHash(t *testing.T) {
	db := newInMemoryDb(t)
	err := addAll(db, []FileInfo{
		{"/a/c", 1000, []byte("a"), []byte("aa")},
		{"/x/c", 1000, []byte("a"), []byte("aa")},
		{"/y/c", 1000, []byte("b"), nil},
	})
	check(t, err)
	expected := InfoSummary{
		Files:     3,
		Unique:    2,
		Duplicate: 1,
		Overhead:  1000,
	}
	got, err := db.Summary()
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestAllDuplicates(t *testing.T) {
	db := newInMemoryDb(t)
	infos := []FileInfo{
		{"/a/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/b/x", 1000, []byte("asdf"), []byte("asdfasdf")},
		{"/c/y", 33, []byte("xxxx"), nil},
		{"/d/z", 2, nil, nil},
	}
	err := addAll(db, infos)
	check(t, err)
	got, err := db.AllDuplicates()
	check(t, err)
	if len(got) != 1 || len(got[0]) != 2 || !reflect.DeepEqual(infos[0], got[0][0]) || !reflect.DeepEqual(infos[1], got[0][1]) {
		t.Fatalf("expected %v %v, got %v %v", infos[0], infos[1], got[0][0], got[0][1])
	}
}

func TestLookup(t *testing.T) {
	db := newInMemoryDb(t)
	infos := []FileInfo{
		{"/a", 133, []byte("a"), []byte("aa")},
		{"/b", 133, []byte("a"), []byte("aa")},
		{"/x", 1234, []byte("a"), []byte("fff")},
		{"/y", 1337, nil, nil},
		{"/z", 1338, nil, nil},
	}
	check(t, addAll(db, infos))
	got, err := db.Lookup("/a")
	check(t, err)
	expected := DuplicateSet{infos[0], infos[1]}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	got, err = db.Lookup("/b")
	check(t, err)
	expected = DuplicateSet{infos[1], infos[0]}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	// non-existent
	got, err = db.Lookup("/c")
	check(t, err)
	if len(got) != 0 {
		t.Fatalf("expected empty set")
	}
	// no matching
	got, err = db.Lookup("/x")
	check(t, err)
	expected = DuplicateSet{infos[2]}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	// no full hash
	got, err = db.Lookup("/y")
	check(t, err)
	expected = DuplicateSet{infos[3]}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestInfosBySize(t *testing.T) {
	db := newInMemoryDb(t)
	infos := []FileInfo{
		{"/a", 133, []byte("a"), []byte("aa")},
		{"/x", 1234, []byte("a"), []byte("fff")},
		{"/y", 1337, nil, nil},
		{"/z", 1338, nil, nil},
	}
	check(t, addAll(db, infos))
	got, err := db.InfosBySize(1234)
	check(t, err)
	expected := []FileInfo{{"/x", 1234, []byte("a"), []byte("fff")}}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestLookupAll(t *testing.T) {
	db := newInMemoryDb(t)
	err := addAll(db, []FileInfo{
		{"/x/y/a", 1000, []byte("a"), []byte("aa")},
		{"/x/z/a", 1000, []byte("a"), []byte("aa")},
		{"/x/y/b", 1000, []byte("b"), []byte("bb")},
		{"/x/z/b", 1000, []byte("b"), []byte("bb")},
		{"/z/.c", 1000, []byte("c"), []byte("cc")},
		{"/y/.c", 1000, []byte("c"), []byte("cc")},
		{"/z/.d/e", 1000, []byte("d"), []byte("dd")},
		{"/y/.d/e", 1000, []byte("d"), []byte("dd")},
		{"/w/x/.a", 1000, []byte("e"), []byte("ee")},
		{"/w/x/.b", 1000, []byte("e"), []byte("ee")},
		{"/x/x", 1234, []byte("x"), []byte("xx")},
		{"/x/foo", 1000, []byte("f"), nil},
		{"/y/bar", 1000, nil, nil},
	})
	check(t, err)

	expected := []DuplicateInfo{
		{"/x/y/a", []byte("aa"), 2},
		{"/x/y/b", []byte("bb"), 2},
		{"/x/z/a", []byte("aa"), 2},
		{"/x/z/b", []byte("bb"), 2},
	}
	got, err := db.LookupAll("/x", false)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/x/y/a", []byte("aa"), 2},
		{"/x/y/b", []byte("bb"), 2},
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
		{"/z/.c", []byte("cc"), 2},
		{"/z/.d/e", []byte("dd"), 2},
	}
	got, err = db.LookupAll("/z", true)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/z/.d/e", []byte("dd"), 2},
	}
	got, err = db.LookupAll("/z/.d", false)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	expected = []DuplicateInfo{
		{"/w/x/.a", []byte("ee"), 2},
		{"/w/x/.b", []byte("ee"), 2},
	}
	got, err = db.LookupAll("/w/", true)
	check(t, err)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestRemove(t *testing.T) {
	db := newInMemoryDb(t)
	err := addAll(db, []FileInfo{
		{"/x/y/a", 1000, []byte("a"), []byte("aa")},
		{"/x/z/a", 1000, []byte("a"), []byte("aa")},
		{"/x/y/b", 1000, []byte("b"), []byte("bb")},
		{"/x/z/b", 1000, []byte("b"), []byte("bb")},
		{"/z/.c", 1000, []byte("c"), []byte("cc")},
	})
	check(t, err)
	check(t, db.Remove("/x/y/a"))
	got, err := db.AllDuplicates()
	check(t, err)
	if len(got) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d", len(got))
	}
	got2, err := db.AllInfos()
	if len(got2) != 4 {
		t.Fatalf("expected 4 infos, got %d", len(got2))
	}
}

func TestRemoveDir(t *testing.T) {
	db := newInMemoryDb(t)
	addAll(db, []FileInfo{
		{"/hello/x", 1000, []byte("a"), []byte("aa")},
		{"/hello/y", 1000, []byte("a"), []byte("aa")},
		{"/helloasdf", 1000, []byte("a"), []byte("aa")},
		{"/goodbye/z", 1000, []byte("b"), []byte("bb")},
		{"/goodbye/w", 1000, []byte("b"), []byte("bb")},
		{"/goodbyeasdf", 1000, []byte("b"), []byte("bb")},
	})
	check(t, db.RemoveDir("/hello", 0, 0))
	got, err := db.AllInfos()
	check(t, err)
	if len(got) != 4 {
		t.Fatalf("expected 4 infos, got %d", len(got))
	}
	check(t, db.RemoveDir("/goodbye/", 0, 0))
	got, err = db.AllInfos()
	check(t, err)
	if len(got) != 2 {
		t.Fatalf("expected 2 infos, got %d", len(got))
	}
}
