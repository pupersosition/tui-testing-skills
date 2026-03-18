## ADDED Requirements

### Requirement: Go installer CLI compatibility
Skill installation MUST be provided by a Go CLI that preserves existing agent selection, destination override, and overwrite safety behavior.

#### Scenario: Install skill for supported agent
- **WHEN** a user runs the Go installer with `--agent codex`
- **THEN** the installer SHALL resolve the codex default destination and install the selected skill

#### Scenario: Destination override with Go installer
- **WHEN** a user provides `--dest <path>`
- **THEN** the installer SHALL install to `<path>` instead of the agent default path

#### Scenario: Safe overwrite behavior with Go installer
- **WHEN** destination exists and `--force` is not set
- **THEN** the installer SHALL fail with guidance to rerun using `--force`

### Requirement: Installer dry-run behavior parity
The Go installer MUST support a no-write preview mode.

#### Scenario: Dry-run installation
- **WHEN** a user runs the installer with `--dry-run`
- **THEN** the installer SHALL report planned source and destination actions without modifying filesystem contents
