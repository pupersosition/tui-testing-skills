## 1. Contract and scaffold (sequential prerequisite)

- [x] 1.1 Create `.codex/skills/bubbletea-tui-visual-test/` scaffold with `SKILL.md`, `scripts/`, and `references/` directories
- [x] 1.2 Define and commit command contract file `.codex/skills/bubbletea-tui-visual-test/references/command-schema.json` (owner: integration lead only)
- [x] 1.3 Add a minimal Bubble Tea fixture app under `.codex/skills/bubbletea-tui-visual-test/assets/fixtures/` for deterministic e2e validation

## 2. Workstream A: session automation engine (Agent A)

- [x] 2.1 Implement PTY session lifecycle commands (`open`, `close`) in `.codex/skills/bubbletea-tui-visual-test/scripts/session_engine.py`
- [x] 2.2 Implement interaction commands (`press`, `type`, `wait`) with timeout handling in `.codex/skills/bubbletea-tui-visual-test/scripts/session_engine.py`
- [x] 2.3 Implement structured JSON response helpers in `.codex/skills/bubbletea-tui-visual-test/scripts/session_contract.py`
- [x] 2.4 Add automated tests for lifecycle and wait behavior in `.codex/skills/bubbletea-tui-visual-test/scripts/tests/test_session_engine.py`

## 3. Workstream B: visual regression pipeline (Agent B)

- [ ] 3.1 Implement PNG checkpoint capture + metadata writer in `.codex/skills/bubbletea-tui-visual-test/scripts/visual_pipeline.py`
- [ ] 3.2 Implement baseline diff command (`assert-visual`) with configurable threshold in `.codex/skills/bubbletea-tui-visual-test/scripts/visual_pipeline.py`
- [ ] 3.3 Implement GIF export command (`record`) in `.codex/skills/bubbletea-tui-visual-test/scripts/visual_pipeline.py`
- [ ] 3.4 Add automated tests for snapshot metadata, pass/fail diff behavior, and renderer-unavailable errors in `.codex/skills/bubbletea-tui-visual-test/scripts/tests/test_visual_pipeline.py`

## 4. Workstream C: skill workflow and references (Agent C)

- [ ] 4.1 Author `.codex/skills/bubbletea-tui-visual-test/SKILL.md` with command-oriented workflow aligned to `command-schema.json`
- [ ] 4.2 Add reference docs for assertions and deterministic runtime settings in `.codex/skills/bubbletea-tui-visual-test/references/assertions.md` and `references/runtime-normalization.md`
- [ ] 4.3 Add parallel-safe artifact conventions in `.codex/skills/bubbletea-tui-visual-test/references/artifact-layout.md` (per-run output directories and naming)
- [ ] 4.4 Add end-to-end usage example script in `.codex/skills/bubbletea-tui-visual-test/references/examples.md` that produces both snapshot and GIF outputs

## 5. Integration and verification (sequential after 2/3/4)

- [ ] 5.1 Integrate dispatcher entrypoint `.codex/skills/bubbletea-tui-visual-test/scripts/agent_tui.py` that routes commands using `command-schema.json`
- [ ] 5.2 Run fixture e2e flow to validate `open -> interact -> wait -> snapshot -> assert-visual -> record` and archive artifacts under a unique run directory
- [ ] 5.3 Verify no path ownership conflicts across workstreams and reconcile any schema mismatches
- [ ] 5.4 Document CI invocation and baseline update workflow in `.codex/skills/bubbletea-tui-visual-test/references/ci-workflow.md`
