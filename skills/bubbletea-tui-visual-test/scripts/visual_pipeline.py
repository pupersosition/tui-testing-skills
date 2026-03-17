"""Visual regression primitives for Bubble Tea TUI skill commands.

This module owns workstream B commands:
- snapshot
- assert-visual
- record
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import struct
from typing import Any, Mapping, Sequence
import uuid
import zlib


PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
REQUIRED_RUNTIME_METADATA = (
    "cols",
    "rows",
    "theme",
    "color_mode",
    "locale",
    "renderer_version",
)
NAME_SANITIZER = re.compile(r"[^A-Za-z0-9._-]+")
ANSI_ESCAPE = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]")


@dataclass
class VisualPipelineError(Exception):
    code: str
    message: str

    def __str__(self) -> str:  # pragma: no cover - Exception formatting utility
        return f"{self.code}: {self.message}"


def _timestamp_utc() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def _make_run_id() -> str:
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    return f"run-{stamp}-{uuid.uuid4().hex[:8]}"


def _sanitize_checkpoint_name(name: str) -> str:
    cleaned = NAME_SANITIZER.sub("-", name).strip("-")
    if not cleaned:
        raise VisualPipelineError("invalid_name", "Checkpoint name must contain letters, numbers, dots, dashes, or underscores.")
    return cleaned


def _ok(session_id: str, data: Mapping[str, Any]) -> dict[str, Any]:
    return {"ok": True, "session_id": session_id, "data": dict(data)}


def _error(session_id: str, code: str, message: str, data: Mapping[str, Any] | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {"ok": False, "session_id": session_id, "error": {"code": code, "message": message}}
    if data:
        payload["data"] = dict(data)
    return payload


def _validate_runtime_metadata(runtime_metadata: Mapping[str, Any]) -> dict[str, Any]:
    missing = [field for field in REQUIRED_RUNTIME_METADATA if not runtime_metadata.get(field)]
    if missing:
        fields = ", ".join(missing)
        raise VisualPipelineError(
            "invalid_runtime_metadata",
            f"Missing required runtime metadata fields: {fields}.",
        )

    cols = runtime_metadata["cols"]
    rows = runtime_metadata["rows"]
    if not isinstance(cols, int) or cols <= 0:
        raise VisualPipelineError("invalid_runtime_metadata", "Runtime metadata field 'cols' must be a positive integer.")
    if not isinstance(rows, int) or rows <= 0:
        raise VisualPipelineError("invalid_runtime_metadata", "Runtime metadata field 'rows' must be a positive integer.")

    return {field: runtime_metadata[field] for field in REQUIRED_RUNTIME_METADATA}


def _strip_ansi_control_sequences(text: str) -> str:
    without_ansi = ANSI_ESCAPE.sub("", text)
    return without_ansi.replace("\r", "\n")


def _normalized_screen_lines(screen_text: str, cols: int, rows: int) -> list[str]:
    if "\x1b" in screen_text:
        try:
            import pyte  # type: ignore

            screen = pyte.Screen(cols, rows)
            stream = pyte.Stream(screen)
            stream.feed(screen_text)
            rendered = [line[:cols].ljust(cols) for line in screen.display]
            if any(line.strip() for line in rendered):
                return rendered
        except Exception:
            pass

    cleaned = _strip_ansi_control_sequences(screen_text)
    logical_lines = cleaned.splitlines()

    # Keep the latest visible region because TUIs repaint frames continuously.
    window = logical_lines[-rows:] if logical_lines else []
    padded: list[str] = []
    for row in range(rows):
        line = window[row] if row < len(window) else ""
        normalized = line.expandtabs(4)[:cols].ljust(cols)
        padded.append(normalized)
    return padded


def _load_monospace_font(cell_height: int) -> Any:
    from PIL import ImageFont  # type: ignore

    candidates = (
        "/System/Library/Fonts/Menlo.ttc",
        "/Library/Fonts/Menlo-Regular.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
        "/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
    )
    size = max(12, int(cell_height * 0.8))
    for candidate in candidates:
        try:
            return ImageFont.truetype(candidate, size=size)
        except Exception:
            continue
    return ImageFont.load_default()


def _render_text_with_pillow(lines: list[str], cols: int, rows: int) -> tuple[int, int, bytes]:
    from PIL import Image, ImageDraw  # type: ignore

    cell_width = 9
    cell_height = 18
    width = max(1, cols * cell_width)
    height = max(1, rows * cell_height)

    image = Image.new("RGB", (width, height), color=(24, 24, 24))
    draw = ImageDraw.Draw(image)
    font = _load_monospace_font(cell_height)
    for idx, line in enumerate(lines):
        draw.text((4, idx * cell_height + 1), line, fill=(236, 236, 236), font=font)
    return width, height, image.tobytes()


def _terminal_buffer_to_rgb(screen_text: str, cols: int, rows: int) -> tuple[int, int, bytes]:
    lines = _normalized_screen_lines(screen_text, cols, rows)
    try:
        return _render_text_with_pillow(lines, cols, rows)
    except Exception:
        # Fallback for environments without Pillow.
        pixels = bytearray()
        for line in lines:
            for char in line:
                if 32 <= ord(char) <= 126 and char != " ":
                    pixels.extend((220, 220, 220))
                else:
                    pixels.extend((30, 30, 30))
        return cols, rows, bytes(pixels)


def _png_chunk(kind: bytes, payload: bytes) -> bytes:
    packed = struct.pack(">I", len(payload)) + kind + payload
    crc = zlib.crc32(kind + payload) & 0xFFFFFFFF
    return packed + struct.pack(">I", crc)


def _write_png(path: Path, width: int, height: int, rgb: bytes) -> None:
    if len(rgb) != width * height * 3:
        raise VisualPipelineError("render_error", "RGB payload length does not match image dimensions.")

    row_stride = width * 3
    raw = bytearray()
    for row in range(height):
        raw.append(0)  # no PNG filter
        start = row * row_stride
        raw.extend(rgb[start : start + row_stride])

    ihdr = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)  # 8-bit RGB
    idat = zlib.compress(bytes(raw), level=9)

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(
        PNG_SIGNATURE
        + _png_chunk(b"IHDR", ihdr)
        + _png_chunk(b"IDAT", idat)
        + _png_chunk(b"IEND", b"")
    )


def _read_png(path: Path) -> tuple[int, int, bytes]:
    blob = path.read_bytes()
    if not blob.startswith(PNG_SIGNATURE):
        raise VisualPipelineError("invalid_png", f"{path} is not a valid PNG file.")

    offset = len(PNG_SIGNATURE)
    width = height = None
    idat_parts: list[bytes] = []

    while offset + 12 <= len(blob):
        length = struct.unpack(">I", blob[offset : offset + 4])[0]
        kind = blob[offset + 4 : offset + 8]
        payload = blob[offset + 8 : offset + 8 + length]
        offset += 12 + length

        if kind == b"IHDR":
            width, height, bit_depth, color_type, compression, filtering, interlace = struct.unpack(">IIBBBBB", payload)
            if bit_depth != 8 or color_type != 2 or compression != 0 or filtering != 0 or interlace != 0:
                raise VisualPipelineError(
                    "invalid_png",
                    f"{path} uses unsupported PNG settings (expected 8-bit RGB, no interlace).",
                )
        elif kind == b"IDAT":
            idat_parts.append(payload)
        elif kind == b"IEND":
            break

    if width is None or height is None:
        raise VisualPipelineError("invalid_png", f"{path} missing IHDR chunk.")

    raw = zlib.decompress(b"".join(idat_parts))
    row_stride = width * 3
    expected = height * (row_stride + 1)
    if len(raw) != expected:
        raise VisualPipelineError("invalid_png", f"{path} payload size does not match image dimensions.")

    rgb = bytearray()
    cursor = 0
    for _ in range(height):
        filter_type = raw[cursor]
        cursor += 1
        if filter_type != 0:
            raise VisualPipelineError("invalid_png", f"{path} uses unsupported PNG filter {filter_type}.")
        rgb.extend(raw[cursor : cursor + row_stride])
        cursor += row_stride

    return width, height, bytes(rgb)


def _pixel_diff(actual_rgb: bytes, baseline_rgb: bytes) -> tuple[float, bytes]:
    total_pixels = len(actual_rgb) // 3
    mismatched = 0
    diff = bytearray()

    for idx in range(total_pixels):
        off = idx * 3
        actual = actual_rgb[off : off + 3]
        baseline = baseline_rgb[off : off + 3]
        if actual != baseline:
            mismatched += 1
            diff.extend((255, 32, 32))
            continue

        neutral = (actual[0] + actual[1] + actual[2]) // 3
        diff.extend((neutral, neutral, neutral))

    ratio = mismatched / total_pixels if total_pixels else 0.0
    return ratio, bytes(diff)


def _load_pillow_image() -> Any:
    try:
        from PIL import Image  # type: ignore
    except Exception as exc:  # pragma: no cover - covered through explicit monkeypatching in tests
        raise VisualPipelineError(
            "renderer_unavailable",
            "GIF renderer is unavailable. Install Pillow with: pip install pillow",
        ) from exc
    return Image


class VisualPipeline:
    """Deterministic visual artifact pipeline for Bubble Tea sessions."""

    def __init__(self, run_dir: str | Path | None = None, root_output_dir: str | Path | None = None) -> None:
        if run_dir is not None:
            self.run_dir = Path(run_dir)
        else:
            base = Path(root_output_dir) if root_output_dir else Path(".context/bubbletea-tui-visual-test/runs")
            self.run_dir = base / _make_run_id()

        self.run_dir.mkdir(parents=True, exist_ok=True)
        self.checkpoints_dir = self.run_dir / "checkpoints"
        self.metadata_dir = self.run_dir / "metadata"
        self.diffs_dir = self.run_dir / "diffs"
        self.recordings_dir = self.run_dir / "recordings"
        for directory in (self.checkpoints_dir, self.metadata_dir, self.diffs_dir, self.recordings_dir):
            directory.mkdir(parents=True, exist_ok=True)

        self._checkpoint_index: dict[str, Path] = {}

    def snapshot(
        self,
        *,
        session_id: str,
        name: str,
        screen_text: str,
        runtime_metadata: Mapping[str, Any],
    ) -> dict[str, Any]:
        try:
            clean_name = _sanitize_checkpoint_name(name)
            runtime = _validate_runtime_metadata(runtime_metadata)
            width, height, rgb = _terminal_buffer_to_rgb(screen_text, runtime["cols"], runtime["rows"])

            png_path = self.checkpoints_dir / f"{clean_name}.png"
            metadata_path = self.metadata_dir / f"{clean_name}.metadata.json"

            _write_png(png_path, width, height, rgb)

            record = {
                "session_id": session_id,
                "checkpoint": clean_name,
                "created_at": _timestamp_utc(),
                "snapshot_path": str(png_path.resolve()),
                "runtime": runtime,
            }
            metadata_path.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
            self._checkpoint_index[clean_name] = png_path

            return _ok(
                session_id,
                {
                    "run_dir": str(self.run_dir.resolve()),
                    "snapshot_path": str(png_path.resolve()),
                    "metadata_path": str(metadata_path.resolve()),
                },
            )
        except VisualPipelineError as exc:
            return _error(session_id, exc.code, exc.message)

    def assert_visual(
        self,
        *,
        session_id: str,
        name: str,
        baseline_path: str | Path,
        threshold: float = 0.0,
    ) -> dict[str, Any]:
        try:
            if threshold < 0 or threshold > 1:
                raise VisualPipelineError("invalid_threshold", "Threshold must be in the [0, 1] range.")

            clean_name = _sanitize_checkpoint_name(name)
            actual_path = self._checkpoint_index.get(clean_name) or (self.checkpoints_dir / f"{clean_name}.png")
            baseline = Path(baseline_path)

            if not actual_path.exists():
                raise VisualPipelineError("missing_snapshot", f"Snapshot for checkpoint '{clean_name}' was not found.")
            if not baseline.exists():
                raise VisualPipelineError("missing_baseline", f"Baseline PNG does not exist: {baseline}")

            actual_width, actual_height, actual_rgb = _read_png(actual_path)
            baseline_width, baseline_height, baseline_rgb = _read_png(baseline)

            if (actual_width, actual_height) != (baseline_width, baseline_height):
                difference_ratio = 1.0
                passed = False
                diff_path = self.diffs_dir / f"{clean_name}.size-mismatch.json"
                diff_payload = {
                    "checkpoint": clean_name,
                    "actual_size": [actual_width, actual_height],
                    "baseline_size": [baseline_width, baseline_height],
                }
                diff_path.write_text(json.dumps(diff_payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
            else:
                difference_ratio, diff_rgb = _pixel_diff(actual_rgb, baseline_rgb)
                passed = difference_ratio <= threshold
                diff_path = self.diffs_dir / f"{clean_name}.diff.png"
                if not passed:
                    _write_png(diff_path, actual_width, actual_height, diff_rgb)

            data = {
                "checkpoint": clean_name,
                "actual_path": str(actual_path.resolve()),
                "baseline_path": str(baseline.resolve()),
                "difference_ratio": difference_ratio,
                "threshold": threshold,
                "passed": passed,
                "diff_artifact": str(diff_path.resolve()) if not passed else None,
            }
            return _ok(session_id, data)
        except VisualPipelineError as exc:
            return _error(session_id, exc.code, exc.message)

    def record(
        self,
        *,
        session_id: str,
        output_path: str | Path,
        frame_paths: Sequence[str | Path] | None = None,
        frame_duration_ms: int = 250,
    ) -> dict[str, Any]:
        try:
            if frame_duration_ms <= 0:
                raise VisualPipelineError("invalid_frame_duration", "frame_duration_ms must be a positive integer.")

            if frame_paths:
                frames = [Path(frame) for frame in frame_paths]
            else:
                frames = list(self._checkpoint_index.values())

            if not frames:
                raise VisualPipelineError("no_frames", "No frames available. Capture snapshots before calling record.")
            for frame in frames:
                if not frame.exists():
                    raise VisualPipelineError("missing_frame", f"Frame PNG does not exist: {frame}")

            image_lib = _load_pillow_image()
            images = [image_lib.open(frame) for frame in frames]
            try:
                destination = Path(output_path)
                destination.parent.mkdir(parents=True, exist_ok=True)
                images[0].save(
                    destination,
                    save_all=True,
                    append_images=images[1:],
                    duration=frame_duration_ms,
                    loop=0,
                )
            finally:
                for image in images:
                    image.close()

            return _ok(
                session_id,
                {
                    "output_path": str(Path(output_path).resolve()),
                    "frame_count": len(frames),
                    "frame_duration_ms": frame_duration_ms,
                },
            )
        except VisualPipelineError as exc:
            return _error(session_id, exc.code, exc.message)
