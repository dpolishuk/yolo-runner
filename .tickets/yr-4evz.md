---
id: yr-4evz
status: closed
deps: [yr-58fn, yr-me4i, yr-txrd, yr-ttw4]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-s0go
---
# E2-T6 Add backend selection policy

STRICT TDD: failing tests first. Support --agent-backend and profile defaults.

## Acceptance Criteria

Given flags and profile defaults, when selecting backend, then chosen adapter matches policy deterministically.


## Notes

**2026-02-13T20:26:22Z**

Implemented backend selection policy with deterministic precedence: --agent-backend > --backend > YOLO_AGENT_BACKEND profile default > opencode fallback. Added strict TDD coverage for policy helper precedence and RunMain behavior, including override semantics and profile-default resolution. Validation: go test ./cmd/yolo-agent && go test ./...
