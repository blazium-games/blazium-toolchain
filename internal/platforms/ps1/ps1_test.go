package ps1

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

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
	if filepath.Base(dest) == "gcc" {
		p := filepath.Join(dest, "bin", "mipsel-none-elf-gcc")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte("gcc"), 0o644)
	}
	elf := filepath.Join(dest, "bin", "elf2x")
	if err := os.MkdirAll(filepath.Dir(elf), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(elf, []byte("elf2x"), 0o644); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(dest, "lib", "libpsn00b"), 0o755)
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
	tool := New()
	err := tool.Build(context.Background(), platforms.BuildOptions{})
	if err == nil {
		t.Fatal("expected usage")
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

func TestGCCPinIs123(t *testing.T) {
	if GCCSeries != "12.3.0" {
		t.Fatalf("GCC pin %s", GCCSeries)
	}
}
