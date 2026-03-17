## Why

The current skill tooling depends on multiple Python entrypoints, which adds runtime setup overhead and complicates packaging across developer and CI environments. Migrating the tooling to Go enables reproducible single-binary workflows while preserving the existing behavior contract used by agents.

## What Changes

- Replace Python-based session automation, visual pipeline, and dispatcher scripts with Go implementations under the canonical skill source.
- Replace the Python installer (`tools/install_skill.py`) with a Go installer CLI that preserves current agent-target behavior and safety flags.
- Introduce a Go module layout, build/test commands, and CI checks for the skill tooling.
- Preserve the command schema and response envelopes so existing agent workflows remain compatible.
- Add a staged migration path with temporary compatibility wrappers and a defined deprecation/removal phase for Python entrypoints.

## Capabilities

### New Capabilities
- `go-skill-toolchain`: Define standard Go module structure, build/test entrypoints, and release artifacts for skill tooling.

### Modified Capabilities
- `bubbletea-session-automation`: Run `open|close|press|type|wait` through Go PTY/session primitives while keeping deterministic behavior and structured JSON responses.
- `terminal-visual-regression`: Run `snapshot|assert-visual|record` through Go-based visual tooling with equivalent metadata and threshold semantics.
- `multi-agent-skill-installation`: Provide installer behavior through a Go CLI while preserving `--agent`, `--dest`, `--force`, and `--dry-run` semantics.
- `repository-hygiene`: Update tracked/ignored files and repository conventions for Go build outputs, test artifacts, and migration-era compatibility shims.

## Impact

- Affected code: `skills/bubbletea-tui-visual-test/scripts/`, `tools/`, repository docs, CI/test commands.
- Dependencies: remove Python runtime dependency for core tooling; add Go toolchain and required Go libraries for PTY and image/diff handling.
- Compatibility: command schema remains stable; migration includes compatibility wrappers during rollout.
- Delivery model: implementation will be partitioned into parallel independent workstreams with explicit file ownership boundaries.
