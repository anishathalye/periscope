package testfs

import (
	"bytes"
	"testing"
)

func TestMkfs(t *testing.T) {
	fs := New([]FileDesc{
		{"/a", 1024, 0},
		{"/b", 1024, 0},
		{"/c", 4096, 0},
	})
	afs := fs.Mkfs()
	info, _ := afs.Stat("/a")
	if info.Size() != 1024 {
		t.Fatal("expected /a to exist")
	}
}

func TestEqual(t *testing.T) {
	fs := New([]FileDesc{
		{"/a", 1024, 0},
		{"/b", 1024, 0},
		{"/c/d", 4096, 0},
		{"/c/e", 4096, 1},
		{"/c/f/g", 4096, 1},
	})
	afs := fs.Mkfs()
	if !Equal(afs, fs) {
		t.Fatalf("expected:\n%sgot:\n%s", fs.ShowIndent(2), ShowIndent(afs, 2))
	}
}

func TestEqualNot(t *testing.T) {
	fs1 := New([]FileDesc{
		{"/a", 1024, 0},
		{"/b", 1024, 0},
	})
	fs2 := New([]FileDesc{
		{"/b", 1024, 0},
		{"/a", 1024, 0},
		{"/c", 1024, 0},
	})
	fs3 := New([]FileDesc{
		{"/b", 1024, 0},
		{"/a", 1024, 1},
		{"/c", 1024, 0},
	})
	if fs1.Equal(fs2) {
		t.Fatal("expected fs1 != fs2")
	}
	if fs2.Equal(fs3) {
		t.Fatal("expected fs2 != fs3")
	}
}

func TestRead(t *testing.T) {
	s := `
/a [1024 0]
/b [1024 1]
/c/d [4096 2]
`
	fs := Read(s)
	expected := New([]FileDesc{
		{"/a", 1024, 0},
		{"/b", 1024, 1},
		{"/c/d", 4096, 2},
	})
	if !fs.Equal(expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), fs.ShowIndent(2))
	}
}

func TestShowRead(t *testing.T) {
	fs := New([]FileDesc{
		{"/a", 0, 0},
		{"/c/e", 4096, 1},
		{"/b", 1024, 0},
		{"/c/f/g", 4096, 1},
		{"/c/d", 4096, 0},
	})
	if !Read(fs.Show()).Equal(fs) {
		t.Fatalf("expected round trip show -> read to work")
	}
}

func TestDirectories(t *testing.T) {
	fs := From(Read(`
/c/d [4096 2]
/c/e/foo [100 4]
	`).Mkfs())
	expected := New([]FileDesc{
		{"/c/d", 4096, 2},
		{"/c/e/foo", 100, 4},
	})
	if !fs.Equal(expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), fs.ShowIndent(2))
	}
}

func TestDirectoriesHidden(t *testing.T) {
	fs := From(Read(`
/c/d [4096 2]
/c/.d/foo [100 4]
	`).Mkfs())
	expected := New([]FileDesc{
		{"/c/d", 4096, 2},
		{"/c/.d/foo", 100, 4},
	})
	if !fs.Equal(expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), fs.ShowIndent(2))
	}
}

func TestDirectoriesDeep(t *testing.T) {
	fs := From(Read(`
/some/deeply/nested/directory/file [4096 2]
/some/deeply/nested/directory/other [100 4]
	`).Mkfs())
	expected := New([]FileDesc{
		{"/some/deeply/nested/directory/file", 4096, 2},
		{"/some/deeply/nested/directory/other", 100, 4},
	})
	if !fs.Equal(expected) {
		t.Fatalf("expected:\n%sgot:\n%s", expected.ShowIndent(2), fs.ShowIndent(2))
	}
}

func TestSameSeedCommonPrefix(t *testing.T) {
	fs := Read(`
/a [10000 1]
/b [12345 1]
	`).Mkfs()
	a, _ := fs.Open("/a")
	var aBuf bytes.Buffer
	aBuf.ReadFrom(a)
	aBytes := aBuf.Bytes()
	b, _ := fs.Open("/b")
	var bBuf bytes.Buffer
	bBuf.ReadFrom(b)
	bBytes := bBuf.Bytes()
	for i := 0; i < 9000; i++ {
		if aBytes[i] != bBytes[i] {
			t.Fatal("prefix mismatch")
		}
	}
}
