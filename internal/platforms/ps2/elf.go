package ps2

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	elfClass32    = 1
	elfData2LSB   = 1
	elfHdrSize32  = 52
	elfShdrSize32 = 40
)

// TextSectionSize returns the ELF32 little-endian .text sh_size.
func TextSectionSize(path string) (uint32, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return textSectionSize(raw)
}

func textSectionSize(raw []byte) (uint32, error) {
	if len(raw) < elfHdrSize32 {
		return 0, fmt.Errorf("ELF too short (%d bytes)", len(raw))
	}
	if string(raw[:4]) != elfMagic {
		return 0, fmt.Errorf("not an ELF")
	}
	if raw[4] != elfClass32 {
		return 0, fmt.Errorf("need ELF32 (EI_CLASS=%d)", raw[4])
	}
	if raw[5] != elfData2LSB {
		return 0, fmt.Errorf("need little-endian ELF (EI_DATA=%d)", raw[5])
	}
	shoff := binary.LittleEndian.Uint32(raw[32:36])
	shentsize := binary.LittleEndian.Uint16(raw[46:48])
	shnum := binary.LittleEndian.Uint16(raw[48:50])
	shstrndx := binary.LittleEndian.Uint16(raw[50:52])
	if shentsize < elfShdrSize32 || shnum == 0 {
		return 0, fmt.Errorf("invalid ELF section header table")
	}
	if int(shstrndx) >= int(shnum) {
		return 0, fmt.Errorf("invalid e_shstrndx %d", shstrndx)
	}
	strHdrOff := int(shoff) + int(shstrndx)*int(shentsize)
	if strHdrOff < 0 || strHdrOff+elfShdrSize32 > len(raw) {
		return 0, fmt.Errorf("shstrtab header out of range")
	}
	strOff := int(binary.LittleEndian.Uint32(raw[strHdrOff+16 : strHdrOff+20]))
	strSize := int(binary.LittleEndian.Uint32(raw[strHdrOff+20 : strHdrOff+24]))
	if strOff < 0 || strSize < 0 || strOff+strSize > len(raw) {
		return 0, fmt.Errorf("shstrtab out of range")
	}
	strtab := raw[strOff : strOff+strSize]

	for i := 0; i < int(shnum); i++ {
		off := int(shoff) + i*int(shentsize)
		if off < 0 || off+elfShdrSize32 > len(raw) {
			return 0, fmt.Errorf("section header %d out of range", i)
		}
		nameOff := int(binary.LittleEndian.Uint32(raw[off : off+4]))
		if nameOff < 0 || nameOff >= len(strtab) {
			continue
		}
		name := cString(strtab[nameOff:])
		if name != ".text" {
			continue
		}
		return binary.LittleEndian.Uint32(raw[off+20 : off+24]), nil
	}
	return 0, fmt.Errorf("ELF has no .text section")
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
