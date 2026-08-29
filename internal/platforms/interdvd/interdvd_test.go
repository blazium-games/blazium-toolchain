package interdvd

import (
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
