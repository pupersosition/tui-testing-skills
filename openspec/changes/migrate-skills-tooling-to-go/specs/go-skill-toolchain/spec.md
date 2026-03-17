## ADDED Requirements

### Requirement: Canonical Go module and entrypoints
The repository MUST provide a canonical Go module layout for skill tooling that includes installer and agent dispatcher entrypoints.

#### Scenario: Contributor locates Go tooling structure
- **WHEN** a contributor inspects the repository
- **THEN** the project SHALL expose documented Go entrypoints for installer and agent command dispatch

### Requirement: Reproducible Go build and test workflow
The skill tooling MUST define deterministic Go build and test commands for local and CI execution.

#### Scenario: CI validates Go tooling
- **WHEN** CI executes the documented Go validation workflow
- **THEN** it SHALL run the defined Go test/build commands and fail on compile or test errors

### Requirement: Migration compatibility window
The migration MUST provide a bounded compatibility window so existing Python entrypoints continue to function until cutover criteria are met.

#### Scenario: Legacy Python entrypoint invoked during migration
- **WHEN** an existing workflow calls the prior Python command path during the compatibility window
- **THEN** the command SHALL execute via the Go-backed flow or return actionable migration guidance without silent failure
