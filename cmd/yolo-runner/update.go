package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/egv/yolo-runner/v2/internal/version"
)

const (
	defaultUpdateReleaseAPI = "https://api.github.com/repos/egv/yolo-runner"
	updateBackupSuffix      = ".yolo-runner-update.bak"
)

var updateCurrentExecutable = os.Executable

type updateRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []updateAsset `json:"assets"`
}

type updateAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type updateOptions struct {
	releaseTag string
	osName     string
	arch       string
	installDir string
	releaseAPI string
	checkOnly  bool
}

type updateInstallRecord struct {
	target string
	backup string
}

func runUpdate(args []string, stdout io.Writer, stderr io.Writer, client *http.Client) int {
	options, err := parseUpdateOptions(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if client == nil {
		client = &http.Client{}
	}

	if options.osName == "windows" {
		fmt.Fprintln(stderr, "Windows self-update is not currently supported. Install updates for yolo-runner tools using your distribution installer.")
		return 1
	}

	if options.checkOnly {
		latestTag, err := resolveUpdateLatestTag(client, options.releaseAPI)
		if err != nil {
			fmt.Fprintf(stderr, "failed to resolve latest release: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "current: %s\nlatest: %s\n", version.Version, latestTag)
		return 0
	}

	if options.installDir != "" {
		if err := validateWindowsUpdatePath(options.osName, options.installDir); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	resolvedRelease, err := resolveUpdateRelease(client, options.releaseTag, options.releaseAPI)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve release %q: %v\n", options.releaseTag, err)
		return 1
	}

	artifactName := updateArtifactName(options.osName, options.arch)
	checksumName := updateChecksumName(artifactName)
	artifactURL, checksumURL, err := updateReleaseAssetURLs(resolvedRelease, artifactName, checksumName)
	if err != nil {
		fmt.Fprintf(stderr, "failed to locate release assets: %v\n", err)
		return 1
	}

	tempDir, err := os.MkdirTemp("", "yolo-runner-update-")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(tempDir)

	artifactPath := filepath.Join(tempDir, artifactName)
	checksumPath := filepath.Join(tempDir, checksumName)
	if err := downloadAsset(client, artifactURL, artifactPath); err != nil {
		fmt.Fprintf(stderr, "failed to download artifact: %v\n", err)
		return 1
	}
	if err := downloadAsset(client, checksumURL, checksumPath); err != nil {
		fmt.Fprintf(stderr, "failed to download checksum: %v\n", err)
		return 1
	}
	if err := verifyArtifactChecksum(artifactPath, checksumPath, artifactName); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "failed to prepare extraction path: %v\n", err)
		return 1
	}
	if err := extractUpdateArtifact(artifactPath, extractDir); err != nil {
		fmt.Fprintf(stderr, "failed to extract artifact: %v\n", err)
		return 1
	}

	installDir := options.installDir
	if installDir == "" {
		installDir = updateDefaultInstallDir()
	}
	if err := validateWindowsUpdatePath(options.osName, installDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := ensureWritableUpdateDirectory(installDir); err != nil {
		fmt.Fprintf(stderr, "failed to use install directory %s: %v\n", installDir, err)
		return 1
	}

	if err := installUpdateArtifacts(extractDir, installDir, resolvedRelease.TagName); err != nil {
		fmt.Fprintf(stderr, "failed to install update: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"yolo-runner update: installed %s from %s\n",
		resolvedRelease.TagName,
		filepath.Join(installDir, updateBinaryName(options.osName)),
	)
	return 0
}

func parseUpdateOptions(args []string) (updateOptions, error) {
	fs := flag.NewFlagSet("yolo-runner update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	releaseTag := fs.String("release", "", "Release tag to install (or latest)")
	releaseTo := fs.String("to", "", "Release tag to install (or latest)")
	checkOnly := fs.Bool("check", false, "Only print current and latest versions")
	osInput := fs.String("os", "", "Target OS (linux, darwin, windows)")
	archInput := fs.String("arch", "", "Target architecture (amd64, arm64)")
	installDir := fs.String("install-dir", "", "Directory to place installed binaries")
	releaseAPI := fs.String("release-api", defaultUpdateReleaseAPI, "GitHub release API base URL")

	if err := fs.Parse(args); err != nil {
		return updateOptions{}, err
	}
	if *checkOnly {
		if strings.TrimSpace(*releaseTag) != "" && !strings.EqualFold(strings.TrimSpace(*releaseTag), "latest") {
			return updateOptions{}, errors.New("--check does not accept --release")
		}
		if strings.TrimSpace(*releaseTo) != "" {
			return updateOptions{}, errors.New("--check does not accept --to")
		}
	}
	if strings.TrimSpace(*releaseAPI) == "" {
		return updateOptions{}, errors.New("--release-api cannot be empty")
	}

	osName, err := resolveUpdateOS(*osInput)
	if err != nil {
		return updateOptions{}, err
	}
	arch, err := resolveUpdateArch(*archInput)
	if err != nil {
		return updateOptions{}, err
	}

	release, err := resolveReleaseTarget(*releaseTag, *releaseTo)
	if err != nil {
		return updateOptions{}, err
	}

	return updateOptions{
		releaseTag: release,
		osName:     osName,
		arch:       arch,
		installDir: strings.TrimSpace(*installDir),
		releaseAPI: strings.TrimRight(strings.TrimSpace(*releaseAPI), "/"),
		checkOnly:  *checkOnly,
	}, nil
}

func resolveReleaseTarget(releaseTag, releaseTo string) (string, error) {
	releaseTag = strings.TrimSpace(strings.ToLower(releaseTag))
	releaseTo = strings.TrimSpace(strings.ToLower(releaseTo))
	if releaseTag != "" && releaseTo != "" && releaseTag != "latest" {
		return "", errors.New("--release and --to are mutually exclusive")
	}
	if releaseTag == "" {
		releaseTag = releaseTo
	}
	if releaseTag == "" {
		return "latest", nil
	}
	if strings.EqualFold(releaseTag, "latest") {
		return "latest", nil
	}
	return updateNormalizeVersionTag(releaseTag), nil
}

func updateNormalizeVersionTag(tag string) string {
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag == "" {
		return tag
	}
	if strings.HasPrefix(tag, "v") {
		return tag
	}
	return "v" + tag
}

func resolveUpdateRelease(ctxClient *http.Client, releaseTag, apiBase string) (updateRelease, error) {
	metadataURL := updateMetadataURL(apiBase, releaseTag)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, nil)
	if err != nil {
		return updateRelease{}, err
	}
	updateApplyGitHubHeaders(req)
	resp, err := ctxClient.Do(req)
	if err != nil {
		return updateRelease{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return updateRelease{}, fmt.Errorf("github release API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release updateRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return updateRelease{}, err
	}
	if release.TagName == "" {
		release.TagName = releaseTag
	}
	if len(release.Assets) == 0 {
		return updateRelease{}, errors.New("release has no assets")
	}
	return release, nil
}

func resolveUpdateLatestTag(ctxClient *http.Client, apiBase string) (string, error) {
	metadataURL := updateMetadataURL(apiBase, "latest")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, nil)
	if err != nil {
		return "", err
	}
	updateApplyGitHubHeaders(req)
	resp, err := ctxClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github release API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release updateRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", errors.New("release has no tag_name")
	}
	return release.TagName, nil
}

func updateApplyGitHubHeaders(req *http.Request) {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "yolo-runner/update")
}

func updateMetadataURL(apiBase, releaseTag string) string {
	base := strings.TrimRight(apiBase, "/")
	switch strings.ToLower(strings.TrimSpace(releaseTag)) {
	case "latest", "":
		return base + "/releases/latest"
	default:
		return base + "/releases/tags/" + url.PathEscape(releaseTag)
	}
}

func updateReleaseAssetURLs(release updateRelease, artifactName, checksumName string) (string, string, error) {
	var artifactURL string
	var checksumURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case artifactName:
			artifactURL = asset.BrowserDownloadURL
		case checksumName:
			checksumURL = asset.BrowserDownloadURL
		}
	}
	if artifactURL == "" {
		return "", "", fmt.Errorf("missing release asset: %s", artifactName)
	}
	if checksumURL == "" {
		return "", "", fmt.Errorf("missing checksum asset: %s", checksumName)
	}
	return artifactURL, checksumURL, nil
}

func updateArtifactName(osName, arch string) string {
	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}
	return "yolo-runner_" + osName + "_" + arch + "." + ext
}

func updateBinaryName(osName string) string {
	if osName == "windows" {
		return "yolo-runner.exe"
	}
	return "yolo-runner"
}

func updateChecksumName(artifact string) string {
	return "checksums-" + artifact + ".txt"
}

func resolveUpdateOS(raw string) (string, error) {
	if raw == "" {
		return resolveUpdateOS(runtime.GOOS)
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "linux":
		return "linux", nil
	case "darwin", "macos", "osx":
		return "darwin", nil
	case "windows", "win", "win32":
		return "windows", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", raw)
	}
}

func resolveUpdateArch(raw string) (string, error) {
	if raw == "" {
		return resolveUpdateArch(runtime.GOARCH)
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "amd64", "x86_64":
		return "amd64", nil
	case "arm64", "aarch64":
		return "arm64", nil
	case "x64":
		return "amd64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", raw)
	}
}

func updateDefaultInstallDir() string {
	exe, err := updateCurrentExecutable()
	if err == nil && exe != "" {
		return filepath.Dir(exe)
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if home == "" {
		home = os.TempDir()
	}
	return filepath.Join(home, ".local", "bin")
}

func validateWindowsUpdatePath(osName, installDir string) error {
	if osName != "windows" {
		return nil
	}
	if isWindowsAbsolutePath(installDir) {
		return nil
	}
	return fmt.Errorf("unsupported Windows install path: %s; set --install-dir to an absolute Windows path", installDir)
}

func isWindowsAbsolutePath(p string) bool {
	if strings.HasPrefix(p, `\\`) {
		return true
	}
	if len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') {
		return true
	}
	return false
}

func ensureWritableUpdateDirectory(installDir string) error {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("cannot create install directory: %w", err)
	}
	testPath := filepath.Join(installDir, ".yolo-runner-update-write-test")
	f, err := os.Create(testPath)
	if err != nil {
		return fmt.Errorf("not writable (%v)", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("not writable (%v)", err)
	}
	if err := os.Remove(testPath); err != nil {
		return fmt.Errorf("cannot remove write test file: %w", err)
	}
	return nil
}

func downloadAsset(client *http.Client, url, destination string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	updateApplyGitHubHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))
	}

	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func verifyArtifactChecksum(artifactPath, checksumPath, artifactName string) error {
	expected, err := parseChecksumManifest(checksumPath, artifactName)
	if err != nil {
		return err
	}
	actual, err := checksumFileSHA256(artifactPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s", artifactName)
	}
	return nil
}

func parseChecksumManifest(checksumPath, artifactName string) (string, error) {
	contents, err := os.ReadFile(checksumPath)
	if err != nil {
		return "", fmt.Errorf("read checksum manifest: %w", err)
	}
	artifactBase := filepath.Base(artifactName)
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		candidate := filepath.Base(fields[1])
		candidate = strings.TrimPrefix(candidate, "dist/")
		if candidate == artifactBase || strings.EqualFold(candidate, artifactBase) {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("checksum entry not found for %s", artifactName)
}

func checksumFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func extractUpdateArtifact(artifactPath, dest string) error {
	if strings.HasSuffix(artifactPath, ".zip") {
		return extractUpdateZip(artifactPath, dest)
	}
	if strings.HasSuffix(artifactPath, ".tar.gz") {
		return extractUpdateTarGz(artifactPath, dest)
	}
	return fmt.Errorf("unsupported artifact type: %s", filepath.Base(artifactPath))
}

func extractUpdateZip(path, dest string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		targetPath := filepath.Join(dest, filepath.FromSlash(file.Name))
		if err := ensureUpdateTargetDir(targetPath); err != nil {
			return err
		}
		if err := copyZipFile(file, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func copyZipFile(file *zip.File, target string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.CreateTemp(filepath.Dir(target), ".yolo-runner-update-")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	_, err = io.Copy(out, src)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	mode := file.Mode()
	if err := os.Chmod(out.Name(), mode); err != nil {
		return err
	}
	return os.Rename(out.Name(), target)
}

func extractUpdateTarGz(path, dest string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		target := filepath.Join(dest, header.Name)
		if err := ensureUpdateTargetDir(target); err != nil {
			return err
		}
		out, err := os.CreateTemp(filepath.Dir(target), ".yolo-runner-update-")
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		if err := os.Chmod(out.Name(), header.FileInfo().Mode()); err != nil {
			_ = os.Remove(out.Name())
			return err
		}
		if err := os.Rename(out.Name(), target); err != nil {
			_ = os.Remove(out.Name())
			return err
		}
	}
	return nil
}

func ensureUpdateTargetDir(target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("prepare target dir for %s: %w", target, err)
	}
	return nil
}

func installUpdateArtifacts(extractDir, installDir, targetTag string) (err error) {
	relPaths, err := updateRelativeFiles(extractDir)
	if err != nil {
		return err
	}

	stagingDir, err := os.MkdirTemp(installDir, ".yolo-runner-update-stage-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	var installed []string
	var backups []updateInstallRecord

	defer func() {
		if err == nil {
			return
		}
		for i := len(installed) - 1; i >= 0; i-- {
			_ = os.Remove(installed[i])
		}
		for i := len(backups) - 1; i >= 0; i-- {
			_ = os.Remove(backups[i].target)
			_ = os.Rename(backups[i].backup, backups[i].target)
		}
	}()

	for _, relPath := range relPaths {
		src := filepath.Join(extractDir, relPath)
		staged := filepath.Join(stagingDir, relPath)
		if err := ensureUpdateTargetDir(staged); err != nil {
			return err
		}
		if err := copyFileForUpdate(src, staged); err != nil {
			return err
		}
	}

	for _, relPath := range relPaths {
		src := filepath.Join(stagingDir, relPath)
		dst := filepath.Join(installDir, relPath)
		if err := ensureUpdateTargetDir(dst); err != nil {
			return err
		}

		backupPath := dst + updateBackupSuffix
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("prepare rollback backup: %w", err)
		}
		if _, err := os.Stat(dst); err == nil {
			if err := os.Rename(dst, backupPath); err != nil {
				return fmt.Errorf("backup existing %s: %w", dst, err)
			}
			backups = append(backups, updateInstallRecord{target: dst, backup: backupPath})
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect target file %s: %w", dst, err)
		}

		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("install %s: %w", relPath, err)
		}
		installed = append(installed, dst)
	}

	if err := verifyInstalledBinariesVersion(installDir, relPaths, targetTag); err != nil {
		return err
	}

	for _, restored := range backups {
		if err := os.Remove(restored.backup); err != nil {
			return fmt.Errorf("cleanup backup %s: %w", restored.backup, err)
		}
	}
	return nil
}

func verifyInstalledBinariesVersion(installDir string, relPaths []string, targetTag string) error {
	target := strings.TrimSpace(targetTag)
	for _, relPath := range relPaths {
		path := filepath.Join(installDir, relPath)
		output, err := exec.Command(path, "--version").CombinedOutput()
		if err != nil {
			return fmt.Errorf("verify version for %s: %w", relPath, err)
		}
		fields := strings.Fields(string(output))
		if len(fields) == 0 {
			return fmt.Errorf("verify version for %s: no output", relPath)
		}
		if fields[len(fields)-1] != target {
			return fmt.Errorf("verify version for %s: expected %s, got %s", relPath, target, fields[len(fields)-1])
		}
	}
	return nil
}

func updateRelativeFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func copyFileForUpdate(src, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	srcInfo, err := file.Stat()
	if err != nil {
		return err
	}

	out, err := os.CreateTemp(filepath.Dir(dst), ".yolo-runner-update-")
	if err != nil {
		return err
	}
	tempPath := out.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(out, file); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, srcInfo.Mode()); err != nil {
		return err
	}
	if err := os.Rename(tempPath, dst); err != nil {
		return err
	}
	return nil
}

func updateIntFromEnv(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return def
	}
	return value
}
