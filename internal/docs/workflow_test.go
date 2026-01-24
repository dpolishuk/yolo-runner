package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadmeMentionsCloseEligible(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	readmePath := filepath.Join(repoRoot, "README.md")
	contents, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}

	if !strings.Contains(string(contents), "bd epic close-eligible") {
		t.Fatalf("README missing bd epic close-eligible step")
	}
}
