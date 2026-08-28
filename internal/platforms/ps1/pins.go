package ps1

// Pinned component versions. setup records these; actual HTTP fetch can be
// filled when official mirrors are published. Do not compile these into Blazium.

const (
	SDKVersion     = "0.24"
	GCCSeries      = "12.2"
	RecommendedISA = "-march=r3000 -msoft-float"
)

// Component is a downloadable (or locally discovered) piece.
type Component struct {
	ID       string `json:"id"`
	License  string `json:"license"`
	Required bool   `json:"required"`
	Notes    string `json:"notes"`
}

func componentsForProfile(profile string) []Component {
	compile := []Component{
		{ID: "mipsel-none-elf-gcc", License: "GPLv3 (unmodified binaries)", Required: true, Notes: "Spawn only; pin " + GCCSeries},
		{ID: "psn00bsdk", License: "MPL 2.0", Required: true, Notes: "libpsn00b linked into guest only, version " + SDKVersion},
		{ID: "elf2x", License: "PSn00bSDK tool", Required: true, Notes: "ELF to PS-X EXE"},
	}
	switch profile {
	case "compile":
		return compile
	case "dev":
		return append(compile,
			Component{ID: "openbios", License: "pcsx-redux tree", Required: true, Notes: "No Sony BIOS"},
			Component{ID: "pcsx-redux-cli", License: "pcsx-redux", Required: true, Notes: "Emulator player"},
		)
	case "iso":
		return append(componentsForProfile("dev"),
			Component{ID: "mkpsxiso", License: "GPLv2+", Required: false, Notes: "Spawn only; never link"},
		)
	default:
		return compile
	}
}
