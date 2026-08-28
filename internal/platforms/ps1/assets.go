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

func defaultCompileAssets() []ZipAsset {
	switch runtime.GOOS {
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
	default:
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
	}
}

// SHA-256 of the official v0.24 zips (unmodified). Empty means verify is skipped
// until the hash is recorded after a trusted first download.
const (
	windowsGCCSHA256 = "51df48145d5ef4b396d3932b72ccba538cc605c7c7745b32c58c277aa00523c5"
	windowsSDKSHA256 = "13a355afa89ecb2505882388841c42d9d231d40bbbe9ae22eb5e16abe19b2ea7"
	linuxGCCSHA256   = ""
	linuxSDKSHA256   = ""

	windowsCMakeURL    = "https://github.com/Kitware/CMake/releases/download/v3.28.6/cmake-3.28.6-windows-x86_64.zip"
	windowsCMakeSHA256 = ""
	windowsNinjaURL    = "https://github.com/ninja-build/ninja/releases/download/v1.12.1/ninja-win.zip"
	windowsNinjaSHA256 = ""
)

func hostBuildAssets() []ZipAsset {
	if runtime.GOOS != "windows" {
		return nil
	}
	return []ZipAsset{
		{ID: "cmake", URL: windowsCMakeURL, SHA256: windowsCMakeSHA256, Dest: "cmake"},
		{ID: "ninja", URL: windowsNinjaURL, SHA256: windowsNinjaSHA256, Dest: "ninja"},
	}
}
