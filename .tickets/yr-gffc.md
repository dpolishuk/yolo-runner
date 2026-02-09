---
id: yr-gffc
status: closed
deps: []
links: []
created: 2026-02-09T23:16:36Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-2y0b
---
# E7-T5a Add task_title to event schema and parser

STRICT TDD: add failing tests first, then extend JSONL event schema with optional task_title and parse it in yolo-tui.

## Acceptance Criteria

Given task lifecycle events, when serialized and parsed, then task_title is preserved and backward compatible when absent.

