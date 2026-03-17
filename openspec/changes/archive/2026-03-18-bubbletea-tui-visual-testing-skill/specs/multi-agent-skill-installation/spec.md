## ADDED Requirements

### Requirement: Agent-selectable skill installation command
The repository MUST provide an installer command that accepts a target agent identifier and installs the Bubble Tea skill from the canonical source directory.

#### Scenario: Install skill for selected agent
- **WHEN** a user runs the installer with `--agent codex`
- **THEN** the installer SHALL copy the canonical skill directory into the codex destination path and report success

#### Scenario: Reject unknown agent identifier
- **WHEN** a user runs the installer with an unsupported `--agent` value
- **THEN** the installer SHALL fail with a clear error listing supported agent values

### Requirement: Agent defaults with destination override
The installer MUST define default installation roots for `claude`, `copilot`, `codex`, and `opencode`, and MUST support explicit destination override.

#### Scenario: Use default destination path
- **WHEN** a user runs the installer with only `--agent claude`
- **THEN** the installer SHALL resolve and use the configured default path for claude

#### Scenario: Use explicit destination override
- **WHEN** a user provides `--dest <path>`
- **THEN** the installer SHALL install to `<path>` instead of the agent default

### Requirement: Safe overwrite behavior
The installer MUST avoid destructive replacement unless explicitly requested.

#### Scenario: Existing destination without force
- **WHEN** the destination skill directory already exists and `--force` is not set
- **THEN** the installer SHALL fail with guidance to rerun with `--force`

#### Scenario: Existing destination with force
- **WHEN** the destination skill directory already exists and `--force` is set
- **THEN** the installer SHALL replace the destination contents with the canonical source
