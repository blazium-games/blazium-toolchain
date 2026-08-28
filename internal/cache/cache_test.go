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

func TestDefaultPrefixHonorsEnv(t *testing.T) {
	t.Setenv("BLAZIUM_TOOLCHAIN_PREFIX", `D:\cache\tc`)
	if DefaultPrefix() != `D:\cache\tc` {
		t.Fatalf("got %q", DefaultPrefix())
	}
}
