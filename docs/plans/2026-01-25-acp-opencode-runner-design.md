# ACP OpenCode Runner Design

**Goal:** Replace JSONL/CLI parsing with ACP protocol so yolo-runner drives OpenCode tasks through ACP using Go.

**Architecture:** Start OpenCode in ACP mode, connect via `acp-go`, select the `yolo` agent, send the task prompt, stream ACP events, auto-approve permissions, respond "decide yourself" to other questions, log all requests to `runner-logs` JSONL, and hand completion back to the existing runner flow.

**Tech Stack:** Go, `acp-go` SDK, existing runner interfaces, JSONL logging, OpenCode ACP mode.

## Requirements

- Use ACP (not CLI JSONL parsing) for OpenCode execution.
- Start OpenCode in ACP mode (flags/transport confirmed via `opencode --help`).
- Connect through ACP, select `yolo` agent, issue the prompt.
- Monitor progress through ACP messages.
- Always grant permissions and log requests.
- For other questions, reply "decide yourself" and log requests.
- When finished, mark the bead complete and proceed to the next task.

## Approach Options

- **ACP-first in-process client (recommended):** Launch OpenCode per task, connect with `acp-go`, and stream events through a Go ACP client. Minimal new runtime complexity and aligns with requirements.
- **Long-lived ACP server:** Keep OpenCode running and connect per task. Fewer process launches but more lifecycle complexity.
- **Dual-path fallback:** Keep CLI JSONL runner as a fallback. Safer rollout but more maintenance.

## Components

- `internal/opencode/acp` (or extension of `internal/opencode`): ACP client wrapper around `acp-go` that handles connection, run lifecycle, and request responses.
- `internal/opencode/client.go`: Updated runner entry points to start OpenCode in ACP mode and invoke the ACP client.
- `internal/logging/jsonl`: Extend to emit ACP request/permission events in the same runner log stream.
- `internal/ui/progress`: Consumes log entries (or callbacks) to reflect ACP progress.

## Data Flow

1. Runner selects bead task and builds prompt.
2. Start OpenCode in ACP mode with repo root + config env.
3. ACP client connects via `acp-go` (transport per `opencode --help`).
4. Select `yolo` agent and start run with prompt.
5. Stream ACP events; emit progress/log entries as they arrive.
6. On permission requests: auto-approve, log decision to JSONL.
7. On other questions: respond "decide yourself", log request.
8. On completion: return to runner, commit/close bead, continue loop.

## Error Handling

- Wrap ACP failures with typed errors (`ErrACPConnect`, `ErrACPRunFailed`).
- If ACP stream drops unexpectedly, return error to runner (no silent success).
- Preserve existing stall handling where possible; avoid marking tasks complete on ACP errors.

## Logging

- Append ACP request/decision entries to `runner-logs/opencode/<issue>.jsonl`.
- Fields: timestamp, issue id, request type, content summary, decision, and run id/session id.
- Keep JSONL compatible with existing log watchers.

## Testing Strategy (Strict TDD)

- Unit tests for ACP client connection lifecycle and event handling using mocked ACP transport.
- Unit tests for permission auto-approval and "decide yourself" responses.
- Unit tests for ACP log writer entries (JSONL payload correctness).
- Integration tests for `internal/opencode` runner wrapper with a fake ACP server.
- Tests are created as separate tasks in beads (test-writing tasks distinct from implementation tasks).

## Open Questions

- Confirm OpenCode ACP flags/transport (stdio vs tcp) via `opencode --help` and docs.
- Confirm ACP lifecycle events required for task completion signal.
