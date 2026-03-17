"""PTY session engine for Bubble Tea automation commands."""

from __future__ import annotations

import os
import re
import time
import uuid
from dataclasses import dataclass
from typing import Any, Callable, Dict, Optional, Pattern, Tuple, Type

from session_contract import error, ok


class SessionTimeoutError(Exception):
    """Raised when a wait operation exceeds timeout."""


class SessionEOFError(Exception):
    """Raised when the underlying process exits while waiting."""


@dataclass
class Session:
    session_id: str
    child: Any
    cmd: str
    cwd: str
    cols: int
    rows: int
    env: Dict[str, str]


def _default_spawn(*, cmd: str, cwd: str, env: Dict[str, str], cols: int, rows: int) -> Any:
    try:
        import pexpect
    except ImportError as exc:  # pragma: no cover - depends on host env.
        raise RuntimeError("pexpect is required for PTY automation") from exc

    child = pexpect.spawn(
        "/bin/sh",
        ["-lc", cmd],
        cwd=cwd,
        env=env,
        encoding="utf-8",
        echo=False,
        dimensions=(rows, cols),
    )
    child.delaybeforesend = 0
    return child


def _default_pexpect_errors() -> Tuple[Type[Exception], Type[Exception]]:
    try:
        import pexpect
    except ImportError:
        return SessionTimeoutError, SessionEOFError
    return pexpect.TIMEOUT, pexpect.EOF


class SessionEngine:
    """Implements open/close/press/type/wait against PTY-backed sessions."""

    _KEY_MAP = {
        "enter": "\r",
        "tab": "\t",
        "esc": "\x1b",
        "backspace": "\x7f",
        "up": "\x1b[A",
        "down": "\x1b[B",
        "right": "\x1b[C",
        "left": "\x1b[D",
    }

    def __init__(
        self,
        *,
        spawner: Optional[Callable[..., Any]] = None,
        timeout_error: Optional[Type[Exception]] = None,
        eof_error: Optional[Type[Exception]] = None,
        uuid_factory: Optional[Callable[[], str]] = None,
        clock: Optional[Callable[[], float]] = None,
    ) -> None:
        default_timeout_error, default_eof_error = _default_pexpect_errors()
        self._spawner = spawner or _default_spawn
        self._timeout_error = timeout_error or default_timeout_error
        self._eof_error = eof_error or default_eof_error
        self._uuid_factory = uuid_factory or (lambda: uuid.uuid4().hex)
        self._clock = clock or time.monotonic
        self._sessions: Dict[str, Session] = {}

    def execute(self, command: str, params: Dict[str, Any]) -> Dict[str, Any]:
        handlers = {
            "open": self.open,
            "close": self.close,
            "press": self.press,
            "type": self.type,
            "wait": self.wait,
        }
        handler = handlers.get(command)
        if handler is None:
            return error("", "UNKNOWN_COMMAND", f"Unsupported command: {command}")
        return handler(params)

    def has_session(self, session_id: str) -> bool:
        return session_id in self._sessions

    def open(self, params: Dict[str, Any]) -> Dict[str, Any]:
        cmd = params.get("cmd")
        cwd = params.get("cwd")
        cols = int(params.get("cols", 0))
        rows = int(params.get("rows", 0))
        if not cmd or not cwd or cols <= 0 or rows <= 0:
            return error("", "INVALID_PARAMS", "open requires cmd, cwd, cols, and rows")

        env = self._normalized_env(params)
        session_id = params.get("session_id") or f"session-{self._uuid_factory()}"
        try:
            child = self._spawner(cmd=cmd, cwd=cwd, env=env, cols=cols, rows=rows)
        except Exception as exc:
            return error("", "OPEN_FAILED", str(exc))

        self._sessions[session_id] = Session(
            session_id=session_id,
            child=child,
            cmd=cmd,
            cwd=cwd,
            cols=cols,
            rows=rows,
            env=env,
        )
        return ok(
            session_id,
            {
                "pid": getattr(child, "pid", None),
                "cmd": cmd,
                "cwd": cwd,
                "cols": cols,
                "rows": rows,
            },
        )

    def close(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session = self._session_or_error(params)
        if isinstance(session, dict):
            return session

        child = session.child
        terminated = False
        close_error: Optional[Exception] = None
        try:
            if hasattr(child, "isalive") and child.isalive():
                try:
                    terminated = bool(child.terminate(force=True))
                except TypeError:
                    terminated = bool(child.terminate(True))
            if hasattr(child, "close"):
                try:
                    child.close(force=True)
                except TypeError:
                    child.close()
        except Exception as exc:
            close_error = exc
        finally:
            self._sessions.pop(session.session_id, None)

        if close_error is not None:
            return error(session.session_id, "CLOSE_FAILED", str(close_error))

        return ok(
            session.session_id,
            {
                "closed": True,
                "terminated": terminated,
                "exitstatus": getattr(child, "exitstatus", None),
                "signalstatus": getattr(child, "signalstatus", None),
            },
        )

    def press(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session = self._session_or_error(params)
        if isinstance(session, dict):
            return session

        key = params.get("key")
        if not isinstance(key, str) or not key:
            return error(session.session_id, "INVALID_PARAMS", "press requires non-empty key")

        try:
            self._send_key(session.child, key)
        except Exception as exc:
            return error(session.session_id, "INTERACTION_FAILED", str(exc))
        return ok(session.session_id, {"action": "press", "key": key})

    def type(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session = self._session_or_error(params)
        if isinstance(session, dict):
            return session

        text = params.get("text")
        if not isinstance(text, str):
            return error(session.session_id, "INVALID_PARAMS", "type requires text")

        try:
            session.child.send(text)
        except Exception as exc:
            return error(session.session_id, "INTERACTION_FAILED", str(exc))
        return ok(session.session_id, {"action": "type", "bytes": len(text)})

    def wait(self, params: Dict[str, Any]) -> Dict[str, Any]:
        session = self._session_or_error(params)
        if isinstance(session, dict):
            return session

        timeout_ms = int(params.get("timeout_ms", 0))
        if timeout_ms <= 0:
            return error(session.session_id, "INVALID_PARAMS", "wait requires timeout_ms > 0")

        match_text = params.get("match_text")
        match_regex = params.get("match_regex")
        if not match_text and not match_regex:
            return error(
                session.session_id,
                "INVALID_PARAMS",
                "wait requires match_text or match_regex",
            )

        timeout_seconds = timeout_ms / 1000.0
        start = self._clock()
        mode = "text" if match_text else "regex"
        try:
            if match_text:
                session.child.expect_exact(match_text, timeout=timeout_seconds)
                matched = match_text
            else:
                pattern = re.compile(str(match_regex))
                session.child.expect(pattern, timeout=timeout_seconds)
                matched = self._extract_regex_match(session.child, pattern)
        except re.error as exc:
            return error(session.session_id, "INVALID_REGEX", str(exc))
        except self._timeout_error:
            return error(
                session.session_id,
                "WAIT_TIMEOUT",
                f"wait timed out after {timeout_ms}ms",
                {"timeout_ms": timeout_ms, "mode": mode},
            )
        except self._eof_error:
            return error(
                session.session_id,
                "SESSION_ENDED",
                "session ended before wait condition matched",
            )
        except Exception as exc:
            return error(session.session_id, "WAIT_FAILED", str(exc))

        elapsed_ms = int((self._clock() - start) * 1000)
        return ok(
            session.session_id,
            {"mode": mode, "matched": matched, "elapsed_ms": elapsed_ms},
        )

    def _normalized_env(self, params: Dict[str, Any]) -> Dict[str, str]:
        env = os.environ.copy()
        user_env = params.get("env", {})
        if isinstance(user_env, dict):
            env.update({str(k): str(v) for k, v in user_env.items()})

        locale = params.get("locale")
        if locale:
            env["LANG"] = str(locale)
            env["LC_ALL"] = str(locale)

        color_mode = params.get("color_mode")
        if color_mode == "truecolor":
            env["COLORTERM"] = "truecolor"
        elif color_mode in {"16", "256"}:
            env["COLORTERM"] = color_mode

        theme = params.get("theme")
        if theme:
            env["BUBBLETEA_THEME"] = str(theme)

        env.setdefault("TERM", "xterm-256color")
        return env

    def _session_or_error(self, params: Dict[str, Any]) -> Session | Dict[str, Any]:
        session_id = params.get("session_id")
        if not isinstance(session_id, str) or not session_id:
            return error("", "INVALID_PARAMS", "session_id is required")

        session = self._sessions.get(session_id)
        if session is None:
            return error(session_id, "SESSION_NOT_FOUND", f"Unknown session: {session_id}")
        return session

    def _send_key(self, child: Any, key: str) -> None:
        normalized = key.strip().lower()
        mapped = self._KEY_MAP.get(normalized)
        if mapped is not None:
            child.send(mapped)
            return

        if normalized.startswith("ctrl+") and len(normalized) == 6:
            child.sendcontrol(normalized[-1])
            return

        child.send(key)

    @staticmethod
    def _extract_regex_match(child: Any, pattern: Pattern[str]) -> str:
        match = getattr(child, "match", None)
        if match is not None and hasattr(match, "group"):
            return str(match.group(0))

        after = getattr(child, "after", "")
        if isinstance(after, str):
            fallback = pattern.search(after)
            if fallback:
                return str(fallback.group(0))
        return ""

