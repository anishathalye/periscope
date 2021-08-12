package periscope

import (
	"bytes"
	"io/ioutil"
	"log"
	"path/filepath"
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
		if len(expected[i]) != len(got[i]) {
			t.Fatalf("duplicate sets have different sizes at index %d, expected %v, got %v", i, len(expected[i]), len(got[i]))
		}
		for j := range expected[i] {
			expectedInfo := expected[i][j]
			gotInfo := got[i][j]
			if expectedInfo.Path != gotInfo.Path {
				t.Fatalf("duplicate set path differs at (%d, %d), expected %v, got %v", i, j, expectedInfo.Path, gotInfo.Path)
			}
			if expectedInfo.Size != gotInfo.Size {
				t.Fatalf("duplicate set size differs at (%d, %d), expected %v, got %v", i, j, expectedInfo.Size, gotInfo.Size)
			}
		}
	}
}

func checkEquivalentInfos(t *testing.T, expected, got []db.FileInfo) {
	if len(expected) != len(got) {
		t.Fatalf("infos differ in length: expected %d, got %d", len(expected), len(got))
	}
	for i := range expected {
		expectedInfo := expected[i]
		gotInfo := got[i]
		if expectedInfo.Path != gotInfo.Path {
			t.Fatalf("info path differs at %d, expected %v, got %v", i, expectedInfo.Path, gotInfo.Path)
		}
		if expectedInfo.Size != gotInfo.Size {
			t.Fatalf("info size differs at %d, expected %v, got %v", i, expectedInfo.Size, gotInfo.Size)
		}
		if (len(expectedInfo.ShortHash) != 0) != (len(gotInfo.ShortHash) != 0) {
			t.Fatalf("info short hash presence differs at %d, expected %v, got %v", i, len(expectedInfo.ShortHash) != 0, len(gotInfo.ShortHash) != 0)
		}
		if (len(expectedInfo.FullHash) != 0) != (len(gotInfo.FullHash) != 0) {
			t.Fatalf("info full hash presence differs at %d, expected %v, got %v", i, len(expectedInfo.FullHash) != 0, len(gotInfo.FullHash) != 0)
		}
	}
}

var dummyHash []byte = []byte{0x01, 0x03, 0x03, 0x07}
