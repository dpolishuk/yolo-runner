# Executor Configuration

This document defines the executor configuration contract used for task-level pipeline orchestration.
Configurations support YAML and JSON encodings.

## Required Top Level Fields

- `name` (`string`, required): Logical executor configuration name.
- `type` (`string`, required): Executor configuration category.
- `backend` (`string`, required): Backend implementation name (for example `codex`, `claude`, `kimi`).
- `pipeline` (`object`, required): Pipeline definition.

## Required Pipeline Stages

- `quality_gate`
- `execute`
- `qc_gate`
- `complete`

Each stage object supports:

- `tools` (array of strings)
- `retry` (object with retry policy)
- `transitions` (object with `on_success` and `on_failure`)

### Retry policy

Each stage must define:

- `max_attempts` (integer, minimum 1): Retry budget for the stage.
- Optional: `initial_delay_ms`, `backoff_ms`, `max_delay_ms` for delay strategy tuning.

### Transition contract

Each stage defines transition behavior for:

- `on_success`
- `on_failure`

Each transition object requires:

- `action`: `next`, `retry`, `fail`, or `complete`
- `condition`: deterministic boolean expression string
- `next_stage`: required when `action` is `next`; must be one of `quality_gate`, `execute`, `qc_gate`, or `complete`.

Allowed condition expressions are deterministic and must be parseable directly without side effects.
Examples:

- `quality_score < threshold`
- `tests_failed`
- `review_failed`

## JSON Schema

The canonical schema is:

`docs/executor-configuration-schema.json`

## Valid example (JSON)

```json
{
  "name": "feature-executor",
  "type": "task",
  "backend": "codex",
  "pipeline": {
    "quality_gate": {
      "tools": ["reviewer"],
      "retry": { "max_attempts": 2, "initial_delay_ms": 500, "backoff_ms": 200 },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "execute",
          "condition": "true"
        },
        "on_failure": {
          "action": "retry",
          "condition": "review_failed"
        }
      }
    },
    "execute": {
      "tools": ["shell", "git", "reviewer"],
      "retry": { "max_attempts": 3, "initial_delay_ms": 200, "backoff_ms": 300 },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "qc_gate",
          "condition": "true"
        },
        "on_failure": {
          "action": "retry",
          "condition": "tests_failed"
        }
      }
    },
    "qc_gate": {
      "tools": ["quality-checker"],
      "retry": { "max_attempts": 2, "initial_delay_ms": 300 },
      "transitions": {
        "on_success": {
          "action": "next",
          "next_stage": "complete",
          "condition": "quality_score >= threshold"
        },
        "on_failure": {
          "action": "fail",
          "condition": "review_failed"
        }
      }
    },
    "complete": {
      "tools": ["git"],
      "retry": { "max_attempts": 1 },
      "transitions": {
        "on_success": {
          "action": "complete",
          "condition": "true"
        },
        "on_failure": {
          "action": "fail",
          "condition": "true"
        }
      }
    }
  }
}
```

## Invalid example (YAML)

```yaml
name: ""
type: task
backend: codex
pipeline:
  quality_gate:
    tools: []
    retry:
      max_attempts: 0
    transitions:
      on_success:
        action: next
        condition: true
```

### Why this is invalid

- `name` is empty.
- `quality_gate.tools` is empty.
- `quality_gate.retry.max_attempts` is below minimum.
- `quality_gate.transitions.on_success` lacks `next_stage` for action `next`.
- `quality_gate.transitions.on_failure` is missing.
- `quality_gate.transitions.on_success.condition` and failure transitions use unsupported expression forms.
