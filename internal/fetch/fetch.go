package fetch

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTP downloads official unmodified zip assets.
type HTTP struct {
	Client *http.Client
}

func (h HTTP) client() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return &http.Client{Timeout: 15 * time.Minute}
}

// FetchZip downloads url to destDir's parent downloads folder, verifies SHA-256
// when wantSHA is non-empty, and extracts into destDir.
func (h HTTP) FetchZip(ctx context.Context, url, wantSHA, destDir string, log io.Writer) error {
	if url == "" {
		return fmt.Errorf("empty download url")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	downloads := filepath.Join(filepath.Dir(destDir), "downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		return err
	}
	name := filepath.Base(url)
	zipPath := filepath.Join(downloads, name)
	if err := h.download(ctx, url, zipPath, wantSHA, log); err != nil {
		return err
	}
	return Unzip(zipPath, destDir)
}

func (h HTTP) download(ctx context.Context, url, dest, wantSHA string, log io.Writer) error {
	if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
		if wantSHA == "" || hashFile(dest) == strings.ToLower(wantSHA) {
			if log != nil {
				fmt.Fprintf(log, "reusing %s\n", dest)
			}
			return nil
		}
		_ = os.Remove(dest)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "blazium-toolchain/0.1.0")
	resp, err := h.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %s", url, resp.Status)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	sum := sha256.New()
	w := io.MultiWriter(f, sum)
	if log != nil {
		fmt.Fprintf(log, "downloading %s\n", url)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if wantSHA != "" && got != strings.ToLower(wantSHA) {
		_ = os.Remove(tmp)
		return fmt.Errorf("sha256 mismatch for %s: got %s want %s", url, got, wantSHA)
	}
	return os.Rename(tmp, dest)
}

func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return ""
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// Unzip extracts zipPath into destDir, rejecting path traversal.
func Unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if err := extractFile(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(f *zip.File, destDir string) error {
	name := filepath.FromSlash(f.Name)
	if name == "" || strings.HasPrefix(name, `..\`) || strings.HasPrefix(name, "../") {
		return fmt.Errorf("refusing zip path %q", f.Name)
	}
	target := filepath.Join(destDir, name)
	rel, err := filepath.Rel(destDir, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing zip path %q", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
