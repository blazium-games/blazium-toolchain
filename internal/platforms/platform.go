package platforms

import (
	"context"
	"io"
	"time"
)

// Status is the public lifecycle of a console target.
type Status string

const (
	StatusSupported Status = "supported"
	StatusPlanned   Status = "planned"
)

// Info is listed by `blazium-toolchain list` and the 3rd-party downloader.
type Info struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      Status   `json:"status"`
	Commands    []string `json:"commands"`
	Description string   `json:"description"`
}

// EnvMap is toolchain paths the editor or a parent installer can consume.
type EnvMap map[string]string

// Platform is one console family. PS1 and PS2 are implemented; others return ErrPlanned.
type Platform interface {
	Info() Info
	Setup(ctx context.Context, opts SetupOptions) error
	Env(opts CommonOptions) (EnvMap, error)
	Status(opts CommonOptions) (map[string]any, error)
	Build(ctx context.Context, opts BuildOptions) error
	Run(ctx context.Context, opts RunOptions) error
	ISO(ctx context.Context, opts ISOOptions) error
}

// CommonOptions apply to most commands.
type CommonOptions struct {
	Prefix string
	JSON   bool
	Stdout io.Writer
	Stderr io.Writer
}

// SetupOptions selects a download/layout profile.
type SetupOptions struct {
	CommonOptions
	Profile string
	Offline bool
}

// BuildOptions compile a guest with the platform SDK (spawn only).
type BuildOptions struct {
	CommonOptions
	Src       string
	Out       string
	Sample    string // template | gte; used when Src is empty
	Overlay   string // optional extra/replacement *.cpp on top of Src or the bundled stub
	ExportSrc string // write the resolved guest C++ tree here (export-guest snapshot)
	Tim    string // optional cooked TIM to embed (P3+)
	Mesh   string // optional cooked SVECTOR mesh to embed (P4+)
	Vag    string // optional cooked VAG to embed (P5+)
	Sprite string // optional cooked SPRITE table to embed (P8+)
	Script string // optional cooked SCRIPT.IR (ABI 2+)
	Gdbc   string // optional cooked SCRIPT.GD.BC
	Luau   string // optional cooked SCRIPT.LU.BC
	Str    string // optional cooked FMV00.STR
	Xa     string // optional cooked FMV00.XA
	Node   string // optional cooked NODE00.bin
	Hud    string // optional cooked HUD00.bin
	Tile   string // optional cooked TILE00.bin
	Scene  string // optional cooked SCENE00.bin
	Anim   string // optional cooked ANIM00.bin
	Cam    string // optional cooked CAM00.bin
	Hit    string // optional cooked HIT00.bin
	Nav    string // optional cooked NAV00.bin
	Path   string // optional cooked PATH00.bin
	Way    string // optional cooked WAY00.bin
}

// RunOptions boot the emulator.
type RunOptions struct {
	CommonOptions
	Exe     string
	ISO     string
	Timeout time.Duration // 0 means default 120s smoke watchdog
	Pcdrv   string        // host dir for -pcdrvbase; empty uses the EXE directory
	UI      bool          // show the pcsx-redux window; smoke stays headless
}

// ISOOptions spawn an ISO packer (ps1) or master a DVD-Video bridge (interdvd).
type ISOOptions struct {
	CommonOptions
	XML      string
	Dir      string
	Out      string
	VolumeID string
	Extra    []string
	ExtrasDir string
	Recursive bool
	Title, Publisher, Preparer, Application, System, Provider string
	Copyright, CopyrightFile, License, LicenseFile, Readme, ReadmeFile string
	Credits, CreditsFile, Author, Studio, Website, Contact, Version, Catalog, Description string
	Abstract, AbstractFile, Bibliographic, BiblioFile string
	Created string
	Disc, Discs int
	RegionMask, ParentalLevel int
	MenuLanguage, AudioLanguage, SubtitleLanguage string
	AutorunLabel, AutorunOpen, AutorunIcon string
	Meta, WriteMeta string
}
