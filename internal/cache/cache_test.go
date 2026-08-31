package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReadState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := State{
		Platform: "ps1",
		Profile:  "compile",
		Env:      map[string]string{"MIPS_GCC": "/opt/gcc/bin/mipsel-none-elf-gcc"},
	}
	if err := WriteState(dir, "ps1", st); err != nil {
		t.Fatal(err)
	}
	got, err := ReadState(dir, "ps1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "compile" || got.Env["MIPS_GCC"] == "" {
		t.Fatalf("round-trip: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "ps1", StateFile)); err != nil {
		t.Fatal(err)
	}
}

func TestVendorRootsArePrefixAndExeOnly(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cache")
	roots := VendorRoots(prefix, "n64")
	want := PlatformDir(prefix, "n64")
	var sawPrefix bool
	for _, r := range roots {
		if r == want {
			sawPrefix = true
		}
		if filepath.Base(filepath.Dir(r)) == "third_party" {
			t.Fatalf("third_party is not a layout root: %v", roots)
		}
	}
	if !sawPrefix {
		t.Fatalf("missing %s in %v", want, roots)
	}
}

func TestInstAndSrcDir(t *testing.T) {
	prefix := filepath.Join("C:", "tc")
	if InstDir(prefix, "n64") != filepath.Join(prefix, "n64", InstRel) {
		t.Fatalf("inst %s", InstDir(prefix, "n64"))
	}
	if SrcDir(prefix, "n64", "libdragon") != filepath.Join(prefix, "n64", SrcRel, "libdragon") {
		t.Fatalf("src %s", SrcDir(prefix, "n64", "libdragon"))
	}
}

func TestDefaultPrefixHonorsEnv(t *testing.T) {
	t.Setenv("BLAZIUM_TOOLCHAIN_PREFIX", `D:\cache\tc`)
	if DefaultPrefix() != `D:\cache\tc` {
		t.Fatalf("got %q", DefaultPrefix())
	}
}
