package embedfs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManifestAndPins(t *testing.T) {
	m := MustManifest()
	if m.Name != "blazium-toolchain" || m.Version == "" || m.License != "GPL-3.0-or-later" {
		t.Fatalf("manifest %+v", m)
	}
	p := MustPins()
	if p.SDKVersion != "0.24" || p.GCCSeries != "12.3.0" {
		t.Fatalf("pins %+v", p)
	}
	for _, goos := range []string{"windows", "linux"} {
		if len(p.Compile[goos]) == 0 {
			t.Fatalf("compile pins missing for %s", goos)
		}
		for _, a := range p.Compile[goos] {
			if a.URL == "" || a.SHA256 == "" || a.Dest == "" {
				t.Fatalf("%s compile pin %+v", goos, a)
			}
		}
	}
	if len(p.Compile["darwin"]) != 0 {
		t.Fatal("darwin must not have compile zips")
	}
}

func TestRequiredFilesPresent(t *testing.T) {
	want := []string{
		pathManifest,
		pathPins,
		pathInterDVDMeta,
		pathLicense,
		pathNotice,
	}
	for _, name := range want {
		b, err := files.ReadFile(name)
		if err != nil || len(b) == 0 {
			t.Fatalf("%s: %v len=%d", name, err, len(b))
		}
	}
	var saw int
	_ = fs.WalkDir(files, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		saw++
		return nil
	})
	if saw < len(want) {
		t.Fatalf("embedded files %d want >= %d", saw, len(want))
	}
}

func TestRepoRootCopiesMatch(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	pairs := []struct{ embed, repo string }{
		{pathManifest, "manifest.json"},
		{pathLicense, "LICENSE"},
		{pathNotice, "NOTICE"},
	}
	for _, p := range pairs {
		got, err := files.ReadFile(p.embed)
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(root, p.repo))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s does not match repo-root %s", p.embed, p.repo)
		}
	}
}

func TestInterDVDTemplate(t *testing.T) {
	b, err := InterDVDMetaTemplate()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("blazium.interdvd.meta/v1")) {
		t.Fatalf("%s", b)
	}
}
