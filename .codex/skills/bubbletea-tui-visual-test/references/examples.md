# Examples

The script below runs an end-to-end flow against the fixture app and emits both snapshot and GIF artifacts.

Prerequisites:

- Dispatcher command exists (for example `.codex/skills/bubbletea-tui-visual-test/scripts/agent_tui.py`)
- `jq` is installed
- Go toolchain is available for the fixture (`go run .`)

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SKILL_DIR="$ROOT/.codex/skills/bubbletea-tui-visual-test"
DISPATCHER="$SKILL_DIR/scripts/agent_tui.py"
FIXTURE_DIR="$SKILL_DIR/assets/fixtures/bubbletea-counter"

RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)-$$-$(LC_ALL=C tr -dc 'a-f0-9' </dev/urandom | head -c 6)"
OUT_DIR="$ROOT/.context/artifacts/bubbletea-tui-visual-test/$RUN_ID"
mkdir -p "$OUT_DIR"/{logs,metadata,snapshots,diffs,gifs}
mkdir -p "$SKILL_DIR/references/baselines"

dispatch() {
  local req="$1"
  python "$DISPATCHER" --request "$req"
}

OPEN_REQ="$(jq -nc \
  --arg cmd "go run ." \
  --arg cwd "$FIXTURE_DIR" \
  '{
    version: "1.0.0",
    command: "open",
    params: {
      cmd: $cmd,
      cwd: $cwd,
      cols: 80,
      rows: 24,
      env: {
        TERM: "xterm-256color",
        LANG: "C.UTF-8",
        LC_ALL: "C.UTF-8",
        TZ: "UTC"
      },
      locale: "C.UTF-8",
      theme: "light",
      color_mode: "256"
    }
  }')"

OPEN_RES="$(dispatch "$OPEN_REQ")"
echo "$OPEN_RES" >>"$OUT_DIR/logs/commands.jsonl"
SESSION_ID="$(echo "$OPEN_RES" | jq -r '.session_id')"

cleanup() {
  CLOSE_REQ="$(jq -nc --arg sid "$SESSION_ID" \
    '{version:"1.0.0",command:"close",params:{session_id:$sid}}')"
  CLOSE_RES="$(dispatch "$CLOSE_REQ" || true)"
  echo "$CLOSE_RES" >>"$OUT_DIR/logs/commands.jsonl"
}
trap cleanup EXIT

WAIT_READY_REQ="$(jq -nc --arg sid "$SESSION_ID" \
  '{version:"1.0.0",command:"wait",params:{session_id:$sid,match_text:"STATUS: READY",timeout_ms:3000}}')"
dispatch "$WAIT_READY_REQ" >>"$OUT_DIR/logs/commands.jsonl"

PRESS_PLUS_REQ="$(jq -nc --arg sid "$SESSION_ID" \
  '{version:"1.0.0",command:"press",params:{session_id:$sid,key:"+"}}')"
dispatch "$PRESS_PLUS_REQ" >>"$OUT_DIR/logs/commands.jsonl"

WAIT_COUNT_REQ="$(jq -nc --arg sid "$SESSION_ID" \
  '{version:"1.0.0",command:"wait",params:{session_id:$sid,match_text:"Counter: 1",timeout_ms:3000}}')"
dispatch "$WAIT_COUNT_REQ" >>"$OUT_DIR/logs/commands.jsonl"

SNAP_REQ="$(jq -nc --arg sid "$SESSION_ID" --arg out "$OUT_DIR/snapshots" \
  '{version:"1.0.0",command:"snapshot",params:{session_id:$sid,name:"counter-1",output_dir:$out}}')"
SNAP_RES="$(dispatch "$SNAP_REQ")"
echo "$SNAP_RES" >>"$OUT_DIR/logs/commands.jsonl"
SNAP_PATH="$(echo "$SNAP_RES" | jq -r '.data.snapshot_path')"

BASELINE_PATH="$SKILL_DIR/references/baselines/counter-1.png"
if [[ ! -f "$BASELINE_PATH" ]]; then
  cp "$SNAP_PATH" "$BASELINE_PATH"
fi

ASSERT_REQ="$(jq -nc --arg sid "$SESSION_ID" --arg bp "$BASELINE_PATH" \
  '{version:"1.0.0",command:"assert-visual",params:{session_id:$sid,name:"counter-1",baseline_path:$bp,threshold:0.005}}')"
dispatch "$ASSERT_REQ" >>"$OUT_DIR/logs/commands.jsonl"

RECORD_REQ="$(jq -nc --arg sid "$SESSION_ID" --arg out "$OUT_DIR/gifs/counter-flow.gif" \
  '{version:"1.0.0",command:"record",params:{session_id:$sid,output_path:$out}}')"
dispatch "$RECORD_REQ" >>"$OUT_DIR/logs/commands.jsonl"

echo "Artifacts written to: $OUT_DIR"
```

Expected outputs:

- Snapshot at `$OUT_DIR/snapshots/counter-1.png`
- GIF at `$OUT_DIR/gifs/counter-flow.gif`
- Per-command JSON transcript at `$OUT_DIR/logs/commands.jsonl`
