package ps2

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	guest "github.com/blazium-games/blazium-toolchain/guest/ps2"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

func plantCompileTools(t *testing.T, prefix string) {
	t.Helper()
	gcc := filepath.Join(prefix, "ps2", "ps2dev", "ee", "bin", "mips64r5900el-ps2-elf-gcc")
	if err := os.MkdirAll(filepath.Dir(gcc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gcc, []byte("gcc"), 0o644); err != nil {
		t.Fatal(err)
	}
	sdk := filepath.Join(prefix, "ps2", "ps2sdk")
	if err := os.MkdirAll(filepath.Join(sdk, "ee", "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sdk, "samples"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestSetupWritesState(t *testing.T) {
	dir := t.TempDir()
	plantCompileTools(t, dir)
	t.Setenv("EE_GCC", "")
	t.Setenv("PS2SDK", "")
	t.Setenv("PS2DEV", "")
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
	if _, err := os.Stat(filepath.Join(dir, "ps2", "guest", "runtime", "main.cpp")); err != nil {
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
	if info.ID != "ps2" || info.Status != platforms.StatusSupported {
		t.Fatalf("%+v", info)
	}
	var exp bool
	for _, c := range info.Commands {
		if c == "export-guest" {
			exp = true
		}
	}
	if !exp {
		t.Fatalf("commands %v", info.Commands)
	}
}

func TestISORequiresDirAndOut(t *testing.T) {
	err := New().ISO(context.Background(), platforms.ISOOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir()},
	})
	if err == nil {
		t.Fatal("expected usage error")
	}
}

func TestISOWritesSYSTEMCNF(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "GAME.ELF"), []byte{0x7f, 'E', 'L', 'F', 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "g.iso")
	err := New().ISO(context.Background(), platforms.ISOOptions{
		CommonOptions: platforms.CommonOptions{Prefix: t.TempDir(), Stdout: os.Stdout},
		Dir:           dir,
		Out:           out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SYSTEM.CNF")); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyMipsELF(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.elf")
	hdr := make([]byte, 20)
	copy(hdr, []byte{0x7f, 'E', 'L', 'F'})
	hdr[18] = 8
	if err := os.WriteFile(p, hdr, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyMipsELF(p); err != nil {
		t.Fatal(err)
	}
}

func TestBiosReady(t *testing.T) {
	dir := t.TempDir()
	if biosReady(map[string]string{"PS2_BIOS_DIR": dir}) {
		t.Fatal("empty dir should not be ready")
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if biosReady(map[string]string{"PS2_BIOS_DIR": dir}) {
		t.Fatal("txt only")
	}
	if err := os.WriteFile(filepath.Join(dir, "scph10000.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !biosReady(map[string]string{"PS2_BIOS_DIR": dir}) {
		t.Fatal("bin should count")
	}
}
