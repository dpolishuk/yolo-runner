package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/version"
)

func TestRunMainSupportsVersionFlag(t *testing.T) {
	original := version.Version
	version.Version = "linear-worker-version-test"
	t.Cleanup(func() {
		version.Version = original
	})

	stdout := captureWorkerStdout(t, func() {
		code := RunMain([]string{"--version"}, nil)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
	})
	if strings.TrimSpace(stdout) != "yolo-linear-worker linear-worker-version-test" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}

func captureWorkerStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = original
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	return string(data)
}
