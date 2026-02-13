---
id: yr-70nw
status: closed
deps: []
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-s0go
---
# E2-T1 Finalize runner backend plugin contract

STRICT TDD: failing tests first. Normalize implement/review/timeouts/result mapping across backends.

## Acceptance Criteria

Given backend contract tests, when all adapters run, then they satisfy required interfaces and statuses.


## Notes

**2026-02-13T20:10:48Z**

Implemented backend contract hardening for timeout/result normalization and opencode adapter nil-safety parity. Validation: go test ./...
