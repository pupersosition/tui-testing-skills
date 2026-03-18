## ADDED Requirements

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
