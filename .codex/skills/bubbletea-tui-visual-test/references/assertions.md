# Assertions Guide

Use two assertion lanes:

1. Behavioral assertions (`wait`) to prove state transitions.
2. Visual assertions (`assert-visual`) to catch rendering regressions.

## Behavioral Assertions (`wait`)

`wait` requires:

- `session_id`
- `timeout_ms`
- Exactly one of `match_text` or `match_regex`

Recommended patterns:

- Use `match_text` for stable labels such as `STATUS: READY`.
- Use `match_regex` for dynamic values such as `Counter: [0-9]+`.
- Keep `timeout_ms` tight (for example 1000-5000) so failures are actionable.

Example:

```json
{
  "version": "1.0.0",
  "command": "wait",
  "params": {
    "session_id": "session-123",
    "match_text": "Counter: 1",
    "timeout_ms": 3000
  }
}
```

## Visual Assertions (`assert-visual`)

`assert-visual` requires:

- `session_id`
- `name` (checkpoint name)
- `baseline_path` (expected PNG)

Optional:

- `threshold` float in `[0, 1]` (default should come from your runner)

Recommended policy:

- PR gate: `threshold` between `0` and `0.01`.
- Platform-variant jobs: allow slightly higher threshold and report diff artifacts.
- Fail if runtime metadata does not match baseline metadata.

Example:

```json
{
  "version": "1.0.0",
  "command": "assert-visual",
  "params": {
    "session_id": "session-123",
    "name": "counter-1",
    "baseline_path": ".codex/skills/bubbletea-tui-visual-test/references/baselines/counter-1.png",
    "threshold": 0.005
  }
}
```

## Failure Handling

- Persist command responses for each step.
- Attach latest snapshot, diff image, and metadata to CI output.
- Always issue `close` even when assertions fail.
