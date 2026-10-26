package guest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The shared runtime is the PS2 guest's, byte for byte.
func TestGDIRSharedMatchesPS2(t *testing.T) {
	for _, name := range GDIRShared {
		ps3, err := gdirFiles.ReadFile("gdir/runtime/" + name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ps2, err := os.ReadFile(filepath.Join("..", "ps2", "runtime", name))
		if err != nil {
			t.Fatalf("ps2 %s: %v", name, err)
		}
		if !bytes.Equal(ps3, ps2) {
			t.Errorf("%s differs from guest/ps2/runtime/%s", name, name)
		}
	}
}

func TestInstallGDIR(t *testing.T) {
	dir := t.TempDir()
	if err := InstallGDIR(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Makefile", "main.cpp", "rsx_draw.cpp", "rsx_shaders.h", "gs_draw.h", "iso_path.h", "pad_io.cpp", "sfx_io.cpp", "mc_io.cpp", "vu1_draw.h", "sfo.xml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not installed: %v", name, err)
		}
	}
}

func TestNotices(t *testing.T) {
	for _, name := range []string{"psl1ght.txt", "newlib.txt"} {
		b, err := Notice(name)
		if err != nil || len(b) < 100 {
			t.Errorf("%s: %v (%d bytes)", name, err, len(b))
		}
	}
}
