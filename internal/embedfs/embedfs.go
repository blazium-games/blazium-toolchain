// Package embedfs ships first-party files inside the CLI: catalog, fetch pins,
// InterDVD metadata template, and the project license texts.
//
// Guest C++ sources live in guest/ps1 (separate embed). GCC and SDK zips are
// not embedded; ps1 setup fetches them into the cache prefix.
package embedfs

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

//go:embed all:files
var files embed.FS

const (
	pathManifest     = "files/manifest.json"
	pathPins         = "files/pins.json"
	pathInterDVDMeta = "files/interdvd/meta.template.json"
	pathLicense      = "files/LICENSE"
	pathNotice       = "files/NOTICE"
)

// Manifest is the installer catalog (same schema as repo-root manifest.json).
type Manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	License     string `json:"license"`
	CLI         string `json:"cli"`
}

// ZipPin is one official unmodified archive the CLI may fetch.
type ZipPin struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Dest   string `json:"dest"`
}

// Pins are official download URLs and hashes for host tools.
type Pins struct {
	SDKVersion     string              `json:"sdk_version"`
	GCCSeries      string              `json:"gcc_series"`
	GCCReleaseBase string              `json:"gcc_release_base"`
	Compile        map[string][]ZipPin `json:"compile"`
	HostBuild      map[string][]ZipPin `json:"host_build"`
}

var (
	loadOnce sync.Once
	loadErr  error
	manifest Manifest
	pins     Pins
)

func load() {
	loadOnce.Do(func() {
		if err := decodeJSON(pathManifest, &manifest); err != nil {
			loadErr = err
			return
		}
		if err := decodeJSON(pathPins, &pins); err != nil {
			loadErr = err
		}
	})
}

func decodeJSON(name string, dest any) error {
	b, err := files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("embedfs: read %s: %w", name, err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("embedfs: parse %s: %w", name, err)
	}
	return nil
}

// FS is the embedded first-party tree (paths start with files/).
func FS() fs.FS {
	return files
}

// MustManifest is the embedded catalog. Panics if the embed is corrupt.
func MustManifest() Manifest {
	load()
	if loadErr != nil {
		panic(loadErr)
	}
	return manifest
}

// MustPins is the embedded fetch pin table. Panics if the embed is corrupt.
func MustPins() Pins {
	load()
	if loadErr != nil {
		panic(loadErr)
	}
	return pins
}

// ReadFile returns an embedded file (name relative to files/, e.g. "LICENSE").
func ReadFile(name string) ([]byte, error) {
	name = strings.TrimPrefix(name, "/")
	return files.ReadFile("files/" + name)
}

// InterDVDMetaTemplate is the JSON written by interdvd meta init.
func InterDVDMetaTemplate() ([]byte, error) {
	return files.ReadFile(pathInterDVDMeta)
}

// LicenseText is the GPL-3.0-or-later license file.
func LicenseText() ([]byte, error) {
	return files.ReadFile(pathLicense)
}

// NoticeText is the project NOTICE.
func NoticeText() ([]byte, error) {
	return files.ReadFile(pathNotice)
}
