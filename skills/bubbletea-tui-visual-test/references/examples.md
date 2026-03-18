# Examples

Go-first one-shot request example:

```bash
go run ./cmd/agent-tui --request '{"version":"1.0.0","command":"wait","params":{"session_id":"session-123","match_text":"STATUS: READY","timeout_ms":3000}}'
```

Go-first REPL example:

```bash
printf '%s\n' \
  '{"version":"1.0.0","command":"close","params":{"session_id":"session-123"}}' \
  | go run ./cmd/agent-tui --repl
```

Migration compatibility path (deprecated wrapper around dispatcher migration):

```bash
python3 skills/bubbletea-tui-visual-test/scripts/agent_tui.py --request '{"version":"1.0.0","command":"close","params":{"session_id":"session-123"}}'
```

Compatibility integration runner (legacy Python runtime) for deterministic end-to-end flow:

```bash
bash skills/bubbletea-tui-visual-test/references/examples.sh
```

The runner performs:

1. `open` fixture Bubble Tea app (`go run .`)
2. `wait` for `STATUS: READY`
3. `press` `+`
4. `wait` for `Counter: 1`
5. `snapshot` checkpoint `ready`
6. `snapshot` checkpoint `counter-1`
7. `assert-visual` against baseline
8. `record` GIF artifact (multi-frame: `ready` -> `counter-1`)
9. `close` session

Prerequisites:

- Go toolchain (`go version`)
- Python 3 with Pillow (`python3 -c 'import PIL'`) for compatibility runner
- Go toolchain for the fixture app

Outputs:

- Per-run artifacts under `.context/artifacts/bubbletea-tui-visual-test/<run_id>/`
- JSON command transcript at `<run>/logs/commands.jsonl`
- Snapshot PNGs at `<run>/checkpoints/ready.png` and `<run>/checkpoints/counter-1.png`
- Snapshot metadata at `<run>/metadata/counter-1.metadata.json`
- GIF at `<run>/gifs/counter-flow.gif`
- Baseline (auto-initialized on first run) at `references/baselines/counter-1.png`

Note: the integration runner uses deterministic fixture screen payloads for `ready` and `counter-1` checkpoints so visual assertions are stable and meaningful across runs.
