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
