# Bubble Tea TUI Visual Test Skill

Reusable skill package for terminal-first visual testing of Bubble Tea applications.

## Repository Layout

- `skills/bubbletea-tui-visual-test/`: canonical reusable skill source (install origin)
- `cmd/install-skill/`: canonical Go installer entrypoint (bootstrap in migration phase)
- `cmd/agent-tui/`: canonical Go dispatcher entrypoint (bootstrap in migration phase)
- `internal/contract/`: shared Go request/response contract models mapped to `references/command-schema.json`
- `tools/install_skill.py`: migration-era Python installer kept for compatibility
- `openspec/changes/bubbletea-tui-visual-testing-skill/`: OpenSpec change artifacts

## Install

Canonical Go installer entrypoint:

```bash
go run ./cmd/install-skill --agent codex
```

Migration-era compatibility installer:

```bash
python3 tools/install_skill.py --agent codex
```

Canonical Go dispatcher entrypoint (bootstrap):

```bash
go run ./cmd/agent-tui
```

Supported agents:

- `claude`
- `copilot`
- `codex`
- `opencode`

Options:

- `--dest <path>`: explicit destination path override
- `--force`: replace destination when it already exists
- `--dry-run`: preview copy operation
- `--skill <name>`: install a different skill folder from `skills/`

## Default Agent Destinations

- `claude` -> `~/.claude/skills/<skill>`
- `copilot` -> `~/.config/copilot/skills/<skill>`
- `codex` -> `~/.codex/skills/<skill>`
- `opencode` -> `~/.config/opencode/skills/<skill>`

## Validate

Run script tests:

```bash
python3 -m pytest skills/bubbletea-tui-visual-test/scripts/tests
```

Run Go contract tests and bootstrap builds:

```bash
go test ./...
go build ./cmd/agent-tui ./cmd/install-skill
```

Run integration smoke test:

```bash
bash skills/bubbletea-tui-visual-test/references/examples.sh
```
