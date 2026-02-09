---
id: yr-hxps
status: open
deps: [yr-lbk0]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-vh4d
---
# E3-T4 Emit merge queue observability events

STRICT TDD: failing tests first. Emit merge_queued/retry/blocked/landed events.

## Acceptance Criteria

Given landing lifecycle, when events stream is consumed, then merge queue phases are visible.

