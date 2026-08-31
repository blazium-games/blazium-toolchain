package ps2

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blazium-games/blazium-toolchain/internal/execx"
	"github.com/blazium-games/blazium-toolchain/internal/report"
)

// WriteCHD packs an ISO9660 image with chdman createcd. Never fetches chdman.
func WriteCHD(ctx context.Context, isoPath, outPath string, stdout, stderr io.Writer) error {
	if err := requireHost(); err != nil {
		return err
	}
	isoPath = strings.TrimSpace(isoPath)
	outPath = strings.TrimSpace(outPath)
	if isoPath == "" || outPath == "" {
		return report.Usage("chd requires --iso and --out")
	}
	if strings.EqualFold(filepath.Ext(isoPath), ".cue") {
		return report.Usage("chd refuses .cue inputs (ISO9660 only)")
	}
	if strings.EqualFold(filepath.Ext(outPath), ".cue") {
		return report.Usage("chd refuses .cue outputs")
	}
	if _, err := os.Stat(isoPath); err != nil {
		return report.Fail("chd iso: "+err.Error(), "pass --iso to an existing ISO9660 file")
	}
	bin := strings.TrimSpace(os.Getenv("CHDMAN"))
	if bin == "" {
		bin = lookFile("chdman")
	}
	if bin == "" {
		return report.Missing("chdman not found (never fetched)", "set CHDMAN or put chdman on PATH")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	args := []string{"createcd", "-f", "-i", isoPath, "-o", outPath}
	run := execx.Host{}
	if err := run.Run(ctx, bin, args, writerOrDiscard(stdout), writerOrDiscard(stderr)); err != nil {
		return report.Fail("chdman createcd: "+err.Error(), "")
	}
	if stdout != nil {
		report.Line(stdout, report.Wrote, outPath)
	}
	return nil
}
