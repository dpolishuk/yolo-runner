---
id: yr-i4ph
status: open
deps: [yr-yhb9]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-lbx1
---
# E5-T1 Linear auth and profile wiring

STRICT TDD: failing tests first. Use env vars + profile config for Linear credentials.

## Acceptance Criteria

Given missing/invalid Linear auth, when startup runs, then errors are explicit; valid auth passes.

