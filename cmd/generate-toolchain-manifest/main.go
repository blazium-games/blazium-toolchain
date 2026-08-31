package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blazium-games/blazium-toolchain/internal/cdnmanifest"
)

func main() {
	version := flag.String("version", "", "Toolchain semver to publish")
	released := flag.String("released-on", "", "RFC3339 release timestamp")
	base := flag.String("base-manifest", "", "Existing toolchain.json to merge")
	cerebroURL := flag.String("cerebro-manifest", "", "Optional Cerebro manifest URL")
	out := flag.String("out", "toolchain.json", "Output manifest path")
	artifactRoot := flag.String("artifacts", "artifacts", "Directory containing platform/arch subdirs")
	flag.Parse()

	if *version == "" {
		fmt.Fprintln(os.Stderr, "--version is required")
		os.Exit(1)
	}

	releasedOn := time.Now().UTC()
	if strings.TrimSpace(*released) != "" {
		parsed, err := time.Parse(time.RFC3339, *released)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --released-on: %v\n", err)
			os.Exit(1)
		}
		releasedOn = parsed.UTC()
	}

	doc := cdnmanifest.Document{Versions: map[string]cdnmanifest.Version{}}
	if data, err := os.ReadFile(*base); err == nil {
		doc, err = cdnmanifest.ParseDocument(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse base manifest: %v\n", err)
			os.Exit(1)
		}
	}
	if *cerebroURL != "" {
		if remote, err := fetchManifest(*cerebroURL); err == nil {
			doc = cdnmanifest.MergeDocuments(doc, remote)
		}
	}

	builds := []cdnmanifest.BuildInput{
		{
			Platform: "linux",
			Arch:     "x86_64",
			Filename: "blazium-toolchain",
			Path:     filepath.Join(*artifactRoot, "linux", "x86_64", "blazium-toolchain"),
			BaseURL:  fmt.Sprintf("https://cdn.blazium.app/toolchain/linux/x86_64/%s", *version),
			SigURL:   fmt.Sprintf("https://cdn.blazium.app/toolchain/linux/x86_64/%s/blazium-toolchain.sig", *version),
			Signing:  "gpg",
		},
		{
			Platform: "linux",
			Arch:     "x86_32",
			Filename: "blazium-toolchain",
			Path:     filepath.Join(*artifactRoot, "linux", "x86_32", "blazium-toolchain"),
			BaseURL:  fmt.Sprintf("https://cdn.blazium.app/toolchain/linux/x86_32/%s", *version),
			SigURL:   fmt.Sprintf("https://cdn.blazium.app/toolchain/linux/x86_32/%s/blazium-toolchain.sig", *version),
			Signing:  "gpg",
		},
		{
			Platform: "windows",
			Arch:     "x86_64",
			Filename: "blazium-toolchain.exe",
			Path:     filepath.Join(*artifactRoot, "windows", "x86_64", "blazium-toolchain.exe"),
			BaseURL:  fmt.Sprintf("https://cdn.blazium.app/toolchain/windows/x86_64/%s", *version),
			Signing:  "sslcom",
		},
		{
			Platform: "windows",
			Arch:     "x86_32",
			Filename: "blazium-toolchain.exe",
			Path:     filepath.Join(*artifactRoot, "windows", "x86_32", "blazium-toolchain.exe"),
			BaseURL:  fmt.Sprintf("https://cdn.blazium.app/toolchain/windows/x86_32/%s", *version),
			Signing:  "sslcom",
		},
	}
	var present []cdnmanifest.BuildInput
	for _, b := range builds {
		if _, err := os.Stat(b.Path); err == nil {
			present = append(present, b)
		}
	}
	if len(present) == 0 {
		fmt.Fprintln(os.Stderr, "no build artifacts found")
		os.Exit(1)
	}
	if err := cdnmanifest.MergeVersion(&doc, *version, releasedOn, present); err != nil {
		fmt.Fprintf(os.Stderr, "merge version: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
}

func fetchManifest(url string) (cdnmanifest.Document, error) {
	resp, err := http.Get(url)
	if err != nil {
		return cdnmanifest.Document{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cdnmanifest.Document{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return cdnmanifest.Document{}, err
	}
	return cdnmanifest.ParseDocument(body)
}
