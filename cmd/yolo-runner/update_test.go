package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/version"
)

func TestYoloRunnerUpdateResolvesLatestAndPinnedRelease(t *testing.T) {
	latestArtifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	pinnedArtifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.2.3"),
	})

	artifactName := updateArtifactName("linux", "amd64")
	checksumName := updateChecksumName(artifactName)
	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName: latestArtifact,
				checksumName: []byte(checksumText(t, artifactName, latestArtifact)),
			},
		},
		"/repos/egv/yolo-runner/releases/tags/v1.2.3": {
			tag: "v1.2.3",
			assets: map[string][]byte{
				artifactName: pinnedArtifact,
				checksumName: []byte(checksumText(t, artifactName, pinnedArtifact)),
			},
		},
	})
	defer server.Close()

	installDirLatest := t.TempDir()
	_, _, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDirLatest,
	}, nil)
	if code != 0 {
		t.Fatalf("latest update should succeed")
	}
	if !server.hasRequestedPathContaining("/repos/egv/yolo-runner/releases/latest") {
		t.Fatal("latest update should request /releases/latest")
	}
	content, err := os.ReadFile(filepath.Join(installDirLatest, "yolo-runner"))
	if err != nil {
		t.Fatalf("read latest binary: %v", err)
	}
	if got := strings.TrimSpace(string(content)); !strings.Contains(got, "yolo-runner v1.0.0") {
		t.Fatalf("latest binary content mismatch = %q", got)
	}

	installDirPinned := t.TempDir()
	_, _, code = runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "v1.2.3",
		"--os", "linux",
		"--arch", "amd64",
		"--install-dir", installDirPinned,
	}, nil)
	if code != 0 {
		t.Fatalf("pinned update should succeed")
	}
	if !server.hasRequestedPathContaining("/repos/egv/yolo-runner/releases/tags/v1.2.3") {
		t.Fatal("pinned update should request /releases/tags/v1.2.3")
	}
	content, err = os.ReadFile(filepath.Join(installDirPinned, "yolo-runner"))
	if err != nil {
		t.Fatalf("read pinned binary: %v", err)
	}
	if got := strings.TrimSpace(string(content)); !strings.Contains(got, "yolo-runner v1.2.3") {
		t.Fatalf("pinned binary content mismatch = %q", got)
	}
}

func TestYoloRunnerUpdateSelectsAssetByOSAndArch(t *testing.T) {
	linuxArtifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	darwinArtifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})

	linuxArtifactName := updateArtifactName("linux", "amd64")
	darwinArtifactName := updateArtifactName("darwin", "arm64")

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				linuxArtifactName:                      linuxArtifact,
				updateChecksumName(linuxArtifactName):  []byte(checksumText(t, linuxArtifactName, linuxArtifact)),
				darwinArtifactName:                     darwinArtifact,
				updateChecksumName(darwinArtifactName): []byte(checksumText(t, darwinArtifactName, darwinArtifact)),
			},
		},
	})
	defer server.Close()

	type expectation struct {
		name        string
		osInput     string
		archInput   string
		target      string
		wantContent string
		installDir  string
	}

	tests := []expectation{
		{
			name:        "linux amd64",
			osInput:     "Linux",
			archInput:   "x86_64",
			target:      "yolo-runner",
			wantContent: "v1.0.0",
			installDir:  t.TempDir(),
		},
		{
			name:        "darwin arm64",
			osInput:     "Darwin",
			archInput:   "arm64",
			target:      "yolo-runner",
			wantContent: "v1.0.0",
			installDir:  t.TempDir(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, code := runUpdateCommand(t, []string{
				"--release-api", server.URL() + "/repos/egv/yolo-runner",
				"--release", "latest",
				"--os", tc.osInput,
				"--arch", tc.archInput,
				"--install-dir", tc.installDir,
			}, nil)
			if code != 0 {
				t.Fatalf("expected update success for %s", tc.name)
			}
			content, err := os.ReadFile(filepath.Join(tc.installDir, tc.target))
			if err != nil {
				t.Fatalf("read installed binary: %v", err)
			}
			if got := strings.TrimSpace(string(content)); !strings.Contains(got, tc.wantContent) {
				t.Fatalf("installed content = %q", got)
			}
		})
	}
}

func TestYoloRunnerUpdateRejectsWindowsOS(t *testing.T) {
	artifact := writeZipArtifact(t, map[string][]byte{
		"yolo-runner.exe": []byte("not-used"),
	})
	artifactName := updateArtifactName("windows", "amd64")
	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(checksumText(t, artifactName, artifact)),
			},
		},
	})
	defer server.Close()

	_, stderr, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Windows",
		"--arch", "amd64",
	}, nil)
	if code == 0 {
		t.Fatalf("windows update should fail: %q", stderr)
	}
	if !strings.Contains(stderr, "Windows self-update is not currently supported") {
		t.Fatalf("expected unsupported windows error, got %q", stderr)
	}
}

func TestYoloRunnerUpdateCheckChecksLatest(t *testing.T) {
	originalVersion := version.Version
	version.Version = "runner-current"
	t.Cleanup(func() {
		version.Version = originalVersion
	})

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag:    "v2.5.1",
			assets: map[string][]byte{},
		},
	})
	defer server.Close()

	stdout, stderr, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--check",
	}, nil)
	if code != 0 {
		t.Fatalf("check should succeed: %q", stderr)
	}
	if !strings.Contains(stdout, "current: runner-current") {
		t.Fatalf("expected current version in check output, got %q", stdout)
	}
	if !strings.Contains(stdout, "latest: v2.5.1") {
		t.Fatalf("expected latest version in check output, got %q", stdout)
	}
}

func TestYoloRunnerUpdateUsesGitHubTokenForRequests(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_example-token")

	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	artifactName := updateArtifactName("linux", "amd64")

	checksum := checksumText(t, artifactName, artifact)
	authHeaders := make([]string, 0, 3)

	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/repos/egv/yolo-runner/releases/latest":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(updateRelease{
				TagName: "v1.0.0",
				Assets: []updateAsset{
					{Name: artifactName, BrowserDownloadURL: ts.URL + "/assets/v1.0.0/" + artifactName},
					{Name: updateChecksumName(artifactName), BrowserDownloadURL: ts.URL + "/assets/v1.0.0/" + updateChecksumName(artifactName)},
				},
			})
		case "/assets/v1.0.0/" + artifactName:
			_, _ = w.Write(artifact)
		case "/assets/v1.0.0/" + updateChecksumName(artifactName):
			_, _ = w.Write([]byte(checksum))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	installDir := t.TempDir()
	stdout, stderr, code := runUpdateCommand(t, []string{
		"--release-api", ts.URL + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code != 0 {
		t.Fatalf("update should succeed: %q", strings.TrimSpace(stderr))
	}
	if !strings.Contains(stdout, "v1.0.0") {
		t.Fatalf("expected install output to mention release version, got %q", stdout)
	}
	if strings.Contains(stdout, "ghp_example-token") || strings.Contains(stderr, "ghp_example-token") {
		t.Fatalf("auth token leaked in command output")
	}
	foundAuth := false
	for _, header := range authHeaders {
		if header == "Bearer ghp_example-token" {
			foundAuth = true
			break
		}
	}
	if !foundAuth {
		t.Fatalf("expected Authorization header on GitHub requests")
	}
}

func TestYoloRunnerUpdateVerifiesChecksumPassAndFail(t *testing.T) {
	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	artifactName := updateArtifactName("linux", "amd64")
	goodChecksum := checksumText(t, artifactName, artifact)
	badChecksum := strings.Repeat("0", 64)

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(goodChecksum),
			},
		},
		"/repos/egv/yolo-runner/releases/tags/v1.0.1": {
			tag: "v1.0.1",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte("" + badChecksum + "  dist/" + artifactName + "\n"),
			},
		},
	})
	defer server.Close()

	installDir := t.TempDir()
	_, stderr, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code != 0 {
		t.Fatalf("valid checksum should succeed: %q", stderr)
	}
	installed, err := os.ReadFile(filepath.Join(installDir, "yolo-runner"))
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if !strings.Contains(string(installed), "v1.0.0") {
		t.Fatalf("expected stable binary, got %q", string(installed))
	}

	// Start from a known baseline to confirm rollback on checksum failure.
	baselineBinary := filepath.Join(installDir, "yolo-runner")
	if err := os.WriteFile(baselineBinary, []byte("pinned-old"), 0o644); err != nil {
		t.Fatalf("seed baseline binary: %v", err)
	}
	_, stderr, code = runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "v1.0.1",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code == 0 {
		t.Fatalf("bad checksum should fail")
	}
	if !strings.Contains(stderr, "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %q", stderr)
	}
	current, err := os.ReadFile(baselineBinary)
	if err != nil {
		t.Fatalf("read baseline binary after failed update: %v", err)
	}
	if got := strings.TrimSpace(string(current)); got != "pinned-old" {
		t.Fatalf("expected rollback to baseline binary, got %q", got)
	}
}

func TestYoloRunnerUpdateRollsBackOnInstallFailure(t *testing.T) {
	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner":           updateBinaryScript(t, "yolo-runner", "v1.0.0"),
		"zzz-blocked/block.bin": updateBinaryScript(t, "zzz-blocked/block.bin", "v9.9.9"),
	})
	artifactName := updateArtifactName("linux", "amd64")

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(checksumText(t, artifactName, artifact)),
			},
		},
	})
	defer server.Close()

	installDir := t.TempDir()
	oldBinary := filepath.Join(installDir, "yolo-runner")
	if err := os.WriteFile(oldBinary, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed existing binary: %v", err)
	}
	_, stderr, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code == 0 {
		t.Fatalf("expected install failure for rollback test: %q", stderr)
	}
	if !strings.Contains(stderr, "verify version for zzz-blocked/block.bin") {
		t.Fatalf("expected version verification failure for rollback test: %q", stderr)
	}
	content, err := os.ReadFile(oldBinary)
	if err != nil {
		t.Fatalf("read post-failure binary: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != "existing" {
		t.Fatalf("expected existing binary to remain, got %q", got)
	}
	if _, err := os.Stat(oldBinary + updateBackupSuffix); err == nil {
		t.Fatalf("expected rollback backup to be removed")
	}
}

func TestYoloRunnerUpdateDefaultInstallDirDetection(t *testing.T) {
	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	artifactName := updateArtifactName("linux", "amd64")

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(checksumText(t, artifactName, artifact)),
			},
		},
	})
	defer server.Close()

	runningDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(runningDir, "yolo-runner"), []byte(""), 0o644)
	origExecutable := updateCurrentExecutable
	t.Cleanup(func() {
		updateCurrentExecutable = origExecutable
	})
	updateCurrentExecutable = func() (string, error) {
		return filepath.Join(runningDir, "yolo-runner"), nil
	}

	_, _, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
	}, nil)
	if code != 0 {
		t.Fatalf("default install dir update should succeed")
	}
	content, err := os.ReadFile(filepath.Join(runningDir, "yolo-runner"))
	if err != nil {
		t.Fatalf("read installed binary from default dir: %v", err)
	}
	if !strings.Contains(string(content), "v1.0.0") {
		t.Fatalf("unexpected binary content: %q", string(content))
	}
}

func TestYoloRunnerUpdateReportsNonWritableInstallDir(t *testing.T) {
	if runtime.GOOS == "linux" && os.Geteuid() == 0 {
		t.Skip("permission checks are not reliable as root")
	}

	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v1.0.0"),
	})
	artifactName := updateArtifactName("linux", "amd64")

	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/latest": {
			tag: "v1.0.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(checksumText(t, artifactName, artifact)),
			},
		},
	})
	defer server.Close()

	installDir := t.TempDir()
	if err := os.Chmod(installDir, 0o500); err != nil {
		t.Fatalf("chmod non-writable dir: %v", err)
	}
	_, stderr, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--release", "latest",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code == 0 {
		t.Fatalf("expected non-writable install dir to fail: %q", stderr)
	}
	if !strings.Contains(stderr, "not writable") {
		t.Fatalf("expected writable error, got %q", stderr)
	}
}

func TestYoloRunnerUpdateRejectsUnsupportedWindowsInstallPath(t *testing.T) {
	if err := validateWindowsUpdatePath("windows", "relative/path"); err == nil {
		t.Fatalf("expected relative path to be rejected for windows install path")
	}
}

func TestYoloRunnerUpdateAcceptsPinnedReleaseWithoutVPrefix(t *testing.T) {
	artifact := writeTarArtifact(t, map[string][]byte{
		"yolo-runner": updateBinaryScript(t, "yolo-runner", "v2.4.0"),
	})
	artifactName := updateArtifactName("linux", "amd64")
	server := newUpdateTestServer(t, map[string]testReleaseFixture{
		"/repos/egv/yolo-runner/releases/tags/v2.4.0": {
			tag: "v2.4.0",
			assets: map[string][]byte{
				artifactName:                     artifact,
				updateChecksumName(artifactName): []byte(checksumText(t, artifactName, artifact)),
			},
		},
	})
	defer server.Close()

	installDir := t.TempDir()
	_, _, code := runUpdateCommand(t, []string{
		"--release-api", server.URL() + "/repos/egv/yolo-runner",
		"--to", "2.4.0",
		"--os", "Linux",
		"--arch", "amd64",
		"--install-dir", installDir,
	}, nil)
	if code != 0 {
		t.Fatalf("pinned --to release should succeed")
	}
	if !server.hasRequestedPathContaining("/repos/egv/yolo-runner/releases/tags/v2.4.0") {
		t.Fatalf("expected pinned release request with v2.4.0")
	}
	content, err := os.ReadFile(filepath.Join(installDir, "yolo-runner"))
	if err != nil {
		t.Fatalf("read pinned binary: %v", err)
	}
	if got := strings.TrimSpace(string(content)); !strings.Contains(got, "yolo-runner v2.4.0") {
		t.Fatalf("expected pinned binary content, got %q", got)
	}
}

type testReleaseFixture struct {
	tag    string
	assets map[string][]byte
}

func newUpdateTestServer(t *testing.T, releases map[string]testReleaseFixture) *updateTestServer {
	t.Helper()

	server := &updateTestServer{
		assets: make(map[string][]byte),
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.recordPath(r.URL.Path)

		if release, ok := releases[r.URL.Path]; ok {
			assets := make([]updateAsset, 0, len(release.assets))
			for name := range release.assets {
				assets = append(assets, updateAsset{
					Name:               name,
					BrowserDownloadURL: server.baseURL() + updateTestAssetPath(release.tag, name),
				})
			}
			sort.Slice(assets, func(i, j int) bool {
				return assets[i].Name < assets[j].Name
			})
			_ = json.NewEncoder(w).Encode(updateRelease{
				TagName: release.tag,
				Assets:  assets,
			})
			return
		}

		if data, ok := server.assets[r.URL.Path]; ok {
			_, _ = w.Write(data)
			return
		}

		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	server.base = ts.URL
	server.server = ts

	for _, fixture := range releases {
		for name, data := range fixture.assets {
			server.assets[updateTestAssetPath(fixture.tag, name)] = data
		}
	}

	return server
}

func updateTestAssetPath(tag, assetName string) string {
	return "/assets/" + url.PathEscape(tag) + "/" + url.PathEscape(assetName)
}

type updateTestServer struct {
	server  *httptest.Server
	base    string
	pathsMu sync.Mutex
	paths   []string
	assets  map[string][]byte
}

func (s *updateTestServer) Close() {
	s.server.Close()
}

func (s *updateTestServer) URL() string {
	return s.base
}

func (s *updateTestServer) baseURL() string {
	return s.base
}

func (s *updateTestServer) recordPath(path string) {
	s.pathsMu.Lock()
	defer s.pathsMu.Unlock()
	s.paths = append(s.paths, path)
}

func (s *updateTestServer) hasRequestedPathContaining(substr string) bool {
	s.pathsMu.Lock()
	defer s.pathsMu.Unlock()
	for _, path := range s.paths {
		if strings.Contains(path, substr) {
			return true
		}
	}
	return false
}

func runUpdateCommand(t *testing.T, args []string, client *http.Client) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runUpdate(args, &stdout, &stderr, client)
	return stdout.String(), stderr.String(), code
}

func updateBinaryScript(t *testing.T, binaryName, version string) []byte {
	t.Helper()
	return []byte(fmt.Sprintf("#!/bin/sh\necho %s %s\n", binaryName, version))
}

func writeTarArtifact(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	gw := gzip.NewWriter(buffer)
	tw := tar.NewWriter(gw)
	for name, content := range entries {
		header := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("tar content: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buffer.Bytes()
}

func writeZipArtifact(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	zw := zip.NewWriter(buffer)
	for name, content := range entries {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		header.SetMode(0o755)
		file, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("zip header: %v", err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatalf("zip write: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buffer.Bytes()
}

func checksumText(t *testing.T, artifactName string, content []byte) string {
	t.Helper()
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%s  dist/%s\n", hex.EncodeToString(sum[:]), artifactName)
}
