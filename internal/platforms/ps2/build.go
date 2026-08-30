package ps2

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	guest "github.com/blazium-games/blazium-toolchain/guest/ps2"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const elfMagic = "\x7fELF"

// GuestDir is <prefix>/ps2/guest/runtime — the bundled MIT stub.
func GuestDir(prefix string) string {
	return filepath.Join(cache.PlatformDir(prefix, ID), "guest", "runtime")
}

func installGuestRuntime(prefix string) error {
	return guest.Install(GuestDir(prefix))
}

type envRunner interface {
	RunEnv(ctx context.Context, name string, args []string, extraPath []string, extraEnv map[string]string, stdout, stderr io.Writer) error
}

func (t *Tool) Build(ctx context.Context, opts platforms.BuildOptions) error {
	if err := requireHost(); err != nil {
		return err
	}
	env, err := t.compileEnv(opts.CommonOptions)
	if err != nil {
		return err
	}
	if !compileReady(env) {
		return fmt.Errorf("%w: run blazium-toolchain ps2 setup --profile compile first", platforms.ErrMissingTool)
	}

	src := opts.Src
	sample := strings.ToLower(strings.TrimSpace(opts.Sample))
	overlay := strings.TrimSpace(opts.Overlay)
	if overlay == "" {
		overlay = strings.TrimSpace(os.Getenv("BLAZIUM_PS2_OVERLAY"))
	}
	if src != "" && !fileExists(filepath.Join(src, "CMakeLists.txt")) && !fileExists(filepath.Join(src, "Makefile")) {
		if overlay == "" {
			overlay = src
		}
		src = ""
	}
	if src == "" && sample != "" {
		src, err = resolveSample(env, sample)
		if err != nil {
			return err
		}
	}
	needWork := (overlay != "" && sample == "") || hasCookSlices(opts)
	if needWork && sample == "" {
		merge := filepath.Join(cache.PlatformDir(opts.Prefix, ID), "work", "guest-src")
		if err := os.RemoveAll(merge); err != nil {
			return err
		}
		if src != "" {
			if err := guest.Overlay(merge, src); err != nil {
				return err
			}
		} else {
			if err := guest.Install(merge); err != nil {
				return err
			}
		}
		if overlay != "" {
			if err := guest.Overlay(merge, overlay); err != nil {
				return err
			}
		}
		if err := stageCookEmbed(merge, opts); err != nil {
			return err
		}
		src = merge
	}
	if src == "" {
		if err := installGuestRuntime(opts.Prefix); err != nil {
			return err
		}
		src = GuestDir(opts.Prefix)
		if err := stageCookEmbed(src, opts); err != nil {
			return err
		}
	}
	if src == "" || opts.Out == "" {
		return fmt.Errorf("%w: build requires --out (bundled guest is used when --src and --sample are omitted)", platforms.ErrUsage)
	}
	if dest := strings.TrimSpace(opts.ExportSrc); dest != "" {
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
		if err := guest.Overlay(dest, src); err != nil {
			return fmt.Errorf("export-src: %w", err)
		}
		if opts.Stdout != nil {
			fmt.Fprintf(opts.Stdout, "exported guest sources %s\n", dest)
		}
	}

	workName := or(sample, filepath.Base(src))
	if opts.Src == "" && sample == "" {
		workName = "blazium-guest"
	}
	buildDir := filepath.Join(cache.PlatformDir(opts.Prefix, ID), "work", workName)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}

	extraPath := []string{filepath.Dir(env["EE_GCC"])}
	if env["IOP_GCC"] != "" {
		extraPath = append(extraPath, filepath.Dir(env["IOP_GCC"]))
	}
	if env["PS2DEV"] != "" {
		extraPath = append(extraPath,
			filepath.Join(env["PS2DEV"], "ee", "bin"),
			filepath.Join(env["PS2DEV"], "iop", "bin"),
			filepath.Join(env["PS2DEV"], "dvp", "bin"),
			filepath.Join(env["PS2DEV"], "bin"),
		)
	}
	extraEnv := map[string]string{
		"PS2SDK": env["PS2SDK"],
		"PS2DEV": or(env["PS2DEV"], filepath.Dir(env["PS2SDK"])),
		"EE_GCC": env["EE_GCC"],
	}
	if mkdir := gnuMkdir(); mkdir != "" {
		extraEnv["MKDIR"] = mkdir
	}

	if fileExists(filepath.Join(src, "Makefile")) || fileExists(filepath.Join(src, "Makefile.sample")) {
		if err := t.buildMake(ctx, src, buildDir, extraPath, extraEnv, opts); err != nil {
			return err
		}
	} else if fileExists(filepath.Join(src, "CMakeLists.txt")) {
		if err := t.buildCMake(ctx, src, buildDir, extraPath, extraEnv, opts); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: no Makefile or CMakeLists.txt in %s", platforms.ErrMissingTool, src)
	}

	elf, err := findMipsELF(buildDir, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return err
	}
	raw, err := os.ReadFile(elf)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.Out, raw, 0o644); err != nil {
		return err
	}
	if err := verifyMipsELF(opts.Out); err != nil {
		return err
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "wrote EE ELF %s\n", opts.Out)
	}
	return nil
}

func (t *Tool) buildMake(ctx context.Context, src, buildDir string, extraPath []string, extraEnv map[string]string, opts platforms.BuildOptions) error {
	makeProg := lookFile("make")
	if makeProg == "" {
		makeProg = lookFile("mingw32-make")
	}
	if makeProg == "" {
		return fmt.Errorf("%w: make (required to build the ps2sdk guest)", platforms.ErrMissingTool)
	}
	makefile := "Makefile"
	if fileExists(filepath.Join(src, "Makefile.sample")) && !fileExists(filepath.Join(src, "Makefile")) {
		makefile = "Makefile.sample"
	}
	args := []string{"-f", makefile, "-C", src}
	return t.runEnv(ctx, makeProg, args, extraPath, extraEnv, opts.Stdout, opts.Stderr)
}

func (t *Tool) buildCMake(ctx context.Context, src, buildDir string, extraPath []string, extraEnv map[string]string, opts platforms.BuildOptions) error {
	cmake := lookFile("cmake")
	if cmake == "" {
		return fmt.Errorf("%w: cmake", platforms.ErrMissingTool)
	}
	tc := filepath.Join(extraEnv["PS2SDK"], "samples", "ps2dev.cmake")
	cfg := []string{"-S", src, "-B", buildDir, "-DCMAKE_BUILD_TYPE=Debug"}
	if fileExists(tc) {
		cfg = append(cfg, "-DCMAKE_TOOLCHAIN_FILE="+tc)
	}
	if opts.Node != "" {
		cfg = append(cfg, "-DBLAZIUM_PS2_NODE="+filepath.ToSlash(opts.Node))
	}
	if opts.Mesh != "" {
		cfg = append(cfg, "-DBLAZIUM_PS2_MESH="+filepath.ToSlash(opts.Mesh))
	}
	if opts.Gtex != "" {
		cfg = append(cfg, "-DBLAZIUM_PS2_GTEX="+filepath.ToSlash(opts.Gtex))
	}
	if opts.Script != "" {
		cfg = append(cfg, "-DBLAZIUM_PS2_SCRIPT="+filepath.ToSlash(opts.Script))
	}
	if err := t.runEnv(ctx, cmake, cfg, extraPath, extraEnv, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("cmake configure: %w", err)
	}
	return t.runEnv(ctx, cmake, []string{"--build", buildDir}, extraPath, extraEnv, opts.Stdout, opts.Stderr)
}

func (t *Tool) compileEnv(opts platforms.CommonOptions) (map[string]string, error) {
	env, err := t.Env(opts)
	if err != nil {
		return nil, err
	}
	if compileReady(env) {
		return env, nil
	}
	env, _ = t.discover(opts.Prefix)
	return env, nil
}

func (t *Tool) runEnv(ctx context.Context, name string, args, extraPath []string, extraEnv map[string]string, stdout, stderr io.Writer) error {
	r := t.runner()
	if er, ok := r.(envRunner); ok {
		return er.RunEnv(ctx, name, args, extraPath, extraEnv, writerOrDiscard(stdout), writerOrDiscard(stderr))
	}
	return r.Run(ctx, name, args, writerOrDiscard(stdout), writerOrDiscard(stderr))
}

func gnuMkdir() string {
	if p := lookFile("mkdir"); p != "" {
		low := strings.ToLower(p)
		if !strings.Contains(low, `\system32\`) && !strings.Contains(low, `/system32/`) {
			return p
		}
	}
	for _, p := range []string{
		`C:\Program Files\Git\usr\bin\mkdir.exe`,
		`C:\Program Files (x86)\Git\usr\bin\mkdir.exe`,
	} {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func resolveSample(env map[string]string, sample string) (string, error) {
	if sample != "cube" {
		return "", fmt.Errorf("%w: --sample must be cube", platforms.ErrUsage)
	}
	sdk := env["PS2SDK"]
	p := filepath.Join(sdk, "ee", "draw", "samples", "cube")
	if fileExists(filepath.Join(p, "Makefile.sample")) || fileExists(filepath.Join(p, "Makefile")) {
		return p, nil
	}
	return "", fmt.Errorf("%w: cube sample not in PS2SDK (%s)", platforms.ErrMissingTool, p)
}

func findMipsELF(buildDir, src string) (string, error) {
	search := []string{buildDir, src}
	var found string
	for _, root := range search {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.HasSuffix(name, ".elf") {
				return nil
			}
			if verifyMipsELF(path) == nil {
				found = path
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found, nil
		}
	}
	return "", fmt.Errorf("no MIPS ELF in %s or %s after build", buildDir, src)
}

func verifyMipsELF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hdr := make([]byte, 20)
	n, err := f.Read(hdr)
	if err != nil && err != io.EOF {
		return err
	}
	if n < 20 || string(hdr[:4]) != elfMagic {
		return fmt.Errorf("%s is not an ELF (got %q)", path, hdr[:min(n, 4)])
	}
	mach := binary.LittleEndian.Uint16(hdr[18:20])
	if mach != 8 && mach != 0 {
		return fmt.Errorf("%s ELF e_machine=%d (want MIPS=8)", path, mach)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
