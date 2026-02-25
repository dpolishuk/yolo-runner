package codingagents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCatalogIncludesBuiltinAndCustomBackendDefinitions(t *testing.T) {
	repoRoot := t.TempDir()
	customDir := filepath.Join(repoRoot, ".yolo-runner", "coding-agents")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("create custom backend directory: %v", err)
	}

	customPath := filepath.Join(customDir, "custom-cli.yaml")
	if err := os.WriteFile(customPath, []byte(`
name: custom-cli
adapter: command
binary: /usr/bin/custom-cli
args:
  - "--prompt"
  - "{{prompt}}"
supports_review: true
supports_stream: true
`), 0o644); err != nil {
		t.Fatalf("write custom backend definition: %v", err)
	}

	catalog, err := LoadCatalog(repoRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if _, ok := catalog.Backend("custom-cli"); !ok {
		t.Fatalf("expected custom backend to be discovered")
	}
	if _, ok := catalog.Backend("opencode"); !ok {
		t.Fatalf("expected builtin opencode backend to be discovered")
	}
}

func TestCatalogValidateBackendUsageChecksModelAndCredentials(t *testing.T) {
	repoRoot := t.TempDir()
	customDir := filepath.Join(repoRoot, ".yolo-runner", "coding-agents")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("create custom backend directory: %v", err)
	}

	customPath := filepath.Join(customDir, "guarded.yaml")
	if err := os.WriteFile(customPath, []byte(`
name: guarded
adapter: command
binary: /usr/bin/guarded
args:
  - "--model"
  - "{{model}}"
supported_models:
  - custom-*
required_credentials:
  - CUSTOM_AGENT_TOKEN
supports_review: true
supports_stream: true
`), 0o644); err != nil {
		t.Fatalf("write custom backend definition: %v", err)
	}

	catalog, err := LoadCatalog(repoRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	if err := catalog.ValidateBackendUsage("guarded", "other-model", func(string) string { return "token" }); err == nil {
		t.Fatalf("expected unsupported model validation error")
	}
	if err := catalog.ValidateBackendUsage("guarded", "custom-model", func(string) string { return "" }); err == nil {
		t.Fatalf("expected missing credential validation error")
	}
	if err := catalog.ValidateBackendUsage("guarded", "custom-model", func(name string) string {
		if name == "CUSTOM_AGENT_TOKEN" {
			return "secret-token"
		}
		return ""
	}); err != nil {
		t.Fatalf("expected backend usage validation to succeed with valid model and credential, got %v", err)
	}
}
