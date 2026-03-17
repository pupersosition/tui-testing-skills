## ADDED Requirements

### Requirement: Parallel workstream file ownership
The change implementation MUST define disjoint file ownership boundaries so multiple Conductor agents can work in parallel without editing the same files.

#### Scenario: Assign non-overlapping ownership map
- **WHEN** tasks are generated for implementation
- **THEN** each task group SHALL declare owned file paths that do not overlap with other task groups

#### Scenario: Shared contract files are sequenced
- **WHEN** a shared interface artifact is required by multiple workstreams
- **THEN** the plan SHALL schedule one owner to publish the contract before dependent workstreams begin integration

### Requirement: Per-run artifact isolation
The skill MUST store runtime outputs under per-run directories to prevent collisions between concurrent agent executions.

#### Scenario: Concurrent runs write artifacts
- **WHEN** two agents execute the same visual workflow concurrently
- **THEN** each run SHALL write outputs to a unique run directory and SHALL NOT overwrite the other run's artifacts

### Requirement: Integration validation before merge
The implementation plan MUST include an integration verification step that confirms all parallel workstreams conform to the shared command contract.

#### Scenario: Contract compatibility check
- **WHEN** parallel branches are combined
- **THEN** the integration checks SHALL validate command schema compatibility and fail on mismatched request or response fields
