package ps1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	guest "github.com/blazium-games/blazium-toolchain/guest/ps1"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const ID = "ps1"

// Tool is the PS1 implementation of platforms.Platform.
type Tool struct {
	Runner  execx.Runner
	Fetcher ZipFetcher
	Assets  []ZipAsset
	JSON    jsonGetter
	CLIURL  string // tests pin a zip URL and skip AppDistrib
}

func New() *Tool {
	return &Tool{Runner: execx.Host{}, Fetcher: fetch.HTTP{}}
}

func (t *Tool) fetcher() ZipFetcher {
	if t.Fetcher != nil {
		return t.Fetcher
	}
	return fetch.HTTP{}
}

func (t *Tool) compileAssets() []ZipAsset {
	if len(t.Assets) > 0 {
		return t.Assets
	}
	return defaultCompileAssets()
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
		Commands:    []string{"setup", "env", "status", "build", "run", "iso", "fmv"},
		Description: "PSn00bSDK guest build + pcsx-redux. Official Blazium PS1 toolchain (host: Windows and Linux).",
	}
}

func (t *Tool) Setup(ctx context.Context, opts platforms.SetupOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
	_ = ctx
	profile := opts.Profile
	if profile == "" {
		profile = "compile"
	}
	if profile != "compile" && profile != "dev" && profile != "iso" {
		return fmt.Errorf("%w: profile must be compile, dev, or iso", platforms.ErrUsage)
	}

	env, notes := t.discover(opts.Prefix)
	if !compileReady(env) && !opts.Offline {
		if err := t.ensureCompile(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
		env, notes = t.discover(opts.Prefix)
	}
	if opts.Offline && env["MIPS_GCC"] == "" && lookFile("mipsel-none-elf-gcc") == "" {
		return fmt.Errorf("%w: no mipsel-none-elf-gcc in vendor tree, env, or PATH and --offline set", platforms.ErrOffline)
	}
	if !opts.Offline && !compileReady(env) {
		return fmt.Errorf("%w: compile profile needs mipsel-none-elf-gcc, elf2x, and PSN00BSDK_LIBS", platforms.ErrMissingTool)
	}
	if needsDev(profile) && !opts.Offline {
		if err := t.ensureDev(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
		env, notes = t.discover(opts.Prefix)
	}
	if err := installGuestRuntime(opts.Prefix); err != nil {
		return err
	}

	if needsDev(profile) && !destReady(env) {
		if opts.Offline {
			return fmt.Errorf("%w: dev profile needs OPENBIOS and PCSX_EXE (vendor or drop --offline)", platforms.ErrOffline)
		}
		return fmt.Errorf("%w: dev profile needs OPENBIOS and PCSX_EXE after fetch", platforms.ErrMissingTool)
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
		if destReady(env) {
			fmt.Fprintln(opts.Stdout, "dev toolchain ready (OpenBIOS + pcsx-redux CLI)")
		} else if compileReady(env) {
			fmt.Fprintln(opts.Stdout, "compile toolchain ready")
		} else if !opts.Offline {
			fmt.Fprintln(opts.Stdout, "note: compile tools incomplete after fetch")
		}
	}
	return nil
}

func (t *Tool) Env(opts platforms.CommonOptions) (platforms.EnvMap, error) {
	st, err := cache.ReadState(opts.Prefix, ID)
	if err != nil {
		if os.IsNotExist(err) {
			env, _ := t.discover(opts.Prefix)
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
	profile := or(st.Profile, "compile")
	supported := HostSupported()
	out := map[string]any{
		"platform":       ID,
		"profile":        st.Profile,
		"env":            env,
		"components":     componentsForProfile(profile),
		"compile_ready":  compileReady(env) && supported,
		"dev_ready":      destReady(env) && supported,
		"ready":          supported && profileReady(profile, env),
		"guest_abi":      guest.CookABI,
		"guest_dir":      GuestDir(opts.Prefix),
		"host_os":        runtime.GOOS,
		"host_supported": supported,
	}
	return out, nil
}

func (t *Tool) Run(ctx context.Context, opts platforms.RunOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
	env, err := t.Env(opts.CommonOptions)
	if err != nil {
		return err
	}
	if !fileExists(env["PCSX_EXE"]) || !fileExists(env["OPENBIOS"]) {
		env, _ = t.discover(opts.Prefix)
	}
	pcsx := env["PCSX_EXE"]
	bios := env["OPENBIOS"]
	if !fileExists(pcsx) {
		return fmt.Errorf("%w: pcsx-redux CLI (run blazium-toolchain ps1 setup --profile dev)", platforms.ErrMissingTool)
	}
	if !fileExists(bios) {
		return fmt.Errorf("%w: OpenBIOS (run blazium-toolchain ps1 setup --profile dev)", platforms.ErrMissingTool)
	}
	if opts.ISO == "" && opts.Exe == "" {
		return fmt.Errorf("%w: run requires an exe or --iso", platforms.ErrUsage)
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"-no-ui", "-run", "-noupdate", "-safe", "-testmode", "-interpreter", "-softgpu", "-webserver", "-webserver-port", "8080", "-bios", bios, "-stdout"}
	pcdrv := opts.Pcdrv
	if pcdrv == "" && opts.Exe != "" {
		pcdrv = filepath.Dir(opts.Exe)
	}
	if pcdrv != "" {
		args = append(args, "-pcdrv", "-pcdrvbase", pcdrv)
	}
	if opts.ISO != "" {
		args = append(args, "-iso", opts.ISO, "-fastboot")
	} else {
		args = append(args, "-loadexe", opts.Exe)
	}
	extraPath := []string{filepath.Dir(pcsx)}
	err = t.runEnv(runCtx, pcsx, args, extraPath, map[string]string{"PCSX_EXE": pcsx, "OPENBIOS": bios}, opts.Stdout, opts.Stderr)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		if opts.Stdout != nil {
			fmt.Fprintf(opts.Stdout, "pcsx-redux smoke timeout after %s (process stopped)\n", timeout)
		}
		return nil
	}
	return err
}

func (t *Tool) ISO(ctx context.Context, opts platforms.ISOOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
	if opts.XML == "" {
		return fmt.Errorf("%w: iso requires --xml", platforms.ErrUsage)
	}
	r := t.runner()
	mk, err := execx.LookPrefersEnv(r, "MKPSXISO", "mkpsxiso")
	if err != nil {
		env, _ := t.discover(opts.Prefix)
		if env["MKPSXISO"] != "" && fileExists(env["MKPSXISO"]) {
			mk = env["MKPSXISO"]
			err = nil
		} else if p := walkNamed(cache.PlatformDir(opts.Prefix, ID), hostNames("mkpsxiso")...); p != "" {
			mk = p
			err = nil
		}
	}
	if err != nil || mk == "" {
		return fmt.Errorf("%w: mkpsxiso (GPL — install separately, spawn only)", platforms.ErrMissingTool)
	}
	args := []string{opts.XML}
	if opts.Out != "" {
		args = append(args, "-o", opts.Out)
	}
	return r.Run(ctx, mk, args, writerOrDiscard(opts.Stdout), writerOrDiscard(opts.Stderr))
}

func (t *Tool) discover(prefix string) (map[string]string, []string) {
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
	copyEnv("ELF2X")
	copyEnv("OPENBIOS")
	copyEnv("PCSX_EXE")
	copyEnv("MKPSXISO")

	plat := cache.PlatformDir(prefix, ID)

	if env["MIPS_GCC"] == "" {
		if p := findVendorFile(prefix, filepath.Join("gcc", "bin", "mipsel-none-elf-gcc")); p != "" {
			env["MIPS_GCC"] = p
			notes = append(notes, "vendored mipsel-none-elf-gcc")
		} else if p := walkNamed(plat, append(hostNames("mipsel-none-elf-gcc"))...); p != "" {
			env["MIPS_GCC"] = p
			notes = append(notes, "found mipsel-none-elf-gcc under prefix")
		} else if p := lookFile("mipsel-none-elf-gcc"); p != "" {
			env["MIPS_GCC"] = p
			notes = append(notes, "discovered mipsel-none-elf-gcc on PATH")
		}
	}
	if env["ELF2X"] == "" {
		if p := findVendorFile(prefix, filepath.Join("elf2x", "elf2x")); p != "" {
			env["ELF2X"] = p
			notes = append(notes, "vendored elf2x")
		} else if p := walkNamed(plat, append(hostNames("elf2x"))...); p != "" {
			env["ELF2X"] = p
			notes = append(notes, "found elf2x under prefix")
		}
		if env["ELF2X"] == "" {
			if p := lookFile("elf2x"); p != "" {
				env["ELF2X"] = p
				notes = append(notes, "discovered elf2x on PATH")
			}
		}
	}
	if env["PSN00BSDK_LIBS"] == "" {
		if p := findLibpsn00b(plat); p != "" {
			env["PSN00BSDK_LIBS"] = p
			notes = append(notes, "found libpsn00b under prefix")
		} else if p := findVendorDir(prefix, "psn00bsdk"); p != "" {
			if lib := filepath.Join(p, "lib", "libpsn00b"); dirExists(lib) {
				env["PSN00BSDK_LIBS"] = lib
			} else {
				env["PSN00BSDK_LIBS"] = p
			}
			notes = append(notes, "vendored psn00bsdk")
		}
	}
	if env["PSN00BSDK_TC"] == "" && env["MIPS_GCC"] != "" {
		env["PSN00BSDK_TC"] = filepath.Dir(filepath.Dir(env["MIPS_GCC"]))
	}
	if env["PCSX_EXE"] == "" {
		if p := findPCSX(filepath.Join(plat, "pcsx-redux"), plat); p != "" {
			env["PCSX_EXE"] = p
			notes = append(notes, "found pcsx-redux under prefix")
		} else if p := findVendorFile(prefix, filepath.Join("pcsx-redux", "pcsx-redux")); p != "" {
			env["PCSX_EXE"] = p
			notes = append(notes, "vendored pcsx-redux")
		} else if p := lookFile("pcsx-redux"); p != "" {
			env["PCSX_EXE"] = p
		}
	}
	if env["OPENBIOS"] == "" {
		canon := filepath.Join(plat, "openbios", "openbios.bin")
		if fileExists(canon) {
			env["OPENBIOS"] = absOr(canon)
			notes = append(notes, "cached OpenBIOS")
		} else if cand := findOpenBIOSFile(filepath.Join(plat, "openbios"), filepath.Join(plat, "pcsx-redux"), plat); cand != "" {
			env["OPENBIOS"] = cand
			notes = append(notes, "found OpenBIOS under prefix")
		} else if cand := findVendorFile(prefix, filepath.Join("openbios", "openbios.bin")); cand != "" {
			env["OPENBIOS"] = cand
			notes = append(notes, "vendored OpenBIOS")
		}
	}
	if env["MKPSXISO"] == "" {
		if p := findVendorFile(prefix, filepath.Join("mkpsxiso", "mkpsxiso")); p != "" {
			env["MKPSXISO"] = p
			notes = append(notes, "vendored mkpsxiso")
		} else if p := walkNamed(plat, hostNames("mkpsxiso")...); p != "" {
			env["MKPSXISO"] = p
			notes = append(notes, "found mkpsxiso under prefix")
		} else if p := lookFile("mkpsxiso"); p != "" {
			env["MKPSXISO"] = p
			notes = append(notes, "discovered mkpsxiso on PATH")
		}
	}
	return env, notes
}

func (t *Tool) ensureCompile(ctx context.Context, prefix string, log io.Writer) error {
	root := cache.PlatformDir(prefix, ID)
	for _, a := range t.compileAssets() {
		dest := filepath.Join(root, a.Dest)
		if compilePiecePresent(dest, a.ID) {
			continue
		}
		if log != nil {
			fmt.Fprintf(log, "fetching %s from %s\n", a.ID, a.URL)
		}
		if err := t.fetcher().FetchZip(ctx, a.URL, a.SHA256, dest, log); err != nil {
			return fmt.Errorf("%w: %s: %v", platforms.ErrMissingTool, a.ID, err)
		}
	}
	return nil
}

func compilePiecePresent(dest, id string) bool {
	switch id {
	case "mipsel-none-elf-gcc":
		return walkNamed(dest, hostNames("mipsel-none-elf-gcc")...) != ""
	case "psn00bsdk":
		return walkNamed(dest, hostNames("elf2x")...) != "" || walkDirNamed(dest, "libpsn00b") != ""
	case "cmake":
		return walkNamed(dest, hostNames("cmake")...) != ""
	case "ninja":
		return walkNamed(dest, hostNames("ninja")...) != ""
	default:
		return false
	}
}

func compileReady(env map[string]string) bool {
	return fileExists(env["MIPS_GCC"]) && fileExists(env["ELF2X"]) && dirExists(env["PSN00BSDK_LIBS"]) && dirExists(env["PSN00BSDK_TC"])
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
