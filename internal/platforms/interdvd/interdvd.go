package interdvd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blazium-games/blazium-toolchain/internal/iso"
	"github.com/blazium-games/blazium-toolchain/internal/platforms"
)

const ID = "interdvd"

// Tool is the Interactive DVD ISO masterer.
type Tool struct{}

func New() *Tool { return &Tool{} }

func (t *Tool) Info() platforms.Info {
	return platforms.Info{
		ID:          ID,
		Name:        "Interactive DVD",
		Status:      platforms.StatusSupported,
		Commands:    []string{"setup", "env", "status", "iso", "meta"},
		Description: "ISO9660+UDF DVD-Video bridge. Pure Go; no mkisofs/oscdimg.",
	}
}

func (t *Tool) Setup(_ context.Context, _ platforms.SetupOptions) error {
	return nil
}

func (t *Tool) Env(_ platforms.CommonOptions) (platforms.EnvMap, error) {
	return platforms.EnvMap{"ISO_TOOL": "builtin"}, nil
}

func (t *Tool) Status(_ platforms.CommonOptions) (map[string]any, error) {
	return map[string]any{
		"platform":   ID,
		"ready":      true,
		"format":     "iso9660+udf",
		"iso_tool":   "builtin",
		"schema":     iso.SchemaV1,
	}, nil
}

func (t *Tool) Build(_ context.Context, _ platforms.BuildOptions) error {
	return fmt.Errorf("%w: interdvd has no build; author VIDEO_TS in the editor", platforms.ErrUsage)
}

func (t *Tool) Run(_ context.Context, _ platforms.RunOptions) error {
	return fmt.Errorf("%w: interdvd has no run; burn or play the ISO externally", platforms.ErrUsage)
}

func (t *Tool) ISO(_ context.Context, opts platforms.ISOOptions) error {
	m, err := MetaFromISOOptions(opts)
	if err != nil {
		return err
	}
	if opts.Out != "" {
		m.Out = opts.Out
	}
	res, err := iso.MasterFromMeta(m, m.Out)
	if err != nil {
		return err
	}
	resolved, rerr := m.Resolve()
	if rerr != nil {
		return rerr
	}
	if opts.Out != "" {
		resolved.Out = opts.Out
	}
	if opts.WriteMeta != "" {
		if err := iso.WriteMeta(opts.WriteMeta, resolved); err != nil {
			return err
		}
	}
	if opts.JSON {
		return json.NewEncoder(opts.Stdout).Encode(map[string]any{
			"out":     resolved.Out,
			"volume":  res.VolumeID,
			"title":   res.Title,
			"format":  "iso9660+udf",
			"sectors": res.TotalSectors,
			"extras":  res.Extras,
			"meta":    opts.Meta,
			"properties": map[string]any{
				"volume": res.VolumeID,
				"title":  res.Title,
			},
		})
	}
	fmt.Fprintf(opts.Stdout, "wrote %s (%d sectors, volume %s)\n", resolved.Out, res.TotalSectors, res.VolumeID)
	return nil
}

// MetaFromISOOptions loads --meta (if set) and applies explicit ISOOptions fields.
func MetaFromISOOptions(opts platforms.ISOOptions) (iso.DiscMeta, error) {
	var m iso.DiscMeta
	if opts.Meta != "" {
		loaded, err := iso.LoadMeta(opts.Meta)
		if err != nil {
			return m, err
		}
		m = loaded
	} else {
		m = iso.InitMeta()
		m.Volume = ""
		m.Preparer = ""
		m.Application = ""
		m.System = ""
		m.Provider = ""
		m.Disc = 0
		m.Discs = 0
		m.MenuLanguage = ""
		m.AudioLanguage = ""
		m.SubtitleLanguage = ""
	}
	if opts.Dir != "" {
		m.Dir = opts.Dir
	}
	if opts.Out != "" {
		m.Out = opts.Out
	}
	if opts.VolumeID != "" {
		m.Volume = opts.VolumeID
	}
	if opts.Title != "" {
		m.Title = opts.Title
	}
	if opts.Publisher != "" {
		m.Publisher = opts.Publisher
	}
	if opts.Preparer != "" {
		m.Preparer = opts.Preparer
	}
	if opts.Application != "" {
		m.Application = opts.Application
	}
	if opts.System != "" {
		m.System = opts.System
	}
	if opts.Provider != "" {
		m.Provider = opts.Provider
	}
	if opts.Copyright != "" {
		m.Copyright = opts.Copyright
	}
	if opts.CopyrightFile != "" {
		m.CopyrightFile = opts.CopyrightFile
	}
	if opts.License != "" {
		m.License = opts.License
	}
	if opts.LicenseFile != "" {
		m.LicenseFile = opts.LicenseFile
	}
	if opts.Readme != "" {
		m.Readme = opts.Readme
	}
	if opts.ReadmeFile != "" {
		m.ReadmeFile = opts.ReadmeFile
	}
	if opts.Credits != "" {
		m.Credits = opts.Credits
	}
	if opts.CreditsFile != "" {
		m.CreditsFile = opts.CreditsFile
	}
	if opts.Author != "" {
		m.Author = opts.Author
	}
	if opts.Studio != "" {
		m.Studio = opts.Studio
	}
	if opts.Website != "" {
		m.Website = opts.Website
	}
	if opts.Contact != "" {
		m.Contact = opts.Contact
	}
	if opts.Version != "" {
		m.Version = opts.Version
	}
	if opts.Catalog != "" {
		m.Catalog = opts.Catalog
	}
	if opts.Description != "" {
		m.Description = opts.Description
	}
	if opts.Abstract != "" {
		m.Abstract = opts.Abstract
	}
	if opts.AbstractFile != "" {
		m.AbstractFile = opts.AbstractFile
	}
	if opts.Bibliographic != "" {
		m.Bibliographic = opts.Bibliographic
	}
	if opts.BiblioFile != "" {
		m.BiblioFile = opts.BiblioFile
	}
	if opts.Created != "" {
		m.Created = opts.Created
	}
	if opts.Disc != 0 {
		m.Disc = opts.Disc
	}
	if opts.Discs != 0 {
		m.Discs = opts.Discs
	}
	if opts.RegionMask != 0 {
		m.RegionMask = opts.RegionMask
	}
	if opts.ParentalLevel != 0 {
		m.ParentalLevel = opts.ParentalLevel
	}
	if opts.MenuLanguage != "" {
		m.MenuLanguage = opts.MenuLanguage
	}
	if opts.AudioLanguage != "" {
		m.AudioLanguage = opts.AudioLanguage
	}
	if opts.SubtitleLanguage != "" {
		m.SubtitleLanguage = opts.SubtitleLanguage
	}
	if opts.AutorunLabel != "" {
		m.Autorun.Label = opts.AutorunLabel
	}
	if opts.AutorunOpen != "" {
		m.Autorun.Open = opts.AutorunOpen
	}
	if opts.AutorunIcon != "" {
		m.Autorun.Icon = opts.AutorunIcon
	}
	if opts.ExtrasDir != "" {
		m.ExtrasDir = opts.ExtrasDir
	}
	if opts.Recursive {
		m.Recursive = true
	}
	for _, spec := range opts.Extra {
		e := parseExtra(spec)
		e.Recursive = e.Recursive || m.Recursive
		m.Extras = append(m.Extras, e)
	}
	return m, nil
}

func parseExtra(spec string) iso.Extra {
	// same split as iso.parseExtraSpec but Extra is exported
	return extraFromSpec(spec)
}
