## Context

The repository currently delivers the skill runtime through Python entrypoints (`tools/install_skill.py` and `skills/bubbletea-tui-visual-test/scripts/*.py`). The command contract for agent workflows is already established and should remain stable, but operational dependency on Python increases setup variability across local machines and CI environments. This change introduces Go as the implementation language for runtime tooling while preserving existing behavior and schema compatibility.

Key constraints:
- Existing command schema (`references/command-schema.json`) must remain the integration contract.
- Migration should avoid downtime for current users relying on Python entrypoints.
- Work must be decomposed so multiple agents can implement in parallel with non-overlapping file ownership.

## Goals / Non-Goals

**Goals:**
- Replace Python runtime tooling with Go implementations for installer, dispatcher, session engine, and visual pipeline.
- Preserve command-level behavior and response envelopes for backward compatibility.
- Standardize Go build/test workflows and CI checks.
- Provide a phased migration path with validation gates and rollback points.
- Define workstreams that can be executed independently by parallel agents.

**Non-Goals:**
- Redesigning the command schema or adding unrelated commands.
- Expanding scope beyond this repository’s skill tooling.
- Solving cross-platform rendering determinism beyond existing requirements.
- Shipping remote update/version management for installed skills.

## Decisions

### Decision 1: Keep command schema stable, swap implementation behind it
- `open|close|press|type|wait|snapshot|assert-visual|record` remain unchanged at the contract layer.
- Go binaries replace Python scripts as the execution engine.
- Rationale: minimizes integration churn and allows incremental rollout.
- Alternative: breaking v2 schema during migration. Rejected to avoid consumer rewrites.

### Decision 2: Introduce a modular Go layout with dedicated entrypoints
- Add `go.mod` at repository root (or `tools/` module root if preferred by maintainers) and organize code into packages:
  - `internal/session` for PTY lifecycle and input/wait logic
  - `internal/visual` for snapshot, diff, and GIF orchestration
  - `internal/contract` for request/response validation and schema mapping
  - `cmd/agent-tui` for command dispatch
  - `cmd/install-skill` for installer CLI
- Rationale: isolates core logic from CLI wrappers and supports independent testing.
- Alternative: one monolithic Go binary. Rejected due to weaker testability and harder parallel ownership.

### Decision 3: Use compatibility shims during migration window
- Keep Python entrypoints temporarily as thin wrappers that delegate to Go binaries (or clearly fail with migration guidance once cutover is complete).
- Define a deprecation phase and removal criteria after parity checks pass.
- Rationale: allows existing automation to continue while migration lands.
- Alternative: immediate Python removal. Rejected due to avoidable disruption.

### Decision 4: Parallelize implementation with strict file ownership
- Workstream A (Go session engine): owns `internal/session`, session-related tests.
- Workstream B (Go visual pipeline): owns `internal/visual`, visual tests.
- Workstream C (Go installer): owns `cmd/install-skill`, installer package/tests, README install section.
- Workstream D (dispatcher and contract): owns `cmd/agent-tui`, `internal/contract`, schema compatibility tests.
- Workstream E (migration/docs/CI): owns compatibility wrappers, CI workflow updates, migration notes.
- Shared contract files are sequenced: schema compatibility fixtures and interface types are published first by Workstream D.
- Rationale: enables concurrent delivery with minimal merge conflicts.
- Alternative: sequential single-agent migration. Rejected due to slower throughput and higher context switching.

### Decision 5: Preserve deterministic artifact and path behavior
- Existing per-run artifact directory semantics remain unchanged.
- Installer destination rules (`--agent`, optional `--dest`, `--force`, `--dry-run`) remain unchanged.
- Rationale: keeps workflows stable while changing implementation language.
- Alternative: alter install layout and artifact conventions during migration. Rejected to reduce blast radius.

## Risks / Trade-offs

- [Go PTY behavior differs from Python `pexpect`] -> Mitigation: parity tests against fixture workflows and contract-level golden responses.
- [Visual diff implementation differences create regressions] -> Mitigation: baseline parity tests and explicit threshold calibration fixtures.
- [Dual-runtime period creates maintenance overhead] -> Mitigation: time-box compatibility phase and define clear removal checklist.
- [Parallel streams drift on interfaces] -> Mitigation: freeze shared contract package first and require integration checks before merge.
- [CI environment missing Go toolchain version parity] -> Mitigation: pin Go version in CI and document local setup requirements.

## Migration Plan

1. Bootstrap Go module and shared contract package; add CI lint/test job for Go.
2. Implement dispatcher contract parsing and response helpers in Go.
3. Implement session engine and visual pipeline in parallel workstreams against shared contract interfaces.
4. Implement installer CLI parity in parallel with runtime workstreams.
5. Add compatibility wrappers and update docs/examples to prefer Go entrypoints.
6. Run parity validation suite (command contract, fixture flow, installer behavior).
7. Cut over defaults to Go; keep wrappers during deprecation window.
8. Remove Python implementations after deprecation criteria are met.

Rollback strategy:
- Keep Python wrappers and original scripts available until parity checks pass; rollback is switching default entrypoints back to Python and disabling Go binaries in CI/release steps.

## Open Questions

- Should Go binaries be committed as artifacts, built at install time, or both?
- Which Go image/diff libraries best match current visual comparison characteristics?
- What is the exact deprecation window for Python wrappers before removal?
