package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func plantCompile(t *testing.T, prefix string) {
	t.Helper()
	gcc := filepath.Join(prefix, "ps1", "gcc", "bin", "mipsel-none-elf-gcc")
	if err := os.MkdirAll(filepath.Dir(gcc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gcc, []byte("gcc"), 0o644); err != nil {
		t.Fatal(err)
	}
	elf := filepath.Join(prefix, "ps1", "elf2x", "elf2x")
	if err := os.MkdirAll(filepath.Dir(elf), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(elf, []byte("elf2x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(prefix, "ps1", "psn00bsdk", "lib", "libpsn00b"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestVersionJSON(t *testing.T) {
	var out bytes.Buffer
	code := Run(context.Background(), []string{"--json", "version"}, &out, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var m map[string]string
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["name"] != "blazium-toolchain" || m["version"] != Version || m["license"] != License {
		t.Fatalf("%v", m)
	}
}

func TestListJSONHasPS1(t *testing.T) {
	var out bytes.Buffer
	code := Run(context.Background(), []string{"--json", "list"}, &out, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var list []platforms.Info
	if err := json.Unmarshal(out.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	var ps1 bool
	for _, info := range list {
		if info.ID == "ps1" && info.Status == platforms.StatusSupported {
			ps1 = true
			var fmv, exp bool
			for _, c := range info.Commands {
				if c == "fmv" {
					fmv = true
				}
				if c == "export-guest" {
					exp = true
				}
			}
			if !fmv || !exp {
				t.Fatalf("ps1 commands missing fmv/export-guest: %v", info.Commands)
			}
		}
	}
	if !ps1 {
		t.Fatalf("list: %s", out.String())
	}
	var interdvd bool
	for _, info := range list {
		if info.ID == "interdvd" && info.Status == platforms.StatusSupported {
			interdvd = true
			var meta bool
			for _, c := range info.Commands {
				if c == "meta" {
					meta = true
				}
			}
			if !meta {
				t.Fatalf("interdvd commands missing meta: %v", info.Commands)
			}
		}
	}
	if !interdvd {
		t.Fatalf("missing interdvd in %s", out.String())
	}
	for _, want := range []string{"ps2", "ps3", "ps4"} {
		var found bool
		for _, info := range list {
			if info.ID == want && info.Status == platforms.StatusPlanned {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing planned %s in %s", want, out.String())
		}
	}
}

func TestPS2IsPlanned(t *testing.T) {
	var errBuf bytes.Buffer
	code := Run(context.Background(), []string{"ps2", "setup"}, &bytes.Buffer{}, &errBuf)
	if code != ExitPlanned {
		t.Fatalf("exit %d body %s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "ps2") {
		t.Fatalf("stderr %s", errBuf.String())
	}
}

func plantDev(t *testing.T, prefix string) {
	t.Helper()
	dir := filepath.Join(prefix, "ps1", "pcsx-redux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"pcsx-redux", "pcsx-redux.exe"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("pcsx"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bios := filepath.Join(prefix, "ps1", "openbios", "openbios.bin")
	if err := os.MkdirAll(filepath.Dir(bios), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bios, []byte("OB"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPS1SetupThenEnv(t *testing.T) {
	t.Setenv("PCSX_EXE", "")
	t.Setenv("OPENBIOS", "")
	dir := t.TempDir()
	plantCompile(t, dir)
	plantDev(t, dir)
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"--prefix", dir, "ps1", "setup", "--profile", "dev", "--offline"}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("setup %d %s", code, errb.String())
	}
	out.Reset()
	code = Run(context.Background(), []string{"--prefix", dir, "--json", "ps1", "env"}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("env %d %s", code, errb.String())
	}
	var env map[string]string
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
}

func TestPS1StatusReadyJSON(t *testing.T) {
	dir := t.TempDir()
	plantCompile(t, dir)
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"--prefix", dir, "ps1", "setup", "--profile", "compile", "--offline"}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("setup %d %s", code, errb.String())
	}
	out.Reset()
	code = Run(context.Background(), []string{"--prefix", dir, "--json", "ps1", "status"}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("status %d %s", code, errb.String())
	}
	var st map[string]any
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	ready, _ := st["ready"].(bool)
	if !ready {
		t.Fatalf("status %s", out.String())
	}
	abi, _ := st["guest_abi"].(float64)
	if int(abi) != 20 {
		t.Fatalf("guest_abi %v in %s", st["guest_abi"], out.String())
	}
	if supported, _ := st["host_supported"].(bool); !supported {
		t.Fatalf("host_supported %s", out.String())
	}
	if _, ok := st["host_os"].(string); !ok {
		t.Fatalf("host_os %s", out.String())
	}
}

func TestHelp(t *testing.T) {
	var out bytes.Buffer
	if Run(context.Background(), []string{"help"}, &out, &bytes.Buffer{}) != ExitOK {
		t.Fatal("help")
	}
	if !strings.Contains(out.String(), "ps1") || !strings.Contains(out.String(), "interdvd") {
		t.Fatal(out.String())
	}
	if !strings.Contains(out.String(), "Windows and Linux") {
		t.Fatal(out.String())
	}
}

func TestPS1FMVJSON(t *testing.T) {
	var out bytes.Buffer
	code := Run(context.Background(), []string{"--json", "ps1", "fmv"}, &out, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("exit %d body %s", code, out.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["encoder"] != "blazium-mit" || m["in_editor"] != true || m["spawn"] != false {
		t.Fatalf("%v", m)
	}
}

func TestUnknownCommand(t *testing.T) {
	code := Run(context.Background(), []string{"ps1", "frobnicate"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitUsage {
		t.Fatalf("exit %d", code)
	}
}

func plantInterDVD(t *testing.T, root string) {
	t.Helper()
	v := filepath.Join(root, "VIDEO_TS")
	if err := os.MkdirAll(v, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v, "VIDEO_TS.IFO"), []byte("IFO"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInterDVDISOAndMeta(t *testing.T) {
	src := t.TempDir()
	plantInterDVD(t, src)
	isoPath := filepath.Join(t.TempDir(), "game.iso")
	extra := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(extra, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{
		"--json", "interdvd", "iso",
		"--dir", src, "--out", isoPath,
		"--title", "CLI Title", "--license", "MIT",
		"--extra", extra + ":NOTES.TXT",
	}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("iso %d %s %s", code, errb.String(), out.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["format"] != "iso9660+udf" || m["title"] != "CLI Title" {
		t.Fatalf("%v", m)
	}

	metaPath := filepath.Join(t.TempDir(), "disc.interdvd.json")
	out.Reset()
	errb.Reset()
	code = Run(context.Background(), []string{"--json", "interdvd", "meta", "init", "--out", metaPath}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("init %d %s", code, errb.String())
	}
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "blazium.interdvd.meta/v1") {
		t.Fatalf("%s", b)
	}

	// Fill dir/out and validate + master via --meta with title override.
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	doc["dir"] = src
	doc["out"] = filepath.Join(t.TempDir(), "frommeta.iso")
	nb, _ := json.Marshal(doc)
	if err := os.WriteFile(metaPath, nb, 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	code = Run(context.Background(), []string{"interdvd", "meta", "validate", "--meta", metaPath}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("validate %d %s", code, errb.String())
	}
	out.Reset()
	errb.Reset()
	writeMeta := filepath.Join(t.TempDir(), "saved.json")
	code = Run(context.Background(), []string{
		"--json", "interdvd", "iso", "--meta", metaPath, "--title", "Override", "--write-meta", writeMeta,
	}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("meta iso %d %s", code, errb.String())
	}
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["title"] != "Override" {
		t.Fatalf("%v", m)
	}
	saved, err := os.ReadFile(writeMeta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "Override") {
		t.Fatalf("%s", saved)
	}

	bad := t.TempDir()
	badMeta := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badMeta, []byte(`{"schema":"blazium.interdvd.meta/v1","dir":"`+strings.ReplaceAll(bad, `\`, `\\`)+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code = Run(context.Background(), []string{"interdvd", "meta", "validate", "--meta", badMeta}, &bytes.Buffer{}, &bytes.Buffer{})
	if code == ExitOK {
		t.Fatal("validate should fail without IFO")
	}

	code = Run(context.Background(), []string{"ps1", "iso"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitUsage {
		t.Fatalf("ps1 iso without xml exit %d", code)
	}
}

func TestExportGuestWritesCPP(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "guest")
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"--json", "ps1", "export-guest", "--out", dest}, &out, &errb)
	if code != ExitOK {
		t.Fatalf("export-guest %d %s", code, errb.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["out"] != dest {
		t.Fatalf("%v", m)
	}
	for _, name := range []string{"CMakeLists.txt", "main.cpp", "script_vm.cpp", "script_vm.h", "fmv_play.cpp"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	cmake, err := os.ReadFile(filepath.Join(dest, "CMakeLists.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cmake), "extra/*.cpp") {
		t.Fatalf("CMakeLists missing extra glob: %s", cmake)
	}
}
