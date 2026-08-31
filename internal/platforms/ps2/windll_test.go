package ps2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindHostDLLRequiresMatchingMachine(t *testing.T) {
	if findHostDLL("definitely-missing-blazium-dll.dll", 0x14c) != "" {
		t.Fatal("missing dll should be empty")
	}
}

func TestMsys2MirrorFallbacksIncludesRepo(t *testing.T) {
	raw := "https://mirror.msys2.org/mingw/mingw32/mingw-w64-i686-gmp-6.3.0-2-any.pkg.tar.zst"
	got := msys2MirrorFallbacks(raw)
	if len(got) < 2 {
		t.Fatalf("want original plus fallback, got %v", got)
	}
	if got[0] != raw {
		t.Fatalf("first candidate must be the configured URL: %v", got)
	}
	foundRepo := false
	for _, u := range got {
		if u == "https://repo.msys2.org/mingw/mingw32/mingw-w64-i686-gmp-6.3.0-2-any.pkg.tar.zst" {
			foundRepo = true
		}
	}
	if !foundRepo {
		t.Fatalf("missing repo.msys2.org fallback: %v", got)
	}
	if len(msys2MirrorFallbacks("not a url")) != 1 {
		t.Fatal("invalid URL should stay a single candidate")
	}
}

func TestPeMachineRejectsText(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.exe")
	if err := os.WriteFile(p, []byte("gcc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := peMachine(p); err == nil {
		t.Fatal("expected pe open error")
	}
}
