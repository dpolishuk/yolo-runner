---
id: yr-5g2h
status: closed
deps: []
links: []
created: 2026-02-12T18:57:13Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-qggb
---
# E7-T28 Surface active opencode progress in yolo-tui during long-running steps

Bug report: yolo-agent shows only runner heartbeats while opencode is actively executing prompt/tool/test steps in stderr logs. yolo-tui should display meaningful in-flight progress (current step/tool/test phase) instead of appearing idle when work is ongoing.

## Acceptance Criteria

Given a task run where opencode emits ongoing prompt/tool updates but no final completion yet, when yolo-tui consumes stream events, then the UI displays active progress details (latest step/tool/test activity) rather than heartbeat-only idle state. Given long-running commands, when stderr/progress activity continues, then yolo-tui updates history/state with that activity at regular intervals. Given no activity at all, heartbeat-only behavior remains unchanged.

