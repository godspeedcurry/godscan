package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPickAsset(t *testing.T) {
	assets := []releaseAsset{
		{Name: "godscan_1.0.0_macos_arm64.tar.gz", URL: "bin-url"},
		{Name: "checksums.txt", URL: "sha-url"},
	}
	bin, sha, err := pickAsset(assets, "darwin", "arm64", "v1.0.0")
	if err != nil {
		t.Fatalf("pickAsset error: %v", err)
	}
	if bin.URL != "bin-url" || sha != "sha-url" {
		t.Fatalf("got %v %v", bin.URL, sha)
	}
}

func TestParseChecksum(t *testing.T) {
	hash := strings.Repeat("a", 64)
	content := hash + "  godscan-darwin-arm64.tar.gz\n"
	got, err := parseChecksum(content, "godscan-darwin-arm64.tar.gz")
	if err != nil {
		t.Fatalf("parseChecksum error: %v", err)
	}
	if got != hash {
		t.Fatalf("hash mismatch got %s", got)
	}
}

func TestInstallBinaryReplacesWithBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "godscan")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("write old: %v", err)
	}
	newFile := filepath.Join(dir, "newbin")
	if err := os.WriteFile(newFile, []byte("new"), 0o755); err != nil {
		t.Fatalf("write new: %v", err)
	}

	backup, err := installBinary(newFile, target)
	if err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "new" {
		t.Fatalf("target content: %s", data)
	}
	if backup == "" {
		t.Fatalf("expected backup path")
	}
	bdata, _ := os.ReadFile(backup)
	if string(bdata) != "old" {
		t.Fatalf("backup content: %s", bdata)
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "godscan.tar.gz")
	if err := makeTarGz(archive, map[string][]byte{
		"godscan": []byte("hello"),
		"LICENSE": []byte("license"),
	}); err != nil {
		t.Fatalf("make tar: %v", err)
	}
	bin, err := extractTarGz(archive)
	if err != nil {
		t.Fatalf("extract tar: %v", err)
	}
	data, _ := os.ReadFile(bin)
	if string(data) != "hello" {
		t.Fatalf("content: %s", data)
	}
	info, _ := os.Stat(bin)
	if info.Mode()&0o111 == 0 {
		t.Fatalf("binary not executable")
	}
}

func makeTarGz(dst string, files map[string][]byte) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, content := range files {
		mode := 0o755
		if strings.HasPrefix(name, "LICENSE") || strings.HasSuffix(strings.ToLower(name), ".txt") {
			mode = 0o644
		}
		hdr := &tar.Header{
			Name: name,
			Mode: int64(mode),
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}
	return nil
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "godscan.zip")
	if err := makeZip(archive, map[string][]byte{
		"godscan": []byte("zipbin"),
		"README":  []byte("text"),
	}); err != nil {
		t.Fatalf("make zip: %v", err)
	}
	bin, err := extractZip(archive)
	if err != nil {
		t.Fatalf("extract zip: %v", err)
	}
	data, _ := os.ReadFile(bin)
	if string(data) != "zipbin" {
		t.Fatalf("zip content: %s", data)
	}
	if info, _ := os.Stat(bin); info.Mode()&0o111 == 0 {
		t.Fatalf("zip binary not executable")
	}
}

func makeZip(dst string, files map[string][]byte) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(content); err != nil {
			return err
		}
	}
	return nil
}

func TestPrepareBinaryWithRawFile(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "godscan")
	if err := os.WriteFile(raw, []byte("raw"), 0o644); err != nil {
		t.Fatalf("write raw: %v", err)
	}
	got, err := prepareBinary(raw, "godscan")
	if err != nil {
		t.Fatalf("prepareBinary: %v", err)
	}
	data, _ := os.ReadFile(got)
	if string(data) != "raw" {
		t.Fatalf("content: %s", data)
	}
	if info, _ := os.Stat(got); info.Mode()&0o111 == 0 {
		t.Fatalf("raw binary not executable")
	}
}

func TestCurrentExecutablePath(t *testing.T) {
	p, err := currentExecutablePath()
	if err != nil {
		t.Fatalf("currentExecutablePath: %v", err)
	}
	if p == "" || !filepath.IsAbs(p) {
		t.Fatalf("unexpected path: %s", p)
	}
	_ = runtime.Version() // touch runtime to avoid unused import if build tags change
	_ = time.Second       // ensure time stays used
}
