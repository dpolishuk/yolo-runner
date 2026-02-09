---
id: yr-tf31
status: closed
deps: [yr-wymf]
links: []
created: 2026-02-09T21:10:21Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.10 Add self-host safety guardrails

Enforce root scoping, max tasks per run, stop signal/file, and dry-run

## Acceptance Criteria

Given safety flags, when loop runs, then work stays within root scope and terminates predictably on limits/stops

