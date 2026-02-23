---
id: yr-70nw
status: closed
deps: []
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-s0go
---
# E2-T1 Finalize runner backend plugin contract

STRICT TDD: failing tests first. Normalize implement/review/timeouts/result mapping across backends.

## Acceptance Criteria

Given backend contract tests, when all adapters run, then they satisfy required interfaces and statuses.


## Notes

**2026-02-11T14:21:29Z**

triage_reason=acp client did not finish after opencode exit
opencode stall category=no_output runner_log=.yolo-runner/clones/yr-70nw/runner-logs/opencode/yr-70nw.jsonl opencode_log=/home/egv/.local/share/opencode/log/2026-02-11T123128.log session=ses_3b34fc9f0ffe6szXIm3jJbv93f last_output_age=10m2.809565747s opencode_tail_path=.yolo-runner/clones/yr-70nw/runner-logs/opencode/yr-70nw.jsonl.tail.txt

**2026-02-11T14:21:29Z**

triage_status=blocked

**2026-02-11T14:27:12Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T14:27:12Z**

triage_status=failed

**2026-02-11T15:14:37Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T15:14:37Z**

triage_status=failed

**2026-02-11T15:19:20Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T15:19:20Z**

triage_status=failed

**2026-02-11T16:07:42Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T16:07:42Z**

triage_status=failed

**2026-02-11T16:20:19Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T16:20:19Z**

triage_status=failed

**2026-02-11T16:35:23Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T16:35:23Z**

triage_status=failed

**2026-02-11T16:52:52Z**

triage_reason=review verdict missing explicit pass

**2026-02-11T16:52:52Z**

triage_status=failed

**2026-02-13T20:10:48Z**

Implemented backend contract hardening for timeout/result normalization and opencode adapter nil-safety parity. Validation: go test ./...
