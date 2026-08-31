package n64

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	guest "github.com/blazium-games/blazium-toolchain/guest/n64"
	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

// GuestDir is <prefix>/n64/guest/runtime — the bundled MIT stub.
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
		return fmt.Errorf("%w: run blazium-toolchain n64 setup --profile compile first", platforms.ErrMissingTool)
	}
	if err := normalizeDisplay(&opts); err != nil {
		return err
	}
	if err := normalizeRdram(&opts); err != nil {
		return err
	}

	src := opts.Src
	sample := strings.ToLower(strings.TrimSpace(opts.Sample))
	overlay := strings.TrimSpace(opts.Overlay)
	if overlay == "" {
		overlay = strings.TrimSpace(os.Getenv("BLAZIUM_N64_OVERLAY"))
	}
	if src != "" && !fileExists(filepath.Join(src, "CMakeLists.txt")) && !fileExists(filepath.Join(src, "Makefile")) {
		if overlay == "" {
			overlay = src
		}
		src = ""
	}
	if src == "" && sample != "" {
		src, err = resolveSample(sample)
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
		return fmt.Errorf("%w: build requires --out FILE.z64 (bundled guest is used when --src and --sample are omitted)", platforms.ErrUsage)
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

	extraPath := []string{filepath.Dir(env["N64_GCC"]), filepath.Join(env["N64_INST"], "bin")}
	extraEnv := map[string]string{
		"N64_INST":          env["N64_INST"],
		"LIBDRAGON_PREVIEW": "2",
	}
	if isT3dSample(sample) {
		t3d := siblingTiny3d()
		if t3d == "" {
			return fmt.Errorf("%w: sibling n64_stuff/tiny3d not found for --sample t3dquad", platforms.ErrMissingTool)
		}
		if err := t.ensureT3dLib(ctx, t3d, extraPath, extraEnv, opts); err != nil {
			return err
		}
		extraEnv["T3D_INST"] = t3d
	}
	if isOvlDemoSample(sample) {
		if err := t.ensureDsoTools(ctx, extraPath, extraEnv, opts); err != nil {
			return err
		}
	}

	if fileExists(filepath.Join(src, "Makefile")) {
		if err := t.buildMake(ctx, src, extraPath, extraEnv, opts); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: no Makefile in %s (N64 guest is Makefile-first via n64.mk)", platforms.ErrMissingTool, src)
	}

	rom, err := findZ64(src, buildDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return err
	}
	raw, err := os.ReadFile(rom)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.Out, raw, 0o644); err != nil {
		return err
	}
	if err := verifyZ64(opts.Out); err != nil {
		// #region agent log
		agentLog("C", "build.go:verify", "product not z64", map[string]any{"out": opts.Out, "err": err.Error()})
		// #endregion
		return err
	}
	// #region agent log
	agentLog("C", "build.go:verify", "wrote z64", map[string]any{"out": opts.Out, "src": rom, "bytes": len(raw)})
	// #endregion
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "wrote N64 ROM %s\n", opts.Out)
	}
	return nil
}

func (t *Tool) buildMake(ctx context.Context, src string, extraPath []string, extraEnv map[string]string, opts platforms.BuildOptions) error {
	unixBin := []string{
		`C:\Program Files\Git\usr\bin`,
		`C:\Program Files\Git\bin`,
		`C:\msys64\usr\bin`,
		`C:\msys64\ucrt64\bin`,
	}
	extraPath = append(unixBin, extraPath...)
	makeProg := ""
	if extraEnv["N64_INST"] != "" {
		makeProg = walkNamed(extraEnv["N64_INST"], hostNames("make")...)
	}
	if makeProg == "" {
		makeProg = lookFile("make")
	}
	if makeProg == "" {
		makeProg = lookFile("mingw32-make")
	}
	if makeProg == "" {
		return fmt.Errorf("%w: make (required to build the libdragon guest; use Git/MSYS2 make)", platforms.ErrMissingTool)
	}
	args := []string{"-f", "Makefile", "-C", src}
	if v := extraEnv["T3D_INST"]; v != "" {
		args = append(args, "T3D_INST="+v)
	}
	return t.runEnv(ctx, makeProg, args, extraPath, extraEnv, opts.Stdout, opts.Stderr)
}

func (t *Tool) ensureT3dLib(ctx context.Context, t3d string, extraPath []string, extraEnv map[string]string, opts platforms.BuildOptions) error {
	if fileExists(t3dLibPath(t3d)) {
		return nil
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "building Tiny3D libt3d.a in %s\n", t3d)
	}
	return t.buildMake(ctx, t3d, extraPath, extraEnv, opts)
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

func normalizeDisplay(opts *platforms.BuildOptions) error {
	s := strings.TrimSpace(opts.Display)
	if s == "" || s == "320" {
		opts.Display = "320"
		return nil
	}
	if s == "640" {
		opts.Display = "640"
		return nil
	}
	return fmt.Errorf("%w: --display must be 320 or 640", platforms.ErrUsage)
}

func normalizeRdram(opts *platforms.BuildOptions) error {
	s := strings.TrimSpace(opts.Rdram)
	if s == "" || s == "8" {
		opts.Rdram = "8"
		return nil
	}
	if s == "4" {
		opts.Rdram = "4"
		return nil
	}
	return fmt.Errorf("%w: --rdram must be 8 or 4", platforms.ErrUsage)
}

func rdramIs4(s string) bool {
	return strings.TrimSpace(s) == "4"
}

func isT3dSample(sample string) bool {
	switch strings.ToLower(strings.TrimSpace(sample)) {
	case "t3dquad", "00_quad", "tiny3d":
		return true
	default:
		return false
	}
}

func isOvlDemoSample(sample string) bool {
	switch strings.ToLower(strings.TrimSpace(sample)) {
	case "ovldemo", "overlay", "dso":
		return true
	default:
		return false
	}
}

func resolveSample(sample string) (string, error) {
	sample = strings.ToLower(strings.TrimSpace(sample))
	if isT3dSample(sample) {
		t3d := siblingTiny3d()
		if t3d == "" {
			return "", fmt.Errorf("%w: sibling n64_stuff/tiny3d not found for --sample t3dquad", platforms.ErrMissingTool)
		}
		p := filepath.Join(t3d, "examples", "00_quad")
		if !fileExists(filepath.Join(p, "Makefile")) {
			return "", fmt.Errorf("%w: sample Makefile missing: %s", platforms.ErrMissingTool, p)
		}
		return p, nil
	}
	allowed := map[string]string{
		"helloworld": "helloworld",
		"hello":      "helloworld",
		"rdpqdemo":   "rdpqdemo",
		"rdpq":       "rdpqdemo",
		"ovldemo":    "ovldemo",
		"overlay":    "ovldemo",
		"dso":        "ovldemo",
	}
	name, ok := allowed[sample]
	if !ok {
		return "", fmt.Errorf("%w: --sample must be helloworld, rdpqdemo, t3dquad, or ovldemo", platforms.ErrUsage)
	}
	lib := siblingLibdragon()
	if lib == "" {
		return "", fmt.Errorf("%w: sibling n64_stuff/libdragon not found for --sample", platforms.ErrMissingTool)
	}
	p := filepath.Join(lib, "examples", name)
	if !fileExists(filepath.Join(p, "Makefile")) {
		return "", fmt.Errorf("%w: sample Makefile missing: %s", platforms.ErrMissingTool, p)
	}
	return p, nil
}

func findZ64(src, buildDir string) (string, error) {
	var hits []string
	add := func(p string) {
		if fileExists(p) && strings.EqualFold(filepath.Ext(p), ".z64") {
			hits = append(hits, p)
		}
	}
	_ = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".z64") {
			hits = append(hits, path)
		}
		return nil
	})
	add(filepath.Join(src, "runtime.z64"))
	add(filepath.Join(src, "helloworld.z64"))
	add(filepath.Join(src, "rdpqdemo.z64"))
	add(filepath.Join(src, "t3d_00_quad.z64"))
	add(filepath.Join(src, "ovldemo.z64"))
	if buildDir != "" {
		_ = filepath.WalkDir(buildDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".z64") {
				hits = append(hits, path)
			}
			return nil
		})
	}
	for _, h := range hits {
		if isZ64(h) {
			return h, nil
		}
	}
	if len(hits) > 0 {
		return "", fmt.Errorf("found %s but it is not a big-endian .z64", hits[0])
	}
	return "", fmt.Errorf("%w: no .z64 produced in %s", platforms.ErrMissingTool, src)
}
