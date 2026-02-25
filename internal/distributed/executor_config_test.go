package distributed

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const executorConfigTestValidJSON = `{
  "name": "test-executor",
  "type": "task",
  "backend": "codex",
  "pipeline": {
    "quality_gate": {
      "tools": ["reviewer"],
      "retry": {
        "max_attempts": 2,
        "initial_delay_ms": 500,
        "backoff_ms": 200
      },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "execute",
          "condition": "true"
        },
        "on_failure": {
          "action": "retry",
          "condition": "review_failed"
        }
      }
    },
    "execute": {
      "tools": ["shell"],
      "retry": {
        "max_attempts": 3
      },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "qc_gate",
          "condition": "tests_failed"
        },
        "on_failure": {
          "action": "retry",
          "condition": "tests_failed"
        }
      }
    },
    "qc_gate": {
      "tools": ["quality-checker"],
      "retry": {
        "max_attempts": 1,
        "initial_delay_ms": 300
      },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "complete",
          "condition": "quality_score >= threshold"
        },
        "on_failure": {
          "action": "fail",
          "condition": "review_failed"
        }
      }
    },
    "complete": {
      "tools": ["git"],
      "retry": {
        "max_attempts": 1
      },
      "transitions": {
        "on_success": {
          "action": "complete",
          "condition": "true"
        },
        "on_failure": {
          "action": "fail",
          "condition": "true"
        }
      }
    }
  }
}`

const executorConfigTestInvalid = `{
  "name": "",
  "type": "task",
  "backend": "codex",
  "pipeline": {
    "quality_gate": {
      "tools": [],
      "retry": {
        "max_attempts": 0
      },
      "transitions": {
        "on_success": {
          "action": "next",
          "condition": "true"
        },
        "on_failure": {
          "action": "retry"
        }
      }
    }
  }
}`

func TestLoadExecutorConfigSupportsJSONAndYAML(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "executor.json")
	if err := os.WriteFile(jsonPath, []byte(executorConfigTestValidJSON), 0o644); err != nil {
		t.Fatalf("write JSON config: %v", err)
	}
	cfg, err := LoadExecutorConfig(jsonPath)
	if err != nil {
		t.Fatalf("load valid JSON config: %v", err)
	}
	if cfg.Name != "test-executor" {
		t.Fatalf("expected name %q, got %q", "test-executor", cfg.Name)
	}

	yamlPath := filepath.Join(tempDir, "executor.yaml")
	yaml := strings.ReplaceAll(executorConfigTestValidJSON, `"test-executor"`, `"yaml-executor"`)
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write YAML config: %v", err)
	}
	cfg, err = LoadExecutorConfig(yamlPath)
	if err != nil {
		t.Fatalf("load valid YAML config: %v", err)
	}
	if cfg.Name != "yaml-executor" {
		t.Fatalf("expected name %q, got %q", "yaml-executor", cfg.Name)
	}
}

func TestLoadExecutorConfigSupportsEnvironmentSubstitution(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "executor.yaml")
	content := `
name: "${EXECUTOR_NAME}"
type: task
backend: ${EXECUTOR_BACKEND}
pipeline:
  quality_gate:
    tools: ["reviewer"]
    retry:
      max_attempts: 2
    transitions:
      on_success:
        action: complete
        condition: "true"
      on_failure:
        action: fail
        condition: "tests_failed"
  execute:
    tools: ["shell"]
    retry:
      max_attempts: 1
    transitions:
      on_success:
        action: next
        next_stage: qc_gate
        condition: "true"
      on_failure:
        action: retry
        condition: "tests_failed"
  qc_gate:
    tools: ["quality-checker"]
    retry:
      max_attempts: 1
    transitions:
      on_success:
        action: next
        next_stage: complete
        condition: "quality_score >= threshold"
      on_failure:
        action: fail
        condition: "review_failed"
  complete:
    tools: ["git"]
    retry:
      max_attempts: 1
    transitions:
      on_success:
        action: complete
        condition: "true"
      on_failure:
        action: fail
        condition: "true"
`

	if err := os.Setenv("EXECUTOR_NAME", "env-executor"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if err := os.Setenv("EXECUTOR_BACKEND", "codex"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("EXECUTOR_NAME")
		_ = os.Unsetenv("EXECUTOR_BACKEND")
	})

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadExecutorConfig(configPath)
	if err != nil {
		t.Fatalf("load config with env substitution: %v", err)
	}
	if cfg.Name != "env-executor" {
		t.Fatalf("expected name %q, got %q", "env-executor", cfg.Name)
	}
	if cfg.Backend != "codex" {
		t.Fatalf("expected backend %q, got %q", "codex", cfg.Backend)
	}
}

func TestLoadExecutorConfigValidationReportsClearErrors(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(configPath, []byte(executorConfigTestInvalid), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	_, err := LoadExecutorConfig(configPath)
	if err == nil {
		t.Fatalf("expected invalid config error")
	}
	if !strings.Contains(err.Error(), "name must be non-empty") {
		t.Fatalf("expected name validation message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "pipeline missing required stage \"execute\"") {
		t.Fatalf("expected missing stage message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "quality_gate.tools must include at least one tool") {
		t.Fatalf("expected tools validation message, got: %v", err)
	}
}

func TestWatchExecutorConfigDetectsChanges(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "executor-watch.yaml")
	if err := os.WriteFile(configPath, []byte(executorConfigTestValidJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	received := make([]ExecutorConfigWatchEvent, 0, 2)
	watchErrCh := make(chan error, 1)
	go func() {
		watchErrCh <- WatchExecutorConfig(ctx, configPath, 20*time.Millisecond, func(event ExecutorConfigWatchEvent) {
			mu.Lock()
			received = append(received, event)
			mu.Unlock()
		}, nil)
	}()

	// Watch should keep running until context is canceled.
	select {
	case watchErr := <-watchErrCh:
		t.Fatalf("watch returned unexpectedly: %v", watchErr)
	case <-time.After(200 * time.Millisecond):
	}

	wait := time.After(200 * time.Millisecond)
	for {
		mu.Lock()
		count := len(received)
		mu.Unlock()
		if count > 0 {
			break
		}
		select {
		case <-wait:
			t.Fatalf("expected initial watch event")
		case <-time.After(10 * time.Millisecond):
		}
	}

	updated := strings.ReplaceAll(executorConfigTestValidJSON, `"test-executor"`, `"updated-executor"`)
	tmp := filepath.Join(tempDir, "executor-watch-updated.yaml.tmp")
	if err := os.WriteFile(tmp, []byte(updated), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	if err := os.Rename(tmp, configPath); err != nil {
		t.Fatalf("replace config file: %v", err)
	}

	wait = time.After(500 * time.Millisecond)
	var foundEvent *ExecutorConfigWatchEvent
	for {
		mu.Lock()
		// Ignore transient invalid transitions while file is being observed and wait for the
		// updated valid payload.
		for _, event := range received {
			if event.Err == nil && event.Config.Name == "updated-executor" {
				e := event
				foundEvent = &e
				break
			}
		}
		count := len(received)
		mu.Unlock()
		if foundEvent != nil || count >= 2 {
			break
		}
		select {
		case <-wait:
			t.Fatalf("expected update watch event")
		case <-time.After(10 * time.Millisecond):
		}
	}

	mu.Lock()
	if foundEvent == nil {
		mu.Unlock()
		t.Fatalf("expected at least one watch event")
	}
	lastEvent := *foundEvent
	mu.Unlock()
	if lastEvent.Err != nil {
		t.Fatalf("unexpected update error: %v", lastEvent.Err)
	}
	if lastEvent.Config.Name != "updated-executor" {
		t.Fatalf("expected updated config name %q, got %q", "updated-executor", lastEvent.Config.Name)
	}
	cancel()
	if watchErr := <-watchErrCh; watchErr != nil {
		t.Fatalf("watch ended with error: %v", watchErr)
	}
}
