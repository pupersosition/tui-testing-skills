## ADDED Requirements

### Requirement: Go-backed session command execution
Session lifecycle and interaction commands MUST be executed by a Go PTY runtime while preserving the established command contract.

#### Scenario: Open session through Go runtime
- **WHEN** an agent executes `open` with explicit runtime parameters
- **THEN** the Go runtime SHALL create a session and return structured JSON with `ok`, `session_id`, and command payload fields

#### Scenario: Interact through Go runtime
- **WHEN** an agent executes `press`, `type`, or `wait` on an active session
- **THEN** the Go runtime SHALL apply input/wait behavior and return contract-compliant operation results

### Requirement: Session parity validation
The migration MUST include parity checks that confirm Go session behavior is compatible with existing fixture expectations.

#### Scenario: Fixture flow parity check
- **WHEN** the session fixture flow is executed in migration validation
- **THEN** results SHALL match the expected lifecycle and wait outcomes defined by the command contract tests
