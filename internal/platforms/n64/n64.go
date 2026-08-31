package n64

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	guest "github.com/blazium-games/blazium-toolchain/guest/n64"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/fetch"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

const ID = "n64"

// Tool is the N64 implementation of platforms.Platform.
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
		Name:        "Nintendo 64",
		Status:      platforms.StatusSupported,
		Commands:    []string{"setup", "env", "status", "build", "export-guest", "run", "rom"},
		Description: n64InfoDescription(),
	}
}

func n64InfoDescription() string {
	base := "Nintendo 64 guest is bundled in this CLI (libdragon preview + Ares/Project64). Product is .z64 (no ISO). Compile/run: Windows and Linux only."
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
	if profile != "compile" && profile != "dev" && profile != "rom" {
		return report.Usage("profile must be compile, dev, or rom")
	}

	env, notes := t.discover(opts.Prefix)
	if !opts.Offline {
		if err := t.ensureLibdragonSrc(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
		if err := t.ensureTiny3dSrc(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
	}
	if !compileReady(env) && !opts.Offline {
		if err := t.ensureCompile(ctx, opts.Prefix, opts.Stdout); err != nil {
			return err
		}
		env, notes = t.discover(opts.Prefix)
	}
	if opts.Offline && env["N64_GCC"] == "" && lookFile("mips64-elf-gcc") == "" {
		return report.Offline("no mips64-elf-gcc in prefix, env, or PATH", "drop --offline or run blazium-toolchain n64 setup --profile compile")
	}
	if !opts.Offline && !compileReady(env) {
		return report.Missing("n64 compile needs N64_INST (mips64-elf-gcc, n64.mk, libdragon.a)", "blazium-toolchain n64 setup --profile compile")
	}
	if err := installGuestRuntime(opts.Prefix); err != nil {
		return err
	}

	if needsDev(profile) {
		if !opts.Offline {
			if err := t.ensureDev(ctx, opts.Prefix, opts.Stdout); err != nil {
				return err
			}
		}
		env, notes = t.discover(opts.Prefix)
		if env["ARES_EXE"] == "" && env["PROJECT64_EXE"] == "" {
			notes = append(notes, "dev profile: no ARES_EXE or PROJECT64_EXE found (boot tests will skip that validator)")
		}
	}

	st := cache.State{
		Platform: ID,
		Profile:  profile,
		Env:      env,
		Notes:    append(notes, "never fetches a PIF/BIOS dump; never searches C:\\ultra"),
	}
	if err := cache.WriteState(opts.Prefix, ID, st); err != nil {
		return err
	}
	report.Setup(opts.Stdout, ID, profile, cache.PlatformDir(opts.Prefix, ID), env)
	if env["ARES_EXE"] == "" {
		report.Line(opts.Stdout, report.Note, "ARES_EXE unset; n64 run --emu ares will skip")
	}
	if env["PROJECT64_EXE"] == "" {
		report.Line(opts.Stdout, report.Note, "PROJECT64_EXE unset; n64 run --emu project64 will skip")
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
	fresh, _ := t.discover(opts.Prefix)
	for _, k := range []string{"N64_INST", "N64_GCC", "N64_MK", "ARES_EXE", "PROJECT64_EXE", "N64_MKDFS", "N64_TOOL", "N64_AUDIOCONV", "T3D_INST", "LIBDRAGON_SRC"} {
		if st.Env[k] == "" || !pathOK(st.Env[k]) {
			if fresh[k] != "" {
				st.Env[k] = fresh[k]
			}
		}
	}
	return st.Env, nil
}

func pathOK(p string) bool {
	return fileExists(p) || dirExists(p)
}

func (t *Tool) Status(opts platforms.CommonOptions) (map[string]any, error) {
	env, err := t.Env(opts)
	if err != nil {
		return nil, err
	}
	st, _ := cache.ReadState(opts.Prefix, ID)
	supported := HostSupported()
	aresOK := fileExists(env["ARES_EXE"])
	pj64OK := fileExists(env["PROJECT64_EXE"])
	t3d := or(env["T3D_INST"], tiny3dRoot(opts.Prefix))
	t3dOK := t3d != "" && fileExists(t3dLibPath(t3d))
	dso := walkNamed(env["N64_INST"], hostNames("n64dso")...)
	dsoOK := dsoToolsReady(env["N64_INST"])
	comp := compileReady(env) && supported
	out := map[string]any{
		"platform":         ID,
		"profile":          st.Profile,
		"env":              env,
		"compile_ready":    comp,
		"dev_ready":        comp && (aresOK || pj64OK),
		"run_ready":        comp && (aresOK || pj64OK),
		"ares_ready":       aresOK,
		"project64_ready":  pj64OK,
		"rom_ready":        comp && fileExists(or(env["N64_MKDFS"], walkNamed(env["N64_INST"], hostNames("mkdfs")...))),
		"ready":            comp,
		"guest_abi":        guest.CookABI,
		"guest_dir":        GuestDir(opts.Prefix),
		"host_os":          runtime.GOOS,
		"host_supported":   supported,
		"product":          "z64",
		"expansion_pak":    true,
		"libdragon_branch": "preview",
		"tiny3d":           t3d,
		"tiny3d_ready":     t3dOK,
		"dso":              dso,
		"dso_ready":        dsoOK,
	}
	return out, nil
}

func (t *Tool) Run(ctx context.Context, opts platforms.RunOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
	rom := strings.TrimSpace(opts.Exe)
	if rom == "" {
		rom = strings.TrimSpace(opts.ISO)
	}
	if rom == "" {
		return report.Usage("run requires a .z64 path")
	}
	if !fileExists(rom) {
		return report.Usage("rom not found: " + rom)
	}
	abs, err := filepath.Abs(rom)
	if err != nil {
		abs = rom
	}
	if err := verifyZ64(abs); err != nil {
		return err
	}

	env, err := t.Env(opts.CommonOptions)
	if err != nil {
		return err
	}
	env2, _ := t.discover(opts.Prefix)
	if env["ARES_EXE"] == "" {
		env["ARES_EXE"] = env2["ARES_EXE"]
	}
	if env["PROJECT64_EXE"] == "" {
		env["PROJECT64_EXE"] = env2["PROJECT64_EXE"]
	}

	emu := strings.ToLower(strings.TrimSpace(opts.Emu))
	if emu == "" {
		emu = strings.ToLower(strings.TrimSpace(settings.Current().N64.Emu))
	}
	if emu == "" {
		emu = "both"
	}
	if emu != "ares" && emu != "project64" && emu != "both" {
		return report.Usage("--emu must be ares, project64, or both")
	}
	if emu == "project64" && runtime.GOOS != "windows" {
		return report.Usage("Project64 is Windows only")
	}
	rdram := strings.TrimSpace(opts.Rdram)
	if rdram == "" {
		rdram = strings.TrimSpace(settings.Current().N64.Rdram)
	}
	if rdram != "" && rdram != "8" && rdram != "4" {
		return report.Usage("--rdram must be 8 or 4")
	}
	opts.Rdram = rdram

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = settings.Current().Smoke()
	}

	wantAres := emu == "ares" || emu == "both"
	wantPJ := emu == "project64" || emu == "both"
	ran := 0
	skipped := []string{}
	aresReady := wantAres && fileExists(env["ARES_EXE"])
	pj64Ready := false

	if wantAres {
		ares := env["ARES_EXE"]
		if !fileExists(ares) {
			skipped = append(skipped, "ares (binary missing)")
			report.Line(opts.Stdout, report.Skip, "ares  ARES_EXE missing")
		} else {
			if err := t.runOneEmu(ctx, "ares", ares, abs, timeout, opts); err != nil {
				return err
			}
			ran++
		}
	}
	if wantPJ && rdramIs4(opts.Rdram) {
		skipped = append(skipped, "project64 (4 MiB no-pak; unknown-ROM pin stays 8 MB)")
		if opts.Stdout != nil {
			report.Line(opts.Stdout, report.Skip, "project64  4 MiB no-pak; unknown-ROM pin stays 8 MB")
		}
	} else if wantPJ {
		pj := env["PROJECT64_EXE"]
		if runtime.GOOS != "windows" {
			skipped = append(skipped, "project64 (not Windows)")
			if opts.Stdout != nil {
				report.Line(opts.Stdout, report.Skip, "project64  not Windows")
			}
		} else if !fileExists(pj) {
			skipped = append(skipped, "project64 (binary missing)")
			if opts.Stdout != nil {
				report.Line(opts.Stdout, report.Skip, "project64  PROJECT64_EXE missing")
			}
		} else if _, err := ensurePj64TestProfile(pj); err != nil {
			skipped = append(skipped, "project64 (video plugin missing)")
			if opts.Stdout != nil {
				report.Line(opts.Stdout, report.Skip, pj64SkipPlugin)
			}
		} else {
			pj64Ready = true
			if err := t.runOneEmu(ctx, "project64", pj, abs, timeout, opts); err != nil {
				if pj64PluginInitFail(err) {
					skipped = append(skipped, "project64 (video plugin failed to init)")
					if opts.Stdout != nil {
						report.Line(opts.Stdout, report.Skip, pj64SkipInit)
					}
				} else {
					return err
				}
			} else {
				ran++
			}
		}
	}

	if opts.Stdout != nil && len(skipped) > 0 {
		report.Linef(opts.Stdout, report.Skip, "%s", strings.Join(skipped, ", "))
	}
	if emu == "both" && aresReady && pj64Ready && ran < 2 {
		return report.Fail("both Ares and Project64 are installed but one did not stay running (CI --emu both requires both)", "")
	}
	if ran == 0 {
		// One-click asks for a single host emu; skip+print is success. CI --emu both still fails if nothing booted.
		if emu != "both" && len(skipped) > 0 {
			return nil
		}
		if runtime.GOOS == "windows" {
			return report.Missing("neither Ares nor Project64 could boot", "set ARES_EXE and/or PROJECT64_EXE")
		}
		return report.Missing("Ares could not boot", "set ARES_EXE")
	}
	return nil
}

// aresEepromPath is the v148 default when Paths/Saves is unset: sibling of the ROM.
func aresEepromPath(rom string) string {
	ext := filepath.Ext(rom)
	if ext == "" {
		return rom + ".eeprom"
	}
	return rom[:len(rom)-len(ext)] + ".eeprom"
}

// pj64SaveDir is Project64's default Save folder next to the exe.
func pj64SaveDir(exe string) string {
	return filepath.Join(filepath.Dir(exe), "Save")
}

func (t *Tool) runOneEmu(ctx context.Context, name, exe, rom string, timeout time.Duration, opts platforms.RunOptions) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var args []string
	if name == "ares" {
		args = aresRunArgs(rom, !rdramIs4(opts.Rdram))
	} else {
		args = []string{rom}
	}
	err := t.runEnv(runCtx, exe, args, []string{filepath.Dir(exe)}, nil, opts.Stdout, opts.Stderr)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		if opts.Stdout != nil {
			report.Linef(opts.Stdout, report.Ready, "%s smoke timeout %s", name, timeout)
		}
		return nil
	}
	if err != nil {
		return err
	}
	return report.Fail(name+" exited before smoke timeout (ROM did not stay running)", "")
}

func (t *Tool) ISO(ctx context.Context, opts platforms.ISOOptions) error {
	_ = ctx
	_ = opts
	return report.Usage("n64 has no ISO/CUE; use n64 rom --dir TREE --out FILE.z64")
}

func (t *Tool) discover(prefix string) (map[string]string, []string) {
	env := map[string]string{}
	var notes []string
	copyEnv := func(key string) {
		if v := os.Getenv(key); v != "" && !forbiddenUltra(v) {
			env[key] = v
		}
	}
	copyEnv("N64_INST")
	copyEnv("N64_GCC")
	copyEnv("ARES_EXE")
	copyEnv("PROJECT64_EXE")
	copyEnv("T3D_INST")

	plat := cache.PlatformDir(prefix, ID)
	instCandidates := []string{
		env["N64_INST"],
		cache.InstDir(prefix, ID),
		toolchainRoot(cache.InstDir(prefix, ID)),
		filepath.Join(plat, "n64-inst"), // pre-unify cache
		toolchainRoot(filepath.Join(plat, "n64-inst")),
	}
	for _, root := range vendorRoots(prefix) {
		if root == plat {
			continue
		}
		instCandidates = append(instCandidates, filepath.Join(root, cache.InstRel), root)
	}
	for _, stuff := range siblingN64Stuff() {
		instCandidates = append(instCandidates, filepath.Join(stuff, cache.InstRel), filepath.Join(stuff, "n64-inst"))
	}

	if env["N64_INST"] == "" || !dirExists(env["N64_INST"]) || !isN64Inst(env["N64_INST"]) {
		for _, cand := range instCandidates {
			if isN64Inst(cand) {
				env["N64_INST"] = absOr(cand)
				notes = append(notes, "found N64_INST")
				break
			}
		}
	}
	if env["N64_INST"] != "" && forbiddenUltra(env["N64_INST"]) {
		env["N64_INST"] = ""
		notes = append(notes, "ignored C:\\ultra (official SDK is out of scope)")
	}

	if env["N64_GCC"] == "" && env["N64_INST"] != "" {
		if p := walkNamed(env["N64_INST"], hostNames("mips64-elf-gcc")...); p != "" {
			env["N64_GCC"] = p
		}
	}
	if env["N64_GCC"] == "" {
		if p := lookFile("mips64-elf-gcc"); p != "" {
			env["N64_GCC"] = p
			notes = append(notes, "discovered mips64-elf-gcc on PATH")
			if env["N64_INST"] == "" {
				// .../bin/mips64-elf-gcc → inst root
				env["N64_INST"] = absOr(filepath.Dir(filepath.Dir(p)))
			}
		}
	}
	if env["N64_INST"] != "" {
		mk := filepath.Join(env["N64_INST"], "include", "n64.mk")
		if fileExists(mk) {
			env["N64_MK"] = mk
		}
		if p := walkNamed(env["N64_INST"], hostNames("mkdfs")...); p != "" {
			env["N64_MKDFS"] = p
		}
		if p := walkNamed(env["N64_INST"], hostNames("n64tool")...); p != "" {
			env["N64_TOOL"] = p
		}
		if p := walkNamed(env["N64_INST"], hostNames("audioconv64")...); p != "" {
			env["N64_AUDIOCONV"] = p
		}
	}

	if env["ARES_EXE"] == "" {
		if p := walkNamed(plat, hostNames("ares")...); p != "" {
			env["ARES_EXE"] = p
		} else if p := lookFile("ares"); p != "" {
			env["ARES_EXE"] = p
		} else {
			for _, stuff := range siblingN64Stuff() {
				if p := walkNamed(filepath.Join(stuff, "ares"), hostNames("ares")...); p != "" {
					env["ARES_EXE"] = p
					notes = append(notes, "found sibling n64_stuff/ares")
					break
				}
			}
		}
	}
	if src := libdragonRoot(prefix); src != "" {
		env["LIBDRAGON_SRC"] = src
	}
	if env["T3D_INST"] == "" {
		if t3d := tiny3dRoot(prefix); t3d != "" {
			env["T3D_INST"] = t3d
			if siblingTiny3d() == t3d {
				notes = append(notes, "found sibling n64_stuff/tiny3d")
			} else {
				notes = append(notes, "found cached tiny3d source")
			}
		}
	}
	if env["PROJECT64_EXE"] == "" && runtime.GOOS == "windows" {
		if p := walkNamed(plat, hostNames("Project64")...); p != "" {
			env["PROJECT64_EXE"] = p
		} else if p := lookFile("Project64"); p != "" {
			env["PROJECT64_EXE"] = p
		} else {
			for _, stuff := range siblingN64Stuff() {
				if p := walkNamed(filepath.Join(stuff, "project64"), hostNames("Project64")...); p != "" {
					env["PROJECT64_EXE"] = p
					notes = append(notes, "found sibling n64_stuff/project64")
					break
				}
			}
		}
	}
	return env, notes
}

func isN64Inst(root string) bool {
	if !dirExists(root) || forbiddenUltra(root) {
		return false
	}
	gcc := walkNamed(root, hostNames("mips64-elf-gcc")...)
	mk := fileExists(filepath.Join(root, "include", "n64.mk"))
	return gcc != "" && mk
}

func (t *Tool) ensureCompile(ctx context.Context, prefix string, log io.Writer) error {
	url := officialToolchainURL()
	dest := cache.InstDir(prefix, ID)
	if url == "" {
		return report.Missing(fmt.Sprintf("no official mips64-elf toolchain for %s/%s", runtime.GOOS, runtime.GOARCH), "")
	}
	if walkNamed(dest, hostNames("mips64-elf-gcc")...) == "" {
		if log != nil {
			report.Linef(log, report.Fetching, "libdragon toolchain  %s", url)
		}
		if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
			if walkNamed(dest, hostNames("mips64-elf-gcc")...) == "" {
				return err
			}
		}
	}
	if err := t.ensureLibdragonSrc(ctx, prefix, log); err != nil {
		return err
	}
	if err := t.ensureTiny3dSrc(ctx, prefix, log); err != nil {
		return err
	}
	inst := toolchainRoot(dest)
	if err := t.ensureLibdragon(ctx, prefix, inst, log); err != nil {
		return err
	}
	return nil
}

func (t *Tool) ensureLibdragonSrc(ctx context.Context, prefix string, log io.Writer) error {
	return t.ensureFetchedSrc(ctx, officialLibdragonSrcURL(), cachedSrcDir(prefix, "libdragon"), "n64.mk", log, "libdragon preview")
}

func (t *Tool) ensureTiny3dSrc(ctx context.Context, prefix string, log io.Writer) error {
	return t.ensureFetchedSrc(ctx, officialTiny3dSrcURL(), cachedSrcDir(prefix, "tiny3d"), "t3d.mk", log, "Tiny3D")
}

func (t *Tool) ensureFetchedSrc(ctx context.Context, url, dest, marker string, log io.Writer, what string) error {
	if findSrcRoot(dest, marker) != "" {
		return nil
	}
	if url == "" {
		return report.Missing("no official "+what+" source URL", "")
	}
	if log != nil {
		report.Linef(log, report.Fetching, "%s  %s", what, url)
	}
	if err := t.fetcher().FetchZip(ctx, url, "", dest, log); err != nil {
		if findSrcRoot(dest, marker) == "" {
			return fmt.Errorf("%s: %w", what, err)
		}
	}
	if findSrcRoot(dest, marker) == "" {
		return report.Missing(fmt.Sprintf("%s archive extracted but %s not found in %s", what, marker, dest), "")
	}
	return nil
}

func (t *Tool) ensureLibdragon(ctx context.Context, prefix, inst string, log io.Writer) error {
	src := libdragonRoot(prefix)
	if fileExists(filepath.Join(inst, "include", "n64.mk")) &&
		fileExists(filepath.Join(inst, "mips64-elf", "lib", "libdragon.a")) {
		if src != "" && !fileExists(filepath.Join(inst, "bin", "n64sym.exe")) && !fileExists(filepath.Join(inst, "bin", "n64sym")) {
			cc := lookFile("gcc")
			_ = t.compileHostTools(ctx, src, inst, []string{filepath.Join(inst, "bin")}, map[string]string{"N64_INST": inst, "CC": cc}, log)
		}
		return nil
	}
	if src == "" {
		return report.Missing("libdragon preview source missing", "blazium-toolchain n64 setup --profile compile")
	}
	if log != nil {
		report.Linef(log, report.Installing, "libdragon preview  %s", inst)
	}
	mkSrc := filepath.Join(src, "n64.mk")
	mkDst := filepath.Join(inst, "include", "n64.mk")
	if fileExists(mkSrc) {
		if err := os.MkdirAll(filepath.Dir(mkDst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(mkSrc)
		if err != nil {
			return err
		}
		if err := os.WriteFile(mkDst, data, 0o644); err != nil {
			return err
		}
	}
	if fileExists(filepath.Join(inst, "mips64-elf", "lib", "libdragon.a")) {
		return nil
	}
	makeProg := filepath.Join(inst, "bin", "make")
	if runtime.GOOS == "windows" {
		makeProg += ".exe"
	}
	if !fileExists(makeProg) {
		makeProg = lookFile("make")
	}
	if makeProg == "" {
		if log != nil {
			report.Line(log, report.Note, "make not found; build libdragon preview with N64_INST set")
		}
		return nil
	}
	extraPath := []string{filepath.Join(inst, "bin")}
	if gcc := lookFile("gcc"); gcc != "" {
		extraPath = append(extraPath, filepath.Dir(gcc))
	}
	for _, p := range makeSearchPath() {
		if dirExists(p) {
			extraPath = append(extraPath, p)
		}
	}
	extraEnv := map[string]string{"N64_INST": inst, "LIBDRAGON_PREVIEW": "2"}
	if gcc := lookFile("gcc"); gcc != "" {
		extraEnv["CC"] = gcc
		if gxx := lookFile("g++"); gxx != "" {
			extraEnv["CXX"] = gxx
		} else {
			extraEnv["CXX"] = gcc
		}
	}
	for _, args := range [][]string{{"install-mk"}, {"libdragon"}, {"install"}} {
		if err := t.runEnvInDir(ctx, makeProg, src, args, extraPath, extraEnv, log, log); err != nil {
			return fmt.Errorf("libdragon make %s: %w", strings.Join(args, " "), err)
		}
	}
	if err := t.runEnvInDir(ctx, makeProg, src, []string{"tools", "tools-install"}, extraPath, extraEnv, log, log); err != nil {
		if log != nil {
			report.Line(log, report.Building, "n64tool mkdfs")
		}
		_ = t.compileHostTools(ctx, src, inst, extraPath, extraEnv, log)
	}
	return nil
}

func (t *Tool) compileHostTools(ctx context.Context, src, inst string, extraPath []string, extraEnv map[string]string, log io.Writer) error {
	gcc := extraEnv["CC"]
	if gcc == "" {
		gcc = lookFile("gcc")
	}
	if gcc == "" {
		return report.Missing("host gcc (MinGW) not found for n64tool", "install gcc and put it on PATH")
	}
	bin := filepath.Join(inst, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	tools := filepath.Join(src, "tools")
	type job struct {
		out  string
		srcs []string
	}
	jobs := []job{
		{"n64tool", []string{filepath.Join(tools, "n64tool.c")}},
		{"ed64romconfig", []string{filepath.Join(tools, "ed64romconfig.c")}},
		{"mkdfs", []string{filepath.Join(tools, "mkdfs", "mkdfs.c")}},
	}
	cflags := []string{"-O2", "-std=gnu11", "-I" + tools, "-I" + filepath.Join(src, "include"), "-Wno-error", "-static"}
	for _, j := range jobs {
		out := filepath.Join(bin, j.out)
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		if fileExists(out) {
			continue
		}
		args := append(append([]string{}, cflags...), "-o", out)
		args = append(args, j.srcs...)
		args = append(args, "-lntdll")
		if err := t.runEnv(ctx, gcc, args, extraPath, extraEnv, log, log); err != nil {
			return fmt.Errorf("%s: %w", j.out, err)
		}
	}
	return writeToolStubs(bin)
}

func dsoToolNames() []string {
	return []string{"n64dso", "n64dso-extern", "n64dso-msym"}
}

func dsoToolsReady(inst string) bool {
	if inst == "" {
		return false
	}
	for _, name := range dsoToolNames() {
		if walkNamed(inst, hostNames(name)...) == "" {
			return false
		}
	}
	return true
}

func (t *Tool) ensureDsoTools(ctx context.Context, extraPath []string, extraEnv map[string]string, opts platforms.BuildOptions) error {
	inst := extraEnv["N64_INST"]
	if dsoToolsReady(inst) {
		return nil
	}
	src := libdragonRoot(opts.Prefix)
	if src == "" {
		return report.Missing("n64dso not found", "blazium-toolchain n64 setup --profile compile")
	}
	if inst == "" {
		return report.Missing("n64dso needs N64_INST", "blazium-toolchain n64 setup --profile compile")
	}
	makeProg := ""
	if inst != "" {
		makeProg = walkNamed(inst, hostNames("make")...)
	}
	if makeProg == "" {
		makeProg = lookFile("make")
	}
	if makeProg == "" {
		makeProg = lookFile("mingw32-make")
	}
	if makeProg == "" {
		return report.Missing("make not found (needed to build n64dso)", "install make and put it on PATH")
	}
	unixBin := makeSearchPath()
	gcc := lookHostMingw("gcc")
	gxx := lookHostMingw("g++")
	if gxx == "" {
		gxx = gcc
	}
	if gcc != "" {
		extraPath = append(extraPath, filepath.Dir(gcc))
	}
	extraPath = append(unixBin, extraPath...)
	env := map[string]string{}
	for k, v := range extraEnv {
		env[k] = v
	}
	env["N64_INST"] = inst
	env["INSTALLDIR"] = inst
	env["LIBDRAGON_PREVIEW"] = "2"
	// Skip tools/Makefile Windows_NT pacman/date checks; host MinGW is enough.
	env["OS"] = "host"
	if gcc != "" {
		env["CC"] = gcc
	}
	if gxx != "" {
		env["CXX"] = gxx
	}
	tools := filepath.Join(src, "tools")
	if opts.Stdout != nil {
		report.Linef(opts.Stdout, report.Building, "n64dso tools  %s", tools)
	}
	args := []string{"n64dso", "n64dso-extern", "n64dso-msym", "n64dso-install", "n64dso-extern-install", "n64dso-msym-install"}
	if err := t.runEnvInDir(ctx, makeProg, tools, args, extraPath, env, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("n64dso tools: %w", err)
	}
	if dsoToolsReady(inst) {
		return nil
	}
	return report.Missing("n64dso tools missing after make", "blazium-toolchain n64 setup --profile compile")
}

func (t *Tool) ensureAudioconv(ctx context.Context, env map[string]string, stdout, stderr io.Writer) error {
	if p := strings.TrimSpace(env["N64_AUDIOCONV"]); p != "" && fileExists(p) {
		return nil
	}
	inst := env["N64_INST"]
	if inst != "" {
		if p := walkNamed(inst, hostNames("audioconv64")...); p != "" {
			env["N64_AUDIOCONV"] = p
			return nil
		}
	}
	src := or(env["LIBDRAGON_SRC"], siblingLibdragon())
	if src == "" && inst != "" {
		src = libdragonRoot(filepath.Dir(filepath.Dir(inst)))
	}
	if src == "" || inst == "" {
		return report.Missing("audioconv64 not found", "blazium-toolchain n64 setup --profile compile")
	}
	makeProg := filepath.Join(inst, "bin", "make")
	if runtime.GOOS == "windows" {
		makeProg += ".exe"
	}
	if !fileExists(makeProg) {
		makeProg = lookFile("make")
	}
	if makeProg == "" {
		return report.Missing("make not found (needed to build audioconv64)", "install make and put it on PATH")
	}
	extraPath := []string{filepath.Join(inst, "bin")}
	if gcc := lookFile("gcc"); gcc != "" {
		extraPath = append(extraPath, filepath.Dir(gcc))
	}
	for _, p := range makeSearchPath() {
		if dirExists(p) {
			extraPath = append(extraPath, p)
		}
	}
	extraEnv := map[string]string{"N64_INST": inst, "INSTALLDIR": inst, "LIBDRAGON_PREVIEW": "2"}
	if gcc := lookFile("gcc"); gcc != "" {
		extraEnv["CC"] = gcc
	}
	if gxx := lookFile("g++"); gxx != "" {
		extraEnv["CXX"] = gxx
	} else if gcc := lookFile("gcc"); gcc != "" {
		extraEnv["CXX"] = gcc
	}
	tools := filepath.Join(src, "tools")
	if stdout != nil {
		report.Linef(stdout, report.Building, "audioconv64  %s", tools)
	}
	if err := t.runEnvInDir(ctx, makeProg, tools, []string{"audioconv64", "audioconv64-install"}, extraPath, extraEnv, stdout, stderr); err != nil {
		return fmt.Errorf("audioconv64: %w", err)
	}
	if p := walkNamed(inst, hostNames("audioconv64")...); p != "" {
		env["N64_AUDIOCONV"] = p
		return nil
	}
	return report.Missing("audioconv64 missing after install", "blazium-toolchain n64 setup --profile compile")
}

func writeToolStubs(bin string) error {
	// n64sym / n64elfcompress are large; hello ROM only needs a .sym file and an in-place elf.
	stubs := map[string]string{
		"n64sym.c": `#include <stdio.h>
int main(int argc, char **argv) {
  const char *out = argc > 0 ? argv[argc-1] : "out.sym";
  FILE *f = fopen(out, "wb");
  if (!f) return 1;
  fclose(f);
  return 0;
}
`,
		"n64elfcompress.c": `#include <stdio.h>
int main(int argc, char **argv) { (void)argc; (void)argv; return 0; }
`,
	}
	gcc := lookFile("gcc")
	if gcc == "" {
		return fmt.Errorf("no host gcc for n64sym/n64elfcompress stubs")
	}
	for name, src := range stubs {
		base := strings.TrimSuffix(name, ".c")
		out := filepath.Join(bin, base)
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		if fileExists(out) {
			continue
		}
		cpath := filepath.Join(bin, name)
		if err := os.WriteFile(cpath, []byte(src), 0o644); err != nil {
			return err
		}
		cmd := exec.Command(gcc, "-O2", "-static", "-o", out, cpath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("stub %s: %w", base, err)
		}
	}
	return nil
}

func (t *Tool) runEnvInDir(ctx context.Context, name, dir string, args, extraPath []string, extraEnv map[string]string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = writerOrDiscard(stdout)
	cmd.Stderr = writerOrDiscard(stderr)
	env := os.Environ()
	if len(extraPath) > 0 {
		joined := strings.Join(extraPath, string(os.PathListSeparator))
		replaced := false
		for i, kv := range env {
			eq := strings.IndexByte(kv, '=')
			if eq > 0 && strings.EqualFold(kv[:eq], "PATH") {
				env[i] = kv[:eq+1] + joined + string(os.PathListSeparator) + kv[eq+1:]
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, "PATH="+joined)
		}
	}
	for k, v := range extraEnv {
		if k == "" {
			continue
		}
		prefix := k + "="
		found := false
		for i, kv := range env {
			if strings.HasPrefix(kv, prefix) {
				env[i] = prefix + v
				found = true
				break
			}
		}
		if !found {
			env = append(env, prefix+v)
		}
	}
	cmd.Env = env
	return cmd.Run()
}

func officialToolchainURL() string {
	return officialToolchainURLFor(runtime.GOOS, runtime.GOARCH)
}

func officialToolchainURLFor(goos, goarch string) string {
	f := settings.Current().N64.Fetch
	base := f.ToolchainBase
	switch goos {
	case "windows":
		if f.ToolchainWindows == "" {
			return ""
		}
		return base + f.ToolchainWindows
	case "linux":
		switch goarch {
		case "amd64":
			if f.ToolchainLinux == "" {
				return ""
			}
			return base + f.ToolchainLinux
		case "arm64":
			if f.ToolchainLinuxARM == "" {
				return ""
			}
			return base + f.ToolchainLinuxARM
		}
	}
	return ""
}

func officialLibdragonSrcURL() string {
	return settings.Current().N64.Fetch.Libdragon
}

func officialTiny3dSrcURL() string {
	return settings.Current().N64.Fetch.Tiny3D
}

func makeSearchPath() []string {
	return settings.Current().N64.MakePath
}

func compileReady(env map[string]string) bool {
	if forbiddenUltra(env["N64_INST"]) {
		return false
	}
	if !fileExists(env["N64_GCC"]) && walkNamed(env["N64_INST"], hostNames("mips64-elf-gcc")...) == "" {
		return false
	}
	if !fileExists(filepath.Join(env["N64_INST"], "include", "n64.mk")) {
		return false
	}
	lib := filepath.Join(env["N64_INST"], "mips64-elf", "lib", "libdragon.a")
	if !fileExists(lib) {
		lib = walkNamed(filepath.Join(env["N64_INST"], "mips64-elf", "lib"), "libdragon.a")
	}
	return fileExists(lib)
}

func needsDev(profile string) bool {
	return profile == "dev" || profile == "rom"
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
