---
id: yr-qggb
status: open
deps: []
links: []
created: 2026-02-11T14:13:21Z
type: epic
priority: 1
assignee: Gennady Evstratov
parent: yr-2y0b
---
# E7: Live-run TUI bugfix wave

Collect small but high-impact UI bugs observed while yolo-agent is actively running and streamed into yolo-tui.

Goal: fix interaction and rendering rough edges quickly without waiting for large redesigns.

Scope:
- visual stability (no jumping layouts)
- readability under narrow terminals
- predictable keyboard interactions while stream is live

This epic will be extended incrementally as new bugs are reported during live runs.

## Acceptance Criteria

Given yolo-agent streams events into yolo-tui,
when known UI bugs are triggered,
then each has a dedicated child task with reproduction, expected behavior, and tests.

Given child tasks in this epic are completed,
when running yolo-tui in live stream mode,
then the UI remains visually stable and predictable for the covered cases.

