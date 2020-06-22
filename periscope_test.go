package periscope

import (
	"bytes"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/anishathalye/periscope/db"

	"github.com/spf13/afero"
)

func newTest(fs afero.Fs) (*Periscope, *bytes.Buffer, *bytes.Buffer) {
	if testDebug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	db, err := db.New(":memory:")
	if err != nil {
		panic(err)
	}
	outStream := new(bytes.Buffer)
	errStream := new(bytes.Buffer)
	_, realFs := fs.(*afero.OsFs)
	return &Periscope{
		fs:        fs,
		realFs:    realFs,
		db:        db,
		dbPath:    "",
		outStream: outStream,
		errStream: errStream,
		options:   &Options{Debug: false},
	}, outStream, errStream
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func checkErr(t *testing.T, err error) {
	if err == nil {
		t.Fatal("expected error")
	}
}

func tempDir() string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		panic(err)
	}
	return resolved
}

func checkEquivalentDuplicateSet(t *testing.T, expected, got []db.DuplicateSet) {
	if len(expected) != len(got) {
		t.Fatalf("duplicate sets differ in length: expected %d, got %d", len(expected), len(got))
	}
	for i := range expected {
		if !reflect.DeepEqual(expected[i].Paths, got[i].Paths) {
			t.Fatalf("duplicate sets have different paths at index %d: expected %v, got %v", i, expected[i].Paths, got[i].Paths)
		}
		if expected[i].Size != got[i].Size {
			t.Fatalf("duplicate sets have different size at index %d: expected %v, got %v", i, expected[i].Size, got[i].Size)
		}
	}
}
