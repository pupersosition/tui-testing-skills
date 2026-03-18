# CI Workflow

This document defines CI invocation and baseline refresh rules for the Bubble Tea visual testing skill.

## CI Steps

1. Validate Go dispatcher/contract tests:

```bash
go test ./...
```

2. Validate Python code and tests for migration compatibility workstreams:

```bash
python3 -m pytest skills/bubbletea-tui-visual-test/scripts/tests
```

3. Validate fixture app compiles:

```bash
(
  cd skills/bubbletea-tui-visual-test/assets/fixtures/bubbletea-counter
  go build ./...
)
```

4. Run integration flow (open -> interact -> wait -> snapshot -> assert-visual -> record):

```bash
bash skills/bubbletea-tui-visual-test/references/examples.sh
```

If your pipeline does not materialize `examples.sh`, run the script body from `references/examples.md`.

## Required Artifacts Per Run

Each run MUST write to a unique run directory under:

`<repo>/.context/artifacts/bubbletea-tui-visual-test/<run_id>/`

Required outputs:

- `logs/commands.jsonl`
- `checkpoints/*.png`
- `metadata/*.metadata.json`
- `gifs/*.gif` (or explicit renderer-unavailable error captured in logs)

## Baseline Update Workflow

Use this workflow only for intentional UI changes:

1. Run the integration flow and inspect produced artifacts.
2. Compare visual diffs and confirm expected design change.
3. Copy approved snapshot to baseline path (for example `references/baselines/<checkpoint>.png`).
4. Re-run visual assertions to ensure green status with the new baseline.
5. Include baseline update rationale in commit message or PR notes.

## Failure Triage

- `WAIT_TIMEOUT`: increase timeout only after confirming app startup latency.
- `renderer_unavailable`: install Pillow (`pip install pillow`) in CI image.
- `missing_baseline`: ensure baseline is checked in for required checkpoints.
- `difference_ratio > threshold`: inspect diff artifact before deciding on baseline refresh.

## Ownership Gate

Before merging parallel workstreams, verify:

- Session automation files remain under `scripts/session_*`.
- Visual pipeline files remain under `scripts/visual_*`.
- Workflow/reference docs remain under `references/` and `SKILL.md`.
- Dispatcher contract remains aligned with `references/command-schema.json`.
