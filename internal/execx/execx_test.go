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

func TestMergeEnvPrependsPATH(t *testing.T) {
	got := mergeEnv([]string{"PATH=/usr/bin", "FOO=1"}, []string{"/opt/tc/bin"}, map[string]string{"PSN00BSDK_TC": "/opt/tc"})
	var path, tc string
	for _, kv := range got {
		if len(kv) >= 5 && kv[:5] == "PATH=" {
			path = kv[5:]
		}
		if len(kv) > 13 && kv[:13] == "PSN00BSDK_TC=" {
			tc = kv[13:]
		}
	}
	if path == "" || path[:len("/opt/tc/bin")] != "/opt/tc/bin" {
		t.Fatalf("PATH %q", path)
	}
	if tc != "/opt/tc" {
		t.Fatalf("TC %q", tc)
	}
}

func TestLookPrefersEnvMissingFallsThrough(t *testing.T) {
	t.Setenv("NO_SUCH_TOOLCHAIN_TOOL", "")
	_, err := LookPrefersEnv(Host{}, "NO_SUCH_TOOLCHAIN_TOOL", "definitely-not-a-real-binary-zzzz")
	if err == nil {
		t.Fatal("expected look path error")
	}
}
