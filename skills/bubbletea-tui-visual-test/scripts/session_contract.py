"""Structured response helpers for the Bubble Tea session command contract."""

from __future__ import annotations

from typing import Any, Dict, Optional

Result = Dict[str, Any]


def ok(session_id: str, data: Optional[Dict[str, Any]] = None) -> Result:
    """Return a successful command response."""
    response: Result = {"ok": True, "session_id": session_id}
    if data is not None:
        response["data"] = data
    return response


def error(
    session_id: str,
    code: str,
    message: str,
    data: Optional[Dict[str, Any]] = None,
) -> Result:
    """Return a failed command response."""
    response: Result = {
        "ok": False,
        "session_id": session_id,
        "error": {"code": code, "message": message},
    }
    if data is not None:
        response["data"] = data
    return response

