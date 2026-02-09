---
id: yr-8btw
status: closed
deps: [yr-wymf, yr-ym07]
links: []
created: 2026-02-09T21:10:21Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.12 Split binaries: yolo-agent/yolo-runner/yolo-task

Separate orchestration, execution, and tracker facade CLIs with shared contracts

## Acceptance Criteria

Given new binaries, when invoked, then each CLI has focused responsibility and composes via contracts

