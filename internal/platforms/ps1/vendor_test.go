package ps1

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVendorRootsStayOnPrefixAndExe(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cache")
	roots := vendorRoots(prefix)
	if len(roots) < 2 {
		t.Fatalf("roots %v", roots)
	}
	wantPrefix := filepath.Join(prefix, ID)
	var sawPrefix bool
	for _, r := range roots {
		if r == wantPrefix {
			sawPrefix = true
		}
		if filepath.Base(filepath.Dir(r)) == "third_party" {
			t.Fatalf("third_party is not a layout root: %v", roots)
		}
		if strings.Contains(r, string(filepath.Separator)+".."+string(filepath.Separator)) {
			t.Fatalf("parent walk in vendor root %q", r)
		}
		if strings.Contains(filepath.ToSlash(r), "/PSn00bSDK") {
			t.Fatalf("sibling SDK root %q", r)
		}
	}
	if !sawPrefix {
		t.Fatalf("missing prefix root %s: %v", wantPrefix, roots)
	}
}

func TestVendorRootsIgnoreCWDThirdParty(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range vendorRoots(t.TempDir()) {
		if r == filepath.Join(wd, "third_party", ID) {
			t.Fatalf("cwd third_party in roots: %v", r)
		}
		if r == filepath.Join(wd, "..", "third_party", ID) {
			t.Fatalf("parent third_party in roots: %v", r)
		}
	}
}

func TestFindVendorFileUsesPrefix(t *testing.T) {
	prefix := t.TempDir()
	name := "elf2x"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(prefix, ID, "elf2x", name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := findVendorFile(prefix, filepath.Join("elf2x", "elf2x"))
	if got == "" {
		t.Fatal("expected prefix vendor hit")
	}
}
