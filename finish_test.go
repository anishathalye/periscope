package periscope

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func dummyFile() *os.File {
	r, w, _ := os.Pipe()
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
	}()
	return w
}

func TestFinish(t *testing.T) {
	oldStdout := os.Stdout
	defer func() {
		os.Stdout = oldStdout
	}()
	os.Stdout = dummyFile()
	fakeHome := tempDir()
	defer os.RemoveAll(fakeHome)
	scanDir := tempDir()
	defer os.RemoveAll(scanDir)
	var err error
	oldCacheHome, wasSet := os.LookupEnv("XDG_CACHE_HOME")
	if wasSet {
		defer os.Setenv("XDG_CACHE_HOME", oldCacheHome)
	}
	os.Unsetenv("XDG_CACHE_HOME")
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", fakeHome)
	err = Finish(&FinishOptions{}) // should be ok with no db
	check(t, err)
	ps, err := New(&Options{Debug: testDebug})
	check(t, err)
	ps.outStream = new(bytes.Buffer)
	ps.errStream = new(bytes.Buffer)
	err = ps.Scan([]string{scanDir}, &ScanOptions{})
	check(t, err)
	dbPath, err := dbPath()
	check(t, err)
	info, err := os.Stat(dbPath)
	check(t, err)
	if !info.Mode().IsRegular() {
		t.Fatal("db was not created")
	}
	err = Finish(&FinishOptions{})
	check(t, err)
	_, err = os.Stat(dbPath)
	if !os.IsNotExist(err) {
		t.Fatal("db was not deleted")
	}
}
