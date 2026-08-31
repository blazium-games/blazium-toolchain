package safepath

import (
	"path/filepath"
	"testing"
)

func TestJoinRejectsDotDot(t *testing.T) {
	root := t.TempDir()
	if _, err := Join(root, "../escape"); err == nil {
		t.Fatal("expected escape")
	}
	got, err := Join(root, "ok/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "file.txt" {
		t.Fatal(got)
	}
}

func TestUnder(t *testing.T) {
	root := t.TempDir()
	if err := Under(root, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := Under(root, filepath.Dir(root)); err == nil {
		t.Fatal("parent should escape")
	}
}
