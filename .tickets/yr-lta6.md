---
id: yr-lta6
status: closed
deps: [yr-z5bz, yr-1d6f]
links: []
created: 2026-02-09T21:11:13Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.20 Merge and push workflow via VCS abstraction

If review succeeds, merge task branch into main and push main; handle merge/push failures with task status updates

## Acceptance Criteria

Given successful task+review runs, when finalize step executes, then branch is merged to main and pushed through VCS interface

