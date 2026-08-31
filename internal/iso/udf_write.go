package iso

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	lbnAnchor      = 256
	lbnMainVDS     = 257
	lbnReserveVDS  = 273
	lbnIntegrity   = 289
	lbnPartitionLB = 291
	vdsSectors     = 16
	udfVRSSector   = 19
)

type udfWriter struct {
	out       io.WriterAt
	props     VolumeProps
	now       time.Time
	partStart uint64
	files     map[string]Location
}

func writeUDF(out io.WriterAt, root *node, props VolumeProps) (*Result, error) {
	next := uint32(2)
	var uid uint64
	assignBlocks(root, &next, &uid)
	w := &udfWriter{
		out:       out,
		props:     props,
		now:       props.Created,
		partStart: lbnPartitionLB,
		files:     map[string]Location{},
	}
	if w.now.IsZero() {
		w.now = time.Now().UTC()
	}
	nf, nd := countTree(root)
	if err := w.writeVolumeStructures(next, nf, nd); err != nil {
		return nil, err
	}
	if err := w.writeFileSet(root); err != nil {
		return nil, err
	}
	if err := w.writeTree(root, root.feBlock, ""); err != nil {
		return nil, err
	}
	totalSectors := lbnPartitionLB + uint64(next)
	backup := make([]byte, SectorSize)
	putAnchor(backup, uint32(totalSectors))
	if err := w.writeSector(totalSectors, backup); err != nil {
		return nil, err
	}
	return &Result{TotalSectors: totalSectors + 1, Files: w.files}, nil
}

func assignBlocks(n *node, next *uint32, uid *uint64) {
	n.feBlock = *next
	*next++
	n.uniqueID = *uid
	*uid++
	if n.isDir {
		n.dataLen = dirFIDBytes(n)
		n.dataLB = *next
		*next += blocks(uint64(n.dataLen))
		for _, c := range n.children {
			assignBlocks(c, next, uid)
		}
		return
	}
	n.dataLen = uint32(n.size)
	if n.size > 0 {
		n.dataLB = *next
		*next += blocks(uint64(n.size))
	}
}

func blocks(n uint64) uint32 {
	return uint32((n + SectorSize - 1) / SectorSize)
}

func countTree(n *node) (files, dirs uint32) {
	if n.isDir {
		dirs = 1
	} else {
		files = 1
	}
	for _, c := range n.children {
		f, d := countTree(c)
		files += f
		dirs += d
	}
	return files, dirs
}

func (w *udfWriter) writeSector(sector uint64, data []byte) error {
	if _, err := w.out.WriteAt(data, int64(sector*SectorSize)); err != nil {
		return fmt.Errorf("iso: write sector %d: %w", sector, err)
	}
	return nil
}

func (w *udfWriter) writeVolumeStructures(partitionBlocks, numFiles, numDirs uint32) error {
	if err := w.writeSector(udfVRSSector, volStructDesc("BEA01")); err != nil {
		return err
	}
	if err := w.writeSector(udfVRSSector+1, volStructDesc("NSR02")); err != nil {
		return err
	}
	if err := w.writeSector(udfVRSSector+2, volStructDesc("TEA01")); err != nil {
		return err
	}
	anchor := make([]byte, SectorSize)
	putAnchor(anchor, lbnAnchor)
	if err := w.writeSector(lbnAnchor, anchor); err != nil {
		return err
	}
	for _, base := range []uint32{lbnMainVDS, lbnReserveVDS} {
		if err := w.writeVDS(base, partitionBlocks); err != nil {
			return err
		}
	}
	if err := w.writeSector(lbnIntegrity, w.integrityDescriptor(partitionBlocks, numFiles, numDirs)); err != nil {
		return err
	}
	return w.writeSector(lbnIntegrity+1, terminatingDescriptor(lbnIntegrity+1))
}

func (w *udfWriter) writeVDS(base, partitionBlocks uint32) error {
	descs := [][]byte{
		w.primaryVolumeDescriptor(base + 0),
		w.implUseVolumeDescriptor(base + 1),
		w.partitionDescriptor(base+2, partitionBlocks),
		w.logicalVolumeDescriptor(base + 3),
		w.unallocatedSpaceDescriptor(base + 4),
		terminatingDescriptor(base + 5),
	}
	for i, d := range descs {
		if err := w.writeSector(uint64(base)+uint64(i), d); err != nil {
			return err
		}
	}
	return nil
}

func (w *udfWriter) writeFileSet(root *node) error {
	fsd := make([]byte, SectorSize)
	le := binary.LittleEndian
	copy(fsd[16:], encodeTimestamp(w.now))
	le.PutUint16(fsd[28:], 3)
	le.PutUint16(fsd[30:], 3)
	le.PutUint32(fsd[32:], 1)
	le.PutUint32(fsd[36:], 1)
	copy(fsd[48:], charSpec())
	copy(fsd[112:], encodeDString(w.props.VolumeID, 128))
	copy(fsd[240:], charSpec())
	copy(fsd[304:], encodeDString(w.props.VolumeID, 32))
	copy(fsd[400:], longAD(SectorSize, root.feBlock, 0))
	copy(fsd[416:], domainEntityID())
	putTag(fsd[:512], tagFileSet, 0)
	if err := w.writeSector(w.partStart+0, fsd); err != nil {
		return err
	}
	return w.writeSector(w.partStart+1, terminatingDescriptor(1))
}

func (w *udfWriter) writeTree(n *node, parentFE uint32, prefix string) error {
	if n.isDir {
		if err := w.writeDirEntry(n, parentFE); err != nil {
			return err
		}
		for _, c := range n.children {
			child := c.name
			if prefix != "" {
				child = prefix + "/" + c.name
			}
			if err := w.writeTree(c, n.feBlock, child); err != nil {
				return err
			}
		}
		return nil
	}
	if n.size > 0 {
		n.absSector = w.partStart + uint64(n.dataLB)
		w.files[prefix] = Location{Sector: n.absSector, Length: n.size}
	}
	return w.writeFileEntry(n)
}

func (w *udfWriter) writeDirEntry(n *node, parentFE uint32) error {
	ents := []struct {
		name string
		fe   uint32
		dir  bool
	}{{"", parentFE, true}}
	for _, c := range n.children {
		ents = append(ents, struct {
			name string
			fe   uint32
			dir  bool
		}{c.name, c.feBlock, c.isDir})
	}
	var fid []byte
	for _, e := range ents {
		fid = appendFID(fid, n.dataLB, e.name, e.fe, e.dir)
	}
	for off := 0; off < len(fid); off += SectorSize {
		end := off + SectorSize
		if end > len(fid) {
			end = len(fid)
		}
		sector := make([]byte, SectorSize)
		copy(sector, fid[off:end])
		if err := w.writeSector(w.partStart+uint64(n.dataLB)+uint64(off/SectorSize), sector); err != nil {
			return err
		}
	}
	return w.writeSector(w.partStart+uint64(n.feBlock), w.fileEntry(n, fileTypeDirectory))
}

func (w *udfWriter) writeFileEntry(n *node) error {
	if n.size > 0 {
		if err := w.streamFile(n); err != nil {
			return err
		}
	}
	return w.writeSector(w.partStart+uint64(n.feBlock), w.fileEntry(n, fileTypeRegular))
}

func (w *udfWriter) streamFile(n *node) error {
	base := int64((w.partStart + uint64(n.dataLB)) * SectorSize)
	if n.data != nil {
		buf := n.data
		off := int64(0)
		for len(buf) > 0 {
			chunk := SectorSize
			if chunk > len(buf) {
				chunk = len(buf)
			}
			sec := make([]byte, SectorSize)
			copy(sec, buf[:chunk])
			if _, err := w.out.WriteAt(sec, base+off); err != nil {
				return err
			}
			buf = buf[chunk:]
			off += SectorSize
		}
		return nil
	}
	src, err := os.Open(n.srcPath)
	if err != nil {
		return fmt.Errorf("iso: open %s: %w", n.srcPath, err)
	}
	defer src.Close()
	buf := make([]byte, SectorSize)
	var off int64
	for {
		nr, rerr := src.Read(buf)
		if nr > 0 {
			if nr < len(buf) {
				for i := nr; i < len(buf); i++ {
					buf[i] = 0
				}
			}
			if _, err := w.out.WriteAt(buf, base+off); err != nil {
				return err
			}
			off += SectorSize
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return fmt.Errorf("iso: read %s: %w", n.srcPath, rerr)
		}
	}
}
