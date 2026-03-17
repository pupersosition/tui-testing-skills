from __future__ import annotations

import json
from pathlib import Path
import sys

SCRIPT_DIR = Path(__file__).resolve().parents[1]
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

import agent_tui  # noqa: E402


def test_dispatcher_command_set_matches_schema_enum() -> None:
    schema_path = SCRIPT_DIR.parent / "references" / "command-schema.json"
    schema = json.loads(schema_path.read_text(encoding="utf-8"))

    expected = set(schema["properties"]["command"]["enum"])
    actual = set(agent_tui.SUPPORTED_COMMANDS)

    assert actual == expected


def test_dispatcher_required_fields_cover_contract_minimum() -> None:
    assert agent_tui.REQUIRED_FIELDS["open"] == ("cmd", "cwd", "cols", "rows")
    assert agent_tui.REQUIRED_FIELDS["snapshot"] == ("session_id", "name")
    assert agent_tui.REQUIRED_FIELDS["assert-visual"] == (
        "session_id",
        "name",
        "baseline_path",
    )
