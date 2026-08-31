package ps2

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func craftELF32MIPS(textSize uint32) []byte {
	const (
		ehSize = 52
		shSize = 40
		shNum  = 3
	)
	shstrtab := []byte("\x00.text\x00.shstrtab\x00")
	shoff := ehSize
	shstrtabOff := shoff + shSize*shNum
	textOff := shstrtabOff + len(shstrtab)
	buf := make([]byte, textOff+int(textSize))

	buf[0], buf[1], buf[2], buf[3] = 0x7f, 'E', 'L', 'F'
	buf[4] = elfClass32
	buf[5] = elfData2LSB
	buf[6] = 1
	binary.LittleEndian.PutUint16(buf[16:], 2) // ET_EXEC
	binary.LittleEndian.PutUint16(buf[18:], 8) // EM_MIPS
	binary.LittleEndian.PutUint32(buf[20:], 1)
	binary.LittleEndian.PutUint32(buf[32:], uint32(shoff))
	binary.LittleEndian.PutUint16(buf[40:], ehSize)
	binary.LittleEndian.PutUint16(buf[46:], shSize)
	binary.LittleEndian.PutUint16(buf[48:], shNum)
	binary.LittleEndian.PutUint16(buf[50:], 2)

	writeSH := func(idx int, name, typ, off, size uint32) {
		o := shoff + idx*shSize
		binary.LittleEndian.PutUint32(buf[o:], name)
		binary.LittleEndian.PutUint32(buf[o+4:], typ)
		binary.LittleEndian.PutUint32(buf[o+16:], off)
		binary.LittleEndian.PutUint32(buf[o+20:], size)
	}
	writeSH(0, 0, 0, 0, 0)
	writeSH(1, 1, 1, uint32(textOff), textSize)
	writeSH(2, 7, 3, uint32(shstrtabOff), uint32(len(shstrtab)))
	copy(buf[shstrtabOff:], shstrtab)
	for i := textOff; i < len(buf); i++ {
		buf[i] = 0x03
	}
	return buf
}

func TestTextSectionSizeCraftedELF(t *testing.T) {
	const want uint32 = 64
	raw := craftELF32MIPS(want)
	got, err := textSectionSize(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("text size %d want %d", got, want)
	}

	path := filepath.Join(t.TempDir(), "tiny.elf")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = TextSectionSize(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("file text size %d want %d", got, want)
	}
}

func TestTextSectionSizeRejectsGarbage(t *testing.T) {
	if _, err := textSectionSize([]byte("not an elf")); err == nil {
		t.Fatal("expected error")
	}
	raw := craftELF32MIPS(8)
	// Wipe .text name so the section is not found.
	copy(raw[52+40+0:], []byte{0, 0, 0, 0})
	if _, err := textSectionSize(raw); err == nil {
		t.Fatal("expected missing .text")
	}
}
