package cdnmanifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeVersion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "blazium-toolchain")
	if err := os.WriteFile(bin, []byte("toolchain-bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := Document{Versions: map[string]Version{}}
	err := MergeVersion(&doc, "0.1.0", time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC), []BuildInput{{
		Platform: "linux",
		Arch:     "x86_64",
		Filename: "blazium-toolchain",
		Path:     bin,
		BaseURL:  "https://cdn.blazium.app/toolchain/linux/x86_64/0.1.0",
		SigURL:   "https://cdn.blazium.app/toolchain/linux/x86_64/0.1.0/blazium-toolchain.sig",
		Signing:  "gpg",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Latest != "0.1.0" {
		t.Fatalf("latest=%s", doc.Latest)
	}
	if len(doc.Versions["0.1.0"].Downloads) != 1 {
		t.Fatalf("downloads=%d", len(doc.Versions["0.1.0"].Downloads))
	}
	dl := doc.Versions["0.1.0"].Downloads[0]
	if dl.DownloadURL != "https://cdn.blazium.app/toolchain/linux/x86_64/0.1.0/blazium-toolchain" {
		t.Fatalf("url=%s", dl.DownloadURL)
	}
	if dl.Size != 13 {
		t.Fatalf("size=%d", dl.Size)
	}
	if dl.Sha256 == "" {
		t.Fatal("empty sha256")
	}
}

func TestParseDocumentEmpty(t *testing.T) {
	doc, err := ParseDocument(nil)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Versions == nil {
		t.Fatal("expected versions map")
	}
}
