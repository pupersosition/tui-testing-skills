## Why

Agents can run terminal commands, but they lack a standardized way to verify Bubble Tea UI behavior and visual design quality. We need a reusable skill that produces deterministic terminal interactions, visual snapshots, and review artifacts so teams can trust TUI changes and parallelize implementation safely.

## What Changes

- Add a new skill for Bubble Tea TUI testing with a command workflow similar to browser-style automation.
- Add deterministic PTY session management and scripted key/input interaction for Bubble Tea apps.
- Add visual testing outputs: checkpoint PNG snapshots with baseline diffing, plus GIF artifacts for human review.
- Add assertion primitives for text/state waits and visual regression checks.
- Add task decomposition and artifact directory conventions so multiple Conductor agents can implement streams in parallel with minimal file conflicts.

## Capabilities

### New Capabilities
- `bubbletea-session-automation`: Start and control Bubble Tea app sessions, send input, wait for state, and capture structured session output.
- `terminal-visual-regression`: Produce deterministic terminal snapshots, compare against baselines with configurable thresholds, and emit GIF artifacts for review.
- `parallel-safe-skill-delivery`: Define implementation boundaries and output ownership rules that support concurrent agent work without overlapping edits.

### Modified Capabilities
- None.

## Impact

- Affected code: new skill directory under `.codex/skills/` with `SKILL.md`, scripts, and references.
- Dependencies: Python runtime (`pexpect`), optional terminal rendering toolchain for PNG/GIF generation.
- Tooling: CI and local workflows will gain snapshot baseline management and visual diff reporting.
- Systems: Conductor agent workflows can assign independent task streams (session control, visual pipeline, docs/references) with low merge conflict risk.
