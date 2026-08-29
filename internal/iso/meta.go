package iso

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/embedfs"
)

// Extra is one host file or folder to pack beside VIDEO_TS.
type Extra struct {
	Host      string `json:"host"`
	Disc      string `json:"disc,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

// Autorun writes a root AUTORUN.INF when any field is set.
type Autorun struct {
	Label string `json:"label,omitempty"`
	Open  string `json:"open,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

func (a Autorun) set() bool {
	return a.Label != "" || a.Open != "" || a.Icon != ""
}

// DiscMeta is the blazium.interdvd.meta/v1 JSON document.
type DiscMeta struct {
	Schema            string  `json:"schema"`
	Dir               string  `json:"dir,omitempty"`
	Out               string  `json:"out,omitempty"`
	Volume            string  `json:"volume,omitempty"`
	Title             string  `json:"title,omitempty"`
	Publisher         string  `json:"publisher,omitempty"`
	Preparer          string  `json:"preparer,omitempty"`
	Application       string  `json:"application,omitempty"`
	System            string  `json:"system,omitempty"`
	Provider          string  `json:"provider,omitempty"`
	Copyright         string  `json:"copyright,omitempty"`
	CopyrightFile     string  `json:"copyright_file,omitempty"`
	License           string  `json:"license,omitempty"`
	LicenseFile       string  `json:"license_file,omitempty"`
	Readme            string  `json:"readme,omitempty"`
	ReadmeFile        string  `json:"readme_file,omitempty"`
	Credits           string  `json:"credits,omitempty"`
	CreditsFile       string  `json:"credits_file,omitempty"`
	Author            string  `json:"author,omitempty"`
	Studio            string  `json:"studio,omitempty"`
	Website           string  `json:"website,omitempty"`
	Contact           string  `json:"contact,omitempty"`
	Version           string  `json:"version,omitempty"`
	Catalog           string  `json:"catalog,omitempty"`
	Description       string  `json:"description,omitempty"`
	Abstract          string  `json:"abstract,omitempty"`
	AbstractFile      string  `json:"abstract_file,omitempty"`
	Bibliographic     string  `json:"bibliographic,omitempty"`
	BiblioFile        string  `json:"biblio_file,omitempty"`
	Created           string  `json:"created,omitempty"`
	Disc              int     `json:"disc,omitempty"`
	Discs             int     `json:"discs,omitempty"`
	RegionMask        int     `json:"region_mask,omitempty"`
	ParentalLevel     int     `json:"parental_level,omitempty"`
	MenuLanguage      string  `json:"menu_language,omitempty"`
	AudioLanguage     string  `json:"audio_language,omitempty"`
	SubtitleLanguage  string  `json:"subtitle_language,omitempty"`
	Autorun           Autorun `json:"autorun,omitempty"`
	Recursive         bool    `json:"recursive,omitempty"`
	ExtrasDir         string  `json:"extras_dir,omitempty"`
	Extras            []Extra `json:"extras,omitempty"`
	baseDir           string  // directory of the metadata file; not serialized
}

// InitMeta returns a template with defaults and empty extras.
func InitMeta() DiscMeta {
	if b, err := embedfs.InterDVDMetaTemplate(); err == nil {
		var m DiscMeta
		if json.Unmarshal(b, &m) == nil && (m.Schema == "" || m.Schema == SchemaV1) {
			if m.Schema == "" {
				m.Schema = SchemaV1
			}
			if m.Extras == nil {
				m.Extras = []Extra{}
			}
			return m
		}
	}
	return DiscMeta{
		Schema:            SchemaV1,
		Volume:            DefaultVolume,
		Preparer:          DefaultPreparer,
		Application:       DefaultApplication,
		System:            DefaultSystem,
		Provider:          DefaultProvider,
		Disc:              1,
		Discs:             1,
		MenuLanguage:      "en",
		AudioLanguage:     "en",
		SubtitleLanguage:  "en",
		Extras:            []Extra{},
	}
}

// LoadMeta reads a DiscMeta JSON file. Paths later resolve relative to the file.
func LoadMeta(path string) (DiscMeta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DiscMeta{}, fmt.Errorf("iso: read meta: %w", err)
	}
	var m DiscMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return DiscMeta{}, fmt.Errorf("iso: parse meta: %w", err)
	}
	if m.Schema != "" && m.Schema != SchemaV1 {
		return DiscMeta{}, fmt.Errorf("iso: unsupported meta schema %q", m.Schema)
	}
	m.Schema = SchemaV1
	m.baseDir = filepath.Dir(filepath.Clean(path))
	return m, nil
}

// WriteMeta writes pretty JSON. Paths under baseDir are stored relative when possible.
func WriteMeta(path string, m DiscMeta) error {
	m.Schema = SchemaV1
	m = m.withRelativePaths(filepath.Dir(filepath.Clean(path)))
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("iso: encode meta: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func (m DiscMeta) withRelativePaths(base string) DiscMeta {
	rel := func(p string) string {
		if p == "" || base == "" || !filepath.IsAbs(p) {
			return p
		}
		r, err := filepath.Rel(base, p)
		if err != nil || strings.HasPrefix(r, "..") {
			return p
		}
		return filepath.ToSlash(r)
	}
	m.Dir = rel(m.Dir)
	m.Out = rel(m.Out)
	m.CopyrightFile = rel(m.CopyrightFile)
	m.LicenseFile = rel(m.LicenseFile)
	m.ReadmeFile = rel(m.ReadmeFile)
	m.CreditsFile = rel(m.CreditsFile)
	m.AbstractFile = rel(m.AbstractFile)
	m.BiblioFile = rel(m.BiblioFile)
	m.ExtrasDir = rel(m.ExtrasDir)
	for i := range m.Extras {
		m.Extras[i].Host = rel(m.Extras[i].Host)
	}
	m.baseDir = ""
	return m
}

// Resolve fills defaults, sanitizes the volume, and makes host paths absolute
// relative to the metadata file (or cwd when baseDir is empty).
func (m DiscMeta) Resolve() (DiscMeta, error) {
	if m.Schema == "" {
		m.Schema = SchemaV1
	}
	m.Volume = SanitizeVolumeID(m.Volume)
	if m.Title == "" {
		m.Title = m.Volume
	}
	if m.Preparer == "" {
		m.Preparer = DefaultPreparer
	}
	if m.Application == "" {
		m.Application = DefaultApplication
	}
	if m.Provider == "" {
		m.Provider = DefaultProvider
	}
	if m.System == "" {
		m.System = DefaultSystem
	}
	if m.Disc == 0 {
		m.Disc = 1
	}
	if m.Discs == 0 {
		m.Discs = 1
	}
	if m.Disc < 1 || m.Discs < 1 || m.Disc > m.Discs {
		return m, fmt.Errorf("iso: disc %d of %d is invalid", m.Disc, m.Discs)
	}
	if m.RegionMask != 0 && (m.RegionMask < 1 || m.RegionMask > 255) {
		return m, fmt.Errorf("iso: region_mask must be 1–255 when set")
	}
	if m.ParentalLevel != 0 && (m.ParentalLevel < 1 || m.ParentalLevel > 8) {
		return m, fmt.Errorf("iso: parental_level must be 1–8 when set")
	}
	if m.Created != "" {
		if _, err := time.Parse(time.RFC3339, m.Created); err != nil {
			return m, fmt.Errorf("iso: created must be RFC3339: %w", err)
		}
	}
	abs := func(p string) string {
		if p == "" {
			return p
		}
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		base := m.baseDir
		if base == "" {
			if cwd, err := os.Getwd(); err == nil {
				base = cwd
			}
		}
		return filepath.Clean(filepath.Join(base, filepath.FromSlash(p)))
	}
	m.Dir = abs(m.Dir)
	m.Out = abs(m.Out)
	m.CopyrightFile = abs(m.CopyrightFile)
	m.LicenseFile = abs(m.LicenseFile)
	m.ReadmeFile = abs(m.ReadmeFile)
	m.CreditsFile = abs(m.CreditsFile)
	m.AbstractFile = abs(m.AbstractFile)
	m.BiblioFile = abs(m.BiblioFile)
	m.ExtrasDir = abs(m.ExtrasDir)
	for i := range m.Extras {
		m.Extras[i].Host = abs(m.Extras[i].Host)
		m.Extras[i].Recursive = m.Extras[i].Recursive || m.Recursive
	}
	return m, nil
}

func (m DiscMeta) createdTime() time.Time {
	if m.Created != "" {
		if t, err := time.Parse(time.RFC3339, m.Created); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func (m DiscMeta) volumeProps() VolumeProps {
	return VolumeProps{
		VolumeID:    m.Volume,
		Title:       m.Title,
		Publisher:   m.Publisher,
		Preparer:    m.Preparer,
		Application: m.Application,
		System:      m.System,
		Provider:    m.Provider,
		Created:     m.createdTime(),
		Disc:        m.Disc,
		Discs:       m.Discs,
	}
}
