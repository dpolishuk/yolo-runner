---
id: yr-ym07
status: closed
deps: [yr-zuw5]
links: []
created: 2026-02-09T21:10:20Z
type: task
priority: 0
assignee: Gennady Evstratov
parent: yr-hd5e
---
# v2.6 Implement OpenCode runner as CLI adapter

Run opencode as external CLI in non-interactive mode; no embedded agent semantics in runner core

## Acceptance Criteria

Given a task prompt, when runner executes, then it launches opencode CLI, captures logs, and returns normalized status

