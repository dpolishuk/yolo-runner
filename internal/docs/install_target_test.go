package docs

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakefileHasInstallTargetThatInstallsYoloAgent(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")

	required := []string{
		"install:",
		"PREFIX ?=",
		"bin/yolo-agent",
		"mkdir -p",
		"chmod +x",
	}

	for _, needle := range required {
		if !strings.Contains(makefile, needle) {
			t.Fatalf("Makefile is missing %q required for install target contract", needle)
		}
	}
}

func TestMakefileInstallTargetHonorsPrefixAndCreatesExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("make install behavior is validated on Unix-family environments only")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	prefix := filepath.Join(t.TempDir(), "prefix")
	cmd := exec.Command("make", "-C", repoRoot, "install", "PREFIX="+prefix)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run make install: %v (%s)", err, output)
	}

	target := filepath.Join(prefix, "bin", "yolo-agent")
	_, err = os.Stat(target)
	if err != nil {
		t.Fatalf("installed binary should exist at custom PREFIX location %q: %v", target, err)
	}

	binDir := filepath.Dir(target)
	info, err := os.Stat(binDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("install should create destination directory %q", binDir)
	}

	binInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("inspect installed binary: %v", err)
	}
	if binInfo.Mode().Perm()&0111 == 0 {
		t.Fatalf("installed binary should be executable, got mode %o", binInfo.Mode())
	}

	helpOutput, err := exec.Command(target, "--help").CombinedOutput()
	if err != nil && !strings.Contains(string(helpOutput), "Usage of yolo-agent:") {
		t.Fatalf("installed binary should expose usage text: %v (%s)", err, helpOutput)
	}
	if !strings.Contains(string(helpOutput), "Usage of yolo-agent:") {
		t.Fatalf("installed binary should expose usage text: got %s", helpOutput)
	}
}
