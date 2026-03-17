---
name: bubbletea-tui-visual-test
description: Validate Bubble Tea TUIs through deterministic terminal interaction, visual snapshots, and GIF review artifacts.
---

Use this skill to run deterministic Bubble Tea UI verification flows with command-based PTY control.

## Inputs

- Bubble Tea app command and working directory
- Dispatcher entrypoint that accepts command JSON (for example `scripts/agent_tui.py`)
  - Use `--repl` mode for multi-step flows so session state is preserved between commands.
- Output root directory for run artifacts
- Optional visual baseline PNGs

## Command Contract

Requests and responses MUST follow `references/command-schema.json`.

Request envelope:

```json
{
  "version": "1.0.0",
  "command": "open|close|press|type|wait|snapshot|assert-visual|record",
  "params": {}
}
```

Response envelope:

```json
{
  "ok": true,
  "session_id": "session-123",
  "data": {}
}
```

Failure responses return `ok=false` plus `error.code` and `error.message`.

## Workflow

1. Normalize runtime settings before opening a session.
2. Run `open` with explicit `cmd`, `cwd`, `cols`, `rows`, and deterministic metadata (`env`, `locale`, `theme`, `color_mode`).
3. Drive interaction using `press` and `type`.
4. Assert behavioral state with `wait` using `match_text` or `match_regex`.
5. Capture checkpoints with `snapshot`.
6. Compare to baselines with `assert-visual`.
7. Export GIF artifact with `record`.
8. Always run `close` in success and failure paths.

## Artifacts

Write outputs to a unique per-run directory to avoid collisions across concurrent agents. Layout and naming rules are defined in `references/artifact-layout.md`.

## References

- `references/assertions.md`
- `references/runtime-normalization.md`
- `references/artifact-layout.md`
- `references/examples.md`
