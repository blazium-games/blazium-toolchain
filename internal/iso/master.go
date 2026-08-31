package iso

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Location is the absolute sector and byte length of a packed file.
type Location struct {
	Sector uint64
	Length int64
}

// Result is the mastered image layout.
type Result struct {
	TotalSectors uint64
	Files        map[string]Location
	VolumeID     string
	Title        string
	Extras       []string
}

// VolumeProps stamps ISO9660 / UDF descriptors.
type VolumeProps struct {
	VolumeID, Title, Publisher, Preparer, Application, System, Provider string
	CopyrightFile, AbstractFile, BibliographicFile                      string
	Created                                                             time.Time
	Disc, Discs                                                         int
}

// Master writes an ISO9660 + UDF 1.02 DVD-Video bridge from srcDir (must contain VIDEO_TS).
func Master(srcDir, outPath string, props VolumeProps, extras []Extra) (Result, error) {
	meta := DiscMeta{
		Dir:         srcDir,
		Out:         outPath,
		Volume:      props.VolumeID,
		Title:       props.Title,
		Publisher:   props.Publisher,
		Preparer:    props.Preparer,
		Application: props.Application,
		System:      props.System,
		Provider:    props.Provider,
		Disc:        props.Disc,
		Discs:       props.Discs,
		Extras:      extras,
	}
	if !props.Created.IsZero() {
		meta.Created = props.Created.UTC().Format(time.RFC3339)
	}
	return MasterFromMeta(meta, outPath)
}

// MasterFromMeta resolves meta, merges extras and generated docs, then writes the image.
func MasterFromMeta(meta DiscMeta, outPath string) (Result, error) {
	resolved, err := meta.Resolve()
	if err != nil {
		return Result{}, err
	}
	if outPath != "" {
		resolved.Out = outPath
	}
	if resolved.Dir == "" {
		return Result{}, fmt.Errorf("iso: dir is required")
	}
	if resolved.Out == "" {
		return Result{}, fmt.Errorf("iso: out is required")
	}
	return masterResolved(resolved)
}

func masterResolved(m DiscMeta) (Result, error) {
	extras, err := collectExtras(m)
	if err != nil {
		return Result{}, err
	}
	dests := map[string]bool{}
	for _, e := range extras {
		dests[strings.ToUpper(e.Disc)] = true
	}
	gens, props, err := generateDocs(m, dests)
	if err != nil {
		return Result{}, err
	}
	if err := checkAutorunTargets(m, dests); err != nil {
		return Result{}, err
	}
	if props.Disc == 0 {
		props.Disc = 1
	}
	if props.Discs == 0 {
		props.Discs = 1
	}
	now := props.Created
	if now.IsZero() {
		now = time.Now().UTC()
		props.Created = now
	}
	root, err := buildTree(m.Dir, extras, gens, now)
	if err != nil {
		return Result{}, err
	}
	assignISONames(root)

	f, err := os.Create(m.Out)
	if err != nil {
		return Result{}, fmt.Errorf("iso: create %s: %w", m.Out, err)
	}
	defer f.Close()

	udfRes, err := writeUDF(f, root, props)
	if err != nil {
		return Result{}, err
	}
	if err := writeISO9660(f, root, props, uint32(udfRes.TotalSectors)); err != nil {
		return Result{}, err
	}
	var extraNames []string
	walkFiles(root, "", func(rel string, _ *node) {
		top := strings.ToUpper(strings.Split(rel, "/")[0])
		if !reservedRoots[top] {
			extraNames = append(extraNames, rel)
		}
	})
	return Result{
		TotalSectors: udfRes.TotalSectors,
		Files:        udfRes.Files,
		VolumeID:     props.VolumeID,
		Title:        props.Title,
		Extras:       extraNames,
	}, nil
}

func assignISONames(n *node) {
	used := map[string]int{}
	for _, c := range n.children {
		c.isoName = mangle83(c.name, c.isDir, used)
		if c.isDir {
			assignISONames(c)
		}
	}
}
