package iso

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ISO9660Entry is one directory record from the primary volume.
type ISO9660Entry struct {
	Name   string
	Sector uint32
	Size   uint32
	Dir    bool
}

// ReadPVDVolume returns the ISO9660 volume identifier from sector 16.
func ReadPVDVolume(r io.ReaderAt) (string, error) {
	sec, err := readSector(r, isoPVDSector)
	if err != nil {
		return "", err
	}
	if string(sec[1:6]) != "CD001" || sec[0] != 1 {
		return "", fmt.Errorf("iso: no ISO9660 PVD at sector 16")
	}
	return strings.TrimSpace(string(sec[40:72])), nil
}

// ReadPVDPublisher returns the publisher identifier.
func ReadPVDPublisher(r io.ReaderAt) (string, error) {
	sec, err := readSector(r, isoPVDSector)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sec[318:446])), nil
}

// HasUDFVRS reports BEA01/NSR02/TEA01 at sectors 19–21.
func HasUDFVRS(r io.ReaderAt) error {
	for i, id := range []string{"BEA01", "NSR02", "TEA01"} {
		sec, err := readSector(r, uint64(udfVRSSector+i))
		if err != nil {
			return err
		}
		if string(sec[1:6]) != id {
			return fmt.Errorf("iso: sector %d want %s got %q", udfVRSSector+i, id, string(sec[1:6]))
		}
	}
	return nil
}

// ListISO9660Dir lists the ISO9660 directory at the given sector.
func ListISO9660Dir(r io.ReaderAt, sector uint32) ([]ISO9660Entry, error) {
	sec, err := readSector(r, uint64(sector))
	if err != nil {
		return nil, err
	}
	var out []ISO9660Entry
	off := 0
	for off < len(sec) {
		l := int(sec[off])
		if l == 0 {
			break
		}
		if off+l > len(sec) {
			break
		}
		rec := sec[off : off+l]
		nameLen := int(rec[32])
		name := string(rec[33 : 33+nameLen])
		if name == "\x00" {
			name = "."
		} else if name == "\x01" {
			name = ".."
		} else if i := strings.Index(name, ";"); i >= 0 {
			name = name[:i]
		}
		out = append(out, ISO9660Entry{
			Name:   name,
			Sector: binary.LittleEndian.Uint32(rec[2:6]),
			Size:   binary.LittleEndian.Uint32(rec[10:14]),
			Dir:    rec[25]&0x02 != 0,
		})
		off += l
	}
	return out, nil
}

// FindISO9660 walks from the PVD root to slash-separated path (Level 1 names).
func FindISO9660(r io.ReaderAt, path string) (ISO9660Entry, error) {
	pvd, err := readSector(r, isoPVDSector)
	if err != nil {
		return ISO9660Entry{}, err
	}
	rootSec := binary.LittleEndian.Uint32(pvd[158:162])
	cur := ISO9660Entry{Name: ".", Sector: rootSec, Dir: true}
	path = strings.Trim(strings.ToUpper(filepathToSlash(path)), "/")
	if path == "" {
		return cur, nil
	}
	for _, part := range strings.Split(path, "/") {
		ents, err := ListISO9660Dir(r, cur.Sector)
		if err != nil {
			return ISO9660Entry{}, err
		}
		var found *ISO9660Entry
		for i := range ents {
			if strings.EqualFold(ents[i].Name, part) {
				found = &ents[i]
				break
			}
		}
		if found == nil {
			return ISO9660Entry{}, fmt.Errorf("iso: %s not in ISO9660 tree", path)
		}
		cur = *found
	}
	return cur, nil
}

// ValidateMeta checks VIDEO_TS.IFO exists after resolve.
func ValidateMeta(m DiscMeta) error {
	resolved, err := m.Resolve()
	if err != nil {
		return err
	}
	if resolved.Dir == "" {
		return fmt.Errorf("iso: dir is required")
	}
	if err := osStatVIDEO(resolved.Dir); err != nil {
		return err
	}
	if _, err := collectExtras(resolved); err != nil {
		return err
	}
	return nil
}

func osStatVIDEO(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "VIDEO_TS", "VIDEO_TS.IFO")); err != nil {
		return fmt.Errorf("iso: VIDEO_TS/VIDEO_TS.IFO is required")
	}
	return nil
}
