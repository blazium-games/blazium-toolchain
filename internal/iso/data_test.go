package iso

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDataISOHasPVD(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SYSTEM.CNF"), []byte("BOOT2 = cdrom0:\\GAME.ELF;1\nVER = 1.00\nVMODE = NTSC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GAME.ELF"), []byte{0x7f, 'E', 'L', 'F', 0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "game.iso")
	if err := WriteDataISO(dir, out, "TESTPS2"); err != nil {
		t.Fatal(err)
	}
	if !HasISO9660PVD(out) {
		t.Fatal("missing CD001 PVD")
	}
}
