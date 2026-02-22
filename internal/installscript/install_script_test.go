package installscript

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallScriptPlanModeReportsPlatformAndInstallCoordinates(t *testing.T) {
	t.Parallel()
	repoRoot := testRepoRoot(t)

	tmpHome := t.TempDir()
	tmpInstallDir := filepath.Join(tmpHome, ".local", "bin")
	releaseBase := "https://example.invalid/releases/latest"

	tests := []struct {
		name         string
		osInput      string
		archInput    string
		wantPlatform string
		wantArch     string
		wantArtifact string
		wantChecksum string
		wantBinary   string
		wantInstall  string
	}{
		{
			name:         "linux amd64",
			osInput:      "Linux",
			archInput:    "x86_64",
			wantPlatform: "linux",
			wantArch:     "amd64",
			wantArtifact: "yolo-runner_linux_amd64.tar.gz",
			wantBinary:   "yolo-agent",
			wantChecksum: "yolo-runner_linux_amd64.tar.gz.sha256",
			wantInstall:  filepath.Join(tmpInstallDir, "yolo-agent"),
		},
		{
			name:         "darwin arm64",
			osInput:      "Darwin",
			archInput:    "arm64",
			wantPlatform: "darwin",
			wantArch:     "arm64",
			wantArtifact: "yolo-runner_darwin_arm64.tar.gz",
			wantBinary:   "yolo-agent",
			wantChecksum: "yolo-runner_darwin_arm64.tar.gz.sha256",
			wantInstall:  filepath.Join(tmpInstallDir, "yolo-agent"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := runInstallScript(repoRoot, []string{
				"--plan",
				"--os", tc.osInput,
				"--arch", tc.archInput,
				"--release-base", releaseBase,
				"--install-dir", tmpInstallDir,
			}, tmpHome)
			if err != nil {
				t.Fatalf("run install.sh --plan: %v\noutput: %s", err, out)
			}

			plan := parsePlanOutput(out)
			gotPlatform, ok := plan["platform"]
			if !ok {
				t.Fatalf("missing platform in plan output: %s", out)
			}
			if gotPlatform != tc.wantPlatform {
				t.Fatalf("platform = %q, want %q", gotPlatform, tc.wantPlatform)
			}

			gotArch := plan["arch"]
			if gotArch != tc.wantArch {
				t.Fatalf("arch = %q, want %q", gotArch, tc.wantArch)
			}

			gotArtifact := plan["artifact"]
			if gotArtifact != tc.wantArtifact {
				t.Fatalf("artifact = %q, want %q", gotArtifact, tc.wantArtifact)
			}

			wantURL := releaseBase + "/" + tc.wantArtifact
			if gotURL := plan["artifact_url"]; gotURL != wantURL {
				t.Fatalf("artifact_url = %q, want %q", gotURL, wantURL)
			}

			wantChecksumURL := releaseBase + "/" + tc.wantChecksum
			if gotChecksum := plan["checksum_url"]; gotChecksum != wantChecksumURL {
				t.Fatalf("checksum_url = %q, want %q", gotChecksum, wantChecksumURL)
			}

			if gotBinary := plan["binary"]; gotBinary != tc.wantBinary {
				t.Fatalf("binary = %q, want %q", gotBinary, tc.wantBinary)
			}

			if gotInstallPath := plan["install_path"]; gotInstallPath != tc.wantInstall {
				t.Fatalf("install_path = %q, want %q", gotInstallPath, tc.wantInstall)
			}
		})
	}
}

func TestInstallScriptRejectsUnsupportedPlatforms(t *testing.T) {
	t.Parallel()
	repoRoot := testRepoRoot(t)

	_, err := runInstallScript(repoRoot, []string{"--plan", "--os", "Plan9", "--arch", "amd64"}, t.TempDir())
	if err == nil {
		t.Fatal("expected plan mode to fail for unsupported OS")
	}
}

func TestInstallScriptInstallModeUsesExpectedUrlsAndInstallPath(t *testing.T) {
	repoRoot := testRepoRoot(t)

	tmpHome := t.TempDir()
	tmpArtifacts := t.TempDir()
	releaseBase := "https://example.invalid/releases/latest"
	installPath := filepath.Join(tmpHome, ".local", "bin", "yolo-agent")
	artifactName := "yolo-runner_linux_amd64.tar.gz"
	artifactURL := releaseBase + "/" + artifactName
	checksumURL := artifactURL + ".sha256"

	artifactPath := writeArtifactTarGz(t, tmpArtifacts, artifactName, "yolo-agent", []byte("#!/bin/sh\necho ok\n"))
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact fixture: %v", err)
	}
	sum := sha256.Sum256(artifactData)
	checksumPath := artifactPath + ".sha256"
	if err := os.WriteFile(
		checksumPath,
		[]byte(fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), artifactName)),
		0o600,
	); err != nil {
		t.Fatalf("write checksum fixture: %v", err)
	}

	fakeCurlLog := filepath.Join(tmpArtifacts, "curl.calls")
	fakeCurlDir := filepath.Join(tmpArtifacts, "bin")
	writeFakeCurl(t, fakeCurlDir, fakeCurlLog)
	out, err := runInstallScript(
		repoRoot,
		[]string{
			"--os", "Linux",
			"--arch", "x86_64",
			"--release-base", releaseBase,
		},
		tmpHome,
		"PATH="+fakeCurlDir+":"+os.Getenv("PATH"),
		"YOLO_FAKE_ARTIFACT_PATH="+artifactPath,
		"YOLO_FAKE_CHECKSUM_PATH="+checksumPath,
		"YOLO_FAKE_ARTIFACT_URL="+artifactURL,
		"YOLO_FAKE_CHECKSUM_URL="+checksumURL,
		"YOLO_FAKE_CURL_LOG="+fakeCurlLog,
	)
	if err != nil {
		t.Fatalf("install should succeed: %v\noutput: %s", err, out)
	}

	calls := strings.Split(strings.TrimSpace(readFileString(t, fakeCurlLog)), "\n")
	if len(calls) != 2 {
		t.Fatalf("expected two downloads, got %d\ncalls: %q", len(calls), calls)
	}
	if calls[0] != artifactURL {
		t.Fatalf("first download = %q, want %q", calls[0], artifactURL)
	}
	if calls[1] != checksumURL {
		t.Fatalf("second download = %q, want %q", calls[1], checksumURL)
	}

	if _, err := os.Stat(installPath); err != nil {
		t.Fatalf("expected installed binary at %s: %v", installPath, err)
	}
	if _, err := os.Stat(filepath.Dir(installPath)); err != nil {
		t.Fatalf("expected install directory %s to exist: %v", filepath.Dir(installPath), err)
	}
	if info, err := os.Stat(installPath); err == nil && info.Mode()&0o111 == 0 {
		t.Fatalf("installed file is not executable: %s", installPath)
	}
	if !strings.Contains(out, "installed yolo-agent to "+filepath.Dir(installPath)) {
		t.Fatalf("install output missing resolved target path:\n%s", out)
	}
}

func TestInstallScriptInstallModeFailsOnChecksumMismatch(t *testing.T) {
	repoRoot := testRepoRoot(t)

	tmpHome := t.TempDir()
	tmpArtifacts := t.TempDir()
	releaseBase := "https://example.invalid/releases/latest"
	artifactName := "yolo-runner_linux_amd64.tar.gz"
	artifactURL := releaseBase + "/" + artifactName
	checksumURL := artifactURL + ".sha256"

	artifactPath := writeArtifactTarGz(t, tmpArtifacts, artifactName, "yolo-agent", []byte("#!/bin/sh\necho ok\n"))
	checksumPath := artifactPath + ".sha256"
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%064s  %s\n", "0", artifactName)), 0o600); err != nil {
		t.Fatalf("write checksum fixture: %v", err)
	}

	fakeCurlLog := filepath.Join(tmpArtifacts, "curl.calls")
	fakeCurlDir := filepath.Join(tmpArtifacts, "bin")
	writeFakeCurl(t, fakeCurlDir, fakeCurlLog)
	_, err := runInstallScript(
		repoRoot,
		[]string{
			"--os", "Linux",
			"--arch", "x86_64",
			"--release-base", releaseBase,
		},
		tmpHome,
		"PATH="+fakeCurlDir+":"+os.Getenv("PATH"),
		"YOLO_FAKE_ARTIFACT_PATH="+artifactPath,
		"YOLO_FAKE_CHECKSUM_PATH="+checksumPath,
		"YOLO_FAKE_ARTIFACT_URL="+artifactURL,
		"YOLO_FAKE_CHECKSUM_URL="+checksumURL,
		"YOLO_FAKE_CURL_LOG="+fakeCurlLog,
	)
	if err == nil {
		t.Fatal("expected checksum mismatch to fail install")
	}
}

func TestInstallScriptChecksumVerification(t *testing.T) {
	repoRoot := testRepoRoot(t)

	tmpDir := t.TempDir()
	artifact := filepath.Join(tmpDir, "artifact.bin")
	manifest := filepath.Join(tmpDir, "artifact.bin.sha256")

	artifactContents := "verified-download"
	if err := os.WriteFile(artifact, []byte(artifactContents), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	sum := sha256.Sum256([]byte(artifactContents))
	hash := hex.EncodeToString(sum[:])
	if err := os.WriteFile(manifest, []byte(fmt.Sprintf("%s  %s\n", hash, filepath.Base(artifact))), 0o600); err != nil {
		t.Fatalf("write checksum manifest: %v", err)
	}

	if out, err := runInstallScript(repoRoot, []string{"--test-checksum", artifact, manifest}, tmpDir); err != nil {
		t.Fatalf("expected checksum to verify, got error: %v\nout: %s", err, out)
	}

	if err := os.WriteFile(manifest, []byte(fmt.Sprintf("%s  %s\n", strings.Repeat("0", len(hash)), filepath.Base(artifact))), 0o600); err != nil {
		t.Fatalf("rewrite checksum manifest: %v", err)
	}

	if out, err := runInstallScript(repoRoot, []string{"--test-checksum", artifact, manifest}, tmpDir); err == nil {
		t.Fatalf("expected checksum mismatch to fail\noutput: %s", out)
	}
}

func runInstallScript(repoRoot string, args []string, homeDir string, extraEnv ...string) (string, error) {
	cmd := exec.Command("bash", append([]string{filepath.Join(repoRoot, "install.sh")}, args...)...)
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func writeArtifactTarGz(t *testing.T, dir, filename, binaryName string, binaryContents []byte) string {
	t.Helper()

	artifactPath := filepath.Join(dir, filename)
	artifact, err := os.Create(artifactPath)
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	defer artifact.Close()

	gw, err := gzip.NewWriterLevel(artifact, gzip.BestCompression)
	if err != nil {
		t.Fatalf("create gzip writer: %v", err)
	}
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: binaryName,
		Mode: 0o755,
		Size: int64(len(binaryContents)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(binaryContents); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return artifactPath
}

func writeFakeCurl(t *testing.T, dir string, callLog string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "curl")
	script := []byte(`#!/usr/bin/env bash
set -euo pipefail

url=""
output=""
while [[ "$#" -gt 0 ]]; do
	case "$1" in
		-o)
			shift
			output="$1"
			;;
		-*)
			;;
		*)
			url="$1"
			;;
	esac
	shift
done

if [[ -z "$url" || -z "$output" ]]; then
	echo "fake curl: invalid arguments" >&2
	exit 1
fi

if [[ -z "${YOLO_FAKE_ARTIFACT_PATH:-}" || -z "${YOLO_FAKE_CHECKSUM_PATH:-}" || -z "${YOLO_FAKE_ARTIFACT_URL:-}" || -z "${YOLO_FAKE_CHECKSUM_URL:-}" || -z "${YOLO_FAKE_CURL_LOG:-}" ]]; then
	echo "fake curl: missing env vars" >&2
	exit 1
fi

echo "$url" >> "$YOLO_FAKE_CURL_LOG"
if [[ "$url" == "$YOLO_FAKE_ARTIFACT_URL" ]]; then
	cp "$YOLO_FAKE_ARTIFACT_PATH" "$output"
	exit 0
fi

if [[ "$url" == "$YOLO_FAKE_CHECKSUM_URL" ]]; then
	cp "$YOLO_FAKE_CHECKSUM_PATH" "$output"
	exit 0
fi

echo "unexpected download URL: $url" >&2
exit 1
`)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create fake curl dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, script, 0o755); err != nil {
		t.Fatalf("write fake curl: %v", err)
	}
	return scriptPath
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(b)
}

func parsePlanOutput(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func testRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}

	root := filepath.Join(wd, "..", "..")
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return root
}
