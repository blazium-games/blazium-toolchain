package iso

import "encoding/binary"

const (
	tagPrimaryVolume        = 0x0001
	tagAnchorVolumePointer  = 0x0002
	tagImplementationUseVol = 0x0004
	tagPartition            = 0x0005
	tagLogicalVolume        = 0x0006
	tagUnallocatedSpace     = 0x0007
	tagTerminating          = 0x0008
	tagLogicalVolumeInteg   = 0x0009
	tagFileSet              = 0x0100
	tagFileIdentifier       = 0x0101
	tagFileEntry            = 0x0105
	tagExtendedAttrHeader   = 0x0106
	descriptorVersion       = 2
)

func crcCCITT(b []byte) uint16 {
	var crc uint16
	for _, v := range b {
		crc ^= uint16(v) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func putTag(desc []byte, ident uint16, tagLocation uint32) {
	le := binary.LittleEndian
	le.PutUint16(desc[0:], ident)
	le.PutUint16(desc[2:], descriptorVersion)
	desc[4] = 0
	desc[5] = 0
	le.PutUint16(desc[6:], 0)
	crcLen := uint16(len(desc) - 16)
	le.PutUint16(desc[8:], crcCCITT(desc[16:]))
	le.PutUint16(desc[10:], crcLen)
	le.PutUint32(desc[12:], tagLocation)
	var sum uint8
	for i := range 16 {
		if i == 4 {
			continue
		}
		sum += desc[i]
	}
	desc[4] = sum
}
