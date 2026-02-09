---
id: yr-lynv
status: closed
deps: []
links: []
created: 2026-02-09T21:11:13Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.16 Define VCS abstraction contract

Define VCS interface independent of git: branch create/switch, commit, review-branch flow, merge-to-main, push, and conflict/error model

## Acceptance Criteria

Given VCS contract, when agent orchestrates a task, then all branch/merge/push operations are expressed via VCS interface without git-specific types

