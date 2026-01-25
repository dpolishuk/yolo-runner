package opencode

import "testing"

func TestBuildACPArgsIncludesPortAndCwd(t *testing.T) {
	args := BuildACPArgs("/repo", 4567)
	expected := []string{"opencode", "acp", "--port", "4567", "--cwd", "/repo"}
	if len(args) != len(expected) {
		t.Fatalf("unexpected args length: %v", args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Fatalf("expected %q at %d, got %q", want, i, args[i])
		}
	}
}
