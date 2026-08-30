package interdvd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func TestInfo(t *testing.T) {
	info := New().Info()
	if info.ID != ID || info.Status != platforms.StatusSupported {
		t.Fatalf("%+v", info)
	}
}

func TestBuildRunUsage(t *testing.T) {
	p := New()
	if err := p.Build(nil, platforms.BuildOptions{}); err == nil {
		t.Fatal("build")
	}
	if err := p.Run(nil, platforms.RunOptions{}); err == nil {
		t.Fatal("run")
	}
}

func TestExtraFromSpec(t *testing.T) {
	e := extraFromSpec(`C:\shots:SCREENSHOTS`)
	if e.Host != `C:\shots` || e.Disc != "SCREENSHOTS" {
		t.Fatalf("%+v", e)
	}
	e = extraFromSpec("note.txt")
	if e.Host != "note.txt" || e.Disc != "" {
		t.Fatalf("%+v", e)
	}
}

func TestISORequiresDir(t *testing.T) {
	err := New().ISO(nil, platforms.ISOOptions{Out: filepath.Join(t.TempDir(), "x.iso")})
	if err == nil {
		t.Fatal("expected error")
	}
	_ = os.ErrNotExist
}

func TestEnvStatusWithoutNetwork(t *testing.T) {
	prefix := t.TempDir()
	p := New()
	env, err := p.Env(platforms.CommonOptions{Prefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	if env["ISO_TOOL"] != "builtin" || env["SCHEMA"] != "blazium.interdvd.meta/v1" {
		t.Fatalf("%v", env)
	}
	if env["FFMPEG"] != "" || env["FFPROBE"] != "" {
		t.Fatalf("empty prefix should not invent tools: %v", env)
	}
	st, err := p.Status(platforms.CommonOptions{Prefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	if ready, _ := st["encode_ready"].(bool); ready {
		t.Fatalf("encode_ready %v", st)
	}
	if ready, _ := st["iso_ready"].(bool); !ready {
		t.Fatalf("iso_ready %v", st)
	}
}

func TestFFmpegMissingTool(t *testing.T) {
	err := New().RunTool(nil, "ffmpeg", []string{"-version"}, platforms.CommonOptions{Prefix: t.TempDir()})
	if !errors.Is(err, platforms.ErrMissingTool) {
		t.Fatalf("got %v", err)
	}
}
