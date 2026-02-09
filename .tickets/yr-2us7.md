---
id: yr-2us7
status: closed
deps: [yr-37ot, yr-gffc]
links: []
created: 2026-02-09T23:15:37Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-2y0b
---
# E7-T5 Show task titles alongside IDs in runner and TUI

STRICT TDD: add failing tests first, then update runner event payload/view rendering so task titles are displayed together with task IDs for readability.

## Acceptance Criteria

Given active and historical tasks, when viewing runner logs and yolo-tui, then each task line shows both task_id and task_title consistently.

