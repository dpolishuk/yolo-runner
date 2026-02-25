package codingagents

import (
	"context"
	"strings"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

func TestResolveCommandArgsRendersBackendPlaceholders(t *testing.T) {
	t.Helper()
	got := CommandSpec{}
	runner := commandRunnerFunc(func(_ context.Context, spec CommandSpec) error {
		got = spec
		return nil
	})
	adapter := NewGenericCLIRunnerAdapter("custom-cli", "/usr/bin/custom-cli", []string{
		"--backend={{backend}}",
		"--backend-name={{backend-name}}",
		"--model={{model}}",
		"{{prompt}}",
	}, runner)

	_, _ = adapter.Run(context.Background(), contracts.RunnerRequest{
		Model:      "custom-model",
		Prompt:     "Implement feature",
		TaskID:     "task-1",
		RepoRoot:   t.TempDir(),
		Metadata:   nil,
		Mode:       contracts.RunnerModeImplement,
	})

	if got.Binary != "/usr/bin/custom-cli" {
		t.Fatalf("expected binary %q, got %q", "/usr/bin/custom-cli", got.Binary)
	}
	if !containsSlice(got.Args, "--backend=custom-cli") {
		t.Fatalf("expected --backend placeholder to render, got %#v", got.Args)
	}
	if !containsSlice(got.Args, "--backend-name=custom-cli") {
		t.Fatalf("expected --backend-name placeholder to render, got %#v", got.Args)
	}
	if !containsSlice(got.Args, "--model=custom-model") {
		t.Fatalf("expected --model placeholder to render, got %#v", got.Args)
	}
	if !containsSlice(got.Args, "Implement feature") {
		t.Fatalf("expected prompt placeholder to render, got %#v", got.Args)
	}
}

func TestResolveCommandArgsPreservesNonTemplateValues(t *testing.T) {
	t.Helper()
	got := CommandSpec{}
	runner := commandRunnerFunc(func(_ context.Context, spec CommandSpec) error {
		got = spec
		return nil
	})
	adapter := NewGenericCLIRunnerAdapter("", "custom", []string{
		"echo",
		"hello",
	}, runner)

	_, _ = adapter.Run(context.Background(), contracts.RunnerRequest{
		Prompt:  "anything",
		Mode:    contracts.RunnerModeImplement,
		Model:   "model-x",
		TaskID:  "task-2",
		RepoRoot: t.TempDir(),
	})

	if len(got.Args) != 2 {
		t.Fatalf("expected unchanged args count=2, got %d", len(got.Args))
	}
	if got.Args[0] != "echo" || got.Args[1] != "hello" {
		t.Fatalf("expected non-template args preserved, got %#v", got.Args)
	}
}

func containsSlice(values []string, needle string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}
