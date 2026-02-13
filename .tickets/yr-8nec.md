---
id: yr-8nec
status: closed
deps: [yr-imaw]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-bbtg
---
# E4-T2 Add tracker profile config model

STRICT TDD: failing tests first. Implement tracker.type and scoped profile settings.

## Acceptance Criteria

Given profile selection, when runner starts, then configured tracker adapter is loaded with validated scope/auth.


## Notes

**2026-02-13T21:48:51Z**

Implemented tracker profile config model in yolo-agent: added --profile/YOLO_PROFILE selection, .yolo-runner/config.yaml loader with tracker.type validation (tk + linear scope/auth checks), startup tracker adapter wiring, and strict TDD coverage. Validation: go test ./cmd/yolo-agent && go test ./... -timeout 120s. Commit: 7387d4e.
