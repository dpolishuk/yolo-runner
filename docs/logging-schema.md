# Logging Schema

Runner logs written to `runner-logs/*.jsonl` should conform to the structured schema below.

## Required fields

Each log line must include all of these fields:

- `timestamp` (`string`): RFC3339 UTC timestamp (for example, `2026-02-22T10:00:00Z`)
- `level` (`string`): log severity (for example, `info`, `warn`, `error`, `debug`)
- `component` (`string`): logical subsystem that emitted the entry (for example, `runner`, `opencode`)
- `task_id` (`string`): ID of the task/work item
- `run_id` (`string`): ID of the run instance

## Optional fields

Any additional fields are allowed for component-specific payloads, such as:

- `issue_id`
- `request_type`
- `decision`
- `message`
- `request_id`

## Example lines

```json
{"timestamp":"2026-02-22T10:00:00Z","level":"info","component":"runner","task_id":"task-99","run_id":"run-99","issue_id":"task-99","title":"Fix logging","status":"started"}
{"timestamp":"2026-02-22T10:00:01Z","level":"info","component":"opencode","task_id":"task-99","run_id":"run-99","issue_id":"task-99","request_type":"update","decision":"allow","message":"tool call completed"}
```

Use `internal/logging.ValidateStructuredLogLine` to validate generated sample lines in tests.
