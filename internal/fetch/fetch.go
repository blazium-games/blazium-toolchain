package fetch

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"

	"github.com/klauspost/compress/zstd"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	return ExtractArchive(zipPath, destDir)
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

// ExtractArchive unpacks a zip or tar.xz into destDir.
func ExtractArchive(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return Unzip(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".tgz"), strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".pkg.tar.zst"):
		return extractTar(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive %s", filepath.Base(archivePath))
	}
}

func extractTar(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return extractTarGz(archivePath, destDir)
	}
	if strings.HasSuffix(lower, ".tar.zst") || strings.HasSuffix(lower, ".pkg.tar.zst") {
		return extractTarZstd(archivePath, destDir)
	}
	return extractTarSystem(archivePath, destDir)
}

func extractTarZstd(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		return err
	}
	defer zr.Close()
	return readTarToDir(zr, destDir)
}

func extractTarGz(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return readTarToDir(gz, destDir)
}

func readTarToDir(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	var links []tarLink
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			return materializeTarLinks(destDir, links)
		}
		if nextErr != nil {
			return nextErr
		}
		link, err := extractTarMember(tr, hdr, destDir)
		if err != nil {
			return err
		}
		if link != nil {
			links = append(links, *link)
		}
	}
}

type tarLink struct {
	dest     string
	linkname string
}

func extractTarMember(tr *tar.Reader, hdr *tar.Header, destDir string) (*tarLink, error) {
	name := filepath.FromSlash(hdr.Name)
	if name == "" || strings.HasPrefix(name, `..\`) || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("refusing tar path %q", hdr.Name)
	}
	target := filepath.Join(destDir, name)
	rel, err := filepath.Rel(destDir, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("refusing tar path %q", hdr.Name)
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return nil, os.MkdirAll(target, 0o755)
	case tar.TypeReg, tar.TypeRegA:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		return nil, closeErr
	case tar.TypeSymlink, tar.TypeLink:
		if hdr.Linkname == "" {
			return nil, nil
		}
		return &tarLink{dest: target, linkname: hdr.Linkname}, nil
	default:
		return nil, nil
	}
}

func materializeTarLinks(destDir string, links []tarLink) error {
	for _, l := range links {
		if fileExists(l.dest) {
			continue
		}
		src := l.linkname
		if !filepath.IsAbs(src) {
			src = filepath.Join(filepath.Dir(l.dest), filepath.FromSlash(l.linkname))
		}
		if !fileExists(src) {
			src = filepath.Join(destDir, filepath.FromSlash(l.linkname))
		}
		if !fileExists(src) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(l.dest), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(l.dest, data, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func extractTarSystem(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	args := []string{"-xf", archivePath, "-C", destDir}
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		args = []string{"-xzf", archivePath, "-C", destDir}
	}
	if strings.HasSuffix(lower, ".tar.xz") {
		args = []string{"-xJf", archivePath, "-C", destDir}
	}
	cmd := exec.Command("tar", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar extract %s: %w: %s", filepath.Base(archivePath), err, strings.TrimSpace(string(out)))
	}
	return nil
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
