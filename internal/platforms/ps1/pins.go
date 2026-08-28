package ps1

// Pinned component versions. This GPL project may vendor, cache, and
// redistribute these trees. Do not compile them into the MIT Blazium editor.

const (
	SDKVersion     = "0.24"
	GCCSeries      = "12.3.0"
	RecommendedISA = "-march=r3000 -msoft-float"
)

// Component is a piece this repo may contain, download, or discover locally.
type Component struct {
	ID        string `json:"id"`
	License   string `json:"license"`
	Required  bool   `json:"required"`
	Contained bool   `json:"contained"`
	Notes     string `json:"notes"`
}

func componentsForProfile(profile string) []Component {
	compile := []Component{
		{ID: "mipsel-none-elf-gcc", License: "GPLv3 (unmodified binaries)", Required: true, Contained: true, Notes: "Official PSn00b v0.24 pin " + GCCSeries},
		{ID: "psn00bsdk", License: "MPL 2.0", Required: true, Contained: true, Notes: "libpsn00b linked into guest only, version " + SDKVersion},
		{ID: "elf2x", License: "MPL 2.0", Required: true, Contained: true, Notes: "ELF to PS-X EXE"},
	}
	switch profile {
	case "compile":
		return compile
	case "dev":
		return append(compile,
			Component{ID: "openbios", License: "GPLv2 (pcsx-redux)", Required: true, Contained: true, Notes: "No Sony BIOS"},
			Component{ID: "pcsx-redux-cli", License: "GPLv2", Required: true, Contained: true, Notes: "Emulator player, spawn only"},
		)
	case "iso":
		return append(componentsForProfile("dev"),
			Component{ID: "mkpsxiso", License: "GPLv2+", Required: false, Contained: true, Notes: "ISO packer, spawn only"},
		)
	default:
		return compile
	}
}
