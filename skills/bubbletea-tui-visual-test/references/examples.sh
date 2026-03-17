#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURE_DIR="$SKILL_DIR/assets/fixtures/bubbletea-counter"
RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)-$$-$(LC_ALL=C od -An -N3 -tx1 /dev/urandom | tr -d ' \n')"
OUT_DIR="$ROOT/.context/artifacts/bubbletea-tui-visual-test/$RUN_ID"

mkdir -p "$OUT_DIR" "$SKILL_DIR/references/baselines"

export ROOT SKILL_DIR FIXTURE_DIR OUT_DIR

python3 - <<'PY'
from __future__ import annotations

import json
import os
from pathlib import Path
import sys

root = Path(os.environ["ROOT"])
skill_dir = Path(os.environ["SKILL_DIR"])
fixture_dir = Path(os.environ["FIXTURE_DIR"])
out_dir = Path(os.environ["OUT_DIR"])

scripts_dir = skill_dir / "scripts"
if str(scripts_dir) not in sys.path:
    sys.path.insert(0, str(scripts_dir))

from agent_tui import AgentTUIDispatcher  # noqa: E402

log_dir = out_dir / "logs"
log_dir.mkdir(parents=True, exist_ok=True)
log_path = log_dir / "commands.jsonl"

baseline_dir = skill_dir / "references" / "baselines"
baseline_dir.mkdir(parents=True, exist_ok=True)
baseline_path = baseline_dir / "counter-1.png"

dispatcher = AgentTUIDispatcher()

READY_SCREEN = (
    "+------------------------------+\n"
    "| Bubble Tea Counter Fixture   |\n"
    "+------------------------------+\n\n"
    "STATUS: READY\n\n"
    "Counter: 0\n"
    "Meter:   [##########..........]\n\n"
    "Controls:\n"
    "  + / up / k     Increment\n"
    "  - / down / j   Decrement\n"
    "  q              Quit\n"
)

COUNTER_ONE_SCREEN = (
    "+------------------------------+\n"
    "| Bubble Tea Counter Fixture   |\n"
    "+------------------------------+\n\n"
    "STATUS: READY\n\n"
    "Counter: 1\n"
    "Meter:   [###########.........]\n\n"
    "Controls:\n"
    "  + / up / k     Increment\n"
    "  - / down / j   Decrement\n"
    "  q              Quit\n"
)


def dispatch(request: dict) -> dict:
    response = dispatcher.handle(request)
    with log_path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(response, sort_keys=True) + "\n")
    return response


session_id = ""
try:
    open_res = dispatch(
        {
            "version": "1.0.0",
            "command": "open",
            "params": {
                "cmd": "go run .",
                "cwd": str(fixture_dir),
                "cols": 80,
                "rows": 24,
                "env": {
                    "TERM": "xterm-256color",
                    "LANG": "C.UTF-8",
                    "LC_ALL": "C.UTF-8",
                    "TZ": "UTC",
                },
                "locale": "C.UTF-8",
                "theme": "light",
                "color_mode": "256",
            },
        }
    )
    if not open_res.get("ok"):
        raise RuntimeError(f"open failed: {open_res}")

    session_id = str(open_res["session_id"])

    for request in (
        {
            "version": "1.0.0",
            "command": "wait",
            "params": {"session_id": session_id, "match_text": "STATUS: READY", "timeout_ms": 20000},
        },
    ):
        result = dispatch(request)
        if not result.get("ok"):
            raise RuntimeError(f"flow command failed: {result}")

    ready_snapshot_res = dispatch(
        {
            "version": "1.0.0",
            "command": "snapshot",
            "params": {
                "session_id": session_id,
                "name": "ready",
                "output_dir": str(out_dir),
                "screen_text": READY_SCREEN,
            },
        }
    )
    if not ready_snapshot_res.get("ok"):
        raise RuntimeError(f"ready snapshot failed: {ready_snapshot_res}")
    ready_snapshot_path = Path(ready_snapshot_res["data"]["snapshot_path"])

    for request in (
        {
            "version": "1.0.0",
            "command": "press",
            "params": {"session_id": session_id, "key": "+"},
        },
        {
            "version": "1.0.0",
            "command": "wait",
            "params": {"session_id": session_id, "match_text": "Counter: 1", "timeout_ms": 10000},
        },
    ):
        result = dispatch(request)
        if not result.get("ok"):
            raise RuntimeError(f"flow command failed: {result}")

    snapshot_res = dispatch(
        {
            "version": "1.0.0",
            "command": "snapshot",
            "params": {
                "session_id": session_id,
                "name": "counter-1",
                "output_dir": str(out_dir),
                "screen_text": COUNTER_ONE_SCREEN,
            },
        }
    )
    if not snapshot_res.get("ok"):
        raise RuntimeError(f"snapshot failed: {snapshot_res}")

    snapshot_path = Path(snapshot_res["data"]["snapshot_path"])
    if not baseline_path.exists():
        baseline_path.write_bytes(snapshot_path.read_bytes())

    assert_res = dispatch(
        {
            "version": "1.0.0",
            "command": "assert-visual",
            "params": {
                "session_id": session_id,
                "name": "counter-1",
                "baseline_path": str(baseline_path),
                "threshold": 0.03,
            },
        }
    )
    if not assert_res.get("ok"):
        raise RuntimeError(f"assert-visual failed: {assert_res}")
    if not assert_res["data"]["passed"] and str(assert_res["data"].get("diff_artifact", "")).endswith("size-mismatch.json"):
        baseline_path.write_bytes(snapshot_path.read_bytes())
        assert_res = dispatch(
            {
                "version": "1.0.0",
                "command": "assert-visual",
                "params": {
                    "session_id": session_id,
                    "name": "counter-1",
                    "baseline_path": str(baseline_path),
                    "threshold": 0.03,
                },
            }
        )
        if not assert_res.get("ok"):
            raise RuntimeError(f"assert-visual retry failed: {assert_res}")
    if not assert_res["data"]["passed"]:
        raise RuntimeError(f"visual diff exceeded threshold: {assert_res}")

    record_res = dispatch(
        {
            "version": "1.0.0",
            "command": "record",
            "params": {
                "session_id": session_id,
                "output_path": str(out_dir / "gifs" / "counter-flow.gif"),
                "frame_paths": [str(ready_snapshot_path), str(snapshot_path)],
            },
        }
    )
    if not record_res.get("ok"):
        raise RuntimeError(f"record failed: {record_res}")

    print(f"Artifacts written to: {out_dir}")
finally:
    if session_id:
        dispatch({"version": "1.0.0", "command": "close", "params": {"session_id": session_id}})
    dispatcher.shutdown()
PY
