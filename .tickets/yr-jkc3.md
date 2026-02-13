---
id: yr-jkc3
status: closed
deps: []
links: []
created: 2026-02-11T14:38:01Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-qggb
---
# E7-T26 Keep yolo-tui open after stream ends (less-like exit)

Bug report: when yolo-agent finishes, yolo-tui exits immediately, so users cannot inspect final state, task outcomes, or last activity without rerunning.

Problem:
- stream EOF currently terminates the UI session right away
- users miss final summary/status context
- no time to scroll history/details after completion

Desired behavior (less-like):
- when input stream ends, yolo-tui should switch to a completed/idle viewing state and stay open
- user explicitly exits with a key (e.g. q / Ctrl+C)
- show a clear footer hint like: "Stream ended — press q to quit"
- keep scroll/navigation active for panels, activity, and history

Implementation notes:
- treat stdin EOF as state transition, not immediate process exit
- preserve current event model/snapshots; just stop ingest loop
- optionally provide an escape hatch flag for current behavior (e.g. --exit-on-eof) if needed for scripts

## Acceptance Criteria

Given `yolo-agent --stream | yolo-tui --events-stdin`,
when yolo-agent finishes and stdin reaches EOF,
then yolo-tui remains open in a read-only final state instead of exiting immediately.

Given stream has ended,
when user navigates/scrolls,
then panels and history remain usable for post-run inspection.

Given stream has ended,
when user presses `q` (or Ctrl+C),
then yolo-tui exits cleanly.

Given a non-interactive/scripted use case,
when auto-exit mode is explicitly enabled (if implemented),
then yolo-tui can still terminate on EOF deterministically.


## Notes

**2026-02-12T08:10:25Z**

triage_reason=acp client did not finish after opencode exit
opencode stall category=permission runner_log=.yolo-runner/clones/yr-jkc3/runner-logs/opencode/yr-jkc3.jsonl opencode_log=/home/egv/.local/share/opencode/log/2026-02-11T123128.log session=ses_3b34fc9f0ffe6szXIm3jJbv93f last_output_age=10m1.376723456s opencode_tail_path=.yolo-runner/clones/yr-jkc3/runner-logs/opencode/yr-jkc3.jsonl.tail.txt

**2026-02-12T08:10:25Z**

triage_status=blocked
