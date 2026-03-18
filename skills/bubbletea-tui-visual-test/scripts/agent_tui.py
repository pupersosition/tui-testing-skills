#!/usr/bin/env python3
"""Dispatcher entrypoint for Bubble Tea TUI automation contract.

Supports:
- one-shot requests via --request / --request-file
- persistent session mode via --repl (JSONL over stdin/stdout)
"""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
from typing import Any, Dict, Iterable

from session_contract import error
from session_engine import SessionEngine
from visual_pipeline import VisualPipeline

SCHEMA_VERSION = "1.0.0"
SUPPORTED_COMMANDS = (
    "open",
    "close",
    "press",
    "type",
    "wait",
    "snapshot",
    "assert-visual",
    "record",
)

REQUIRED_FIELDS: Dict[str, tuple[str, ...]] = {
    "open": ("cmd", "cwd", "cols", "rows"),
    "close": ("session_id",),
    "press": ("session_id", "key"),
    "type": ("session_id", "text"),
    "wait": ("session_id", "timeout_ms"),
    "snapshot": ("session_id", "name"),
    "assert-visual": ("session_id", "name", "baseline_path"),
    "record": ("session_id", "output_path"),
}

DEPRECATION_GUIDANCE = (
    "Deprecated: python3 skills/bubbletea-tui-visual-test/scripts/agent_tui.py "
    "is a migration compatibility path. Prefer `go run ./cmd/agent-tui`."
)


class AgentTUIDispatcher:
    def __init__(self, *, root_output_dir: str | Path | None = None) -> None:
        self.engine = SessionEngine()
        self.root_output_dir = Path(root_output_dir) if root_output_dir else None
        self._pipelines: Dict[str, VisualPipeline] = {}

    def handle(self, request: Dict[str, Any]) -> Dict[str, Any]:
        normalized = self._validate_request(request)
        if isinstance(normalized, dict) and normalized.get("ok") is False:
            return normalized

        command = normalized["command"]
        params = normalized["params"]

        if command in {"open", "close", "press", "type", "wait"}:
            result = self.engine.execute(command, params)
            if command == "close" and result.get("ok"):
                self._pipelines.pop(result["session_id"], None)
            return result

        if command == "snapshot":
            return self._snapshot(params)

        if command == "assert-visual":
            return self._assert_visual(params)

        if command == "record":
            return self._record(params)

        return error("", "UNKNOWN_COMMAND", f"Unsupported command: {command}")

    def shutdown(self) -> None:
        for session_id in list(self._active_session_ids()):
            self.engine.close({"session_id": session_id})

    def _active_session_ids(self) -> Iterable[str]:
        # Accessing _sessions is acceptable here because dispatcher is the owning integration layer.
        return tuple(self.engine._sessions.keys())  # noqa: SLF001

    def _pipeline_for(self, session_id: str, output_dir: str | None = None) -> VisualPipeline:
        pipeline = self._pipelines.get(session_id)
        if pipeline is not None:
            return pipeline

        if output_dir:
            pipeline = VisualPipeline(run_dir=output_dir)
        elif self.root_output_dir:
            pipeline = VisualPipeline(root_output_dir=self.root_output_dir)
        else:
            pipeline = VisualPipeline()

        self._pipelines[session_id] = pipeline
        return pipeline

    def _snapshot(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session_id = str(params["session_id"])
        runtime_metadata = self.engine.runtime_metadata(session_id)
        if runtime_metadata is None:
            return error(session_id, "SESSION_NOT_FOUND", f"Unknown session: {session_id}")

        screen_text = params.get("screen_text")
        if not isinstance(screen_text, str) or not screen_text:
            screen_text = self.engine.screen_text(session_id) or ""

        if not screen_text.strip():
            return error(
                session_id,
                "MISSING_SCREEN_TEXT",
                "No screen text available. Run a wait command first or provide params.screen_text.",
            )

        pipeline = self._pipeline_for(session_id, params.get("output_dir"))
        return pipeline.snapshot(
            session_id=session_id,
            name=str(params["name"]),
            screen_text=screen_text,
            runtime_metadata=runtime_metadata,
        )

    def _assert_visual(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session_id = str(params["session_id"])
        pipeline = self._pipelines.get(session_id)
        if pipeline is None:
            return error(
                session_id,
                "MISSING_PIPELINE",
                "No visual pipeline for session. Capture a snapshot before assert-visual.",
            )

        threshold = float(params.get("threshold", 0.0))
        return pipeline.assert_visual(
            session_id=session_id,
            name=str(params["name"]),
            baseline_path=str(params["baseline_path"]),
            threshold=threshold,
        )

    def _record(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session_id = str(params["session_id"])
        pipeline = self._pipelines.get(session_id)
        if pipeline is None:
            return error(
                session_id,
                "MISSING_PIPELINE",
                "No visual pipeline for session. Capture a snapshot before record.",
            )

        frame_duration_ms = int(params.get("frame_duration_ms", 250))
        frame_paths = params.get("frame_paths")
        if frame_paths is not None and not isinstance(frame_paths, list):
            return error(
                session_id,
                "INVALID_PARAMS",
                "record params.frame_paths must be an array of paths when provided.",
            )

        return pipeline.record(
            session_id=session_id,
            output_path=str(params["output_path"]),
            frame_paths=frame_paths,
            frame_duration_ms=frame_duration_ms,
        )

    def _validate_request(self, request: Dict[str, Any]) -> Dict[str, Any]:
        if not isinstance(request, dict):
            return error("", "INVALID_REQUEST", "Request must be a JSON object.")

        version = request.get("version")
        if version != SCHEMA_VERSION:
            return error("", "INVALID_VERSION", f"Unsupported request version: {version!r}")

        command = request.get("command")
        if command not in SUPPORTED_COMMANDS:
            return error("", "UNKNOWN_COMMAND", f"Unsupported command: {command!r}")

        params = request.get("params")
        if not isinstance(params, dict):
            return error("", "INVALID_PARAMS", "Request params must be an object.")

        missing = [field for field in REQUIRED_FIELDS[command] if field not in params]
        if missing:
            return error(
                str(params.get("session_id", "")),
                "INVALID_PARAMS",
                f"Missing required params for {command}: {', '.join(missing)}",
            )

        if command == "wait" and not (params.get("match_text") or params.get("match_regex")):
            return error(
                str(params.get("session_id", "")),
                "INVALID_PARAMS",
                "wait requires match_text or match_regex",
            )

        return {"command": command, "params": params}


def _parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Bubble Tea TUI command dispatcher")
    parser.add_argument("--request", help="JSON request string")
    parser.add_argument("--request-file", help="Path to JSON request file")
    parser.add_argument("--repl", action="store_true", help="Read one JSON request per line from stdin")
    parser.add_argument(
        "--root-output-dir",
        help="Optional root output directory for per-session visual artifacts",
    )
    return parser.parse_args(argv)


def _load_request(args: argparse.Namespace) -> Dict[str, Any]:
    if args.request:
        return json.loads(args.request)
    if args.request_file:
        return json.loads(Path(args.request_file).read_text(encoding="utf-8"))
    raise ValueError("Provide --request or --request-file, or use --repl")


def _repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def _try_go_dispatcher(argv: list[str]) -> int | None:
    command = ["go", "run", "./cmd/agent-tui", *argv]
    try:
        completed = subprocess.run(command, cwd=_repo_root(), check=False)
    except OSError:
        return None
    if completed.returncode != 0:
        return None
    return completed.returncode


def main(argv: list[str] | None = None) -> int:
    raw_argv = argv or sys.argv[1:]

    if os.environ.get("BUBBLETEA_TUI_FORCE_PYTHON") != "1":
        print(f"[compat] {DEPRECATION_GUIDANCE}", file=sys.stderr)
        go_exit = _try_go_dispatcher(raw_argv)
        if go_exit is not None:
            return go_exit
        print(
            "[compat] Go dispatcher unavailable; falling back to legacy Python runtime.",
            file=sys.stderr,
        )
    else:
        print(
            "[compat] BUBBLETEA_TUI_FORCE_PYTHON=1 set; running legacy Python dispatcher.",
            file=sys.stderr,
        )

    args = _parse_args(raw_argv)
    dispatcher = AgentTUIDispatcher(root_output_dir=args.root_output_dir)

    try:
        if args.repl:
            for raw_line in sys.stdin:
                line = raw_line.strip()
                if not line:
                    continue
                try:
                    request = json.loads(line)
                except json.JSONDecodeError as exc:
                    response = error("", "INVALID_JSON", str(exc))
                else:
                    response = dispatcher.handle(request)
                print(json.dumps(response, sort_keys=True), flush=True)
            return 0

        request = _load_request(args)
        response = dispatcher.handle(request)
        print(json.dumps(response, sort_keys=True))
        return 0
    except ValueError as exc:
        print(json.dumps(error("", "INVALID_ARGS", str(exc))), file=sys.stderr)
        return 2
    finally:
        dispatcher.shutdown()


if __name__ == "__main__":
    raise SystemExit(main())
