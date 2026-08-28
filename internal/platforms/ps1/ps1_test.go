package ps1

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func TestSetupWritesState(t *testing.T) {
	dir := t.TempDir()
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: os.Stdout},
		Profile:       "compile",
		Offline:       false,
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if env == nil {
		t.Fatal("nil env")
	}
	if _, err := os.Stat(filepath.Join(dir, "ps1", "components.json")); err != nil {
		t.Fatal(err)
	}
}

func TestSetupRejectsBadProfile(t *testing.T) {
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
		Profile:       "dreamcast",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOfflineOKWithVendoredGCC(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	gccDir := filepath.Join(dir, "ps1", "gcc", "bin")
	if err := os.MkdirAll(gccDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(gccDir, "mipsel-none-elf-gcc")
	if err := os.WriteFile(fake, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if env["MIPS_GCC"] == "" {
		t.Fatal("expected vendored MIPS_GCC")
	}
}

func TestOfflineFailsWithoutGCC(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("PATH", t.TempDir())
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
		Profile:       "compile",
		Offline:       true,
	})
	if err == nil {
		t.Fatal("expected offline error")
	}
}

func TestBuildRequiresFlags(t *testing.T) {
	tool := New()
	err := tool.Build(context.Background(), platforms.BuildOptions{})
	if err == nil {
		t.Fatal("expected usage")
	}
}

func TestComponentsAreContainable(t *testing.T) {
	for _, c := range componentsForProfile("iso") {
		if !c.Contained {
			t.Fatalf("%s should be containable in this repo", c.ID)
		}
	}
}

func TestComponentsISOIncludesMkpsxiso(t *testing.T) {
	cs := componentsForProfile("iso")
	var saw bool
	for _, c := range cs {
		if c.ID == "mkpsxiso" {
			saw = true
			if c.License != "GPLv2+" {
				t.Fatalf("license %s", c.License)
			}
		}
	}
	if !saw {
		t.Fatal("iso profile missing mkpsxiso")
	}
}
