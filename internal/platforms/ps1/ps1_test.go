package ps1

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func plantCompileTools(t *testing.T, prefix string) {
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
	libs := filepath.Join(prefix, "ps1", "psn00bsdk", "lib", "libpsn00b")
	if err := os.MkdirAll(libs, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestSetupWritesState(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
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
	if _, err := os.Stat(filepath.Join(dir, "ps1", "components.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ps1", "guest", "runtime", "CMakeLists.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ps1", "guest", "runtime", "main.cpp")); err != nil {
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

func TestOfflineOKWithVendoredGCC(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	plantCompileTools(t, dir)
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
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
	if env["MIPS_GCC"] == "" || env["ELF2X"] == "" {
		t.Fatalf("expected vendored tools: %+v", env)
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

func TestStatusReady(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	tool := New()
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	st, err := tool.Status(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	ready, _ := st["ready"].(bool)
	if !ready {
		t.Fatalf("status %+v", st)
	}
}

type memFetch struct{}

func (memFetch) FetchZip(_ context.Context, url, _ string, dest string, _ io.Writer) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	switch filepath.Base(dest) {
	case "gcc":
		p := filepath.Join(dest, "bin", "mipsel-none-elf-gcc")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte("gcc"), 0o644)
	case "pcsx-redux":
		for _, n := range []string{"pcsx-redux", "pcsx-redux.exe"} {
			if err := os.WriteFile(filepath.Join(dest, n), []byte("pcsx"), 0o644); err != nil {
				return err
			}
		}
		bios := filepath.Join(dest, "resources", "openbios.bin")
		if err := os.MkdirAll(filepath.Dir(bios), 0o755); err != nil {
			return err
		}
		return os.WriteFile(bios, []byte("OB"), 0o644)
	default:
		elf := filepath.Join(dest, "bin", "elf2x")
		if err := os.MkdirAll(filepath.Dir(elf), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(elf, []byte("elf2x"), 0o644); err != nil {
			return err
		}
		return os.MkdirAll(filepath.Join(dest, "lib", "libpsn00b"), 0o755)
	}
}

func plantDevTools(t *testing.T, prefix string) {
	t.Helper()
	dir := filepath.Join(prefix, "ps1", "pcsx-redux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range hostNames("pcsx-redux") {
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

func TestSetupFetchesWhenMissing(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("ELF2X", "")
	t.Setenv("PSN00BSDK_LIBS", "")
	t.Setenv("PSN00BSDK_TC", "")
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	tool := &Tool{Fetcher: memFetch{}, Assets: []ZipAsset{
		{ID: "mipsel-none-elf-gcc", URL: "http://example.test/gcc.zip", Dest: "gcc"},
		{ID: "psn00bsdk", URL: "http://example.test/sdk.zip", Dest: "psn00bsdk"},
	}}
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !compileReady(env) {
		t.Fatalf("not ready after fetch: %+v", env)
	}
}

func TestBuildRequiresFlags(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantSDKCMake(t, dir)
	tool := New()
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
	})
	if err == nil {
		t.Fatal("expected usage")
	}
}

func TestBuildRequiresSetup(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("ELF2X", "")
	t.Setenv("PSN00BSDK_LIBS", "")
	t.Setenv("PATH", t.TempDir())
	tool := New()
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
		Sample:        "template",
		Out:           filepath.Join(t.TempDir(), "hello.exe"),
	})
	if err == nil {
		t.Fatal("expected missing setup")
	}
}

type recRunner struct {
	calls [][]string
}

func (r *recRunner) LookPath(name string) (string, error) {
	return name, nil
}

func (r *recRunner) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	return r.RunEnv(ctx, name, args, nil, nil, stdout, stderr)
}

func (r *recRunner) RunEnv(_ context.Context, name string, args []string, _ []string, _ map[string]string, _ io.Writer, _ io.Writer) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	target := "template"
	for i, a := range args {
		if a == "--target" && i+1 < len(args) {
			target = args[i+1]
		}
	}
	for i, a := range args {
		if a == "--build" && i+1 < len(args) {
			out := filepath.Join(args[i+1], target+".exe")
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			return os.WriteFile(out, []byte("PS-X EXE\x00payload"), 0o644)
		}
	}
	return nil
}

func plantSDKCMake(t *testing.T, prefix string) {
	t.Helper()
	sdk := filepath.Join(prefix, "ps1", "psn00bsdk", "lib", "libpsn00b", "cmake")
	if err := os.MkdirAll(sdk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sdk, "sdk.cmake"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func plantSample(t *testing.T, prefix, sample string) {
	t.Helper()
	rel := filepath.Join("ps1", "psn00bsdk", "share", "psn00bsdk", "template")
	if sample == "gte" {
		rel = filepath.Join("ps1", "psn00bsdk", "share", "psn00bsdk", "examples", "graphics", "gte")
	}
	dir := filepath.Join(prefix, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte("project(test)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func plantHostTools(t *testing.T, prefix string) {
	t.Helper()
	cmake := filepath.Join(prefix, "ps1", "cmake", "bin", "cmake")
	ninja := filepath.Join(prefix, "ps1", "ninja", "ninja")
	if err := os.MkdirAll(filepath.Dir(cmake), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ninja), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cmake, []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ninja, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildWritesPSXEXE(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantSDKCMake(t, dir)
	plantSample(t, dir, "template")
	plantHostTools(t, dir)
	rec := &recRunner{}
	tool := &Tool{Runner: rec, Fetcher: memFetch{}}
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "hello.exe")
	sprite := filepath.Join(dir, "SPRITE00.bin")
	if err := os.WriteFile(sprite, []byte{2, 0, 1, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "SCRIPT.IR")
	if err := os.WriteFile(script, []byte{0, 0, 0xcd, 0xcc, 0xcc, 0x3e, 0, 0, 0, 2, 160, 0, 0, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Sample:        "template",
		Out:           out,
		Sprite:        sprite,
		Script:        script,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPSXEXE(out); err != nil {
		t.Fatal(err)
	}
	var sawNinja, sawTarget, sawSprite, sawScript bool
	for _, c := range rec.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "-G Ninja") {
			sawNinja = true
		}
		if strings.Contains(joined, "--target template") {
			sawTarget = true
		}
		if strings.Contains(joined, "BLAZIUM_PS1_SPRITE=") {
			sawSprite = true
		}
		if strings.Contains(joined, "BLAZIUM_PS1_SCRIPT=") {
			sawScript = true
		}
	}
	if !sawNinja || !sawTarget || !sawSprite || !sawScript {
		t.Fatalf("cmake calls: %v", rec.calls)
	}
}

func TestBuildInstallsGuestWithoutSetup(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantSDKCMake(t, dir)
	plantHostTools(t, dir)
	rec := &recRunner{}
	tool := &Tool{Runner: rec, Fetcher: memFetch{}}
	out := filepath.Join(dir, "game.exe")
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Out:           out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(GuestDir(dir), "CMakeLists.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestBuildUsesBundledGuest(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantSDKCMake(t, dir)
	plantHostTools(t, dir)
	rec := &recRunner{}
	tool := &Tool{Runner: rec, Fetcher: memFetch{}}
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "game.exe")
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Out:           out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPSXEXE(out); err != nil {
		t.Fatal(err)
	}
	guest := GuestDir(dir)
	var sawSrc, sawTarget bool
	for _, c := range rec.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, guest) {
			sawSrc = true
		}
		if strings.Contains(joined, "--target runtime") {
			sawTarget = true
		}
	}
	var sawWork bool
	for _, c := range rec.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "blazium-guest") {
			sawWork = true
		}
	}
	if !sawSrc || !sawTarget || !sawWork {
		t.Fatalf("expected bundled guest %s, --target runtime, work/blazium-guest; cmake calls: %v", guest, rec.calls)
	}
}

func TestGTEMissingTim(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantSDKCMake(t, dir)
	plantSample(t, dir, "gte")
	tool := New()
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "compile",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	err := tool.Build(context.Background(), platforms.BuildOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Sample:        "gte",
		Out:           filepath.Join(dir, "gte.exe"),
	})
	if err == nil {
		t.Fatal("expected missing texture.tim")
	}
}

func TestComponentsAreContainable(t *testing.T) {
	for _, c := range componentsForProfile("iso") {
		if !c.Contained {
			t.Fatalf("%s should be containable in this repo", c.ID)
		}
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

func TestFindLibpsn00bPrefersLib(t *testing.T) {
	dir := t.TempDir()
	inc := filepath.Join(dir, "include", "libpsn00b")
	lib := filepath.Join(dir, "lib", "libpsn00b")
	if err := os.MkdirAll(inc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	got := findLibpsn00b(dir)
	if got != lib && filepath.Clean(got) != filepath.Clean(lib) {
		t.Fatalf("got %q want %q", got, lib)
	}
}

func TestCMakeTargetUsesSrcBasename(t *testing.T) {
	if cmakeTarget("", filepath.Join("blazium", "platform", "ps1", "runtime")) != "runtime" {
		t.Fatal(cmakeTarget("", "runtime"))
	}
	if cmakeTarget("template", "") != "template" {
		t.Fatal("template")
	}
	if cmakeTarget("gte", "") != "gte" {
		t.Fatal("gte")
	}
}

func TestGCCPinIs123(t *testing.T) {
	if GCCSeries != "12.3.0" {
		t.Fatalf("GCC pin %s", GCCSeries)
	}
}

func TestSetupDevFetchesCLIAndOpenBIOS(t *testing.T) {
	t.Setenv("MIPS_GCC", "")
	t.Setenv("ELF2X", "")
	t.Setenv("PSN00BSDK_LIBS", "")
	t.Setenv("PSN00BSDK_TC", "")
	t.Setenv("PCSX_EXE", "")
	t.Setenv("OPENBIOS", "")
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	plantCompileTools(t, dir)
	tool := &Tool{Fetcher: memFetch{}, CLIURL: "http://example.test/pcsx.zip"}
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	env, err := tool.Env(platforms.CommonOptions{Prefix: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !destReady(env) {
		t.Fatalf("dev not ready: %+v", env)
	}
	if _, err := os.Stat(filepath.Join(dir, "ps1", "openbios", "openbios.bin")); err != nil {
		t.Fatal(err)
	}
}

func TestOfflineDevFailsWithoutEmu(t *testing.T) {
	t.Setenv("PCSX_EXE", "")
	t.Setenv("OPENBIOS", "")
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	plantCompileTools(t, dir)
	tool := New()
	err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "dev",
		Offline:       true,
	})
	if err == nil {
		t.Fatal("expected offline dest error")
	}
}

func TestRunUsesSetupEnv(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	plantDevTools(t, dir)
	rec := &recRunner{}
	tool := &Tool{Runner: rec}
	if err := tool.Setup(context.Background(), platforms.SetupOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Profile:       "dev",
		Offline:       true,
	}); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "hello.exe")
	if err := os.WriteFile(exe, []byte("PS-X EXE"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := tool.Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Exe:           exe,
		Timeout:       time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls %v", rec.calls)
	}
	joined := strings.Join(rec.calls[0], " ")
	if !strings.Contains(joined, "-testmode") || !strings.Contains(joined, "-no-ui") || !strings.Contains(joined, "-loadexe") || !strings.Contains(joined, "-bios") {
		t.Fatalf("args %s", joined)
	}
	rec.calls = nil
	if err := tool.Run(context.Background(), platforms.RunOptions{
		CommonOptions: platforms.CommonOptions{Prefix: dir},
		Exe:           exe,
		Timeout:       time.Minute,
		UI:            true,
	}); err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("ui calls %v", rec.calls)
	}
	ui := strings.Join(rec.calls[0], " ")
	if strings.Contains(ui, "-no-ui") || strings.Contains(ui, "-testmode") || !strings.Contains(ui, "-loadexe") {
		t.Fatalf("ui args %s", ui)
	}
}
