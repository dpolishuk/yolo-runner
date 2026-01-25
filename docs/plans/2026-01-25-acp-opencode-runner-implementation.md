# ACP OpenCode Runner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Run OpenCode tasks via ACP using the Go SDK, replacing JSONL/CLI parsing and auto-handling ACP permission/questions.

**Architecture:** Start `opencode acp` as a local server, connect via `acp-go`, select the `yolo` agent, stream events, and log ACP requests/decisions to `runner-logs` JSONL while reusing existing runner orchestration.

**Tech Stack:** Go, `acp-go` SDK, OpenCode ACP server, existing runner interfaces, JSONL logging.

## Task 1: Add ACP server command tests

**Files:**
- Create: `internal/opencode/acp_server_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/opencode -run TestBuildACPArgsIncludesPortAndCwd`
Expected: FAIL with `undefined: BuildACPArgs`.

**Step 3: Commit tests**

```bash
git add internal/opencode/acp_server_test.go
git commit -m "test: cover opencode acp args"
```

## Task 2: Implement ACP server command builder

**Files:**
- Create: `internal/opencode/acp_server.go`

**Step 1: Run tests to see failure**

Run: `go test ./internal/opencode -run TestBuildACPArgsIncludesPortAndCwd`
Expected: FAIL with `undefined: BuildACPArgs`.

**Step 2: Write minimal implementation**

```go
func BuildACPArgs(repoRoot string, port int) []string {
    return []string{"opencode", "acp", "--port", strconv.Itoa(port), "--cwd", repoRoot}
}
```

**Step 3: Run tests to verify it passes**

Run: `go test ./internal/opencode -run TestBuildACPArgsIncludesPortAndCwd`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/opencode/acp_server.go
git commit -m "feat: add opencode acp args helper"
```

## Task 3: Add ACP request logging tests

**Files:**
- Create: `internal/logging/acp_test.go`

**Step 1: Write the failing test**

```go
package logging

import (
    "os"
    "path/filepath"
    "testing"
)

func TestAppendACPRequestWritesJSONL(t *testing.T) {
    dir := t.TempDir()
    logPath := filepath.Join(dir, "runner-logs", "opencode", "issue-1.jsonl")
    if err := AppendACPRequest(logPath, ACPRequestEntry{
        IssueID: "issue-1",
        RequestType: "permission",
        Decision: "allow",
    }); err != nil {
        t.Fatalf("append error: %v", err)
    }
    content, err := os.ReadFile(logPath)
    if err != nil {
        t.Fatalf("read log: %v", err)
    }
    if len(content) == 0 || content[len(content)-1] != '\n' {
        t.Fatalf("expected newline-terminated jsonl")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/logging -run TestAppendACPRequestWritesJSONL`
Expected: FAIL with `undefined: AppendACPRequest`.

**Step 3: Commit tests**

```bash
git add internal/logging/acp_test.go
git commit -m "test: cover acp request logging"
```

## Task 4: Implement ACP request logging

**Files:**
- Create: `internal/logging/acp.go`

**Step 1: Run tests to see failure**

Run: `go test ./internal/logging -run TestAppendACPRequestWritesJSONL`
Expected: FAIL with `undefined: AppendACPRequest`.

**Step 2: Write minimal implementation**

```go
type ACPRequestEntry struct {
    Timestamp   string `json:"timestamp"`
    IssueID     string `json:"issue_id"`
    RequestType string `json:"request_type"`
    Decision    string `json:"decision"`
    Message     string `json:"message,omitempty"`
    RequestID   string `json:"request_id,omitempty"`
}

func AppendACPRequest(logPath string, entry ACPRequestEntry) error {
    if entry.Timestamp == "" {
        entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05Z")
    }
    if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
        return err
    }
    payload, err := json.Marshal(entry)
    if err != nil {
        return err
    }
    file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()
    _, err = file.Write(append(payload, '\n'))
    return err
}
```

**Step 3: Run tests to verify it passes**

Run: `go test ./internal/logging -run TestAppendACPRequestWritesJSONL`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/logging/acp.go
git commit -m "feat: append acp request logs"
```

## Task 5: Add ACP client tests (permission + question handling)

**Files:**
- Create: `internal/opencode/acp_client_test.go`

**Step 1: Write the failing test**

```go
package opencode

import (
    "context"
    "testing"
)

func TestACPHandlerAutoApprovesPermission(t *testing.T) {
    handler := NewACPHandler("issue-1", "/tmp/log.jsonl", nil)
    decision := handler.HandlePermission(context.Background(), "perm-1", "repo.write")
    if decision != ACPDecisionAllow {
        t.Fatalf("expected allow, got %v", decision)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/opencode -run TestACPHandlerAutoApprovesPermission`
Expected: FAIL with `undefined: NewACPHandler`.

**Step 3: Commit tests**

```bash
git add internal/opencode/acp_client_test.go
git commit -m "test: cover acp request handling"
```

## Task 6: Implement ACP handler + client wrapper

**Files:**
- Create: `internal/opencode/acp_client.go`
- Modify: `go.mod`

**Step 1: Run tests to see failure**

Run: `go test ./internal/opencode -run TestACPHandlerAutoApprovesPermission`
Expected: FAIL with `undefined: NewACPHandler`.

**Step 2: Write minimal implementation**

```go
type ACPDecision string

const (
    ACPDecisionAllow ACPDecision = "allow"
    ACPDecisionDecide ACPDecision = "decide"
)

type ACPHandler struct {
    issueID string
    logPath string
    logger  func(string, string, string, string) error
}

func NewACPHandler(issueID string, logPath string, logger func(string, string, string, string) error) *ACPHandler {
    return &ACPHandler{issueID: issueID, logPath: logPath, logger: logger}
}

func (h *ACPHandler) HandlePermission(ctx context.Context, requestID string, scope string) ACPDecision {
    if h != nil && h.logger != nil {
        _ = h.logger(h.logPath, h.issueID, "permission", "allow")
    }
    return ACPDecisionAllow
}

func (h *ACPHandler) HandleQuestion(ctx context.Context, requestID string, prompt string) string {
    if h != nil && h.logger != nil {
        _ = h.logger(h.logPath, h.issueID, "question", "decide yourself")
    }
    return "decide yourself"
}
```

Also wire `acp-go` (per SDK) in this file or a sibling `internal/opencode/acp_run.go`:
- Start client connection to the ACP server endpoint.
- Select `yolo` agent for the run.
- Stream events until completion or error.
- Delegate permission/question messages to `ACPHandler`.

**Step 3: Run tests to verify it passes**

Run: `go test ./internal/opencode -run TestACPHandlerAutoApprovesPermission`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/opencode/acp_client.go go.mod go.sum
git commit -m "feat: add acp handler and client wrapper"
```

## Task 7: Add opencode runner ACP integration tests

**Files:**
- Modify: `internal/opencode/client_test.go`

**Step 1: Write the failing test**

```go
func TestRunUsesACPClient(t *testing.T) {
    called := false
    runner := RunnerFunc(func(args []string, env map[string]string, stdoutPath string) (Process, error) {
        proc := newFakeProcess()
        close(proc.waitCh)
        return proc, nil
    })
    acpClient := ACPClientFunc(func(ctx context.Context, issueID string, logPath string) error {
        called = true
        return nil
    })
    if err := RunWithACP(context.Background(), "issue-1", "/repo", "prompt", "", "", "", "", runner, acpClient); err != nil {
        t.Fatalf("RunWithACP error: %v", err)
    }
    if !called {
        t.Fatalf("expected ACP client to be called")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/opencode -run TestRunUsesACPClient`
Expected: FAIL with `undefined: RunWithACP`.

**Step 3: Commit tests**

```bash
git add internal/opencode/client_test.go
git commit -m "test: cover acp runner integration"
```

## Task 8: Implement ACP runner integration

**Files:**
- Modify: `internal/opencode/client.go`
- Modify: `internal/opencode/watchdog.go` (if no longer used, keep but stop wiring)

**Step 1: Run tests to see failure**

Run: `go test ./internal/opencode -run TestRunUsesACPClient`
Expected: FAIL with `undefined: RunWithACP`.

**Step 2: Write minimal implementation**

- Add `ACPClient` interface with `Run(ctx, issueID, logPath, endpoint)` or similar.
- Add `RunWithACP` wrapper that:
  - Builds `opencode acp` args using `BuildACPArgs`.
  - Starts the ACP server with the existing `Runner` interface.
  - Resolves the endpoint (use `--port` + `127.0.0.1`).
  - Calls the ACP client with handler for permission/question.
- Update `Run` and `RunWithContext` to call ACP path rather than JSONL watchdog.

**Step 3: Run tests to verify it passes**

Run: `go test ./internal/opencode -run TestRunUsesACPClient`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/opencode/client.go internal/opencode/watchdog.go
git commit -m "feat: run opencode via acp"
```

## Task 9: Update runner logging tests if needed

**Files:**
- Modify: `internal/ui/progress_test.go` (if log format changes)

**Step 1: Add tests only if ACP log entries affect progress behavior**

If needed, add a minimal test that a JSONL append bumps file size so progress ticks.

**Step 2: Run tests**

Run: `go test ./internal/ui -run TestProgress` (or full suite)

**Step 3: Commit**

```bash
git add internal/ui/progress_test.go
git commit -m "test: keep progress in sync with acp logs"
```

## Verification Checklist

- `go test ./internal/opencode`
- `go test ./internal/logging`
- `go test ./...`

## Notes

- Use `opencode acp --port <port> --cwd <repo>` based on `opencode acp --help`.
- Log ACP requests to `runner-logs/opencode/<issue>.jsonl` (same path as current).
- Keep JSONL newline-terminated for progress spinner updates.
