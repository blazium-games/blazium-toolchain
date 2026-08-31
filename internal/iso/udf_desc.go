package iso

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

const (
	fileTypeDirectory = 4
	fileTypeRegular   = 5
	eaHeaderLen       = 24
	fileTimesEALen    = 32
	filePermsReadAll  = 0x14A5
	maxExtentLen      = 0x3FFFF800
)

func putAnchor(b []byte, location uint32) {
	copy(b[16:], extentAD(vdsSectors*SectorSize, lbnMainVDS))
	copy(b[24:], extentAD(vdsSectors*SectorSize, lbnReserveVDS))
	putTag(b[:512], tagAnchorVolumePointer, location)
}

func (w *udfWriter) primaryVolumeDescriptor(loc uint32) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	copy(b[24:], encodeDString(w.props.VolumeID, 32))
	le.PutUint16(b[56:], uint16(w.props.Disc))
	le.PutUint16(b[58:], uint16(w.props.Discs))
	le.PutUint16(b[60:], 2)
	le.PutUint16(b[62:], 3)
	le.PutUint32(b[64:], 1)
	le.PutUint32(b[68:], 1)
	setID := fmt.Sprintf("%08X%s", uint32(w.now.Unix()), w.props.VolumeID)
	copy(b[72:], encodeDString(setID, 128))
	copy(b[200:], charSpec())
	copy(b[264:], charSpec())
	copy(b[376:], encodeTimestamp(w.now))
	copy(b[388:], implEntityID(w.props.Provider))
	putTag(b[:512], tagPrimaryVolume, loc)
	return b
}

func (w *udfWriter) implUseVolumeDescriptor(loc uint32) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	le.PutUint32(b[16:], 1)
	suffix := make([]byte, 8)
	le.PutUint16(suffix[0:], 0x0102)
	copy(b[20:], entityID("*UDF LV Info", suffix))
	copy(b[52:], charSpec())
	title := w.props.Title
	if title == "" {
		title = w.props.VolumeID
	}
	copy(b[116:], encodeDString(title, 128))
	copy(b[116+128:], encodeDString(w.props.Publisher, 36))
	copy(b[116+128+36*3:], implEntityID(w.props.Provider))
	putTag(b[:512], tagImplementationUseVol, loc)
	return b
}

func (w *udfWriter) partitionDescriptor(loc, partitionBlocks uint32) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	le.PutUint32(b[16:], 2)
	le.PutUint16(b[20:], 1)
	copy(b[24:], entityID("+NSR02", nil))
	le.PutUint32(b[184:], 1)
	le.PutUint32(b[188:], lbnPartitionLB)
	le.PutUint32(b[192:], partitionBlocks)
	copy(b[196:], implEntityID(w.props.Provider))
	putTag(b[:512], tagPartition, loc)
	return b
}

func (w *udfWriter) logicalVolumeDescriptor(loc uint32) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	le.PutUint32(b[16:], 3)
	copy(b[20:], charSpec())
	title := w.props.Title
	if title == "" {
		title = w.props.VolumeID
	}
	copy(b[84:], encodeDString(title, 128))
	le.PutUint32(b[212:], SectorSize)
	copy(b[216:], domainEntityID())
	copy(b[248:], longAD(2*SectorSize, 0, 0))
	le.PutUint32(b[264:], 6)
	le.PutUint32(b[268:], 1)
	copy(b[272:], implEntityID(w.props.Provider))
	copy(b[432:], extentAD(2*SectorSize, lbnIntegrity))
	b[440] = 1
	b[441] = 6
	le.PutUint16(b[442:], 1)
	putTag(b[:446], tagLogicalVolume, loc)
	return b
}

func (w *udfWriter) unallocatedSpaceDescriptor(loc uint32) []byte {
	b := make([]byte, SectorSize)
	binary.LittleEndian.PutUint32(b[16:], 4)
	putTag(b[:24], tagUnallocatedSpace, loc)
	return b
}

func terminatingDescriptor(loc uint32) []byte {
	b := make([]byte, SectorSize)
	putTag(b[:512], tagTerminating, loc)
	return b
}

func (w *udfWriter) integrityDescriptor(partitionBlocks, numFiles, numDirs uint32) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	copy(b[16:], encodeTimestamp(w.now))
	le.PutUint32(b[28:], 1)
	le.PutUint64(b[40:], 16)
	le.PutUint32(b[72:], 1)
	le.PutUint32(b[76:], 48)
	le.PutUint32(b[84:], partitionBlocks)
	iu := 80 + 8
	copy(b[iu:], implEntityID(w.props.Provider))
	le.PutUint32(b[iu+32:], numFiles)
	le.PutUint32(b[iu+36:], numDirs)
	le.PutUint16(b[iu+40:], 0x0102)
	le.PutUint16(b[iu+42:], 0x0102)
	le.PutUint16(b[iu+44:], 0x0102)
	putTag(b[:iu+48], tagLogicalVolumeInteg, lbnIntegrity)
	return b
}

func (w *udfWriter) fileEntry(n *node, fileType uint8) []byte {
	b := make([]byte, SectorSize)
	le := binary.LittleEndian
	le.PutUint16(b[16+4:], 4)
	le.PutUint16(b[16+8:], 1)
	b[16+11] = fileType
	le.PutUint32(b[36:], 0xFFFFFFFF)
	le.PutUint32(b[40:], 0xFFFFFFFF)
	le.PutUint32(b[44:], filePermsReadAll)
	linkCount := uint16(1)
	if n.isDir {
		linkCount = 1 + uint16(countChildDirs(n))
	}
	le.PutUint16(b[48:], linkCount)
	infoLen := uint64(n.dataLen)
	if fileType == fileTypeRegular {
		infoLen = uint64(n.size)
	}
	le.PutUint64(b[56:], infoLen)
	le.PutUint64(b[64:], uint64(blocks(infoLen)))
	copy(b[72:], encodeTimestamp(n.modTime))
	copy(b[84:], encodeTimestamp(n.modTime))
	copy(b[96:], encodeTimestamp(n.modTime))
	le.PutUint32(b[108:], 1)
	copy(b[128:], implEntityID(w.props.Provider))
	le.PutUint64(b[160:], n.uniqueID)
	const eaLen = eaHeaderLen + fileTimesEALen
	le.PutUint32(b[176+16:], eaLen)
	le.PutUint32(b[176+20:], eaLen)
	putTag(b[176:176+eaHeaderLen], tagExtendedAttrHeader, n.feBlock)
	ft := 176 + eaHeaderLen
	le.PutUint32(b[ft:], 5)
	b[ft+4] = 1
	le.PutUint32(b[ft+8:], fileTimesEALen)
	le.PutUint32(b[ft+12:], 12)
	le.PutUint32(b[ft+16:], 1)
	copy(b[ft+20:], encodeTimestamp(n.modTime))
	le.PutUint32(b[168:], eaLen)
	adStart := 176 + eaLen
	contentLen := adStart
	if infoLen > 0 {
		adOff := adStart
		block := n.dataLB
		remaining := infoLen
		for remaining > 0 {
			ext := remaining
			if ext > maxExtentLen {
				ext = maxExtentLen
			}
			copy(b[adOff:], shortAD(uint32(ext), block))
			block += blocks(ext)
			adOff += 8
			remaining -= ext
		}
		le.PutUint32(b[172:], uint32(adOff-adStart))
		contentLen = adOff
	}
	putTag(b[:contentLen], tagFileEntry, n.feBlock)
	return b
}

func countChildDirs(n *node) int {
	c := 0
	for _, ch := range n.children {
		if ch.isDir {
			c++
		}
	}
	return c
}

func dirFIDBytes(n *node) uint32 {
	off := fidLen("")
	for _, c := range n.children {
		off += fidLen(c.name)
	}
	return uint32(off)
}

func fidLen(name string) int {
	base := 38 + dcharsLen(name)
	return (base + 3) / 4 * 4
}

func dcharsLen(name string) int {
	if name == "" {
		return 0
	}
	eightBit := true
	n := 0
	for _, r := range name {
		if r > 0xFF {
			eightBit = false
		}
		n++
	}
	if eightBit {
		return 1 + n
	}
	return 1 + 2*len(utf16.Encode([]rune(name)))
}

func appendFID(buf []byte, baseLB uint32, name string, childFE uint32, isDir bool) []byte {
	dchars := encodeDChars(name)
	fid := make([]byte, fidLen(name))
	le := binary.LittleEndian
	le.PutUint16(fid[16:], 1)
	var chars uint8
	if name == "" {
		chars |= 0x08
	}
	if isDir {
		chars |= 0x02
	}
	fid[18] = chars
	fid[19] = byte(len(dchars))
	copy(fid[20:], longAD(SectorSize, childFE, 0))
	copy(fid[38:], dchars)
	tagLoc := baseLB + uint32(len(buf)/SectorSize)
	putTag(fid, tagFileIdentifier, tagLoc)
	return append(buf, fid...)
}

func encodeDChars(name string) []byte {
	if name == "" {
		return nil
	}
	eightBit := true
	for _, r := range name {
		if r > 0xFF {
			eightBit = false
			break
		}
	}
	if eightBit {
		out := make([]byte, 1+len(name))
		out[0] = 8
		copy(out[1:], name)
		return out
	}
	u := utf16Encode(name)
	out := make([]byte, 1+2*len(u))
	out[0] = 16
	for i, c := range u {
		binary.BigEndian.PutUint16(out[1+2*i:], c)
	}
	return out
}
