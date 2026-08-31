package n64

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

// WriteROM packs a DragonFS tree with mkdfs and wraps an ELF with n64tool.
// This is the disc analog. There is no ISO/CUE product.
func (t *Tool) WriteROM(ctx context.Context, dir, elf, out string, stdout, stderr io.Writer) error {
	if strings.TrimSpace(out) == "" {
		return report.Usage("rom requires --out FILE.z64")
	}
	env, err := t.compileEnv(platforms.CommonOptions{})
	if err != nil {
		return err
	}
	if !compileReady(env) {
		return report.Missing("n64 compile tools not ready", "blazium-toolchain n64 setup --profile compile")
	}
	mkdfs := or(env["N64_MKDFS"], walkNamed(env["N64_INST"], hostNames("mkdfs")...))
	n64tool := or(env["N64_TOOL"], walkNamed(env["N64_INST"], hostNames("n64tool")...))
	if !fileExists(mkdfs) {
		return report.Missing("mkdfs not found", "blazium-toolchain n64 setup --profile compile")
	}
	if !fileExists(n64tool) {
		return report.Missing("n64tool not found", "blazium-toolchain n64 setup --profile compile")
	}
	if strings.TrimSpace(dir) == "" && strings.TrimSpace(elf) == "" {
		return report.Usage("rom requires --dir TREE and/or --elf FILE.elf")
	}

	extraPath := []string{filepath.Join(env["N64_INST"], "bin")}
	extraEnv := map[string]string{"N64_INST": env["N64_INST"]}

	dfsOut := ""
	if strings.TrimSpace(dir) != "" {
		if !dirExists(dir) {
			return report.Usage("--dir is not a directory: " + dir)
		}
		dfsOut = strings.TrimSuffix(out, filepath.Ext(out)) + ".dfs"
		if err := t.runEnv(ctx, mkdfs, []string{dfsOut, dir}, extraPath, extraEnv, stdout, stderr); err != nil {
			return fmt.Errorf("mkdfs: %w", err)
		}
	}

	if strings.TrimSpace(elf) != "" {
		title := strings.TrimSpace(settings.Current().N64.RomTitle)
		if title == "" {
			title = "Blazium N64"
		}
		args := []string{"-o", out, "-t", title}
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
		return report.Usage("rom --dir also needs --elf to wrap a .z64")
	}

	if err := verifyZ64(out); err != nil {
		return err
	}
	if stdout != nil {
		report.Line(stdout, report.Wrote, out)
	}
	return nil
}
