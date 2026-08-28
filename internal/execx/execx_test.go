package execx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPrefersEnvFile(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "pcsx-redux.exe")
	if err := os.WriteFile(fake, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PCSX_EXE", fake)
	got, err := LookPrefersEnv(Host{}, "PCSX_EXE", "pcsx-redux")
	if err != nil {
		t.Fatal(err)
	}
	if got != fake {
		t.Fatalf("got %q", got)
	}
}

func TestLookPrefersEnvMissingFallsThrough(t *testing.T) {
	t.Setenv("NO_SUCH_TOOLCHAIN_TOOL", "")
	_, err := LookPrefersEnv(Host{}, "NO_SUCH_TOOLCHAIN_TOOL", "definitely-not-a-real-binary-zzzz")
	if err == nil {
		t.Fatal("expected look path error")
	}
}
