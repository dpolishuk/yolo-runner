---
id: yr-cohs
status: open
deps: [yr-kx2t]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-0xy1
---
# E1-T5 Add task lock and landing lock

STRICT TDD: failing tests first. Prevent duplicate execution and serialize landing.

## Acceptance Criteria

Given duplicate picks and parallel completions, when locking is active, then no task runs twice and landing remains serialized.

