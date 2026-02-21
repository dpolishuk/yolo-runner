---
id: yr-abz7
status: open
deps: []
links: []
created: 2026-02-21T20:22:46Z
type: epic
priority: -1
assignee: Gennady Evstratov
---
# Epic: Task Subsystem Architecture Refactoring

Refactor task management into two distinct layers: Storage Backend (pluggable data access) and Task Engine (graph-based scheduler with dependency resolution and smart concurrency). This addresses the current bug where GitHub tracker treats all repository issues as children of the root epic.

## Problem
Current TaskManager interface conflates storage and scheduling:
- GitHub fetches ALL issues as flat list
- No proper parent-child relationship handling
- No dependency resolution
- Static concurrency instead of graph-based calculation

## Solution
Split into two layers:
1. StorageBackend - authentication, data fetching, format conversion
2. TaskEngine - graph construction, dependency resolution, scheduling, concurrency calculation

## Success Criteria
- GitHub backend correctly fetches only sub-issues of root
- Task engine builds accurate dependency graph
- Concurrency calculated from graph structure
- All existing tests pass
- New architecture used to implement itself (dogfooding)

## Acceptance Criteria
- Given Epic #52 with 8 sub-tasks, when agent runs, only those 8 tasks are processed
- Given tasks with dependencies, when graph is built, execution respects dependency order
- Given a complex task tree, when concurrency is calculated, it equals max parallelizable tasks
- Given external status change, when tree is refreshed, graph reflects new state

## References
- ADR: docs/adr/ADR-001-task-subsystem-refactoring.md
- Design Doc: docs/design/task-subsystem-refactor.md

## Run Commands

### Execute this epic (tk tracker - no GitHub token needed):
```bash
./bin/yolo-agent --repo . --root yr-abz7 --stream 2>&1 | tee runner-logs/yr-abz7-$(date +%Y%m%d-%H%M%S).jsonl | ./bin/yolo-tui
```

### Execute first ready task:
```bash
./bin/yolo-agent --repo . --root yr-b3z3 --stream 2>&1 | tee runner-logs/yr-b3z3-$(date +%Y%m%d-%H%M%S).jsonl | ./bin/yolo-tui
```

### Monitor logs (AI assistant can tail these):
```bash
# Watch latest log file
tail -f runner-logs/yr-abz7-*.jsonl | jq -c '{type: .type, task: .task_title, phase: .phase}'

# Or watch events in real-time
tail -f runner-logs/yr-abz7-*.jsonl
```

**Note:** This epic uses **tk tracker** (local markdown tickets), so no GitHub token is required. The logs are saved to `runner-logs/` for observability.

