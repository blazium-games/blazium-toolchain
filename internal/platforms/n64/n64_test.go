package n64

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	guest "github.com/blazium-games/blazium-toolchain/guest/n64"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func plantCompileTools(t *testing.T, prefix string) {
	t.Helper()
	inst := filepath.Join(prefix, "n64", "n64-inst")
	gcc := filepath.Join(inst, "bin", "mips64-elf-gcc")
	if runtime.GOOS == "windows" {
		gcc += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(gcc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gcc, []byte("gcc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inst, "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst, "include", "n64.mk"), []byte("# n64.mk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(inst, "mips64-elf", "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "libdragon.a"), []byte("lib"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSetupWritesState(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	t.Setenv("N64_INST", "")
	t.Setenv("N64_GCC", "")
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: os.Stdout},
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
	if !compileReady(env) {
		t.Fatalf("not ready: %+v", env)
	}
	if forbiddenUltra(env["N64_INST"]) {
		t.Fatal("must never select C:\\ultra")
	}
	if _, err := os.Stat(filepath.Join(dir, "n64", "guest", "runtime", "main.cpp")); err != nil {
		t.Fatal(err)
	}
	if guest.CookABI != 1 {
		t.Fatalf("abi %d", guest.CookABI)
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

func TestInfoSupported(t *testing.T) {
	info := New().Info()
	if info.ID != "n64" || info.Status != platforms.StatusSupported {
		t.Fatalf("%+v", info)
	}
	var exp, rom bool
	for _, c := range info.Commands {
		if c == "export-guest" {
			exp = true
		}
		if c == "rom" {
			rom = true
		}
		if c == "iso" {
			t.Fatal("n64 must not list iso as a product command")
		}
	}
	if !exp || !rom {
		t.Fatalf("commands %v", info.Commands)
	}
	if !strings.Contains(info.Description, "Windows") || !strings.Contains(info.Description, "Linux") {
		t.Fatalf("description must name Windows/Linux: %s", info.Description)
	}
	if !strings.Contains(info.Description, "bundled") {
		t.Fatalf("description must say guest is bundled: %s", info.Description)
	}
	if !strings.Contains(info.Description, ".z64") {
		t.Fatalf("description must name .z64: %s", info.Description)
	}
}

func TestISOUsageError(t *testing.T) {
	err := New().ISO(context.Background(), platforms.ISOOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
		Dir:           t.TempDir(),
		Out:           "game.iso",
	})
	if err == nil || !strings.Contains(err.Error(), "rom") {
		t.Fatalf("ISO must usage-error toward n64 rom: %v", err)
	}
}

func TestLookBashSkipsWSLLauncher(t *testing.T) {
	p := lookBash()
	if p == "" {
		t.Skip("no Git/MSYS bash on this host")
	}
	if strings.Contains(strings.ToLower(p), `system32\bash`) {
		t.Fatalf("must not use WSL launcher: %s", p)
	}
}

func TestForbiddenUltra(t *testing.T) {
	if !forbiddenUltra(`C:\ultra`) || !forbiddenUltra(`C:\ultra\GCC\MIPSE\BIN`) {
		t.Fatal("must reject official SDK paths")
	}
	if forbiddenUltra(`C:\n64-toolchain`) || forbiddenUltra(t.TempDir()) {
		t.Fatal("must not reject a normal prefix")
	}
}

func TestVerifyZ64(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(ok, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyZ64(ok); err != nil {
		t.Fatal(err)
	}
	pe := filepath.Join(dir, "bad.exe")
	if err := os.WriteFile(pe, []byte("MZ\x00\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyZ64(pe); err == nil {
		t.Fatal("PE must fail")
	}
	elf := filepath.Join(dir, "bad.elf")
	if err := os.WriteFile(elf, []byte{0x7f, 'E', 'L', 'F', 1, 2, 3, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyZ64(elf); err == nil {
		t.Fatal("ELF must fail as product")
	}
}

func TestStageCookEmbedNtex(t *testing.T) {
	dir := t.TempDir()
	ntex := filepath.Join(dir, "in-ntex.bin")
	if err := os.WriteFile(ntex, []byte("NTEX\x01\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := platforms.BuildOptions{Ntex: ntex}
	if !hasCookSlices(opts) {
		t.Fatal("expected ntex cook slice")
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "NTEX00.bin")); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_HAS_NTEX") {
		t.Fatalf("%s", flags)
	}
}

func TestRunRequiresRom(t *testing.T) {
	err := New().Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
	})
	if err == nil {
		t.Fatal("expected usage error")
	}
}

type memFetch struct{}

func (memFetch) FetchZip(_ context.Context, _, _, destDir string, _ io.Writer) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	name := "ares"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return os.WriteFile(filepath.Join(destDir, name), []byte("ares"), 0o644)
}

func TestAresRunArgsUseSettingsTree(t *testing.T) {
	args := aresRunArgs("game.z64")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "General/HomebrewMode=true") {
		t.Fatalf("homebrew key: %v", args)
	}
	if !strings.Contains(joined, "Nintendo64/ExpansionPak=true") {
		t.Fatalf("expansion key: %v", args)
	}
	if strings.Contains(joined, "Homebrew Mode=") || strings.Contains(joined, "Expansion Pak=") {
		t.Fatalf("must use settings tree keys, not UI labels: %v", args)
	}
}

func TestOfficialAresURLWindows(t *testing.T) {
	url := officialAresURLFor("windows", "amd64")
	if !strings.Contains(url, "ares-emulator/ares") || !strings.HasSuffix(url, "ares-windows-x64.zip") {
		t.Fatalf("official Ares zip: %s", url)
	}
	if officialAresURLFor("linux", "amd64") != "" {
		t.Fatal("linux has no official GitHub Ares zip in v148")
	}
}

func TestSetupDevFetchesAres(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	t.Setenv("N64_INST", "")
	t.Setenv("N64_GCC", "")
	t.Setenv("ARES_EXE", "")
	t.Setenv("PROJECT64_EXE", "")
	tool := &Tool{Fetcher: memFetch{}}
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: os.Stdout},
		Profile:       "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !fileExists(env["ARES_EXE"]) {
		t.Fatalf("dev profile must fetch Ares: %+v", env)
	}
	st, err := tool.Status(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !st["ares_ready"].(bool) {
		t.Fatalf("status ares_ready: %+v", st)
	}
}

func TestProject64LinuxRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("linux-only assertion")
	}
	rom := filepath.Join(t.TempDir(), "g.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(rom, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err := New().Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
		Exe:           rom,
		Emu:           "project64",
	})
	if err == nil {
		t.Fatal("expected project64 linux refusal")
	}
}
