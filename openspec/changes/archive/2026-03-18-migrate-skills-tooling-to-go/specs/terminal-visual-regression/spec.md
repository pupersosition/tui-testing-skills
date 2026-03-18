## ADDED Requirements

### Requirement: Go-backed visual checkpoint pipeline
Snapshot, visual assertion, and record commands MUST be executed by a Go visual pipeline while preserving existing deterministic metadata rules.

#### Scenario: Capture snapshot with metadata via Go pipeline
- **WHEN** an agent executes `snapshot` for an active run
- **THEN** the Go pipeline SHALL write PNG and metadata artifacts with required deterministic runtime fields

#### Scenario: Visual threshold assertion via Go pipeline
- **WHEN** an agent executes `assert-visual` with a configured threshold
- **THEN** the Go pipeline SHALL return pass/fail results and emit a diff artifact when the threshold is exceeded

### Requirement: Renderer unavailability handling parity
The Go visual pipeline MUST preserve clear failure behavior when rendering dependencies are unavailable.

#### Scenario: Record command without renderer dependency
- **WHEN** an agent executes `record` in an environment missing required renderer support
- **THEN** the pipeline SHALL return a structured error with actionable guidance and preserve previously generated artifacts
