package main

import (
	"strings"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/version"
)

func TestRunMainSupportsVersionFlag(t *testing.T) {
	original := version.Version
	version.Version = "agent-version-test"
	t.Cleanup(func() {
		version.Version = original
	})

	stdout := captureStdout(t, func() {
		code := RunMain([]string{"--version"}, nil)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
	})
	if strings.TrimSpace(stdout) != "yolo-agent agent-version-test" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}
