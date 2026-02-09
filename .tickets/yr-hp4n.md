---
id: yr-hp4n
status: closed
deps: [yr-wymf, yr-qe3a]
links: []
created: 2026-02-09T21:10:21Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.9 Persist retry/block metadata via tk

Store retry count and block reasons through tk operations only

## Acceptance Criteria

Given retries or blocks, when loop updates task, then metadata is written through tk add-note/status and visible in tk show/query

