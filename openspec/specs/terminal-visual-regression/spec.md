## ADDED Requirements

### Requirement: Deterministic terminal snapshot capture
The skill MUST capture visual checkpoint snapshots as PNG files with associated metadata describing terminal dimensions, theme, color mode, locale, and renderer version.

#### Scenario: Capture named snapshot checkpoint
- **WHEN** an agent executes a snapshot command with a checkpoint name
- **THEN** the system SHALL write a PNG file and metadata record for that checkpoint

#### Scenario: Reject snapshot with incomplete runtime metadata
- **WHEN** snapshot capture is requested without required deterministic metadata
- **THEN** the system SHALL fail the command with actionable validation errors

### Requirement: Visual baseline comparison
The skill MUST compare checkpoint PNG outputs to stored baselines and report pass/fail using a configurable diff threshold.

#### Scenario: Snapshot comparison passes threshold
- **WHEN** computed visual difference is less than or equal to the configured threshold
- **THEN** the system SHALL return a passing visual assertion result

#### Scenario: Snapshot comparison exceeds threshold
- **WHEN** computed visual difference is greater than the configured threshold
- **THEN** the system SHALL return a failing result and write a diff artifact highlighting mismatches

### Requirement: GIF review artifact generation
The skill MUST support generating a GIF artifact from an executed Bubble Tea interaction flow for human design review.

#### Scenario: Generate GIF artifact for completed run
- **WHEN** an agent executes the record command for a completed interaction flow
- **THEN** the system SHALL write a GIF artifact and return its output path in structured command output

#### Scenario: Renderer unavailable
- **WHEN** GIF generation is requested in an environment without the required renderer
- **THEN** the system SHALL return a clear error with installation guidance and preserve existing text/snapshot artifacts
