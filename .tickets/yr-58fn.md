---
id: yr-58fn
status: closed
deps: [yr-70nw]
links: []
created: 2026-02-09T23:07:07Z
type: task
priority: 1
assignee: Gennady Evstratov
parent: yr-s0go
---
# E2-T2 Add backend capability matrix

STRICT TDD: failing tests first. Record capabilities like supports_review and supports_stream.

## Acceptance Criteria

Given backend configs, when selecting backend, then unsupported modes are rejected predictably.


## Notes

**2026-02-11T18:31:54Z**

landing_status=blocked

**2026-02-11T18:31:54Z**

triage_reason=git push origin main failed: remote: error: refusing to update checked out branch: refs/heads/main        
remote: error: By default, updating the current branch in a non-bare repository        
remote: is denied, because it will make the index and work tree inconsistent        
remote: with what you pushed, and will require 'git reset --hard' to match        
remote: the work tree to HEAD.        
remote: 
remote: You can set the 'receive.denyCurrentBranch' configuration variable        
remote: to 'ignore' or 'warn' in the remote repository to allow pushing into        
remote: its current branch; however, this is not recommended unless you        
remote: arranged to update its work tree to match what you pushed in some        
remote: other way.        
remote: 
remote: To squelch this message and still keep the default behaviour, set        
remote: 'receive.denyCurrentBranch' configuration variable to 'refuse'.        
To /home/egv/dev/yolo-runner/.
 ! [remote rejected] main -> main (branch is currently checked out)
error: failed to push some refs to '/home/egv/dev/yolo-runner/.': exit status 1

**2026-02-11T18:31:54Z**

triage_status=blocked

**2026-02-12T20:34:26Z**

triage_reason=exit status 1

**2026-02-12T20:34:26Z**

triage_status=failed

**2026-02-12T20:45:55Z**

triage_reason=review verdict missing explicit pass

**2026-02-12T20:45:55Z**

triage_status=failed

**2026-02-12T20:54:49Z**

triage_reason=review verdict missing explicit pass

**2026-02-12T20:54:49Z**

triage_status=failed

**2026-02-12T21:20:02Z**

triage_reason=review verdict returned fail

**2026-02-12T21:20:02Z**

triage_status=failed

**2026-02-13T08:00:03Z**

triage_reason=review verdict returned fail

**2026-02-13T08:00:03Z**

triage_status=failed

**2026-02-13T20:16:09Z**

Implemented backend capability matrix in yolo-agent with supports_review/supports_stream, added selector validation for backend+mode compatibility, and added strict TDD coverage for unknown backend plus unsupported review/stream modes. Validation: go test ./cmd/yolo-agent && go test ./...
