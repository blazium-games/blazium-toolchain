package steam

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func TestInfo(t *testing.T) {
	info := New().Info()
	if info.ID != ID || info.Status != platforms.StatusSupported {
		t.Fatalf("%+v", info)
	}
	var setup, env, status bool
	for _, c := range info.Commands {
		switch c {
		case "setup":
			setup = true
		case "env":
			env = true
		case "status":
			status = true
		}
	}
	if !setup || !env || !status {
		t.Fatalf("commands %v", info.Commands)
	}
}

func TestPinsPresent(t *testing.T) {
	p := embedfs.MustPins()
	for _, goos := range []string{"windows", "linux", "darwin"} {
		if len(p.Steam[goos]) == 0 {
			t.Fatalf("missing steam pin for %s", goos)
		}
		for _, a := range p.Steam[goos] {
			if a.URL == "" || a.Dest == "" {
				t.Fatalf("%s pin %+v", goos, a)
			}
			if !strings.Contains(a.URL, "steamcmd") {
				t.Fatalf("unexpected url %s", a.URL)
			}
		}
	}
}

func TestOfflineMissing(t *testing.T) {
	prefix := t.TempDir()
	err := New().Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: prefix},
		Offline:       true,
	})
	if err == nil {
		t.Fatal("expected offline error")
	}
}

func TestOfflineReady(t *testing.T) {
	prefix := t.TempDir()
	dest := steamcmdDest(prefix)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "steamcmd"
	if runtime.GOOS == "windows" {
		name = "steamcmd.exe"
	}
	path := filepath.Join(dest, name)
	if err := os.WriteFile(path, []byte("steamcmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	tool := New()
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: prefix},
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	if env["STEAMCMD"] == "" {
		t.Fatalf("%v", env)
	}
	st, err := tool.Status(platforms.CommonOptions{Prefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	if ready, _ := st["ready"].(bool); !ready {
		t.Fatalf("%v", st)
	}
}

func TestUnusedCommands(t *testing.T) {
	tool := New()
	if err := tool.Build(context.Background(), platforms.BuildOptions{}); err == nil {
		t.Fatal("build")
	}
	if err := tool.Run(context.Background(), platforms.RunOptions{}); err == nil {
		t.Fatal("run")
	}
	if err := tool.ISO(context.Background(), platforms.ISOOptions{}); err == nil {
		t.Fatal("iso")
	}
}
