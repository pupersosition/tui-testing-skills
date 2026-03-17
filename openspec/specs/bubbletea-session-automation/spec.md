## ADDED Requirements

### Requirement: Deterministic Bubble Tea session lifecycle
The skill MUST provide commands to start and stop a Bubble Tea application in a pseudo-terminal with explicit runtime parameters including working directory, environment overrides, terminal width, and terminal height.

#### Scenario: Open session with explicit runtime parameters
- **WHEN** an agent executes the session open command with command string and terminal parameters
- **THEN** the system SHALL create a session and return a unique session identifier plus normalized runtime metadata

#### Scenario: Close session after completion
- **WHEN** an agent executes the session close command for an active session
- **THEN** the system SHALL terminate the process tree and mark the session state as closed

### Requirement: Interactive input and state wait controls
The skill MUST support deterministic interaction primitives for Bubble Tea applications, including key presses, text input, and wait-until checks using text or regular expressions.

#### Scenario: Send key input to active session
- **WHEN** an agent sends a key press command to an active session
- **THEN** the system SHALL deliver the input to the pseudo-terminal and return an operation result with timestamp

#### Scenario: Wait for expected UI state
- **WHEN** an agent runs a wait command with a text or regex condition and timeout
- **THEN** the system SHALL return success when the condition appears or a timeout failure when it does not

### Requirement: Structured command responses
All session automation commands MUST return structured JSON output containing operation status, session identifier, and command-specific payload data.

#### Scenario: Successful command response shape
- **WHEN** a session automation command succeeds
- **THEN** the response SHALL include `ok=true`, `session_id`, and `data` fields

#### Scenario: Failed command response shape
- **WHEN** a session automation command fails
- **THEN** the response SHALL include `ok=false`, `error.code`, and `error.message` fields
