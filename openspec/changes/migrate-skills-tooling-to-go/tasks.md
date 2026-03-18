## 1. Shared bootstrap and contract freeze (sequential prerequisite)

- [x] 1.1 Add Go module bootstrap (`go.mod`, base package layout) and document canonical entrypoints in `README.md`
- [x] 1.2 Create shared command contract package (`internal/contract`) with typed request/response models mapped to `skills/bubbletea-tui-visual-test/references/command-schema.json`
- [x] 1.3 Add contract compatibility fixtures/golden responses for `open|close|press|type|wait|snapshot|assert-visual|record`
- [x] 1.4 Add CI job for Go lint/test/build validation and keep existing migration-era checks passing

## 2. Workstream A: session runtime migration (Agent A, owns `internal/session/**`, `tests/session/**`)

- [ ] 2.1 Implement Go PTY session lifecycle (`open`, `close`) with deterministic runtime parameters
- [ ] 2.2 Implement Go interaction commands (`press`, `type`, `wait`) with timeout and structured error handling
- [ ] 2.3 Add session parity tests against fixture workflows and contract golden responses
- [ ] 2.4 Provide integration hooks consumed by dispatcher without editing non-owned files

## 3. Workstream B: visual pipeline migration (Agent B, owns `internal/visual/**`, `tests/visual/**`)

- [x] 3.1 Implement Go snapshot capture producing PNG plus deterministic metadata records
- [x] 3.2 Implement Go baseline comparison (`assert-visual`) with threshold-based pass/fail and diff artifact output
- [x] 3.3 Implement Go GIF export (`record`) including structured renderer-unavailable failures
- [x] 3.4 Add visual parity tests for metadata shape, diff pass/fail behavior, and renderer failure handling

## 4. Workstream C: installer migration (Agent C, owns `cmd/install-skill/**`, `internal/install/**`, `tests/install/**`)

- [ ] 4.1 Implement Go installer CLI with `--agent`, `--skill`, `--source-root`, `--dest`, `--force`, and `--dry-run`
- [ ] 4.2 Implement destination resolution parity for `claude`, `copilot`, `codex`, and `opencode`
- [ ] 4.3 Implement safe overwrite behavior and dry-run reporting parity with existing installer semantics
- [ ] 4.4 Add installer tests for supported-agent success, unknown-agent rejection, destination override, and overwrite guardrails

## 5. Workstream D: dispatcher and migration wrappers (Agent D, owns `cmd/agent-tui/**`, `skills/bubbletea-tui-visual-test/scripts/**`, docs migration notes)

- [ ] 5.1 Implement Go dispatcher entrypoint that validates requests and routes commands to session/visual packages via shared contract types
- [ ] 5.2 Add migration-era compatibility wrappers for legacy Python command paths with explicit deprecation guidance
- [ ] 5.3 Update skill docs/examples to use Go-first invocation while documenting temporary compatibility paths
- [ ] 5.4 Add dispatcher contract tests for success/failure envelope parity across all commands

## 6. Parallel integration and cutover (sequential after 2-5)

- [ ] 6.1 Merge workstreams and run full parity suite (session flow, visual regression flow, installer behavior)
- [ ] 6.2 Validate non-overlapping ownership assumptions and resolve any contract mismatches discovered at integration time
- [ ] 6.3 Flip default tooling references from Python to Go in repository docs and CI workflows
- [ ] 6.4 Define and execute Python deprecation checklist (cutoff date, removal PR, rollback notes)
