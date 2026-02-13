---
id: yr-yg5e
status: closed
deps: [yr-vhmh]
links: []
created: 2026-02-09T23:07:08Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-lbx1
---
# E5-T4 Implement Linear write operations

STRICT TDD: failing tests first. Implement status/data updates including triage metadata.

## Acceptance Criteria

Given blocked/failed outcomes, when writing updates, then Linear reflects standardized triage fields.


## Notes

**2026-02-13T22:55:24Z**

Implemented strict-TDD Linear write operations in internal/linear/task_manager.go: SetTaskStatus now resolves team workflow states and updates issue state via issueUpdate; SetTaskData now writes deterministic key=value metadata comments via commentCreate (including triage_status/triage_reason). Added failing-first tests in internal/linear/task_manager_test.go for state mutation mapping, missing-state errors, and sorted metadata writes. Also fixed internal/linear package build by removing graphQLError name collision in task manager. Validation: go test ./internal/linear -count=1; go test ./... -count=1. Commit: a6bebc6.
