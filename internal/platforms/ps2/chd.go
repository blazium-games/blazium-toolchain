package ps2

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

// WriteCHD packs an ISO9660 image with chdman createcd. Never fetches chdman.
func WriteCHD(ctx context.Context, isoPath, outPath string, stdout, stderr io.Writer) error {
	if err := requireHost(); err != nil {
		return err
	}
	isoPath = strings.TrimSpace(isoPath)
	outPath = strings.TrimSpace(outPath)
	if isoPath == "" || outPath == "" {
		return fmt.Errorf("%w: chd requires --iso and --out", platforms.ErrUsage)
	}
	if strings.EqualFold(filepath.Ext(isoPath), ".cue") {
		return fmt.Errorf("%w: chd refuses .cue inputs (ISO9660 only)", platforms.ErrUsage)
	}
	if strings.EqualFold(filepath.Ext(outPath), ".cue") {
		return fmt.Errorf("%w: chd refuses .cue outputs", platforms.ErrUsage)
	}
	if _, err := os.Stat(isoPath); err != nil {
		return fmt.Errorf("chd iso: %w", err)
	}
	bin := strings.TrimSpace(os.Getenv("CHDMAN"))
	if bin == "" {
		bin = lookFile("chdman")
	}
	if bin == "" {
		return fmt.Errorf("%w: chdman (set CHDMAN or put chdman on PATH; never fetched)", platforms.ErrMissingTool)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	args := []string{"createcd", "-f", "-i", isoPath, "-o", outPath}
	run := execx.Host{}
	if err := run.Run(ctx, bin, args, writerOrDiscard(stdout), writerOrDiscard(stderr)); err != nil {
		return fmt.Errorf("chdman createcd: %w", err)
	}
	if stdout != nil {
		fmt.Fprintf(stdout, "wrote CHD %s\n", outPath)
	}
	return nil
}
