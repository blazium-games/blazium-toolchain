package platforms_test

import (
	"errors"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/platforms"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/future"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/ps1"
	"github.com/blazium-games/blazium-toolchain/internal/platforms/ps2"
)

func TestListIncludesPS1AndPlanned(t *testing.T) {
	platforms.Register(ps1.New())
	platforms.Register(ps2.New())
	future.Register()
	list := platforms.List()
	ids := map[string]platforms.Status{}
	for _, info := range list {
		ids[info.ID] = info.Status
	}
	if ids["ps1"] != platforms.StatusSupported {
		t.Fatalf("ps1 status %q", ids["ps1"])
	}
	if ids["ps2"] != platforms.StatusSupported {
		t.Fatalf("ps2 status %q", ids["ps2"])
	}
	for _, id := range []string{"ps3", "ps4"} {
		if ids[id] != platforms.StatusPlanned {
			t.Fatalf("%s want planned, got %q", id, ids[id])
		}
	}
}

func TestLookupPlanned(t *testing.T) {
	future.Register()
	_, err := platforms.Lookup("ps3")
	if !errors.Is(err, platforms.ErrPlanned) {
		t.Fatalf("got %v", err)
	}
}

func TestLookupUnknown(t *testing.T) {
	_, err := platforms.Lookup("n64")
	if !errors.Is(err, platforms.ErrUnknownPlatform) {
		t.Fatalf("got %v", err)
	}
}
