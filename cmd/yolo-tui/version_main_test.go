package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/version"
)

func TestRunMainSupportsVersionFlag(t *testing.T) {
	original := version.Version
	version.Version = "tui-version-test"
	t.Cleanup(func() {
		version.Version = original
	})

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMain([]string{"--version"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(out.String()) != "yolo-tui tui-version-test" {
		t.Fatalf("unexpected version output: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected error output: %q", errOut.String())
	}
}
