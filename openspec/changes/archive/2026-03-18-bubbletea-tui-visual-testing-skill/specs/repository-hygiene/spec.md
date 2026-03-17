## ADDED Requirements

### Requirement: Canonical reusable skill source layout
The repository MUST maintain a canonical reusable source path for the skill that is independent of local agent runtime directories.

#### Scenario: Locate canonical skill source
- **WHEN** a contributor inspects the repository layout
- **THEN** the canonical skill source SHALL exist under `skills/bubbletea-tui-visual-test/`

### Requirement: Runtime artifact isolation
Generated runtime artifacts MUST be isolated from reusable source files to keep commits clean.

#### Scenario: Execute integration flow
- **WHEN** the integration example runner is executed
- **THEN** generated artifacts SHALL be written under `.context/` and SHALL NOT modify canonical source files

### Requirement: Ignore transient local files
Repository ignore rules MUST exclude common local cache/runtime outputs related to skill execution and testing.

#### Scenario: Check repository status after local runs
- **WHEN** a user runs local tests and fixture integrations
- **THEN** transient files (for example cache directories and runtime outputs) SHALL be ignored by default
