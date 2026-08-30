package n64

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

// WriteROM packs a DragonFS tree with mkdfs and wraps an ELF with n64tool.
// This is the disc analog. There is no ISO/CUE product.
func (t *Tool) WriteROM(ctx context.Context, dir, elf, out string, stdout, stderr io.Writer) error {
	if strings.TrimSpace(out) == "" {
		return fmt.Errorf("%w: rom requires --out FILE.z64", platforms.ErrUsage)
	}
	env, err := t.compileEnv(platforms.CommonOptions{})
	if err != nil {
		return err
	}
	if !compileReady(env) {
		return fmt.Errorf("%w: run blazium-toolchain n64 setup --profile compile first", platforms.ErrMissingTool)
	}
	mkdfs := or(env["N64_MKDFS"], walkNamed(env["N64_INST"], hostNames("mkdfs")...))
	n64tool := or(env["N64_TOOL"], walkNamed(env["N64_INST"], hostNames("n64tool")...))
	if !fileExists(mkdfs) {
		return fmt.Errorf("%w: mkdfs (libdragon tools; run libdragon build.sh on preview)", platforms.ErrMissingTool)
	}
	if !fileExists(n64tool) {
		return fmt.Errorf("%w: n64tool", platforms.ErrMissingTool)
	}
	if strings.TrimSpace(dir) == "" && strings.TrimSpace(elf) == "" {
		return fmt.Errorf("%w: rom requires --dir TREE and/or --elf FILE.elf", platforms.ErrUsage)
	}

	extraPath := []string{filepath.Join(env["N64_INST"], "bin")}
	extraEnv := map[string]string{"N64_INST": env["N64_INST"]}

	dfsOut := ""
	if strings.TrimSpace(dir) != "" {
		if !dirExists(dir) {
			return fmt.Errorf("%w: --dir is not a directory: %s", platforms.ErrUsage, dir)
		}
		dfsOut = strings.TrimSuffix(out, filepath.Ext(out)) + ".dfs"
		if err := t.runEnv(ctx, mkdfs, []string{dfsOut, dir}, extraPath, extraEnv, stdout, stderr); err != nil {
			return fmt.Errorf("mkdfs: %w", err)
		}
	}

	if strings.TrimSpace(elf) != "" {
		args := []string{"-o", out, "-t", "Blazium N64"}
		if dfsOut != "" {
			args = append(args, "-f", dfsOut)
		}
		args = append(args, elf)
		if err := t.runEnv(ctx, n64tool, args, extraPath, extraEnv, stdout, stderr); err != nil {
			return fmt.Errorf("n64tool: %w", err)
		}
	} else if dfsOut != "" && fileExists(out) {
		// DFS-only refresh onto an existing ROM is not supported without an ELF.
		_ = dfsOut
	} else if dfsOut != "" {
		return fmt.Errorf("%w: rom --dir also needs --elf to wrap a .z64", platforms.ErrUsage)
	}

	if err := verifyZ64(out); err != nil {
		return err
	}
	if stdout != nil {
		fmt.Fprintf(stdout, "wrote N64 ROM %s\n", out)
	}
	return nil
}
