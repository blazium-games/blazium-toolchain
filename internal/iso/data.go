package iso

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/report"
	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

// WriteDataISO writes a simple ISO9660 data disc (no UDF, no VIDEO_TS) from srcDir.
// Used for PS2 BOOT2 images. Files keep 8.3-mangled names.
func WriteDataISO(srcDir, outPath, volumeID string) error {
	if srcDir == "" || outPath == "" {
		return report.Usage("iso requires --dir and --out")
	}
	st, err := os.Stat(srcDir)
	if err != nil {
		return report.Fail("iso: stat "+srcDir+": "+err.Error(), "pass --dir to an existing folder")
	}
	if !st.IsDir() {
		return report.Fail(srcDir+" is not a directory", "pass --dir to a folder")
	}
	now := time.Now().UTC()
	root := &node{name: "", isDir: true, modTime: st.ModTime()}
	if err := addDirChildren(root, srcDir, true); err != nil {
		return err
	}
	if len(root.children) == 0 {
		return report.Fail(srcDir+" has no files", "put GAME.ELF / SYSTEM.CNF in the tree")
	}
	assignISONames(root)

	volumeID = VolumeOr(volumeID, settings.Current().PS2.ISOVolume)
	s := settings.Current().PS2
	props := VolumeProps{
		VolumeID:    volumeID,
		System:      s.ISOSystem,
		Application: s.ISOApp,
		Created:     now,
		Disc:        1,
		Discs:       1,
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return report.Fail("iso: create "+outPath+": "+err.Error(), "")
	}
	defer f.Close()

	total, err := layoutDataISO(root)
	if err != nil {
		return err
	}
	if err := writeISO9660(f, root, props, total); err != nil {
		return err
	}
	if err := writeDataFileExtents(f, root); err != nil {
		return err
	}
	end := int64(total) * int64(SectorSize)
	if err := f.Truncate(end); err != nil {
		return err
	}
	return nil
}

func layoutDataISO(root *node) (uint32, error) {
	// path tables (L+M ISO + L+M Joliet) — writeISO9660 allocates from isoMetaStart
	// File data must start after the metadata window so PVD directory records
	// can point at extents. Place files after isoMetaLimit.
	next := uint32(isoMetaLimit)
	var walk func(n *node)
	walk = func(n *node) {
		for _, c := range n.children {
			if c.isDir {
				walk(c)
				continue
			}
			c.absSector = uint64(next)
			c.dataLen = uint32(c.size)
			next += blocks32(uint64(c.size))
		}
	}
	walk(root)
	if next < isoMetaLimit+1 {
		next = isoMetaLimit + 1
	}
	return next, nil
}

func writeDataFileExtents(out io.WriterAt, root *node) error {
	var walk func(n *node) error
	walk = func(n *node) error {
		for _, c := range n.children {
			if c.isDir {
				if err := walk(c); err != nil {
					return err
				}
				continue
			}
			var data []byte
			var err error
			if c.data != nil {
				data = c.data
			} else if c.srcPath != "" {
				data, err = os.ReadFile(c.srcPath)
				if err != nil {
					return err
				}
			}
			if c.absSector == 0 {
				return fmt.Errorf("iso: file %s has no sector", c.name)
			}
			if _, err := out.WriteAt(data, int64(c.absSector)*int64(SectorSize)); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

// HasISO9660PVD reports whether path looks like an ISO9660 image (CD001 at sector 16).
func HasISO9660PVD(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8)
	if _, err := f.ReadAt(buf, int64(isoPVDSector)*int64(SectorSize)+1); err != nil {
		return false
	}
	return string(buf[:5]) == "CD001"
}
