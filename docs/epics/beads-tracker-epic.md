# Epic: beads as first-class tracker in yolo-agent

## Requirements (IMMUTABLE)
- yolo-agent supports beads as a tracker type alongside tk, linear, and github
- Config format: `tracker.type: beads` in .yolo-runner/config.yaml profiles
- Optional `beads.scope.root` for restricting to a specific root task ID
- No authentication required (beads is local file-based)
- Auto-detection of .beads/ directory as beads tracker

## Success Criteria
- [ ] yolo-agent --tracker beads works with --profile pointing to beads config
- [ ] TaskManager and StorageBackend interfaces implemented for beads
- [ ] Scope validation works (optional beads.scope.root)
- [ ] Auto-detection prefers beads when .beads/ directory exists
- [ ] Tests pass for beads tracker integration
- [ ] Config validate command supports beads

## Anti-Patterns (FORBIDDEN)
- ❌ NO separate auth config (beads is local-only, no auth needed)
- ❌ NO scope requirement (beads.scope.root is optional)
- ❌ NO new adapter - reuse existing internal/beads Adapter

## Approach
Add beads tracker support to yolo-agent following the existing tk/linear/github patterns:
1. Add trackerTypeBeads constant and config structs
2. Create TaskManager and StorageBackend in internal/beads (wrapping existing Adapter)
3. Add cases to buildTaskManagerForTracker and buildStorageBackendForTracker
4. Add validation in validateTrackerModel
5. Add tests

## Architecture
- cmd/yolo-agent/tracker_profile.go: Add beads type, config structs, builder cases
- internal/beads/task_manager.go: New TaskManager implementation
- internal/beads/storage_backend.go: New StorageBackend implementation

## Design Rationale

### Problem
yolo-runner currently supports tk, linear, and github as issue trackers. Users want to use beads (distributed git-backed graph issue tracker) as their tracker, but yolo-agent lacks beads support.

### Research Findings
**Codebase:**
- internal/beads/beads.go has Adapter with Ready(), Tree(), Show(), UpdateStatus(), Close() methods
- tk uses pattern: Adapter wrapped by TaskManager and StorageBackend
- tracker_profile.go shows how to add new tracker types

**External:**
- beads uses Dolt SQL backend (.beads/ directory)
- CLI commands: bd ready, bd list, bd show, bd update, bd close

### Approaches Considered

#### 1. New TaskManager/StorageBackend in internal/beads ✓

**What it is:** Create new task_manager.go and storage_backend.go in internal/beads that wrap the existing Adapter and implement contracts interfaces

**Pros:**
- Follows tk pattern (Adapter wrapped by TaskManager/StorageBackend)
- Reuses existing Adapter code
- Clean separation

**Cons:**
- More files to create

**Chosen because:** Matches existing tk pattern, maximum code reuse

#### 2. Modify existing Adapter to implement interfaces

**What it is:** Add contracts interface methods directly to existing Adapter

**Why rejected:** Adapter is in different package, mixing concerns, harder to test

### Scope Boundaries

**In scope:**
- beads as tracker type
- Optional scope.root validation
- Auto-detection of .beads/ directory
- Config validation

**Out of scope:**
- beads server/sync (local-only operation)
- Any authentication (not needed)

### Open Questions
- Should .beads/ be preferred over .tickets/ when both exist? (proposed: yes, prefer newer format)
