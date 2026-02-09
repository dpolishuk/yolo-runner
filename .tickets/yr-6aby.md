---
id: yr-6aby
status: closed
deps: [yr-1enf]
links: []
created: 2026-02-09T21:10:20Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.3 Implement tk adapter via tk CLI only

Implement next/show/status/data operations by invoking tk commands; no reimplementation of readiness/deps logic

## Acceptance Criteria

Given adapter operations, when executed, then adapter calls tk ready/show/status/add-note/query and never computes readiness internally

