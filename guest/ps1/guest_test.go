package guest

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallWritesRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CMakeLists.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.cpp")); err != nil {
		t.Fatal(err)
	}
	if CookABI != 18 {
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

func TestLockstepWithEditorRuntime(t *testing.T) {
	if os.Getenv("BLAZIUM_EDITOR_LOCKSTEP") != "1" {
		t.Skip("set BLAZIUM_EDITOR_LOCKSTEP=1 to compare guest sources with a sibling editor tree")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	toolchainRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	editor := filepath.Join(toolchainRoot, "..", "blazium", "platform", "ps1", "runtime")
	if _, err := os.Stat(filepath.Join(editor, "main.cpp")); err != nil {
		t.Skip("editor runtime tree absent")
	}
	embedded := filepath.Join(filepath.Dir(thisFile), "runtime")
	for _, name := range []string{"main.cpp", "CMakeLists.txt", "script_vm.cpp", "script_vm.h", "fmv_play.cpp", "fmv_play.h"} {
		want, err := os.ReadFile(filepath.Join(editor, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(embedded, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("%s differs from editor copy at %s", name, editor)
		}
	}
}
