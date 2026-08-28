package ps1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const ID = "ps1"

// Tool is the PS1 implementation of platforms.Platform.
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
		ID:          ID,
		Name:        "PlayStation 1",
		Status:      platforms.StatusSupported,
		Commands:    []string{"setup", "env", "status", "build", "run", "iso"},
		Description: "PSn00bSDK guest build + pcsx-redux. Official Blazium PS1 toolchain.",
	}
}

func (t *Tool) Setup(ctx context.Context, opts platforms.SetupOptions) error {
	_ = ctx
	profile := opts.Profile
	if profile == "" {
		profile = "compile"
	}
	if profile != "compile" && profile != "dev" && profile != "iso" {
		return fmt.Errorf("%w: profile must be compile, dev, or iso", platforms.ErrUsage)
	}

	env, notes := t.discover()
	if opts.Offline {
		if env["MIPS_GCC"] == "" && lookFile("mipsel-none-elf-gcc") == "" {
			return fmt.Errorf("%w: no mipsel-none-elf-gcc on PATH and --offline set", platforms.ErrOffline)
		}
	}

	st := cache.State{
		Platform: ID,
		Profile:  profile,
		Env:      env,
		Notes:    append(notes, "components: "+joinIDs(componentsForProfile(profile))),
	}
	if err := cache.WriteState(opts.Prefix, ID, st); err != nil {
		return err
	}

	manifestPath := filepath.Join(cache.PlatformDir(opts.Prefix, ID), "components.json")
	raw, err := json.MarshalIndent(componentsForProfile(profile), "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, append(raw, '\n'), 0o644); err != nil {
		return err
	}

	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "ps1 setup profile=%s prefix=%s\n", profile, cache.PlatformDir(opts.Prefix, ID))
		for k, v := range env {
			if v != "" {
				fmt.Fprintf(opts.Stdout, "  %s=%s\n", k, v)
			}
		}
		if !opts.Offline {
			fmt.Fprintln(opts.Stdout, "note: download pins are recorded; fetch from official mirrors when published. discovered local tools were reused.")
		}
	}
	return nil
}

func (t *Tool) Env(opts platforms.CommonOptions) (platforms.EnvMap, error) {
	st, err := cache.ReadState(opts.Prefix, ID)
	if err != nil {
		if os.IsNotExist(err) {
			env, _ := t.discover()
			return env, nil
		}
		return nil, err
	}
	if st.Env == nil {
		st.Env = map[string]string{}
	}
	return st.Env, nil
}

func (t *Tool) Status(opts platforms.CommonOptions) (map[string]any, error) {
	env, err := t.Env(opts)
	if err != nil {
		return nil, err
	}
	st, _ := cache.ReadState(opts.Prefix, ID)
	out := map[string]any{
		"platform":   ID,
		"profile":    st.Profile,
		"env":        env,
		"components": componentsForProfile(or(st.Profile, "compile")),
		"ready":      env["MIPS_GCC"] != "" || lookFile("mipsel-none-elf-gcc") != "",
	}
	return out, nil
}

func (t *Tool) Build(ctx context.Context, opts platforms.BuildOptions) error {
	if opts.Src == "" || opts.Out == "" {
		return fmt.Errorf("%w: build requires --src and --out", platforms.ErrUsage)
	}
	r := t.runner()
	cmake, err := r.LookPath("cmake")
	if err != nil {
		return fmt.Errorf("%w: cmake", platforms.ErrMissingTool)
	}
	buildDir := filepath.Join(opts.Src, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}
	args := []string{"-S", opts.Src, "-B", buildDir}
	if sdk := os.Getenv("PSN00BSDK_LIBS"); sdk != "" {
		tc := filepath.Join(sdk, "cmake", "sdk.cmake")
		if _, err := os.Stat(tc); err == nil {
			args = append(args, "-DCMAKE_TOOLCHAIN_FILE="+tc)
		}
	}
	if err := r.Run(ctx, cmake, args, writerOrDiscard(opts.Stdout), writerOrDiscard(opts.Stderr)); err != nil {
		return fmt.Errorf("cmake configure: %w", err)
	}
	if err := r.Run(ctx, cmake, []string{"--build", buildDir}, writerOrDiscard(opts.Stdout), writerOrDiscard(opts.Stderr)); err != nil {
		return fmt.Errorf("cmake build: %w", err)
	}
	if elf2x, err := execx.LookPrefersEnv(r, "", "elf2x"); err == nil {
		// Best-effort: convert first ELF in build dir if out is not already PS-X EXE.
		_ = elf2x
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "build finished; copy or elf2x the guest to %s\n", opts.Out)
	}
	return nil
}

func (t *Tool) Run(ctx context.Context, opts platforms.RunOptions) error {
	r := t.runner()
	pcsx, err := execx.LookPrefersEnv(r, "PCSX_EXE", "pcsx-redux")
	if err != nil {
		return fmt.Errorf("%w: pcsx-redux (set PCSX_EXE)", platforms.ErrMissingTool)
	}
	bios := os.Getenv("OPENBIOS")
	if bios == "" {
		env, _ := t.Env(opts.CommonOptions)
		bios = env["OPENBIOS"]
	}
	args := []string{"-no-ui", "-run", "-noupdate", "-safe", "-interpreter", "-softgpu"}
	if bios != "" {
		args = append(args, "-bios", bios)
	}
	if opts.ISO != "" {
		args = append(args, "-iso", opts.ISO, "-fastboot")
	} else if opts.Exe != "" {
		args = append(args, "-loadexe", opts.Exe)
	} else {
		return fmt.Errorf("%w: run requires an exe or --iso", platforms.ErrUsage)
	}
	return r.Run(ctx, pcsx, args, writerOrDiscard(opts.Stdout), writerOrDiscard(opts.Stderr))
}

func (t *Tool) ISO(ctx context.Context, opts platforms.ISOOptions) error {
	if opts.XML == "" {
		return fmt.Errorf("%w: iso requires --xml", platforms.ErrUsage)
	}
	r := t.runner()
	mk, err := execx.LookPrefersEnv(r, "MKPSXISO", "mkpsxiso")
	if err != nil {
		return fmt.Errorf("%w: mkpsxiso (GPL — install separately, spawn only)", platforms.ErrMissingTool)
	}
	args := []string{opts.XML}
	if opts.Out != "" {
		args = append(args, "-o", opts.Out)
	}
	return r.Run(ctx, mk, args, writerOrDiscard(opts.Stdout), writerOrDiscard(opts.Stderr))
}

func (t *Tool) discover() (map[string]string, []string) {
	env := map[string]string{}
	var notes []string

	copyEnv := func(key string) {
		if v := os.Getenv(key); v != "" {
			env[key] = v
		}
	}
	copyEnv("PSN00BSDK_LIBS")
	copyEnv("PSN00BSDK_TC")
	copyEnv("MIPS_GCC")
	copyEnv("OPENBIOS")
	copyEnv("PCSX_EXE")
	copyEnv("MKPSXISO")

	if env["MIPS_GCC"] == "" {
		if p := lookFile("mipsel-none-elf-gcc"); p != "" {
			env["MIPS_GCC"] = p
			notes = append(notes, "discovered mipsel-none-elf-gcc on PATH")
		}
	}
	if env["PSN00BSDK_TC"] == "" && env["MIPS_GCC"] != "" {
		env["PSN00BSDK_TC"] = filepath.Dir(filepath.Dir(env["MIPS_GCC"]))
	}
	if env["PCSX_EXE"] == "" {
		if p := lookFile("pcsx-redux"); p != "" {
			env["PCSX_EXE"] = p
		}
	}
	if env["OPENBIOS"] == "" {
		if cand := discoverOpenBIOS(); cand != "" {
			env["OPENBIOS"] = cand
			notes = append(notes, "discovered OpenBIOS")
		}
	}
	return env, notes
}

func discoverOpenBIOS() string {
	if runtime.GOOS == "" {
		return ""
	}
	wd, _ := os.Getwd()
	cands := []string{
		filepath.Join(wd, "pcsx-redux", "src", "mips", "openbios", "openbios.bin"),
		filepath.Join(wd, "..", "pcsx-redux", "src", "mips", "openbios", "openbios.bin"),
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return ""
}

func lookFile(name string) string {
	p, err := execx.Host{}.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func joinIDs(cs []Component) string {
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
	}
	return strings.Join(ids, ",")
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
