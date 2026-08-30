package future

import "github.com/blazium-games/blazium-toolchain/internal/platforms"

// Register reserves console ids. Implementations land in later releases.
func Register() {
	platforms.RegisterPlanned(platforms.Info{
		ID:          "ps3",
		Name:        "PlayStation 3",
		Status:      platforms.StatusPlanned,
		Description: "Reserved. Not implemented.",
	})
	platforms.RegisterPlanned(platforms.Info{
		ID:          "ps4",
		Name:        "PlayStation 4",
		Status:      platforms.StatusPlanned,
		Description: "Reserved. Not implemented.",
	})
	platforms.RegisterPlanned(platforms.Info{
		ID:          "n64",
		Name:        "Nintendo 64",
		Status:      platforms.StatusPlanned,
		Description: "Reserved. libdragon preview guest + Ares/Project64 validation. Not implemented.",
	})
}
