package ps1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/cache"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const psxMagic = "PS-X EXE"

type envRunner interface {
	RunEnv(ctx context.Context, name string, args []string, extraPath []string, extraEnv map[string]string, stdout, stderr io.Writer) error
}

func (t *Tool) Build(ctx context.Context, opts platforms.BuildOptions) error {
	env, err := t.compileEnv(opts.CommonOptions)
	if err != nil {
		return err
	}
	if !compileReady(env) {
		return fmt.Errorf("%w: run blazium-toolchain ps1 setup --profile compile first", platforms.ErrMissingTool)
	}

	src := opts.Src
	sample := strings.ToLower(strings.TrimSpace(opts.Sample))
	if src == "" && sample != "" {
		src, err = resolveSample(opts.Prefix, sample)
		if err != nil {
			return err
		}
	}
	if src == "" || opts.Out == "" {
		return fmt.Errorf("%w: build requires --out and either --src or --sample", platforms.ErrUsage)
	}

	target := cmakeTarget(sample, src)
	if sample == "gte" {
		if walkNamed(src, "texture.tim") == "" && !fileExists(filepath.Join(src, "texture.tim")) {
			return fmt.Errorf("%w: gte sample is missing texture.tim in the official zip; skip or supply --src with the asset", platforms.ErrMissingTool)
		}
	}

	cmake, ninja, err := t.ensureHostBuildTools(ctx, opts.Prefix, opts.Stdout)
	if err != nil {
		return err
	}

	workName := or(sample, filepath.Base(src))
	buildDir := filepath.Join(cache.PlatformDir(opts.Prefix, ID), "work", workName)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}

	sdkFile := filepath.Join(env["PSN00BSDK_LIBS"], "cmake", "sdk.cmake")
	if !fileExists(sdkFile) {
		return fmt.Errorf("%w: sdk.cmake not under PSN00BSDK_LIBS (%s)", platforms.ErrMissingTool, env["PSN00BSDK_LIBS"])
	}

	extraPath := []string{
		filepath.Dir(env["MIPS_GCC"]),
		filepath.Dir(env["ELF2X"]),
		filepath.Dir(cmake),
		filepath.Dir(ninja),
	}
	extraEnv := map[string]string{
		"PSN00BSDK_LIBS": env["PSN00BSDK_LIBS"],
		"PSN00BSDK_TC":   env["PSN00BSDK_TC"],
		"MIPS_GCC":       env["MIPS_GCC"],
		"ELF2X":          env["ELF2X"],
	}

	cfg := []string{
		"-S", src,
		"-B", buildDir,
		"-G", "Ninja",
		"-DCMAKE_TOOLCHAIN_FILE=" + sdkFile,
		"-DPSN00BSDK_TC=" + env["PSN00BSDK_TC"],
		"-DCMAKE_BUILD_TYPE=Debug",
		"-DCMAKE_MAKE_PROGRAM=" + ninja,
	}
	if opts.Tim != "" {
		cfg = append(cfg, "-DBLAZIUM_PS1_TIM="+filepath.ToSlash(opts.Tim))
	}
	if err := t.runEnv(ctx, cmake, cfg, extraPath, extraEnv, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("cmake configure: %w", err)
	}
	if err := t.runEnv(ctx, cmake, []string{"--build", buildDir, "--target", target}, extraPath, extraEnv, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("cmake build: %w", err)
	}

	guest, err := findPSXEXE(buildDir, target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return err
	}
	raw, err := os.ReadFile(guest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.Out, raw, 0o644); err != nil {
		return err
	}
	if err := verifyPSXEXE(opts.Out); err != nil {
		return err
	}
	if opts.Stdout != nil {
		fmt.Fprintf(opts.Stdout, "wrote PS-X EXE %s\n", opts.Out)
	}
	return nil
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

func (t *Tool) ensureHostBuildTools(ctx context.Context, prefix string, log io.Writer) (cmake, ninja string, err error) {
	plat := cache.PlatformDir(prefix, ID)
	cmake = t.findHostTool(prefix, "cmake", "cmake")
	ninja = t.findHostTool(prefix, "ninja", "ninja")
	if cmake != "" && ninja != "" {
		return cmake, ninja, nil
	}
	for _, a := range hostBuildAssets() {
		if a.ID == "cmake" && cmake != "" {
			continue
		}
		if a.ID == "ninja" && ninja != "" {
			continue
		}
		dest := filepath.Join(plat, a.Dest)
		if compilePiecePresent(dest, a.ID) {
			continue
		}
		if log != nil {
			fmt.Fprintf(log, "fetching %s from %s\n", a.ID, a.URL)
		}
		if err := t.fetcher().FetchZip(ctx, a.URL, a.SHA256, dest, log); err != nil {
			return "", "", fmt.Errorf("%w: %s: %v", platforms.ErrMissingTool, a.ID, err)
		}
	}
	cmake = t.findHostTool(prefix, "cmake", "cmake")
	ninja = t.findHostTool(prefix, "ninja", "ninja")
	if cmake == "" {
		return "", "", fmt.Errorf("%w: cmake (install CMake 3.21+ or allow ps1 build to fetch it)", platforms.ErrMissingTool)
	}
	if ninja == "" {
		return "", "", fmt.Errorf("%w: ninja (install Ninja or allow ps1 build to fetch it)", platforms.ErrMissingTool)
	}
	return cmake, ninja, nil
}

func (t *Tool) findHostTool(prefix, id, command string) string {
	plat := cache.PlatformDir(prefix, ID)
	if p := walkNamed(filepath.Join(plat, id), hostNames(command)...); p != "" {
		return p
	}
	if p, err := t.runner().LookPath(command); err == nil {
		return p
	}
	return lookFile(command)
}

func resolveSample(prefix, sample string) (string, error) {
	plat := cache.PlatformDir(prefix, ID)
	var marker string
	switch sample {
	case "template":
		marker = filepath.Join("share", "psn00bsdk", "template", "CMakeLists.txt")
	case "gte":
		marker = filepath.Join("share", "psn00bsdk", "examples", "graphics", "gte", "CMakeLists.txt")
	default:
		return "", fmt.Errorf("%w: --sample must be template or gte", platforms.ErrUsage)
	}
	if hit := findRelFile(plat, marker); hit != "" {
		return filepath.Dir(hit), nil
	}
	return "", fmt.Errorf("%w: official %s sample not in the SDK cache; run ps1 setup --profile compile", platforms.ErrMissingTool, sample)
}

func findRelFile(root, rel string) string {
	if root == "" {
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(filepath.ToSlash(path), filepath.ToSlash(rel)) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return ""
	}
	if abs, err := filepath.Abs(found); err == nil {
		return abs
	}
	return found
}

func cmakeTarget(sample, src string) string {
	if sample == "gte" || strings.EqualFold(filepath.Base(src), "gte") {
		return "gte"
	}
	if sample == "template" || strings.EqualFold(filepath.Base(src), "template") {
		return "template"
	}
	base := filepath.Base(src)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "template"
	}
	return base
}

func findPSXEXE(buildDir, target string) (string, error) {
	prefer := hostNames(target + ".exe")
	if p := walkNamed(buildDir, prefer...); p != "" && verifyPSXEXE(p) == nil {
		return p, nil
	}
	var found string
	_ = filepath.WalkDir(buildDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(name, ".elf") {
			return nil
		}
		if verifyPSXEXE(path) == nil {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("no PS-X EXE in %s after cmake --target %s", buildDir, target)
	}
	return found, nil
}

func verifyPSXEXE(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	if !bytes.Equal(buf[:n], []byte(psxMagic)) {
		return fmt.Errorf("%s is not a PS-X EXE (got %q)", path, buf[:n])
	}
	return nil
}
