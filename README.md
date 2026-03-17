# Bubble Tea TUI Visual Test Skill

Reusable skill package for terminal-first visual testing of Bubble Tea applications.

## Repository Layout

- `skills/bubbletea-tui-visual-test/`: canonical reusable skill source (install origin)
- `tools/install_skill.py`: installer that copies a selected skill into a target agent skill directory
- `openspec/changes/bubbletea-tui-visual-testing-skill/`: OpenSpec change artifacts

## Install

Use the installer and select an agent target:

```bash
python3 tools/install_skill.py --agent codex
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

Run integration smoke test:

```bash
bash skills/bubbletea-tui-visual-test/references/examples.sh
```
