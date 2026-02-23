package main

import (
	"os"
	"strings"
	"testing"
)

func TestGoModModulePath(t *testing.T) {
	modBytes, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	firstLine := strings.Fields(strings.SplitN(string(modBytes), "\n", 2)[0])
	if len(firstLine) != 2 || firstLine[0] != "module" {
		t.Fatalf("unexpected go.mod header: %q", firstLine)
	}

	if got, want := firstLine[1], "github.com/egv/yolo-runner/v2"; got != want {
		t.Fatalf("go.mod module path %q, want %q", got, want)
	}
}
