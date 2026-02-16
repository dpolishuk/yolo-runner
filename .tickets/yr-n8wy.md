---
id: yr-n8wy
status: open
deps: [yr-10ub]
links: []
created: 2026-02-15T17:56:44Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-bs0w
---
# E12-T2 Add CLI routing for config subcommands

Add yolo-agent config command routing for validate/init while preserving existing run behavior.

## Acceptance Criteria

Given parser tests, config subcommands route correctly and legacy run path remains backward compatible; implementation follows strict TDD (RED->GREEN->REFACTOR) with test evidence in notes.


## Notes

**2026-02-15T20:08:17Z**

auto_commit_sha=dc3df5c0b05e30f536d8290a895fe6e91ba5ae38

**2026-02-15T20:08:17Z**

landing_status=blocked

**2026-02-15T20:08:17Z**

triage_reason=git checkout main failed: error: you need to resolve your current index first
.tickets/yr-n8wy.md: needs merge: exit status 1

**2026-02-15T20:08:17Z**

triage_status=blocked
