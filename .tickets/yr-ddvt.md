---
id: yr-ddvt
status: closed
deps: [yr-ym07]
links: []
created: 2026-02-09T21:10:20Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.7 Add runner watchdog/timeout/error mapping

Detect hangs, classify timeout/init failures, and kill stuck subprocesses cleanly

## Acceptance Criteria

Given stalled agent process, when timeout expires, then runner terminates subprocess and reports blocked/failed with reason

