package fetch

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/blazium-games/blazium-toolchain/internal/settings"
)

func TestFetchZipAndUnzip(t *testing.T) {
	zipBytes, sum := mustTestZip(t, map[string]string{
		"bin/mipsel-none-elf-gcc": "gcc",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "gcc")
	var log bytes.Buffer
	err := HTTP{Client: srv.Client()}.FetchZip(context.Background(), srv.URL+"/gcc.zip", sum, dest, &log)
	if err != nil {
		t.Fatal(err)
	}
	got := filepath.Join(dest, "bin", "mipsel-none-elf-gcc")
	raw, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "gcc" {
		t.Fatalf("content %q", raw)
	}
}

func TestFetchRejectsFileURL(t *testing.T) {
	err := HTTP{}.FetchZip(context.Background(), "file:///C:/secret.zip", "", t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("got %v", err)
	}
}

func TestDownloadRespectsMaxBytes(t *testing.T) {
	t.Cleanup(settings.Reset)
	s := settings.Defaults()
	s.DownloadMaxBytes = 8
	settings.Set(s)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789abcdef"))
	}))
	t.Cleanup(srv.Close)
	err := HTTP{Client: srv.Client()}.FetchZip(context.Background(), srv.URL+"/big.zip", "", filepath.Join(t.TempDir(), "out"), nil)
	if err == nil || (!strings.Contains(err.Error(), "exceeded") && !strings.Contains(err.Error(), "max 8")) {
		t.Fatalf("got %v", err)
	}
}

func TestUnzipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "bad.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("no"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := Unzip(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestExtractTarGzSkipsSymlink(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "sdk.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	body := []byte("gcc")
	if err := tw.WriteHeader(&tar.Header{Name: "ee/bin/mips64r5900el-ps2-elf-gcc", Mode: 0755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "ps2sdk/ports/bin/bzcat", Typeflag: tar.TypeSymlink, Linkname: "bzip2", Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := ExtractArchive(archive, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "ee", "bin", "mips64r5900el-ps2-elf-gcc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "gcc" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "ps2sdk", "ports", "bin", "bzcat")); err == nil {
		got, err := os.ReadFile(filepath.Join(dest, "ps2sdk", "ports", "bin", "bzcat"))
		if err != nil || string(got) != "gcc" {
			t.Fatalf("hard/symlink should copy target or be skipped, got %q err %v", got, err)
		}
	}
}

func TestExtractTarGzMaterializesHardlink(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "sdk.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	body := []byte("real-gcc")
	if err := tw.WriteHeader(&tar.Header{Name: "ee/bin/gcc-15.exe", Mode: 0755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "ee/bin/gcc.exe", Typeflag: tar.TypeLink, Linkname: "ee/bin/gcc-15.exe", Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := ExtractArchive(archive, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "ee", "bin", "gcc.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "real-gcc" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractTarRejectsOutsideLink(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "sdk.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "ee/bin/stolen", Typeflag: tar.TypeLink, Linkname: secret, Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ExtractArchive(archive, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected outside link refusal")
	}
}

func TestExtractDeb(t *testing.T) {
	dir := t.TempDir()
	var tarbuf bytes.Buffer
	gz := gzip.NewWriter(&tarbuf)
	tw := tar.NewWriter(gz)
	body := []byte("mips-gcc")
	if err := tw.WriteHeader(&tar.Header{Name: "opt/libdragon/bin/mips64-elf-gcc", Mode: 0755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	data := tarbuf.Bytes()
	deb := buildDeb(t, "data.tar.gz", data)
	path := filepath.Join(dir, "gcc.deb")
	if err := os.WriteFile(path, deb, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := ExtractDeb(path, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "opt", "libdragon", "bin", "mips64-elf-gcc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "mips-gcc" {
		t.Fatalf("got %q", got)
	}
}

func buildDeb(t *testing.T, member string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("!<arch>\n")
	writeAr := func(name string, data []byte) {
		hdr := bytes.Repeat([]byte{' '}, 60)
		copy(hdr[0:16], []byte(name))
		copy(hdr[16:28], []byte("0"))
		copy(hdr[28:34], []byte("0"))
		copy(hdr[34:40], []byte("0"))
		copy(hdr[40:48], []byte("100644"))
		copy(hdr[48:58], []byte(strconv.Itoa(len(data))))
		hdr[58], hdr[59] = '`', '\n'
		buf.Write(hdr)
		buf.Write(data)
		if len(data)%2 == 1 {
			buf.WriteByte(0)
		}
	}
	writeAr("debian-binary", []byte("2.0\n"))
	writeAr(member, body)
	return buf.Bytes()
}

func mustTestZip(t *testing.T, files map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:])
}
