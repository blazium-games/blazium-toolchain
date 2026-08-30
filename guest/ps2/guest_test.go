package guest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWritesRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	if CookABI != 1 {
		t.Fatalf("CookABI %d", CookABI)
	}
	for _, name := range RuntimeNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestEmbeddedRuntimeComplete(t *testing.T) {
	for _, name := range RuntimeNames {
		if _, err := files.ReadFile("runtime/" + name); err != nil {
			t.Fatalf("embed missing %s: %v", name, err)
		}
	}
}

func TestRuntimeNamesHasVU1(t *testing.T) {
	var cpp, hdr, vsm bool
	for _, name := range RuntimeNames {
		switch name {
		case "vu1_draw.cpp":
			cpp = true
		case "vu1_draw.h":
			hdr = true
		case "draw_3D.vsm":
			vsm = true
		}
	}
	if !cpp || !hdr || !vsm {
		t.Fatalf("RuntimeNames missing VU1 files: %v", RuntimeNames)
	}
}

func TestRuntimeNamesHasScriptVM(t *testing.T) {
	var vm, hdr bool
	for _, name := range RuntimeNames {
		if name == "script_vm.cpp" {
			vm = true
		}
		if name == "script_vm.h" {
			hdr = true
		}
	}
	if !vm || !hdr {
		t.Fatalf("RuntimeNames missing script_vm: %v", RuntimeNames)
	}
}

func TestOverlayReplacesAndAdds(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	over := t.TempDir()
	if err := os.MkdirAll(filepath.Join(over, "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "main.cpp"), []byte("// overlay main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(over, "extra", "hooks.cpp"), []byte("// extra hook\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Overlay(dir, over); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "main.cpp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "// overlay main\n" {
		t.Fatalf("main.cpp not replaced: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "extra", "hooks.cpp")); err != nil {
		t.Fatal(err)
	}
}
