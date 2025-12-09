package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var ErrAlreadyLatest = errors.New("already at latest version")

type SelfUpdateOptions struct {
	Owner          string
	Repo           string
	CurrentVersion string
	TargetVersion  string
	DownloadURL    string
	OS             string
	Arch           string
	DryRun         bool
	Force          bool
	SkipChecksum   bool
	HTTPClient     *http.Client
	UserAgent      string
	Token          string
}

type releaseInfo struct {
	Tag    string
	Assets []releaseAsset
}

type releaseAsset struct {
	Name string
	URL  string
}

// SelfUpdate downloads and atomically replaces the current executable.
// It returns the version that was checked/installed.
func SelfUpdate(ctx context.Context, opts SelfUpdateOptions) (string, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return "", fmt.Errorf("owner and repo are required")
	}
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "godscan-selfupdate"
	}

	var (
		selected releaseAsset
		tag      string
		shaURL   string
	)

	if opts.DownloadURL != "" {
		selected = releaseAsset{Name: path.Base(opts.DownloadURL), URL: opts.DownloadURL}
		tag = opts.TargetVersion
	} else {
		rel, err := fetchRelease(ctx, client, fetchReleaseOpts{
			Owner: opts.Owner,
			Repo:  opts.Repo,
			Tag:   opts.TargetVersion,
			Token: opts.Token,
			Agent: opts.UserAgent,
		})
		if err != nil {
			return "", err
		}
		tag = rel.Tag
		selected, shaURL, err = pickAsset(rel.Assets, opts.OS, opts.Arch, tag)
		if err != nil {
			return tag, err
		}
	}

	if tag != "" && tag == opts.CurrentVersion && !opts.Force {
		return tag, ErrAlreadyLatest
	}

	if opts.DryRun {
		Info("Latest version: %s; asset: %s", tag, selected.Name)
		return tag, nil
	}

	archivePath, err := downloadToTemp(ctx, client, selected.URL, opts.UserAgent)
	if err != nil {
		return tag, err
	}
	defer os.Remove(archivePath)

	if shaURL != "" && !opts.SkipChecksum {
		expected, err := downloadChecksum(ctx, client, shaURL, filepath.Base(selected.Name), opts.UserAgent)
		if err != nil {
			return tag, err
		}
		if err := verifySHA256(archivePath, expected); err != nil {
			return tag, err
		}
	} else if shaURL == "" {
		Warning("No checksum found for %s; proceeding without verification", selected.Name)
	}

	newBinaryPath, err := prepareBinary(archivePath, selected.Name)
	if err != nil {
		return tag, err
	}
	defer os.Remove(newBinaryPath)

	exePath, err := currentExecutablePath()
	if err != nil {
		return tag, err
	}
	backupPath, err := installBinary(newBinaryPath, exePath)
	if err != nil {
		return tag, err
	}
	if backupPath != "" {
		Info("Backup saved to %s", backupPath)
	}
	Success("Updated %s to %s", filepath.Base(exePath), tag)
	return tag, nil
}

type fetchReleaseOpts struct {
	Owner string
	Repo  string
	Tag   string
	Token string
	Agent string
}

func fetchRelease(ctx context.Context, client *http.Client, opts fetchReleaseOpts) (releaseInfo, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", opts.Owner, opts.Repo)
	if opts.Tag != "" && strings.ToLower(opts.Tag) != "latest" {
		endpoint = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", opts.Owner, opts.Repo, opts.Tag)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if opts.Agent != "" {
		req.Header.Set("User-Agent", opts.Agent)
	}
	if opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+opts.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("GitHub release query failed: %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return releaseInfo{}, err
	}

	out := releaseInfo{Tag: payload.TagName}
	for _, a := range payload.Assets {
		out.Assets = append(out.Assets, releaseAsset{Name: a.Name, URL: a.BrowserDownloadURL})
	}
	return out, nil
}

func pickAsset(assets []releaseAsset, osName, arch, tag string) (releaseAsset, string, error) {
	osName = normalizeOS(osName)
	arch = normalizeArch(arch)
	version := strings.TrimPrefix(tag, "v")

	candidates := []string{
		fmt.Sprintf("godscan-%s-%s.tar.gz", osName, arch),
		fmt.Sprintf("godscan_%s_%s_%s.tar.gz", version, osName, arch),
		fmt.Sprintf("godscan_%s_%s_%s.zip", version, osName, arch),
	}

	var (
		bin releaseAsset
		sha string
	)
	for _, a := range assets {
		for _, want := range candidates {
			if a.Name == want {
				bin = a
			}
			if a.Name == want+".sha256" || a.Name == want+".sha256.txt" {
				sha = a.URL
			}
		}
		if a.Name == "checksums.txt" {
			sha = a.URL
		}
	}
	if bin.Name == "" {
		return releaseAsset{}, "", fmt.Errorf("no asset for %s/%s (looked for %s)", osName, arch, strings.Join(candidates, ", "))
	}
	return bin, sha, nil
}

func downloadToTemp(ctx context.Context, client *http.Client, url, agent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if agent != "" {
		req.Header.Set("User-Agent", agent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "godscan-update-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	var downloaded int64
	contentLen := resp.ContentLength
	lastLogged := int64(-10)
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := tmp.Write(buf[:n]); err != nil {
				return "", err
			}
			downloaded += int64(n)
			if contentLen > 0 {
				percent := downloaded * 100 / contentLen
				if percent >= lastLogged+10 {
					Info("下载中 %d%% (%.2f/%.2f MB)", percent, float64(downloaded)/1e6, float64(contentLen)/1e6)
					lastLogged = percent
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return tmp.Name(), nil
}

func downloadChecksum(ctx context.Context, client *http.Client, url string, assetName string, agent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if agent != "" {
		req.Header.Set("User-Agent", agent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download failed: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	hash, err := parseChecksum(string(body), assetName)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func parseChecksum(content, assetName string) (string, error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields[0]) < 64 {
			continue
		}
		if len(fields) == 1 || strings.Contains(fields[len(fields)-1], assetName) {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s", assetName)
}

func normalizeOS(osName string) string {
	switch strings.ToLower(osName) {
	case "darwin":
		return "macos"
	default:
		return strings.ToLower(osName)
	}
}

func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	default:
		return arch
	}
}

func verifySHA256(filePath, expected string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, strings.TrimSpace(expected)) {
		return fmt.Errorf("checksum mismatch: got %s want %s", sum, expected)
	}
	return nil
}

func prepareBinary(archivePath, assetName string) (string, error) {
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractTarGz(archivePath)
	case strings.HasSuffix(assetName, ".zip"):
		return extractZip(archivePath)
	default:
	}

	out := filepath.Join(os.TempDir(), filepath.Base(assetName))
	if err := copyFile(archivePath, out, 0o755); err != nil {
		return "", err
	}
	return out, nil
}

func extractTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	tmpDir, err := os.MkdirTemp("", "godscan-bin-*")
	if err != nil {
		return "", err
	}

	var bestPath string
	var bestScore int = -1
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		binPath := filepath.Join(tmpDir, filepath.Base(hdr.Name))
		mode := hdr.FileInfo().Mode()
		if err := writeToFile(tr, binPath, mode); err != nil {
			return "", err
		}
		if mode&0o111 == 0 {
			if err := os.Chmod(binPath, mode|0o755); err != nil {
				return "", err
			}
		}
		score := scoreCandidate(filepath.Base(hdr.Name), mode)
		if score > bestScore {
			bestScore = score
			bestPath = binPath
		}
	}
	if bestPath == "" {
		return "", errors.New("no executable found in archive")
	}
	return bestPath, nil
}

func extractZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	tmpDir, err := os.MkdirTemp("", "godscan-bin-*")
	if err != nil {
		return "", err
	}

	var (
		bestPath  string
		bestScore = -1
	)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		target := filepath.Join(tmpDir, filepath.Base(f.Name))
		mode := f.Mode()
		if err := writeToFile(rc, target, mode); err != nil {
			rc.Close()
			return "", err
		}
		rc.Close()
		if mode&0o111 == 0 {
			_ = os.Chmod(target, mode|0o755)
		}
		score := scoreCandidate(filepath.Base(f.Name), mode)
		if score > bestScore {
			bestScore = score
			bestPath = target
		}
	}
	if bestPath == "" {
		return "", errors.New("no executable found in zip archive")
	}
	return bestPath, nil
}

func writeToFile(r io.Reader, path string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return nil
}

func scoreCandidate(name string, mode os.FileMode) int {
	score := 0
	lower := strings.ToLower(name)
	if mode&0o111 != 0 {
		score += 10
	}
	if strings.Contains(lower, "godscan") {
		score += 5
	}
	if strings.HasSuffix(lower, ".exe") {
		score += 3
	}
	switch {
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".md"), strings.HasPrefix(lower, "license"):
		score -= 5
	}
	return score
}

func installBinary(newBinaryPath, targetPath string) (string, error) {
	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if err := copyFile(newBinaryPath, tmpPath, 0o755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	tmp.Close()

	backup := targetPath + ".bak"
	if _, err := os.Stat(targetPath); err == nil {
		_ = os.Remove(backup)
		if err := os.Rename(targetPath, backup); err != nil {
			os.Remove(tmpPath)
			return "", fmt.Errorf("rename backup: %w", err)
		}
	} else {
		backup = ""
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Rename(backup, targetPath)
		return backup, fmt.Errorf("replace binary: %w", err)
	}
	_ = os.Chmod(targetPath, 0o755)
	return backup, nil
}

func currentExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return exe, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
