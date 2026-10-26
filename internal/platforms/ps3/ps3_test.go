package ps3

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func TestInfo(t *testing.T) {
	info := New().Info()
	if info.ID != "ps3" || info.Status != platforms.StatusSupported {
		t.Fatalf("info: %+v", info)
	}
}

func TestBuildNeedsTools(t *testing.T) {
	t.Setenv("PS3DEV", filepath.Join(t.TempDir(), "none"))
	t.Setenv("PSL1GHT", "")
	t.Setenv("PS3_WSL", "0")
	err := New().Build(context.Background(), platforms.BuildOptions{CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()}, Out: "x.pkg", Sample: "gdir"})
	if err == nil {
		t.Fatal("build without ps3dev should fail")
	}
}

func TestGDIRDataName(t *testing.T) {
	for name, want := range map[string]bool{"MESH00.bin": true, "DISP.BIN": true, "BOOTLOGO.RAW": false, "PENGUIN.ELF": false, "BUDGET.txt": false} {
		if got := gdirDataName(name); got != want {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}
}

func TestStageGDIR(t *testing.T) {
	export := t.TempDir()
	if err := os.WriteFile(filepath.Join(export, "SCRP00.bin"), []byte("SCRP"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "src")
	if _, err := stageGDIR(src, export, "", nil); err == nil {
		t.Fatal("an export without MESH00 should fail")
	}
	for _, f := range []string{"MESH00.bin", "ICON0.PNG", "PENGUIN.ELF"} {
		if err := os.WriteFile(filepath.Join(export, f), []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := stageGDIR(src, export, "", nil)
	if err != nil || n != 2 {
		t.Fatalf("staged %d, %v", n, err)
	}
	for _, f := range []string{"pkg/USRDIR/DATA/MESH00.BIN", "pkg/USRDIR/DATA/SCRP00.BIN", "pkg/ICON0.PNG", "Makefile", "rsx_draw.cpp"} {
		if _, err := os.Stat(filepath.Join(src, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "pkg/USRDIR/DATA/PENGUIN.ELF")); err == nil {
		t.Error("the ELF should not be staged")
	}
}

func TestOutputs(t *testing.T) {
	pkg, dir := outputs(filepath.Join("a", "PENGUIN.pkg"), "BLAZ00001")
	if pkg != filepath.Join("a", "PENGUIN.pkg") || dir != filepath.Join("a", "BLAZ00001") {
		t.Fatalf("got %q %q", pkg, dir)
	}
	pkg, dir = outputs("out", "PENG00001")
	if pkg != filepath.Join("out", "PENG00001.pkg") || dir != filepath.Join("out", "PENG00001") {
		t.Fatalf("got %q %q", pkg, dir)
	}
}

func TestContentID(t *testing.T) {
	got := contentID("PENG00001", "Penguin Arcade Kart")
	if got != "UP0001-PENG00001_00-PENGUINARCADEKAR" || len(got) != 36 {
		t.Fatalf("got %q (%d)", got, len(got))
	}
	if got := contentID("BLAZ00001", "Go!"); got != "UP0001-BLAZ00001_00-GO00000000000000" {
		t.Fatalf("got %q", got)
	}
}

func TestTitleID(t *testing.T) {
	for id, ok := range map[string]bool{"BLAZ00001": true, "PENG12345": true, "BLA00001": false, "blaz00001": false, "BLAZ0001A": false} {
		if titleIDRe.MatchString(id) != ok {
			t.Errorf("%s: want %v", id, ok)
		}
	}
}

func TestWSLPath(t *testing.T) {
	got, err := wslPath(`C:\Users\me\AppData\ps3`)
	if err == nil && !strings.HasPrefix(got, "/mnt/c/Users/me") && !strings.HasPrefix(got, "/") {
		t.Fatalf("got %q", got)
	}
	if q := shellQuote("it's"); q != `'it'\''s'` {
		t.Fatalf("quote: %s", q)
	}
	if q := shellQuote("/mnt/c/a_b"); q != "/mnt/c/a_b" {
		t.Fatalf("quote: %s", q)
	}
	if d := wslDir("~/ps3dev"); d != `"$HOME"/ps3dev` {
		t.Fatalf("dir: %s", d)
	}
}

func TestNotices(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "GAME.pkg")
	path, err := writeNotices(out)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "GAME.THIRD-PARTY-NOTICES.txt") {
		t.Fatalf("path %s", path)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"PSL1GHT", "newlib", "Permission is hereby granted"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("notices lack %q", want)
		}
	}
}
