package ps2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	guest "github.com/blazium-games/blazium-toolchain/guest/ps2"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/iso"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const ID = "ps2"

// Tool is the PS2 implementation of platforms.Platform.
type Tool struct {
	Runner  execx.Runner
	Fetcher zipFetcher
}

type zipFetcher interface {
	FetchZip(ctx context.Context, url, sha256, destDir string, log io.Writer) error
}

func New() *Tool {
	return &Tool{Runner: execx.Host{}, Fetcher: fetch.HTTP{}}
}

func (t *Tool) runner() execx.Runner {
	if t.Runner == nil {
		return execx.Host{}
	}
	return t.Runner
}

func (t *Tool) fetcher() zipFetcher {
	if t.Fetcher != nil {
		return t.Fetcher
	}
	return fetch.HTTP{}
}

func (t *Tool) Info() platforms.Info {
	return platforms.Info{
		ID:          ID,
		Name:        "PlayStation 2",
		Status:      platforms.StatusSupported,
		Commands:    []string{"setup", "env", "status", "build", "export-guest", "run", "iso", "elf-info", "chd"},
		Description: ps2InfoDescription(),
	}
}

func ps2InfoDescription() string {
	base := "PlayStation 2 EE guest is bundled in this CLI (ps2sdk graph/draw/dma/packet + PCSX2). Compile/run/iso: Windows and Linux only."
	if HostSupported() {
		return base
	}
	return base + " This host cannot compile or run; env/status/export-guest still work."
}

func (t *Tool) Setup(ctx context.Context, opts platforms.SetupOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
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
	if opts.Offline && env["EE_GCC"] == "" && lookFile("mips64r5900el-ps2-elf-gcc") == "" {
		return fmt.Errorf("%w: no mips64r5900el-ps2-elf-gcc in vendor tree, env, or PATH and --offline set", platforms.ErrOffline)
	}
	if !opts.Offline && !compileReady(env) {
		return fmt.Errorf("%w: compile profile needs EE_GCC, PS2SDK, and PS2DEV", platforms.ErrMissingTool)
	}
	if err := ensureWindowsHostDLLs(env["EE_GCC"], opts.Stdout); err != nil {
		return err
	}
	if err := installGuestRuntime(opts.Prefix); err != nil {
		return err
	}

	if needsDev(profile) {
		env, notes = t.discover(opts.Prefix)
		if env["PCSX2_EXE"] == "" && !opts.Offline {
			return fmt.Errorf("%w: dev profile needs PCSX2_EXE (install pcsx2-qt or set PCSX2_EXE). BIOS is never fetched", platforms.ErrMissingTool)
		}
	}

	st := cache.State{
		Platform: ID,
		Profile:  profile,
		Env:      env,
		Notes:    append(notes, "never fetches a PS2 BIOS"),
	}
	if err := cache.WriteState(opts.Prefix, ID, st); err != nil {
		return err
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "ps2 setup profile=%s prefix=%s\n", profile, cache.PlatformDir(opts.Prefix, ID))
		for k, v := range env {
			if v != "" {
				fmt.Fprintf(opts.Stdout, "  %s=%s\n", k, v)
			}
		}
		if compileReady(env) {
			fmt.Fprintln(opts.Stdout, "compile toolchain ready")
		}
		if env["PS2_BIOS_DIR"] == "" {
			fmt.Fprintln(opts.Stdout, "note: PS2_BIOS_DIR unset — ps2 run requires a user-supplied BIOS")
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
	biosOK := biosReady(env)
	out := map[string]any{
		"platform":       ID,
		"profile":        st.Profile,
		"env":            env,
		"compile_ready":  compileReady(env) && supported,
		"dev_ready":      compileReady(env) && fileExists(env["PCSX2_EXE"]) && supported,
		"run_ready":      compileReady(env) && fileExists(env["PCSX2_EXE"]) && biosOK && supported,
		"ready":          supported && compileReady(env),
		"guest_abi":      guest.CookABI,
		"guest_dir":      GuestDir(opts.Prefix),
		"host_os":        runtime.GOOS,
		"host_supported": supported,
		"bios_ready":     biosOK,
	}
	_ = profile
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
	if !fileExists(env["PCSX2_EXE"]) {
		env, _ = t.discover(opts.Prefix)
	}
	pcsx := env["PCSX2_EXE"]
	if !fileExists(pcsx) {
		return fmt.Errorf("%w: pcsx2-qt (set PCSX2_EXE or run blazium-toolchain ps2 setup --profile dev)", platforms.ErrMissingTool)
	}
	if !biosReady(env) {
		return fmt.Errorf("%w: PS2 BIOS not found (set PS2_BIOS_DIR to a folder with a legal dump; never committed)", platforms.ErrMissingTool)
	}
	if opts.ISO == "" && opts.Exe == "" {
		return fmt.Errorf("%w: run requires an elf or --iso", platforms.ErrUsage)
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"-batch"}
	if !opts.UI {
		args = append([]string{"-nogui"}, args...)
	}
	if opts.ISO != "" {
		args = append(args, opts.ISO)
	} else {
		args = append(args, "-elf", opts.Exe)
		hostDir := hostFsDir(opts.Exe, opts.ISO, opts.Pcdrv)
		if hostDir != "" {
			prev, err := os.Getwd()
			if err == nil {
				if chdirErr := os.Chdir(hostDir); chdirErr == nil {
					defer func() { _ = os.Chdir(prev) }()
					if opts.Stdout != nil {
						fmt.Fprintf(opts.Stdout, "HostFs cwd %s (host: resolves to ELF dir)\n", hostDir)
					}
				}
			}
		}
	}
	err = t.runEnv(runCtx, pcsx, args, []string{filepath.Dir(pcsx)}, map[string]string{"PCSX2_EXE": pcsx}, opts.Stdout, opts.Stderr)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		if opts.Stdout != nil {
			fmt.Fprintf(opts.Stdout, "PCSX2 smoke timeout after %s (process stopped)\n", timeout)
		}
		return nil
	}
	return err
}

func (t *Tool) ISO(ctx context.Context, opts platforms.ISOOptions) error {
	_ = ctx
	if err := requireHost(); err != nil {
		return err
	}
	if opts.Dir == "" || opts.Out == "" {
		return fmt.Errorf("%w: iso requires --dir and --out", platforms.ErrUsage)
	}
	if err := ensureSystemCNF(opts.Dir); err != nil {
		return err
	}
	vol := opts.VolumeID
	if vol == "" {
		vol = "BLAZIUM2"
	}
	if err := iso.WriteDataISO(opts.Dir, opts.Out, vol); err != nil {
		return err
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "wrote ISO9660 %s\n", opts.Out)
	}
	return nil
}

func (t *Tool) discover(prefix string) (map[string]string, []string) {
	env := map[string]string{}
	var notes []string
	copyEnv := func(key string) {
		if v := os.Getenv(key); v != "" {
			env[key] = v
		}
	}
	copyEnv("PS2SDK")
	copyEnv("PS2DEV")
	copyEnv("EE_GCC")
	copyEnv("IOP_GCC")
	copyEnv("PCSX2_EXE")
	copyEnv("PS2_BIOS_DIR")

	plat := cache.PlatformDir(prefix, ID)

	if env["PS2SDK"] == "" || !dirExists(env["PS2SDK"]) {
		if p := findSDKRoot(plat); p != "" {
			env["PS2SDK"] = p
			notes = append(notes, "found ps2sdk under prefix")
		} else {
			for _, stuff := range siblingPS2Stuff() {
				if p := findSDKRoot(stuff); p != "" {
					env["PS2SDK"] = p
					notes = append(notes, "found sibling ps2_stuff/ps2sdk")
					break
				}
			}
		}
	}
	if env["PS2DEV"] == "" {
		if env["PS2SDK"] != "" {
			parent := filepath.Dir(env["PS2SDK"])
			if walkNamed(parent, hostNames("mips64r5900el-ps2-elf-gcc")...) != "" {
				env["PS2DEV"] = absOr(parent)
				notes = append(notes, "PS2DEV inferred from PS2SDK parent")
			}
		}
		if env["PS2DEV"] == "" {
			if p := os.Getenv("PS2DEV"); p != "" && dirExists(p) {
				env["PS2DEV"] = p
			}
		}
		if env["PS2DEV"] == "" && dirExists(filepath.Join(plat, "ps2dev")) {
			env["PS2DEV"] = absOr(filepath.Join(plat, "ps2dev"))
		}
	}
	if env["EE_GCC"] == "" {
		if env["PS2DEV"] != "" {
			if p := walkNamed(filepath.Join(env["PS2DEV"], "ee"), hostNames("mips64r5900el-ps2-elf-gcc")...); p != "" {
				env["EE_GCC"] = p
			}
		}
		if env["EE_GCC"] == "" {
			if p := walkNamed(plat, hostNames("mips64r5900el-ps2-elf-gcc")...); p != "" {
				env["EE_GCC"] = p
				notes = append(notes, "found EE gcc under prefix")
			} else if p := lookFile("mips64r5900el-ps2-elf-gcc"); p != "" {
				env["EE_GCC"] = p
				notes = append(notes, "discovered EE gcc on PATH")
			}
		}
	}
	if env["IOP_GCC"] == "" {
		if env["PS2DEV"] != "" {
			if p := walkNamed(filepath.Join(env["PS2DEV"], "iop"), hostNames("mipsel-none-elf-gcc")...); p != "" {
				env["IOP_GCC"] = p
			}
		}
		if env["IOP_GCC"] == "" {
			if p := lookFile("mipsel-none-elf-gcc"); p != "" {
				env["IOP_GCC"] = p
			}
		}
	}
	if env["PCSX2_EXE"] == "" {
		if p := walkNamed(plat, hostNames("pcsx2-qt")...); p != "" {
			env["PCSX2_EXE"] = p
		} else if p := lookFile("pcsx2-qt"); p != "" {
			env["PCSX2_EXE"] = p
		} else if p := lookFile("pcsx2"); p != "" {
			env["PCSX2_EXE"] = p
		}
	}
	if env["PS2_BIOS_DIR"] == "" {
		canon := filepath.Join(plat, "bios")
		if biosReady(map[string]string{"PS2_BIOS_DIR": canon}) {
			env["PS2_BIOS_DIR"] = absOr(canon)
		}
	}
	return env, notes
}

func (t *Tool) ensureCompile(ctx context.Context, prefix string, log io.Writer) error {
	url := officialTarballURL()
	if url == "" {
		return nil
	}
	dest := filepath.Join(cache.PlatformDir(prefix, ID), "ps2dev")
	if compileTreeComplete(dest) {
		return ensureWindowsHostDLLs(walkNamed(dest, hostNames("mips64r5900el-ps2-elf-gcc")...), log)
	}
	if walkNamed(dest, hostNames("mips64r5900el-ps2-elf-gcc")...) != "" {
		if log != nil {
			fmt.Fprintf(log, "ps2dev tree is incomplete (missing cc1); re-extracting\n")
		}
		_ = os.RemoveAll(dest)
	}
	if log != nil {
		fmt.Fprintf(log, "fetching ps2dev release from %s\n", url)
	}
	if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
		if compileTreeComplete(dest) {
			if log != nil {
				fmt.Fprintf(log, "extract reported %v; compile tools are present, continuing\n", err)
			}
			return ensureWindowsHostDLLs(walkNamed(dest, hostNames("mips64r5900el-ps2-elf-gcc")...), log)
		}
		return err
	}
	return ensureWindowsHostDLLs(walkNamed(dest, hostNames("mips64r5900el-ps2-elf-gcc")...), log)
}

func compileTreeComplete(dest string) bool {
	if walkNamed(dest, hostNames("mips64r5900el-ps2-elf-gcc")...) == "" {
		return false
	}
	if findSDKRoot(dest) == "" {
		return false
	}
	return walkNamed(dest, hostNames("cc1")...) != ""
}

func officialTarballURL() string {
	return officialTarballURLFor(runtime.GOOS)
}

func officialTarballURLFor(goos string) string {
	switch goos {
	case "windows":
		return "https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-windows-latest.tar.gz"
	case "linux":
		return "https://github.com/ps2dev/ps2dev/releases/download/latest/ps2dev-ubuntu-latest.tar.gz"
	default:
		return ""
	}
}

func compileReady(env map[string]string) bool {
	if !fileExists(env["EE_GCC"]) || !dirExists(env["PS2SDK"]) || !dirExists(filepath.Join(env["PS2SDK"], "ee", "include")) {
		return false
	}
	root := env["PS2DEV"]
	if root == "" {
		root = filepath.Dir(env["EE_GCC"])
	}
	return walkNamed(root, hostNames("cc1")...) != "" || walkNamed(filepath.Dir(env["EE_GCC"]), hostNames("cc1")...) != ""
}

func needsDev(profile string) bool {
	return profile == "dev" || profile == "iso"
}

func biosReady(env map[string]string) bool {
	dir := env["PS2_BIOS_DIR"]
	if !dirExists(dir) {
		return false
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := strings.ToLower(e.Name())
		if strings.HasSuffix(n, ".bin") || strings.HasSuffix(n, ".rom") {
			return true
		}
	}
	return false
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// hostFsDir is the folder PCSX2 host: should resolve to for ELF runs (not ISO).
func hostFsDir(exe, iso, override string) string {
	if strings.TrimSpace(iso) != "" {
		return ""
	}
	if strings.TrimSpace(override) != "" {
		return override
	}
	if strings.TrimSpace(exe) == "" {
		return ""
	}
	return filepath.Dir(exe)
}

func ensureSystemCNF(dir string) error {
	cnf := filepath.Join(dir, "SYSTEM.CNF")
	if fileExists(cnf) {
		return nil
	}
	elfName := "GAME.ELF"
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.EqualFold(filepath.Ext(e.Name()), ".elf") {
			elfName = strings.ToUpper(e.Name())
			break
		}
	}
	body := fmt.Sprintf("BOOT2 = cdrom0:\\%s;1\nVER = 1.00\nVMODE = NTSC\n", elfName)
	return os.WriteFile(cnf, []byte(body), 0o644)
}
