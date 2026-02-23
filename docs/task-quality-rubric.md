# Task Quality Rubric (Readiness Review v1)

## Purpose

This rubric defines how to rate whether a task is ready for implementation and
how to validate task requirements before starting work.

Use this rubric for task description quality, not code quality.

## Required Fields (must be present)

- `Title`: One line that states the work in an outcome-focused way.
- `Description`: Plain-language problem statement with concrete scope.
- `Acceptance Criteria`: One or more bullets using `Given/When/Then` format.
- `Deliverables`: Explicit list of outputs to be produced.
- `Testing Plan`: Specific validation steps to confirm behavior.
- `Definition of Done`: Explicit commands/checks before marking complete.
- `Dependencies/Context`: External systems, profiles, files, and assumptions.

## Clarity Rules (quality gates)

1. **No ambiguity**
   - Every objective should be specific and unambiguous.
2. **Measurable outcomes**
   - Acceptance criteria must be testable and observable.
3. **Scoped**
   - Required files and behavior boundaries must be explicit.
4. **Deterministic wording**
   - Avoid “improve”, “handle all cases”, and “make better” without detail.
5. **Dependency-aware**
   - Mention impacted subsystems, feature flags, and environment assumptions.
6. **Test-first intent**
   - Include at least one command or test that validates each acceptance bullet.

## Scoring

Score each task on the table below. Each row is 0–20 points. Total score is out of 100.

- **Required Fields**: 0–20 (every required field present = 20, missing field = 0)
- **Clarity Rules**: 0–20 (all rules satisfied = 20)
- **Acceptance Coverage**: 0–20 (every acceptance bullet has explicit input/output expectation = 20)
- **Validation Rigor**: 0–20 (tests or commands per bullet = 20)
- **Non-goals and risks**: 0–20 (explicit non-goals, risks, or blockers = 20)

Readiness decision:

- `90–100`: Start implementation
- `80–89`: Start only after explicit reviewer pass
- `< 80`: Blocked until improved
- Missing fields: automatic fail regardless of total score

## Sample Applications

### Good Example

**Title:** Implement task timeout classification for runner stalls  
**Required field completeness:** All required fields present  
**Acceptance Criteria sample:**
- Given a root task, when `yolo-agent --runner-timeout 10m` runs with no output for 30s, then status is marked `blocked` and reason includes `stall`.
- Given a task with ACP permission prompt, when running headless, then it is marked `blocked` and no user input is required.

**Expected score:** `93/100`  
**Why:** All fields present, specific wording, testable outcomes, and explicit validation commands.

### Bad Example

**Title:** Improve reliability  
**Required field completeness:** Missing acceptance criteria and testing plan  
**Acceptance Criteria sample:** “Make the system more stable and robust.”  

**Expected score:** `28/100`  
**Why:** Vague goal, no measurable criteria, no boundaries, no validation plan.
