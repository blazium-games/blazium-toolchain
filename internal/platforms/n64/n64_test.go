package n64

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	asm, err := os.ReadFile(filepath.Join(dest, "cook_embed.S"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(asm), "size_cooked_") {
		t.Fatal("size word after incbin breaks mips64-elf GP relocs")
	}
	if !strings.Contains(string(asm), "cooked_ntex_end") {
		t.Fatalf("need end label: %s", asm)
	}
}

func TestStageCookEmbedInp(t *testing.T) {
	dir := t.TempDir()
	inp := filepath.Join(dir, "in-inp.bin")
	if err := os.WriteFile(inp, []byte("INP6\x01\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := platforms.BuildOptions{Inp: inp}
	if !hasCookSlices(opts) {
		t.Fatal("expected inp cook slice")
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "INP600.bin")); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_HAS_INP") {
		t.Fatalf("%s", flags)
	}
	asm, err := os.ReadFile(filepath.Join(dest, "cook_embed.S"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(asm), "size_cooked_") {
		t.Fatal("size word after incbin breaks mips64-elf GP relocs")
	}
	if !strings.Contains(string(asm), "cooked_inp_end") {
		t.Fatalf("need end label: %s", asm)
	}
}

func TestStageCookAudioCopiesWav(t *testing.T) {
	dir := t.TempDir()
	wav := filepath.Join(dir, "beep.wav")
	// Minimal RIFF WAVE header + 4 bytes of silence.
	raw := []byte{
		'R', 'I', 'F', 'F', 36, 0, 0, 0, 'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ', 16, 0, 0, 0, 1, 0, 1, 0,
		0x22, 0x56, 0, 0, 0x44, 0xAC, 0, 0, 2, 0, 16, 0,
		'd', 'a', 't', 'a', 4, 0, 0, 0, 0, 0, 0, 0,
	}
	if err := os.WriteFile(wav, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := platforms.BuildOptions{Sfx: wav, Music: wav}
	if !hasCookAudio(opts) {
		t.Fatal("expected sfx/music cook")
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "assets", "SFX00.wav")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "assets", "MUSIC00.wav")); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_HAS_SFX") || !strings.Contains(string(flags), "BLAZIUM_N64_HAS_MUSIC") {
		t.Fatalf("%s", flags)
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "SFX00.wav64") || !strings.Contains(string(mk), "$(ROMNAME).z64") {
		t.Fatalf("%s", mk)
	}
	w64, err := os.ReadFile(filepath.Join(dest, "filesystem", "SFX00.wav64"))
	if err != nil {
		t.Fatal(err)
	}
	if len(w64) < 4 || string(w64[0:4]) != "WV64" {
		t.Fatalf("wav64 magic: %q", w64)
	}
}

func TestStageCookPackCopiesDfs(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "PACK01.bin")
	raw := []byte{'P', 'A', 'C', 'K', 1, 0, 1, 0, 4, 0, 0, 0, 'p', 'k', '0', '1'}
	if err := os.WriteFile(pack, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := platforms.BuildOptions{Pack: pack}
	if !hasCookDfs(opts) {
		t.Fatal("expected pack cook")
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "filesystem", "PACK01.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got[0:4]) != "PACK" {
		t.Fatalf("%q", got)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_HAS_PACK") {
		t.Fatalf("%s", flags)
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mk), "cooked_pack") || strings.Contains(string(mk), ".incbin") {
		t.Fatal("extra pack must be DFS, not incbin")
	}
	if !strings.Contains(string(mk), "filesystem/PACK01.bin") {
		t.Fatalf("%s", mk)
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
	if !strings.Contains(joined, "General/AutoSaveMemory=true") {
		t.Fatalf("autosave key: %v", args)
	}
	if strings.Contains(joined, "Homebrew Mode=") || strings.Contains(joined, "Expansion Pak=") {
		t.Fatalf("must use settings tree keys, not UI labels: %v", args)
	}
}

func TestAresEepromPathIsSibling(t *testing.T) {
	got := aresEepromPath(`D:\export\Game.z64`)
	if !strings.HasSuffix(got, "Game.eeprom") {
		t.Fatalf("%s", got)
	}
	if strings.Contains(got, ".z64") {
		t.Fatalf("must replace extension: %s", got)
	}
	pj := pj64SaveDir(`C:\emu\Project64.exe`)
	if !strings.HasSuffix(filepath.ToSlash(pj), "Save") {
		t.Fatalf("%s", pj)
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

func TestRunSkipsMissingSingleEmu(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows One-click skip")
	}
	dir := t.TempDir()
	rom := filepath.Join(dir, "g.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(rom, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_EXE", "")
	t.Setenv("PROJECT64_EXE", filepath.Join(dir, "missing-pj64.exe"))
	var buf bytes.Buffer
	err := New().Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "project64",
		Timeout:       time.Second,
	})
	if err != nil {
		t.Fatalf("missing Project64 must skip, not fail: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "skip project64") {
		t.Fatalf("expected skip print, got %q", buf.String())
	}
}

func TestCICDRequiresN64CompileHello(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var yaml []byte
	for i := 0; i < 8; i++ {
		p := filepath.Join(dir, ".github", "workflows", "cicd.yml")
		if raw, readErr := os.ReadFile(p); readErr == nil {
			yaml = raw
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if len(yaml) == 0 {
		t.Fatal("cicd.yml not found")
	}
	src := string(yaml)
	for _, want := range []string{
		"n64-compile",
		"n64 setup --profile compile",
		"n64 build --out hello.z64",
		"80 37 12 40",
		"hello.z64",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("cicd.yml missing %q", want)
		}
	}
	start := strings.Index(src, "n64-compile:")
	if start < 0 {
		t.Fatal("n64-compile job missing")
	}
	compileBlock := src[start:]
	if end := strings.Index(compileBlock, "\n  build:"); end >= 0 {
		compileBlock = compileBlock[:end]
	}
	if strings.Contains(compileBlock, "n64 run") || strings.Contains(compileBlock, "--emu") {
		t.Fatal("n64-compile must not spawn n64 run / --emu")
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
