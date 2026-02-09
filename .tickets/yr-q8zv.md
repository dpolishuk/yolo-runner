---
id: yr-q8zv
status: closed
deps: [yr-lynv, yr-wymf]
links: []
created: 2026-02-09T21:11:13Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.18 Per-task branch orchestration in agent loop

Before each task run, create/switch to task branch from main via VCS abstraction

## Acceptance Criteria

Given open task T, when loop starts execution, then agent runs task on dedicated VCS branch for T and never directly on main

