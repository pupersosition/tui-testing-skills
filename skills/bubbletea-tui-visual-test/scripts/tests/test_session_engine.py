from __future__ import annotations

import re
import sys
import unittest
from pathlib import Path
from typing import Any, Dict, List

SCRIPT_DIR = Path(__file__).resolve().parents[1]
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from session_engine import SessionEOFError, SessionEngine, SessionTimeoutError


class FakeChild:
    def __init__(
        self,
        *,
        text_timeout: bool = False,
        regex_timeout: bool = False,
        regex_source: str = "STATUS: READY",
    ) -> None:
        self.text_timeout = text_timeout
        self.regex_timeout = regex_timeout
        self.regex_source = regex_source
        self.events: List[Any] = []
        self.pid = 1001
        self.exitstatus = 0
        self.signalstatus = None
        self._alive = True
        self.match = None
        self.after = ""

    def send(self, text: str) -> None:
        self.events.append(("send", text))

    def sendcontrol(self, key: str) -> None:
        self.events.append(("sendcontrol", key))

    def expect_exact(self, text: str, timeout: float) -> int:
        self.events.append(("expect_exact", text, timeout))
        if self.text_timeout:
            raise SessionTimeoutError("timed out")
        self.after = text
        return 0

    def expect(self, pattern: re.Pattern[str], timeout: float) -> int:
        self.events.append(("expect", pattern.pattern, timeout))
        if self.regex_timeout:
            raise SessionTimeoutError("timed out")
        match = pattern.search(self.regex_source)
        if not match:
            raise SessionEOFError("no match before EOF")
        self.match = match
        self.after = match.group(0)
        return 0

    def isalive(self) -> bool:
        return self._alive

    def terminate(self, force: bool = True) -> bool:
        self.events.append(("terminate", force))
        self._alive = False
        return True

    def close(self, force: bool = True) -> None:
        self.events.append(("close", force))


class FakeSpawner:
    def __init__(self, child: FakeChild) -> None:
        self.child = child
        self.calls: List[Dict[str, Any]] = []

    def __call__(self, *, cmd: str, cwd: str, env: Dict[str, str], cols: int, rows: int) -> FakeChild:
        self.calls.append({"cmd": cmd, "cwd": cwd, "env": env, "cols": cols, "rows": rows})
        return self.child


class SessionEngineTests(unittest.TestCase):
    def make_engine(self, child: FakeChild) -> tuple[SessionEngine, FakeSpawner]:
        spawner = FakeSpawner(child)
        engine = SessionEngine(
            spawner=spawner,
            timeout_error=SessionTimeoutError,
            eof_error=SessionEOFError,
            uuid_factory=lambda: "unit",
        )
        return engine, spawner

    def open_fixture_session(self, engine: SessionEngine) -> str:
        result = engine.open(
            {
                "cmd": "go run ./main.go",
                "cwd": "/tmp/fixture",
                "cols": 100,
                "rows": 32,
                "locale": "C",
                "color_mode": "256",
            }
        )
        self.assertTrue(result["ok"])
        return result["session_id"]

    def test_open_and_close_session_lifecycle(self) -> None:
        child = FakeChild()
        engine, spawner = self.make_engine(child)

        session_id = self.open_fixture_session(engine)
        self.assertEqual(session_id, "session-unit")
        self.assertTrue(engine.has_session(session_id))
        self.assertEqual(spawner.calls[0]["cols"], 100)
        self.assertEqual(spawner.calls[0]["rows"], 32)
        self.assertEqual(spawner.calls[0]["env"]["LANG"], "C")

        closed = engine.close({"session_id": session_id})
        self.assertTrue(closed["ok"])
        self.assertFalse(engine.has_session(session_id))
        self.assertIn(("terminate", True), child.events)
        self.assertIn(("close", True), child.events)

    def test_wait_with_text_match(self) -> None:
        child = FakeChild()
        engine, _ = self.make_engine(child)
        session_id = self.open_fixture_session(engine)

        result = engine.wait(
            {"session_id": session_id, "match_text": "STATUS: READY", "timeout_ms": 250}
        )
        self.assertTrue(result["ok"])
        self.assertEqual(result["data"]["mode"], "text")
        self.assertEqual(result["data"]["matched"], "STATUS: READY")

    def test_wait_timeout_returns_structured_error(self) -> None:
        child = FakeChild(regex_timeout=True)
        engine, _ = self.make_engine(child)
        session_id = self.open_fixture_session(engine)

        result = engine.wait(
            {"session_id": session_id, "match_regex": r"Counter:\s+\d+", "timeout_ms": 25}
        )
        self.assertFalse(result["ok"])
        self.assertEqual(result["error"]["code"], "WAIT_TIMEOUT")
        self.assertEqual(result["session_id"], session_id)


if __name__ == "__main__":
    unittest.main()

