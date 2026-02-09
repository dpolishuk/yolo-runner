---
id: yr-w66o
status: open
deps: [yr-5reb]
links: []
created: 2026-02-09T23:07:08Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-5543
---
# E8-T2 Self-host demo: Claude + conflict retry path

STRICT TDD: add acceptance test for one-retry conflict behavior with Claude backend.

## Acceptance Criteria

Given intentional conflict, when queue retries once, then final state is landed or blocked with triage metadata.

