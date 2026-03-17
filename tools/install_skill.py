#!/usr/bin/env python3
"""Install a reusable skill into an agent-specific skill directory."""

from __future__ import annotations

import argparse
import shutil
import sys
from pathlib import Path

DEFAULT_AGENT_ROOTS = {
    "claude": Path("~/.claude/skills").expanduser(),
    "copilot": Path("~/.config/copilot/skills").expanduser(),
    "codex": Path("~/.codex/skills").expanduser(),
    "opencode": Path("~/.config/opencode/skills").expanduser(),
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Install a skill for a selected coding agent")
    parser.add_argument(
        "--agent",
        required=True,
        choices=sorted(DEFAULT_AGENT_ROOTS.keys()),
        help="Target agent",
    )
    parser.add_argument(
        "--skill",
        default="bubbletea-tui-visual-test",
        help="Skill folder name under the source root",
    )
    parser.add_argument(
        "--source-root",
        default="skills",
        help="Root directory containing reusable skills",
    )
    parser.add_argument(
        "--dest",
        help=(
            "Explicit destination directory for the skill. "
            "If omitted, uses the agent default root and appends <skill>."
        ),
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Replace destination if it already exists",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show planned actions without writing files",
    )
    return parser.parse_args()


def resolve_paths(args: argparse.Namespace) -> tuple[Path, Path]:
    repo_root = Path(__file__).resolve().parents[1]
    source_dir = (repo_root / args.source_root / args.skill).resolve()
    if args.dest:
        destination_dir = Path(args.dest).expanduser().resolve()
    else:
        destination_dir = (DEFAULT_AGENT_ROOTS[args.agent] / args.skill).resolve()
    return source_dir, destination_dir


def install_skill(source_dir: Path, destination_dir: Path, *, force: bool, dry_run: bool) -> None:
    if not source_dir.exists() or not source_dir.is_dir():
        raise FileNotFoundError(f"Skill source not found: {source_dir}")

    if destination_dir.exists():
        if not force:
            raise FileExistsError(
                f"Destination already exists: {destination_dir}. "
                "Use --force to replace it."
            )
        if dry_run:
            print(f"[dry-run] Would remove existing destination: {destination_dir}")
        else:
            shutil.rmtree(destination_dir)

    if dry_run:
        print(f"[dry-run] Would copy: {source_dir} -> {destination_dir}")
        return

    destination_dir.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(source_dir, destination_dir)
    print(f"Installed skill to: {destination_dir}")


def main() -> int:
    args = parse_args()
    source_dir, destination_dir = resolve_paths(args)

    print(f"Agent: {args.agent}")
    print(f"Skill: {args.skill}")
    print(f"Source: {source_dir}")
    print(f"Destination: {destination_dir}")

    try:
        install_skill(source_dir, destination_dir, force=args.force, dry_run=args.dry_run)
    except (FileNotFoundError, FileExistsError) as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:  # pragma: no cover
        print(f"Unexpected error: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
