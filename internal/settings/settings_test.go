package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultsMatchOfficialURLs(t *testing.T) {
	d := Defaults()
	if !strings.Contains(d.N64.Fetch.Libdragon, "DragonMinded/libdragon") {
		t.Fatal(d.N64.Fetch.Libdragon)
	}
	if !strings.HasSuffix(d.N64.Fetch.ToolchainLinux, ".deb") {
		t.Fatal(d.N64.Fetch.ToolchainLinux)
	}
	if d.Smoke() != 120*time.Second || d.Download() != 15*time.Minute {
		t.Fatalf("timeouts %v %v", d.Smoke(), d.Download())
	}
	if d.PS2.ElfTextMax != 512*1024 || d.N64.CartMax != 64*1024*1024 {
		t.Fatalf("size gates %d %d", d.PS2.ElfTextMax, d.N64.CartMax)
	}
}

func TestLoadFileOverlays(t *testing.T) {
	t.Cleanup(Reset)
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	body := []byte("smoke_timeout: 30s\nn64:\n  emu: ares\n  display: \"640\"\n  rom_title: Demo\n  pj64:\n    vi_refresh: \"1600\"\nps2:\n  cnf_vmode: PAL\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadFile(path); err != nil {
		t.Fatal(err)
	}
	s := Current()
	if s.Smoke() != 30*time.Second {
		t.Fatalf("smoke %v", s.Smoke())
	}
	if s.N64.Emu != "ares" || s.N64.Display != "640" || s.N64.RomTitle != "Demo" {
		t.Fatalf("%+v", s.N64)
	}
	if s.N64.PJ64.ViRefresh != "1600" || s.PS2.CNFVMode != "PAL" {
		t.Fatalf("pj64/ps2 %+v %+v", s.N64.PJ64, s.PS2)
	}
	if s.N64.Fetch.Libdragon == "" || s.PS2.ISOVolume != "BLAZIUM2" {
		t.Fatalf("defaults dropped: %+v", s)
	}
	if LoadedPath() != path {
		t.Fatalf("loaded %q", LoadedPath())
	}
}

func TestFindWalksParents(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("n64:\n  emu: ares\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(child); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	got := Find("")
	if filepath.Clean(got) != filepath.Clean(filepath.Join(root, FileName)) {
		t.Fatalf("got %q", got)
	}
}

func TestValidateRejectsBadInputs(t *testing.T) {
	if err := CheckURL("file:///tmp/x"); err == nil {
		t.Fatal("file: url")
	}
	if err := CheckURL("https://user:pass@example.com/x"); err == nil {
		t.Fatal("credentials")
	}
	if err := CheckFetchURL("http://example.com/x"); err == nil {
		t.Fatal("non-loopback http")
	}
	if err := CheckFetchURL("http://127.0.0.1:9/x"); err != nil {
		t.Fatal(err)
	}
	d := Defaults()
	d.SmokeTimeout = "nope"
	if err := Validate(d); err == nil || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("duration: %v", err)
	}
	d = Defaults()
	d.Env = map[string]string{"not-ok": "1"}
	if err := Validate(d); err == nil || !strings.Contains(err.Error(), "POSIX") {
		t.Fatalf("env: %v", err)
	}
	d = Defaults()
	d.N64.Emu = "dolphin"
	if err := Validate(d); err == nil {
		t.Fatal("emu")
	}
	if err := Validate(Defaults()); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFileRejectsBadYAML(t *testing.T) {
	t.Cleanup(Reset)
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("smoke_timeout: banana\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadFile(path); err == nil {
		t.Fatal("expected duration error")
	}
}

func TestProfileFor(t *testing.T) {
	d := Defaults()
	if d.ProfileFor("n64") != "compile" {
		t.Fatal(d.ProfileFor("n64"))
	}
	d.N64.Profile = "rom"
	if d.ProfileFor("n64") != "rom" {
		t.Fatal(d.ProfileFor("n64"))
	}
}
