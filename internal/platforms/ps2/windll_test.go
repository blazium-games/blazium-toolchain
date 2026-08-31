package ps2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindHostDLLRequiresMatchingMachine(t *testing.T) {
	if findHostDLL("definitely-missing-blazium-dll.dll", 0x14c) != "" {
		t.Fatal("missing dll should be empty")
	}
}

func TestPeMachineRejectsText(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.exe")
	if err := os.WriteFile(p, []byte("gcc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := peMachine(p); err == nil {
		t.Fatal("expected pe open error")
	}
}
