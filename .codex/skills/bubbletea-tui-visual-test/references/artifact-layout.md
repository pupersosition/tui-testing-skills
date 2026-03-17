# Artifact Layout

Use per-run directories so concurrent agents never overwrite each other.

## Root Convention

Default root:

`<repo>/.context/artifacts/bubbletea-tui-visual-test/<run_id>/`

`run_id` format:

`YYYYMMDDTHHMMSSZ-<pid>-<shortid>`

Example:

`20260317T214900Z-43122-a7c2f1`

## Directory Structure

```text
<run_id>/
  logs/
  metadata/
  snapshots/
  diffs/
  gifs/
```

## Naming Rules

- Snapshot PNG: `snapshots/<checkpoint>.png`
- Snapshot metadata: `metadata/<checkpoint>.json`
- Visual diff PNG: `diffs/<checkpoint>.diff.png`
- Visual diff metadata: `diffs/<checkpoint>.diff.json`
- GIF output: `gifs/<flow-name>.gif`
- Command transcript: `logs/commands.jsonl`

## Parallel Safety Rules

1. Never write to another run's directory.
2. Never reuse a `run_id`.
3. Treat baselines as read-only during assertion runs.
4. Copy/update baselines only in an explicit baseline-refresh workflow.
