---
id: yr-py4x
status: closed
deps: [yr-96qx, yr-ma1p]
links: []
created: 2026-02-09T23:07:08Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-4jk8
---
# E6-T3 Implement GitHub read operations

STRICT TDD: failing tests first. Implement next/show against configured repo.

## Acceptance Criteria

Given repo issues, when querying, then next/selectable tasks are returned correctly.


## Notes

**2026-02-13T23:23:14Z**

Implemented GitHub read operations with strict TDD. Added NextTasks/GetTask coverage for dependency filtering, priority ordering, pull-request exclusion, and parent leaf fallback. Implemented GitHub REST read flow (repo issues pagination + issue fetch), task-graph mapping, dependency metadata extraction, and status mapping for open/closed issues. Validation: go test ./internal/github -count=1; go test ./... -count=1.
