---
id: yr-nnog
status: closed
deps: [yr-4evz]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-s0go
---
# E2-T7 Add backend conformance suite

STRICT TDD: establish shared contract tests all adapters must pass.

## Acceptance Criteria

Given opencode/codex/claude/kimi, when conformance suite runs, then all pass required behaviors.


## Notes

**2026-02-13T20:34:41Z**

Implemented shared backend conformance suite under internal/contracts/conformance and wired codex/claude/kimi/opencode adapters to it. Added cross-backend timeout normalization when runner returns nil after context deadline (claude, kimi, opencode). Validation: go test ./...
