# ACP Session Update Callback Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Forward ACP session update notifications to an optional callback.

**Architecture:** Add an optional callback field to the ACP client wrapper and invoke it in SessionUpdate. Thread the callback through RunACPClient while keeping nil-safe defaults.

**Tech Stack:** Go, acp-go SDK, existing opencode ACP client wrapper.

## Task 1: Add failing test for session update callback

**Files:**
- Modify: `internal/opencode/acp_client_test.go`

**Step 1: Write the failing test**

```go
func TestACPClientSessionUpdateCallback(t *testing.T) {
    called := false
    client := &acpClient{
        onUpdate: func(_ *acp.SessionNotification) {
            called = true
        },
    }

    if err := client.SessionUpdate(context.Background(), &acp.SessionNotification{}); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if !called {
        t.Fatalf("expected session update callback to be called")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/opencode -run TestACPClientSessionUpdateCallback`
Expected: FAIL with `unknown field onUpdate` or `called` remains false.

## Task 2: Implement session update callback

**Files:**
- Modify: `internal/opencode/acp_client.go`

**Step 1: Run test to see failure**

Run: `go test ./internal/opencode -run TestACPClientSessionUpdateCallback`
Expected: FAIL.

**Step 2: Write minimal implementation**

```go
type acpClient struct {
    handler  *ACPHandler
    onUpdate func(*acp.SessionNotification)
}

func (c *acpClient) SessionUpdate(ctx context.Context, params *acp.SessionNotification) error {
    if c != nil && c.onUpdate != nil {
        c.onUpdate(params)
    }
    return nil
}
```

Also update `RunACPClient` signature to accept an optional callback and pass it into the client.

**Step 3: Run test to verify it passes**

Run: `go test ./internal/opencode -run TestACPClientSessionUpdateCallback`
Expected: PASS.

## Task 3: Verify and commit

**Step 1: Run targeted tests**

Run:
- `go test ./internal/opencode -run TestACPClientSessionUpdateCallback`
- `go test ./internal/opencode -run TestACPHandlerAutoApprovesPermission`

Expected: PASS.

**Step 2: Commit**

```bash
git add internal/opencode/acp_client.go internal/opencode/acp_client_test.go
git commit -m "feat: forward acp session updates"
```
