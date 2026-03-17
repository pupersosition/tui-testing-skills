## Context

This change introduces a new skill for validating Bubble Tea applications through automated terminal interaction and visual checks. The target user experience is similar to browser automation: open a session, interact, assert state, and capture visual artifacts. Today there is no shared contract for these steps, which leads to inconsistent testing and difficult handoffs across parallel Conductor agents.

Constraints:
- Bubble Tea is the primary framework target for v1.
- Terminal rendering can vary by platform, so outputs must be normalized.
- The implementation should allow multiple agents to work concurrently without touching the same files.

## Goals / Non-Goals

**Goals:**
- Provide deterministic Bubble Tea session automation (start, input, wait, snapshot, close).
- Provide visual regression primitives based on PNG checkpoint snapshots.
- Produce GIF artifacts for human design review.
- Define a stable command contract that can be used by agents in scripted flows.
- Partition implementation into independent workstreams to minimize merge conflicts.

**Non-Goals:**
- Supporting non-Bubble Tea TUI frameworks in v1.
- Pixel-perfect guarantees across all terminals, fonts, and OS themes.
- Replacing Bubble Tea unit/integration tests; this complements them.
- Building a hosted service; scope is local/CI skill execution.

## Decisions

### Decision 1: Use a Python `pexpect` session engine
- The session controller will run Bubble Tea commands in a pseudo-terminal and support:
  - start session with fixed env/size
  - send keys/text
  - wait for text/regex conditions
  - capture full buffer snapshots
  - enforce timeouts and process cleanup
- Rationale: `pexpect` is available in the target environment and provides deterministic PTY control with minimal setup.
- Alternative considered: tmux-based control. Rejected for v1 because tmux is not guaranteed to be present and adds operational coupling.

### Decision 2: Split assertions into behavioral and visual lanes
- Behavioral lane: text/regex assertions for state transitions and key flow correctness.
- Visual lane: deterministic PNG checkpoint snapshots compared to baselines with thresholded diffs.
- GIF lane: generate a playback artifact for design review and PR communication.
- Rationale: GIF is valuable for human review but unstable as a strict CI gate; PNG checkpoints provide better deterministic comparison.
- Alternative considered: GIF-only assertions. Rejected due to high flakiness and noisy diffs.

### Decision 3: Define a command-oriented interface analogous to browser automation
- The skill will expose command steps such as `open`, `press`, `type`, `wait`, `snapshot`, `assert-visual`, and `record`.
- Commands return structured JSON for machine readability and chaining.
- Rationale: keeps the agent workflow predictable and easy to compose into test tapes.
- Alternative considered: direct ad-hoc shell snippets in SKILL.md. Rejected due to poor reusability and harder validation.

### Decision 4: Enforce deterministic rendering contract
- All runs must declare terminal dimensions, color mode, locale, and theme settings.
- Snapshot metadata must include these parameters and tool versions.
- Baseline comparisons are valid only when metadata matches.
- Rationale: reduces false positives from environmental drift.

### Decision 5: Parallel-safe repository ownership boundaries
- Workstream A owns session control scripts and command schema.
- Workstream B owns visual pipeline scripts and snapshot diff tooling.
- Workstream C owns SKILL.md orchestration, references, and examples.
- Shared touchpoints are restricted to documented interfaces to avoid file overlap.
- Rationale: enables multiple Conductor agents to execute in parallel with predictable merge behavior.

## Risks / Trade-offs

- [Terminal rendering variance may still cause flaky visual diffs] -> Mitigation: strict environment normalization, tolerant diff thresholds, and metadata pinning.
- [External renderer/tool dependency may be unavailable in some environments] -> Mitigation: keep renderer pluggable and provide text-only fallback assertions.
- [Long-running sessions may leak processes] -> Mitigation: enforce global timeouts, trap signals, and always execute cleanup on failure paths.
- [Parallel workstreams can diverge in interface expectations] -> Mitigation: define command JSON schema early and treat it as the contract across streams.

## Migration Plan

1. Create skill scaffold and command contract docs.
2. Implement session engine and text assertions behind stable command JSON.
3. Implement visual snapshot + diff + GIF pipeline with deterministic metadata.
4. Integrate into SKILL workflow examples and add baseline artifact conventions.
5. Validate in CI with a sample Bubble Tea app; document fallback behavior when visual tooling is unavailable.

Rollback strategy:
- Because this is additive, rollback is removal of the new skill folder and any CI hooks referencing it.

## Open Questions

- Which renderer should be the default for PNG/GIF generation in CI (and what is the minimum supported version)?
- Should visual baseline storage live in-repo or external artifact storage for larger projects?
- Do we require a strict perceptual diff threshold globally, or per-snapshot override in test tapes?
