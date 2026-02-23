package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/version"
)

func TestRunOnceMainSupportsVersionFlag(t *testing.T) {
	original := version.Version
	version.Version = "runner-version-test"
	t.Cleanup(func() {
		version.Version = original
	})

	out := &bytes.Buffer{}
	code := RunOnceMain([]string{"--version"}, nil, nil, out, io.Discard, nil, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(out.String()) != "yolo-runner runner-version-test" {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}
