package n64

import (
	"bytes"
	"context"
	"errors"
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
	if strings.Contains(string(flags), "BLAZIUM_N64_DISPLAY_640") {
		t.Fatal("default embed must not set 640")
	}
	if strings.Contains(string(flags), "BLAZIUM_N64_RUMBLE") {
		t.Fatal("default embed must not set rumble")
	}
	if strings.Contains(string(flags), "BLAZIUM_N64_RDRAM_4") {
		t.Fatal("default embed must not set 4 MiB no-pak")
	}
}

func TestStageCookEmbedDisplay640(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageCookEmbed(dest, platforms.BuildOptions{Display: "640"}); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_DISPLAY_640") {
		t.Fatalf("missing 640 flag: %s", flags)
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "BLAZIUM_N64_DISPLAY_640") {
		t.Fatalf("cook.mk missing 640: %s", mk)
	}
}

func TestStageCookEmbedRumble(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageCookEmbed(dest, platforms.BuildOptions{Rumble: true}); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_RUMBLE") {
		t.Fatalf("missing rumble flag: %s", flags)
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "BLAZIUM_N64_RUMBLE") {
		t.Fatalf("cook.mk missing rumble: %s", mk)
	}
}

func TestStageCookEmbedRdram4(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageCookEmbed(dest, platforms.BuildOptions{Rdram: "4"}); err != nil {
		t.Fatal(err)
	}
	flags, err := os.ReadFile(filepath.Join(dest, "cook_flags.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(flags), "BLAZIUM_N64_RDRAM_4") {
		t.Fatalf("missing 4 MiB flag: %s", flags)
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "BLAZIUM_N64_RDRAM_4") {
		t.Fatalf("cook.mk missing 4 MiB: %s", mk)
	}
}

func TestNormalizeRdram(t *testing.T) {
	opts := platforms.BuildOptions{}
	if err := normalizeRdram(&opts); err != nil || opts.Rdram != "8" {
		t.Fatalf("empty -> 8: %v %q", err, opts.Rdram)
	}
	opts.Rdram = "4"
	if err := normalizeRdram(&opts); err != nil || opts.Rdram != "4" {
		t.Fatalf("4: %v %q", err, opts.Rdram)
	}
	opts.Rdram = "16"
	if err := normalizeRdram(&opts); err == nil || !errors.Is(err, platforms.ErrUsage) {
		t.Fatalf("16 must be ErrUsage, got %v", err)
	}
}

func TestNormalizeDisplay(t *testing.T) {
	opts := platforms.BuildOptions{}
	if err := normalizeDisplay(&opts); err != nil || opts.Display != "320" {
		t.Fatalf("empty -> 320: %v %q", err, opts.Display)
	}
	opts.Display = "640"
	if err := normalizeDisplay(&opts); err != nil || opts.Display != "640" {
		t.Fatalf("640: %v %q", err, opts.Display)
	}
	opts.Display = "800"
	if err := normalizeDisplay(&opts); err == nil || !errors.Is(err, platforms.ErrUsage) {
		t.Fatalf("800 must be ErrUsage, got %v", err)
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

func TestStageCookPackDirStagesExtras(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "export")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	pack := []byte{'P', 'A', 'C', 'K', 1, 0, 1, 0, 4, 0, 0, 0, 'r', 'o', 'o', 'm'}
	if err := os.WriteFile(filepath.Join(src, "PACK01.bin"), pack, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "MESH01.bin"), []byte("MESH01rebind"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "NODE01.bin"), []byte("NODE01rebind"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "NTEX01.bin"), []byte("NTEX01rebind"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "src")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := platforms.BuildOptions{PackDir: src}
	if !hasCookDfs(opts) {
		t.Fatal("expected pack-dir cook")
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"PACK01.bin", "MESH01.bin", "NODE01.bin", "NTEX01.bin"} {
		if _, err := os.Stat(filepath.Join(dest, "filesystem", name)); err != nil {
			t.Fatalf("missing staged %s: %v", name, err)
		}
	}
	mk, err := os.ReadFile(filepath.Join(dest, "cook.mk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mk), "filesystem/MESH01.bin") {
		t.Fatalf("%s", mk)
	}
	cam := []byte{'C', 'A', 'M', 'N', 1, 0, 0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(src, "CAM00.bin"), cam, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stageCookEmbed(dest, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "filesystem", "CAM00.bin")); err != nil {
		t.Fatalf("missing staged CAM00.bin: %v", err)
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
	args := aresRunArgs("game.z64", true)
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

func TestAresRunArgsRdram4(t *testing.T) {
	args := aresRunArgs("game.z64", false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "Nintendo64/ExpansionPak=false") {
		t.Fatalf("4 MiB no-pak must set ExpansionPak=false: %v", args)
	}
	if strings.Contains(joined, "Nintendo64/ExpansionPak=true") {
		t.Fatalf("4 MiB no-pak must not keep ExpansionPak=true: %v", args)
	}
	if strings.Contains(joined, "Expansion Pak=") {
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

func TestPickPj64GfxPrefersParallelRejectsJabo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Jabo_Direct3D8.dll"), []byte("j"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "angrylion-rdp-plus.dll"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "parallel-rdp.dll"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := pickPj64Gfx(dir)
	if got != "parallel-rdp.dll" {
		t.Fatalf("prefer Parallel-RDP, got %q", got)
	}
}

func TestPickPj64GfxAcceptsAngrylion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Jabo_Direct3D8.dll"), []byte("j"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "angrylion-plus.dll"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := pickPj64Gfx(dir)
	if got != "angrylion-plus.dll" {
		t.Fatalf("got %q", got)
	}
}

func TestPickPj64GfxRejectsJaboOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Jabo_Direct3D8.dll"), []byte("j"), 0o644); err != nil {
		t.Fatal(err)
	}
	if pickPj64Gfx(dir) != "" {
		t.Fatal("Jabo-only GFX must not be picked")
	}
}

func TestWritePj64TestProfilePinsLimiterAndNotJabo(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "Config", "Project64.cfg")
	if err := writePj64TestProfile(cfg, `GFX\parallel-rdp.dll`); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	for _, want := range []string{
		"[Plugin]", "Graphics Dll=GFX\\parallel-rdp.dll", "Graphics Dll Default=GFX\\parallel-rdp.dll",
		"[Defaults]", "Unknown RDRAM Size=8388608", "Fixed Audio=1", "Audio-Sync Audio=1", "ViRefresh=1500",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("profile missing %q\n%s", want, src)
		}
	}
	if strings.Contains(strings.ToLower(src), "jabo") || strings.Contains(strings.ToLower(src), "direct3d8") {
		t.Fatalf("profile must not pin Jabo:\n%s", src)
	}
}

func TestWritePj64TestProfileRefusesJabo(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "Project64.cfg")
	if err := writePj64TestProfile(cfg, `GFX\Jabo_Direct3D8.dll`); err == nil {
		t.Fatal("must refuse Jabo")
	}
}

func TestRunSkipsPj64WithoutVideoPlugin(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Project64 skip")
	}
	dir := t.TempDir()
	rom := filepath.Join(dir, "g.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(rom, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	pj := filepath.Join(dir, "Project64.exe")
	if err := os.WriteFile(pj, []byte("mz"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "GFX"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GFX", "Jabo_Direct3D8.dll"), []byte("j"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_EXE", "")
	t.Setenv("PROJECT64_EXE", pj)
	var buf bytes.Buffer
	err := New().Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "project64",
		Timeout:       time.Second,
	})
	if err != nil {
		t.Fatalf("missing plugin must skip, not fail compile: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "skip project64") || !strings.Contains(buf.String(), "Parallel-RDP") {
		t.Fatalf("expected plugin skip print, got %q", buf.String())
	}
}

func TestRunSkipsPj64Nopak(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Project64 skip")
	}
	dir := t.TempDir()
	rom := filepath.Join(dir, "g.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(rom, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	pj := filepath.Join(dir, "Project64.exe")
	if err := os.WriteFile(pj, []byte("mz"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_EXE", "")
	t.Setenv("PROJECT64_EXE", pj)
	var buf bytes.Buffer
	err := New().Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "project64",
		Rdram:         "4",
		Timeout:       time.Second,
	})
	if err != nil {
		t.Fatalf("4 MiB no-pak must skip Project64, not fail: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "4 MiB no-pak skip project64: unknown-ROM pin stays 8 MB") {
		t.Fatalf("expected 4 MiB no-pak skip print, got %q", out)
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
	low := strings.ToLower(compileBlock)
	if strings.Contains(low, "t3dquad") || strings.Contains(low, "tiny3d") || strings.Contains(low, "00_quad") {
		t.Fatal("n64-compile must not build Tiny3D")
	}
	if strings.Contains(compileBlock, "--display 640") || strings.Contains(low, "hires") {
		t.Fatal("n64-compile must not require 640x480")
	}
	if strings.Contains(compileBlock, "--rumble") || strings.Contains(low, "rumble") {
		t.Fatal("n64-compile must not require rumble")
	}
	if strings.Contains(compileBlock, "ovldemo") || strings.Contains(low, "n64dso") {
		t.Fatal("n64-compile must not require DSO overlays")
	}
	if strings.Contains(compileBlock, "--rdram 4") || strings.Contains(low, "nopak") {
		t.Fatal("n64-compile must not require --rdram 4")
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

// fakeEmuRunner records spawned names. hang=true waits on ctx (smoke success).
type fakeEmuRunner struct {
	names []string
	hang  func(name string) bool
}

func (f *fakeEmuRunner) LookPath(name string) (string, error) { return name, nil }

func (f *fakeEmuRunner) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	return f.RunEnv(ctx, name, args, nil, nil, stdout, stderr)
}

func (f *fakeEmuRunner) RunEnv(ctx context.Context, name string, args []string, _ []string, _ map[string]string, _, _ io.Writer) error {
	f.names = append(f.names, name)
	if f.hang == nil || f.hang(name) {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func writeTinyZ64(t *testing.T, dir string) string {
	t.Helper()
	rom := filepath.Join(dir, "g.z64")
	raw := make([]byte, 64)
	copy(raw, z64Magic)
	if err := os.WriteFile(rom, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return rom
}

func plantAresDummy(t *testing.T, dir string) string {
	t.Helper()
	ares := filepath.Join(dir, "ares.exe")
	if err := os.WriteFile(ares, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ares
}

func plantPj64ReadyTree(t *testing.T, dir string) string {
	t.Helper()
	pj := filepath.Join(dir, "Project64.exe")
	if err := os.WriteFile(pj, []byte("mz"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "GFX"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GFX", "parallel-rdp.dll"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	return pj
}

func emuBaseHas(names []string, part string) bool {
	part = strings.ToLower(part)
	for _, n := range names {
		if strings.Contains(strings.ToLower(filepath.Base(n)), part) {
			return true
		}
	}
	return false
}

func TestRunBothSucceedsWhenBothHang(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Project64 dual-emu")
	}
	dir := t.TempDir()
	rom := writeTinyZ64(t, dir)
	ares := plantAresDummy(t, dir)
	pj := plantPj64ReadyTree(t, dir)
	t.Setenv("ARES_EXE", ares)
	t.Setenv("PROJECT64_EXE", pj)
	fake := &fakeEmuRunner{hang: func(string) bool { return true }}
	var buf bytes.Buffer
	err := (&Tool{Runner: fake}).Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "both",
		Timeout:       50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("both hanging must succeed: %v\n%s", err, buf.String())
	}
	if !emuBaseHas(fake.names, "ares") || !emuBaseHas(fake.names, "project64") {
		t.Fatalf("must invoke both, got %v", fake.names)
	}
}

func TestRunBothFailsWhenOnlyAresRan(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Project64 dual-emu")
	}
	dir := t.TempDir()
	rom := writeTinyZ64(t, dir)
	ares := plantAresDummy(t, dir)
	pj := plantPj64ReadyTree(t, dir)
	t.Setenv("ARES_EXE", ares)
	t.Setenv("PROJECT64_EXE", pj)
	fake := &fakeEmuRunner{hang: func(name string) bool {
		return strings.Contains(strings.ToLower(filepath.Base(name)), "ares")
	}}
	var buf bytes.Buffer
	err := (&Tool{Runner: fake}).Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "both",
		Timeout:       50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("one-emu must not be CI green when both validators are installed")
	}
	if errors.Is(err, platforms.ErrMissingTool) {
		t.Fatalf("must not wrap ErrMissingTool: %v", err)
	}
	if !strings.Contains(err.Error(), "one-emu is not CI green when both validators are installed") {
		t.Fatalf("got %v\n%s", err, buf.String())
	}
	if !emuBaseHas(fake.names, "ares") || !emuBaseHas(fake.names, "project64") {
		t.Fatalf("must invoke both, got %v", fake.names)
	}
}

func TestRunBothSkipsMissingPj64(t *testing.T) {
	dir := t.TempDir()
	rom := writeTinyZ64(t, dir)
	ares := plantAresDummy(t, dir)
	t.Setenv("ARES_EXE", ares)
	t.Setenv("PROJECT64_EXE", filepath.Join(dir, "missing-pj64.exe"))
	fake := &fakeEmuRunner{hang: func(string) bool { return true }}
	var buf bytes.Buffer
	err := (&Tool{Runner: fake}).Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir, Stdout: &buf},
		Exe:           rom,
		Emu:           "both",
		Timeout:       50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("missing PJ64 must skip+print, not fail: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "skip project64") {
		t.Fatalf("expected skip print, got %q", buf.String())
	}
	if emuBaseHas(fake.names, "project64") {
		t.Fatalf("must not spawn missing PJ64: %v", fake.names)
	}
}

func TestResolveSampleT3dQuad(t *testing.T) {
	root := t.TempDir()
	t3d := filepath.Join(root, "n64_stuff", "tiny3d")
	quad := filepath.Join(t3d, "examples", "00_quad")
	if err := os.MkdirAll(quad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(t3d, "t3d.mk"), []byte("# t3d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(quad, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	got, err := resolveSample("t3dquad")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(quad)
	if filepath.Clean(got) != want {
		t.Fatalf("t3dquad -> %q want %q", got, want)
	}
	for _, alias := range []string{"00_quad", "tiny3d"} {
		g, err := resolveSample(alias)
		if err != nil {
			t.Fatalf("alias %s: %v", alias, err)
		}
		if filepath.Clean(g) != want {
			t.Fatalf("alias %s -> %q want %q", alias, g, want)
		}
	}
}

func TestResolveSampleOvlDemo(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "n64_stuff", "libdragon")
	demo := filepath.Join(lib, "examples", "ovldemo")
	if err := os.MkdirAll(demo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "n64.mk"), []byte("# n64\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	got, err := resolveSample("ovldemo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(demo)
	if filepath.Clean(got) != want {
		t.Fatalf("ovldemo -> %q want %q", got, want)
	}
	for _, alias := range []string{"overlay", "dso"} {
		g, err := resolveSample(alias)
		if err != nil {
			t.Fatalf("alias %s: %v", alias, err)
		}
		if filepath.Clean(g) != want {
			t.Fatalf("alias %s -> %q want %q", alias, g, want)
		}
	}
}

func TestResolveSampleUnknownRejected(t *testing.T) {
	_, err := resolveSample("gltf")
	if err == nil {
		t.Fatal("unknown sample must fail")
	}
	if !errors.Is(err, platforms.ErrUsage) {
		t.Fatalf("want ErrUsage, got %v", err)
	}
}
