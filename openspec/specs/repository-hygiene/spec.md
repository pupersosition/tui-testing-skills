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

### Requirement: Go build artifact isolation
Repository hygiene rules MUST isolate transient Go build and test artifacts from tracked reusable skill sources.

#### Scenario: Local Go validation run
- **WHEN** a contributor runs Go build and test workflows
- **THEN** transient caches and generated outputs SHALL remain outside tracked canonical skill source files or be ignored by repository rules

### Requirement: Migration-era source-of-truth clarity
During migration, the repository MUST document and enforce canonical ownership between reusable source, generated artifacts, and compatibility wrappers.

#### Scenario: Contributor verifies migration layout
- **WHEN** a contributor reviews repository docs and file layout during migration
- **THEN** the canonical reusable skill source and temporary compatibility paths SHALL be explicitly identified to prevent accidental edits in generated/runtime directories
