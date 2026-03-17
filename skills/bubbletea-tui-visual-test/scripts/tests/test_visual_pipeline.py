from __future__ import annotations

import json
from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import visual_pipeline  # noqa: E402


def _runtime_metadata() -> dict[str, object]:
    return {
        "cols": 24,
        "rows": 5,
        "theme": "light",
        "color_mode": "256",
        "locale": "en_US.UTF-8",
        "renderer_version": "builtin-terminal-rasterizer/1.0",
    }


def test_snapshot_writes_png_and_metadata(tmp_path: Path) -> None:
    pipeline = visual_pipeline.VisualPipeline(run_dir=tmp_path / "run-a")
    result = pipeline.snapshot(
        session_id="session-1",
        name="counter-home",
        screen_text="Counter: 1\nPress q to quit",
        runtime_metadata=_runtime_metadata(),
    )

    assert result["ok"] is True
    snapshot_path = Path(result["data"]["snapshot_path"])
    metadata_path = Path(result["data"]["metadata_path"])

    assert snapshot_path.exists()
    assert metadata_path.exists()

    metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
    assert metadata["session_id"] == "session-1"
    assert metadata["checkpoint"] == "counter-home"
    assert metadata["runtime"]["cols"] == 24
    assert metadata["runtime"]["rows"] == 5
    assert metadata["runtime"]["renderer_version"] == "builtin-terminal-rasterizer/1.0"


def test_assert_visual_pass_and_fail_diff_behavior(tmp_path: Path) -> None:
    baseline_pipeline = visual_pipeline.VisualPipeline(run_dir=tmp_path / "baseline-run")
    baseline = baseline_pipeline.snapshot(
        session_id="baseline-session",
        name="screen",
        screen_text="Counter: 1\nPress q to quit",
        runtime_metadata=_runtime_metadata(),
    )
    baseline_path = Path(baseline["data"]["snapshot_path"])

    test_pipeline = visual_pipeline.VisualPipeline(run_dir=tmp_path / "test-run")
    test_pipeline.snapshot(
        session_id="test-session",
        name="screen",
        screen_text="Counter: 1\nPress q to quit",
        runtime_metadata=_runtime_metadata(),
    )

    passing = test_pipeline.assert_visual(
        session_id="test-session",
        name="screen",
        baseline_path=baseline_path,
        threshold=0.0,
    )
    assert passing["ok"] is True
    assert passing["data"]["passed"] is True
    assert passing["data"]["difference_ratio"] == 0.0
    assert passing["data"]["diff_artifact"] is None

    test_pipeline.snapshot(
        session_id="test-session",
        name="screen",
        screen_text="Counter: 9\nPress q to quit",
        runtime_metadata=_runtime_metadata(),
    )
    failing = test_pipeline.assert_visual(
        session_id="test-session",
        name="screen",
        baseline_path=baseline_path,
        threshold=0.0,
    )
    assert failing["ok"] is True
    assert failing["data"]["passed"] is False
    assert failing["data"]["difference_ratio"] > 0.0
    assert failing["data"]["diff_artifact"] is not None
    assert Path(failing["data"]["diff_artifact"]).exists()


def test_record_reports_renderer_unavailable(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    pipeline = visual_pipeline.VisualPipeline(run_dir=tmp_path / "run-record")
    snapshot = pipeline.snapshot(
        session_id="session-record",
        name="frame-1",
        screen_text="Frame 1",
        runtime_metadata=_runtime_metadata(),
    )
    assert snapshot["ok"] is True
    snapshot_path = Path(snapshot["data"]["snapshot_path"])
    assert snapshot_path.exists()

    def _raise_renderer_error() -> object:
        raise visual_pipeline.VisualPipelineError(
            "renderer_unavailable",
            "GIF renderer is unavailable. Install Pillow with: pip install pillow",
        )

    monkeypatch.setattr(visual_pipeline, "_load_pillow_image", _raise_renderer_error)

    result = pipeline.record(
        session_id="session-record",
        output_path=tmp_path / "review.gif",
    )
    assert result["ok"] is False
    assert result["error"]["code"] == "renderer_unavailable"
    assert "pip install pillow" in result["error"]["message"]
    assert snapshot_path.exists()
