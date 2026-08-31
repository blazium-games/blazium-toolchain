package iso

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type genFile struct {
	name    string
	data    []byte
	modTime time.Time
}

func generateDocs(m DiscMeta, dests map[string]bool) ([]genFile, VolumeProps, error) {
	now := m.createdTime()
	props := m.volumeProps()
	var files []genFile
	addText := func(name, body, host string, ident *string) error {
		if host == "" && strings.TrimSpace(body) == "" {
			return nil
		}
		key := strings.ToUpper(name)
		if dests[key] {
			return fmt.Errorf("iso: generated %s conflicts with an extra", name)
		}
		if reservedRoots[key] {
			return fmt.Errorf("iso: generated name %s is reserved", name)
		}
		var data []byte
		var mt time.Time
		if host != "" {
			b, err := os.ReadFile(host)
			if err != nil {
				return fmt.Errorf("iso: read %s: %w", host, err)
			}
			data = b
			if st, err := os.Stat(host); err == nil {
				mt = st.ModTime()
			}
		} else {
			data = []byte(body)
			if !strings.HasSuffix(body, "\n") {
				data = append(data, '\n')
			}
			mt = now
		}
		dests[key] = true
		files = append(files, genFile{name: name, data: data, modTime: mt})
		if ident != nil {
			*ident = name
		}
		return nil
	}

	if err := addText("COPYRIGHT.TXT", m.Copyright, m.CopyrightFile, &props.CopyrightFile); err != nil {
		return nil, props, err
	}
	if err := addText("LICENSE.TXT", m.License, m.LicenseFile, nil); err != nil {
		return nil, props, err
	}
	readme := m.Readme
	if readme == "" && m.ReadmeFile == "" {
		if hasIdentity(m) {
			readme = identityHeader(m)
		}
	} else if m.ReadmeFile == "" {
		h := identityHeader(m)
		if h != "" && !strings.HasPrefix(readme, h) {
			readme = h + "\n" + readme
		}
	}
	if err := addText("README.TXT", readme, m.ReadmeFile, nil); err != nil {
		return nil, props, err
	}
	credits := m.Credits
	if credits == "" && m.CreditsFile == "" && (m.Author != "" || m.Studio != "") {
		credits = strings.TrimSpace(m.Author + "\n" + m.Studio)
	}
	if err := addText("CREDITS.TXT", credits, m.CreditsFile, nil); err != nil {
		return nil, props, err
	}
	abstract := m.Abstract
	if abstract == "" {
		abstract = m.Description
	}
	if err := addText("ABSTRACT.TXT", abstract, m.AbstractFile, &props.AbstractFile); err != nil {
		return nil, props, err
	}
	if err := addText("BIBLIOGR.TXT", m.Bibliographic, m.BiblioFile, &props.BibliographicFile); err != nil {
		return nil, props, err
	}
	if m.Autorun.set() {
		if dests["AUTORUN.INF"] {
			return nil, props, fmt.Errorf("iso: generated AUTORUN.INF conflicts with an extra")
		}
		files = append(files, genFile{name: "AUTORUN.INF", data: []byte(autorunINF(m.Autorun)), modTime: now})
		dests["AUTORUN.INF"] = true
	}
	return files, props, nil
}

func hasIdentity(m DiscMeta) bool {
	return m.Author != "" || m.Studio != "" || m.Website != "" || m.Contact != "" ||
		m.Version != "" || m.Catalog != "" || m.Description != "" || m.Readme != ""
}

func identityHeader(m DiscMeta) string {
	var lines []string
	add := func(k, v string) {
		if v != "" {
			lines = append(lines, k+": "+v)
		}
	}
	add("Title", m.Title)
	add("Author", m.Author)
	add("Studio", m.Studio)
	add("Website", m.Website)
	add("Contact", m.Contact)
	add("Version", m.Version)
	add("Catalog", m.Catalog)
	if m.Description != "" && m.Abstract == "" {
		lines = append(lines, "", m.Description)
	}
	return strings.Join(lines, "\n")
}

func autorunINF(a Autorun) string {
	var b strings.Builder
	b.WriteString("[autorun]\r\n")
	if a.Label != "" {
		fmt.Fprintf(&b, "label=%s\r\n", a.Label)
	}
	if a.Open != "" {
		fmt.Fprintf(&b, "open=%s\r\n", strings.ReplaceAll(a.Open, "/", "\\"))
	}
	if a.Icon != "" {
		fmt.Fprintf(&b, "icon=%s\r\n", strings.ReplaceAll(a.Icon, "/", "\\"))
	}
	return b.String()
}

func checkAutorunTargets(m DiscMeta, dests map[string]bool) error {
	for _, p := range []string{m.Autorun.Open, m.Autorun.Icon} {
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(filepathToSlash(p), "/")
		top := strings.ToUpper(strings.Split(p, "/")[0])
		if reservedRoots[top] {
			continue // VIDEO_TS files exist after merge
		}
		if !dests[strings.ToUpper(p)] && !dests[strings.ToUpper(strings.Split(p, "/")[0])] {
			return fmt.Errorf("iso: autorun path %s is not packed", p)
		}
	}
	return nil
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
