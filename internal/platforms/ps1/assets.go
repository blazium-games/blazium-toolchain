package ps1

import (
	"context"
	"io"
	"runtime"
)

const gccReleaseBase = "https://github.com/Lameguy64/PSn00bSDK/releases/download/v0.24/"

// ZipAsset is an official unmodified archive this CLI may fetch.
type ZipAsset struct {
	ID     string
	URL    string
	SHA256 string
	Dest   string // relative to <prefix>/ps1/
}

// ZipFetcher downloads and unpacks a zip.
type ZipFetcher interface {
	FetchZip(ctx context.Context, url, sha256, destDir string, log io.Writer) error
}

func compileAssetsFor(goos string) []ZipAsset {
	switch goos {
	case "windows":
		return []ZipAsset{
			{
				ID:     "mipsel-none-elf-gcc",
				URL:    gccReleaseBase + "gcc-mipsel-none-elf-12.3.0-windows.zip",
				SHA256: windowsGCCSHA256,
				Dest:   "gcc",
			},
			{
				ID:     "psn00bsdk",
				URL:    gccReleaseBase + "PSn00bSDK-0.24-win32.zip",
				SHA256: windowsSDKSHA256,
				Dest:   "psn00bsdk",
			},
		}
	case "linux":
		return []ZipAsset{
			{
				ID:     "mipsel-none-elf-gcc",
				URL:    gccReleaseBase + "gcc-mipsel-none-elf-12.3.0-linux.zip",
				SHA256: linuxGCCSHA256,
				Dest:   "gcc",
			},
			{
				ID:     "psn00bsdk",
				URL:    gccReleaseBase + "PSn00bSDK-0.24-Linux.zip",
				SHA256: linuxSDKSHA256,
				Dest:   "psn00bsdk",
			},
		}
	default:
		return nil
	}
}

func defaultCompileAssets() []ZipAsset {
	return compileAssetsFor(runtime.GOOS)
}

// SHA-256 of the official v0.24 zips (unmodified). Empty means verify is skipped
// until the hash is recorded after a trusted first download.
const (
	windowsGCCSHA256 = "51df48145d5ef4b396d3932b72ccba538cc605c7c7745b32c58c277aa00523c5"
	windowsSDKSHA256 = "13a355afa89ecb2505882388841c42d9d231d40bbbe9ae22eb5e16abe19b2ea7"
	linuxGCCSHA256   = "228f031a25cf2687d8845fd1421f625bafc211fa27da428e458e80d030a726f8"
	linuxSDKSHA256   = "5ada7c9478d795b22bde96ee23d6869290c14a740bcde3d57c01c342a26ded35"

	windowsCMakeURL    = "https://github.com/Kitware/CMake/releases/download/v3.28.6/cmake-3.28.6-windows-x86_64.zip"
	windowsCMakeSHA256 = "a8f2e684ead94a64fd3517a38857a5b3f7f8d68d15c49ca1143d18797eaf9cac"
	windowsNinjaURL    = "https://github.com/ninja-build/ninja/releases/download/v1.12.1/ninja-win.zip"
	windowsNinjaSHA256 = ""

	// Official CMake Linux package is .tar.gz; FetchZip cannot unpack it. Linux
	// setup uses cmake on PATH. Ninja ships a zip.
	linuxCMakeURL    = "https://github.com/Kitware/CMake/releases/download/v3.28.6/cmake-3.28.6-linux-x86_64.tar.gz"
	linuxCMakeSHA256 = "931e3c0d546ee03ca72bb147ccd9b49e3b6252f765f66bf21b9d165519940458"
	linuxNinjaURL    = "https://github.com/ninja-build/ninja/releases/download/v1.12.1/ninja-linux.zip"
	linuxNinjaSHA256 = "6f98805688d19672bd699fbbfa2c2cf0fc054ac3df1f0e6a47664d963d530255"
)

func hostBuildAssets() []ZipAsset {
	switch runtime.GOOS {
	case "windows":
		return []ZipAsset{
			{ID: "cmake", URL: windowsCMakeURL, SHA256: windowsCMakeSHA256, Dest: "cmake"},
			{ID: "ninja", URL: windowsNinjaURL, SHA256: windowsNinjaSHA256, Dest: "ninja"},
		}
	case "linux":
		return []ZipAsset{
			{ID: "ninja", URL: linuxNinjaURL, SHA256: linuxNinjaSHA256, Dest: "ninja"},
		}
	default:
		return nil
	}
}
