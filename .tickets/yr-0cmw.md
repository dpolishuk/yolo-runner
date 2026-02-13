---
id: yr-0cmw
status: closed
deps: []
links: []
created: 2026-02-11T14:13:21Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-qggb
---
# E7-T24 Keep top bar single-line by truncating to terminal width

Bug report: top status bar sometimes wraps from 1 line to 2 lines when content is long or terminal width changes, causing visible vertical jumping.

Required behavior:
- Top bar must always render as exactly one visual line.
- If content does not fit, truncate with ellipsis instead of wrapping.
- This should remain stable during live updates while events stream in.

Suggested implementation direction:
- Use display-cell aware width/truncation (not rune-count only) to handle emoji and wide glyphs correctly.
- Keep consistent horizontal padding while preserving one-line height.

## Acceptance Criteria

Given a narrow terminal width,
when status text exceeds available top-bar width,
then top bar stays one visual line and uses truncation.

Given live event updates,
when status content length oscillates around width limit,
then top area height does not jump between one and two lines.

Given automated tests,
when rendering top bar at constrained widths,
then assertions prove single-line output and no wrap regressions.


## Notes

**2026-02-12T08:19:37Z**

triage_reason=context deadline exceeded
signal: killed

**2026-02-12T08:19:37Z**

triage_status=failed

**2026-02-12T08:49:21Z**

triage_reason=acp client did not finish after opencode exit
opencode stall category=no_output runner_log=.yolo-runner/clones/yr-0cmw/runner-logs/opencode/yr-0cmw.jsonl opencode_log=/home/egv/.local/share/opencode/log/2026-02-11T123128.log last_output_age=10m4.611956136s opencode_tail_path=.yolo-runner/clones/yr-0cmw/runner-logs/opencode/yr-0cmw.jsonl.tail.txt

**2026-02-12T08:49:21Z**

triage_status=blocked
