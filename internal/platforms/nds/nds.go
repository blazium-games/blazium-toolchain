// Package nds is the Nintendo DS platform: a DS-mode .nds (which a DSi runs
// as well) built with BlocksDS, from the GDIR guest (guest/nds) and a PS2
// GDIR export converted to the PS1 forms (internal/platforms/ps1).
package nds

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	guest "github.com/blazium-games/blazium-toolchain/guest/nds"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

const ID = "nds"

// Tool is the DS implementation of platforms.Platform.
type Tool struct {
	Runner execx.Runner
}

func New() *Tool {
	return &Tool{Runner: execx.Host{}}
}

func (t *Tool) runner() execx.Runner {
	if t.Runner == nil {
		return execx.Host{}
	}
	return t.Runner
}

func (t *Tool) Info() platforms.Info {
	return platforms.Info{
		ID:       ID,
		Name:     "Nintendo DS",
		Status:   platforms.StatusSupported,
		Commands: []string{"setup", "env", "status", "build", "run"},
		Description: "Nintendo DS (DS mode; a DSi runs it too). BlocksDS builds the GDIR guest " +
			"(build --sample gdir) into a .nds with the export in NitroFS. Run: melonDS or DeSmuME.",
	}
}

// envKeys are what discover finds and Env reports.
var envKeys = []string{"BLOCKSDS", "ARM_NONE_EABI_PATH", "WONDERFUL_TOOLCHAIN", "NDS_MAKE", "MELONDS_EXE", "DESMUME_EXE"}

// discover finds BlocksDS, the ARM compiler and an emulator: the
// environment first, then the usual install places.
func (t *Tool) discover() map[string]string {
	env := map[string]string{}
	for _, k := range envKeys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			env[k] = v
		}
	}
	if env["WONDERFUL_TOOLCHAIN"] == "" && dirExists("/opt/wonderful") {
		env["WONDERFUL_TOOLCHAIN"] = "/opt/wonderful"
	}
	if env["BLOCKSDS"] == "" {
		for _, c := range []string{
			"/opt/blocksds/core",
			filepath.Join(env["WONDERFUL_TOOLCHAIN"], "thirdparty", "blocksds", "core"),
		} {
			if c != "" && fileExists(filepath.Join(c, "sys", "crts", "ds_arm9.specs")) {
				env["BLOCKSDS"] = c
				break
			}
		}
	}
	if env["ARM_NONE_EABI_PATH"] == "" {
		if wt := env["WONDERFUL_TOOLCHAIN"]; wt != "" {
			p := filepath.Join(wt, "toolchain", "gcc-arm-none-eabi", "bin")
			if fileExists(filepath.Join(p, exeName("arm-none-eabi-gcc"))) {
				env["ARM_NONE_EABI_PATH"] = p + string(os.PathSeparator)
			}
		}
	}
	if env["NDS_MAKE"] == "" {
		if p, err := exec.LookPath("make"); err == nil {
			env["NDS_MAKE"] = p
		}
	}
	if env["MELONDS_EXE"] == "" {
		for _, n := range []string{"melonDS", "melonds"} {
			if p, err := exec.LookPath(n); err == nil {
				env["MELONDS_EXE"] = p
				break
			}
		}
	}
	if env["DESMUME_EXE"] == "" {
		for _, n := range []string{"desmume", "desmume-cli", "/usr/games/desmume", "/usr/games/desmume-cli"} {
			if p, err := exec.LookPath(n); err == nil {
				env["DESMUME_EXE"] = p
				break
			}
		}
	}
	return env
}

func exeName(n string) string {
	if runtime.GOOS == "windows" {
		return n + ".exe"
	}
	return n
}

func compileReady(env map[string]string) bool {
	if env["BLOCKSDS"] == "" || !fileExists(filepath.Join(env["BLOCKSDS"], "sys", "crts", "ds_arm9.specs")) {
		return false
	}
	gcc := exeName("arm-none-eabi-gcc")
	if p := env["ARM_NONE_EABI_PATH"]; p != "" {
		return fileExists(filepath.Join(p, gcc))
	}
	_, err := exec.LookPath(gcc)
	return err == nil
}

func (t *Tool) Setup(ctx context.Context, opts platforms.SetupOptions) error {
	_ = ctx
	env := t.discover()
	profile := opts.Profile
	if profile == "" {
		profile = "compile"
	}
	if !compileReady(env) {
		return report.Missing("BlocksDS and an arm-none-eabi GCC (BLOCKSDS, ARM_NONE_EABI_PATH or WONDERFUL_TOOLCHAIN)",
			"install BlocksDS (Wonderful Toolchain: wf-pacman -S toolchain-gcc-arm-none-eabi blocksds-toolchain), or set BLOCKSDS and ARM_NONE_EABI_PATH")
	}
	if err := guest.InstallGDIR(GDIRGuestDir(opts.Prefix)); err != nil {
		return err
	}
	st := cache.State{Platform: ID, Profile: profile, Env: env}
	if err := cache.WriteState(opts.Prefix, ID, st); err != nil {
		return err
	}
	report.Setup(opts.Stdout, ID, profile, cache.PlatformDir(opts.Prefix, ID), env)
	if env["MELONDS_EXE"] == "" && env["DESMUME_EXE"] == "" {
		report.Line(opts.Stdout, report.Note, "no MELONDS_EXE or DESMUME_EXE: nds run will not find an emulator")
	}
	return nil
}

func (t *Tool) Env(opts platforms.CommonOptions) (platforms.EnvMap, error) {
	fresh := t.discover()
	st, err := cache.ReadState(opts.Prefix, ID)
	if err != nil || st.Env == nil {
		return fresh, nil
	}
	for _, k := range envKeys {
		if fresh[k] != "" {
			st.Env[k] = fresh[k]
		}
	}
	return st.Env, nil
}

func (t *Tool) Status(opts platforms.CommonOptions) (map[string]any, error) {
	env, err := t.Env(opts)
	if err != nil {
		return nil, err
	}
	comp := compileReady(env)
	emu := fileExists(env["MELONDS_EXE"]) || fileExists(env["DESMUME_EXE"])
	return map[string]any{
		"platform":      ID,
		"env":           env,
		"compile_ready": comp,
		"run_ready":     comp && emu,
		"ready":         comp,
		"product":       "nds",
		"mode":          "DS (runs on DSi)",
		"guest_dir":     GDIRGuestDir(opts.Prefix),
		"host_os":       runtime.GOOS,
	}, nil
}

// GDIRGuestDir is <prefix>/nds/guest/gdir: the GDIR guest, staged per build.
func GDIRGuestDir(prefix string) string {
	return filepath.Join(cache.PlatformDir(prefix, ID), "guest", "gdir")
}

func (t *Tool) Build(ctx context.Context, opts platforms.BuildOptions) error {
	env, _ := t.Env(opts.CommonOptions)
	if !compileReady(env) {
		return report.Missing("nds compile tools not ready", "blazium-toolchain nds setup")
	}
	if strings.ToLower(strings.TrimSpace(opts.Sample)) != "gdir" {
		return report.Usage("nds build needs --sample gdir (the GDIR guest with a PS2 export)")
	}
	if opts.Out == "" {
		return report.Usage("build requires --out FILE.nds")
	}
	export, err := gdirExportDir(opts.PackDir, opts.Out)
	if err != nil {
		return err
	}
	src := GDIRGuestDir(opts.Prefix)
	if _, err := stageGDIR(src, export, opts.Stdout); err != nil {
		return err
	}
	if ov := strings.TrimSpace(opts.Overlay); ov != "" {
		if err := guest.Overlay(src, ov); err != nil {
			return err
		}
	}
	makeProg := env["NDS_MAKE"]
	if makeProg == "" {
		makeProg = "make"
	}
	extra := map[string]string{"BLOCKSDS": env["BLOCKSDS"]}
	if v := env["ARM_NONE_EABI_PATH"]; v != "" {
		extra["ARM_NONE_EABI_PATH"] = v
	}
	if v := env["WONDERFUL_TOOLCHAIN"]; v != "" {
		extra["WONDERFUL_TOOLCHAIN"] = v
	}
	if v := os.Getenv("GDIR_EXTRA_FLAGS"); v != "" {
		extra["GDIR_EXTRA_FLAGS"] = v
	}
	if err := t.runEnv(ctx, makeProg, []string{"-C", src}, extra, opts.Stdout, opts.Stderr); err != nil {
		return report.Fail("make: "+err.Error(), "")
	}
	rom := filepath.Join(src, "gdir.nds")
	raw, err := os.ReadFile(rom)
	if err != nil {
		return report.Fail("no gdir.nds was made", "")
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(opts.Out, raw, 0o644); err != nil {
		return err
	}
	if opts.Stdout != nil {
		report.Line(opts.Stdout, report.Wrote, opts.Out)
	}
	return nil
}

func (t *Tool) runEnv(ctx context.Context, name string, args []string, extraEnv map[string]string, stdout, stderr io.Writer) error {
	type envRunner interface {
		RunEnv(ctx context.Context, name string, args []string, extraPath []string, extraEnv map[string]string, stdout, stderr io.Writer) error
	}
	if er, ok := t.runner().(envRunner); ok {
		return er.RunEnv(ctx, name, args, nil, extraEnv, writerOr(stdout), writerOr(stderr))
	}
	return t.runner().Run(ctx, name, args, writerOr(stdout), writerOr(stderr))
}

func writerOr(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func (t *Tool) Run(ctx context.Context, opts platforms.RunOptions) error {
	env, _ := t.Env(opts.CommonOptions)
	rom := strings.TrimSpace(opts.Exe)
	if rom == "" {
		rom = strings.TrimSpace(opts.ISO)
	}
	if rom == "" {
		return report.Usage("nds run needs --exe GAME.nds")
	}
	emu := env["MELONDS_EXE"]
	if strings.EqualFold(opts.Emu, "desmume") || emu == "" {
		emu = env["DESMUME_EXE"]
	}
	if emu == "" {
		return report.Missing("no DS emulator (MELONDS_EXE or DESMUME_EXE)", "install melonDS and set MELONDS_EXE")
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := t.runner().Run(cctx, emu, []string{rom}, writerOr(opts.Stdout), writerOr(opts.Stderr))
	if cctx.Err() == context.DeadlineExceeded {
		return nil // the smoke run lasted its time
	}
	if err != nil {
		return report.Fail(fmt.Sprintf("%s: %v", filepath.Base(emu), err), "")
	}
	return nil
}

func (t *Tool) ISO(ctx context.Context, opts platforms.ISOOptions) error {
	_ = ctx
	_ = opts
	return report.Usage("nds has no ISO; nds build --sample gdir writes the .nds")
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
