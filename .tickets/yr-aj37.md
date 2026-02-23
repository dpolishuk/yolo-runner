---
id: yr-aj37
status: closed
deps: []
links: []
created: 2026-02-11T14:27:02Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-qggb
---
# E7-T25 Prevent raw ACP stderr lines from breaking yolo-tui

Bug report (live stream mode): while yolo-agent is piped into yolo-tui, raw lines like `ACP[yr-70nw] agent_message ...` are written directly to terminal and corrupt the fullscreen layout.

Observed symptoms:
- top status bar redraw gets interrupted
- random ACP message lines appear between frames
- full-screen UI looks broken/flickery

Likely source:
- opencode client currently writes selected ACP lines directly to stderr (`writeConsoleLine(os.Stderr, ...)`) in addition to emitting structured events.
- In `--stream | yolo-tui --events-stdin` mode, those direct stderr writes bypass the event pipeline and interfere with TUI rendering.

Required behavior:
- In stream/TUI mode, all runner information must go through structured events only.
- No unsolicited raw ACP output should be printed directly to terminal while TUI is active.
- Keep log persistence intact in JSONL files.

Implementation options:
1) Add a quiet/structured-only mode to opencode client and enable it from yolo-agent when `--stream` is set.
2) Route these lines into event sink as `runner_output` / `runner_warning` and suppress direct stderr writes.
3) Keep direct stderr only for non-TUI contexts (explicit fallback mode).,

## Acceptance Criteria

Given `yolo-agent --stream | yolo-tui --events-stdin`,
when ACP emits agent messages/updates,
then no raw `ACP[...]` lines are printed directly into the terminal.

Given live event flow,
when the top bar and panels redraw,
then the fullscreen layout remains intact (no out-of-band line injections).

Given non-stream/non-TUI runs,
when useful diagnostics are needed,
then logging still appears in JSONL artifacts and optional fallback console output remains available where intended.

