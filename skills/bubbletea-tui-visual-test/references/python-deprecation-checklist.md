# Python Deprecation Checklist

Date opened: 2026-03-18
Scope: Migration from Python runtime tooling to Go runtime tooling for installer and dispatcher flows.

## Milestones

- Go-first defaults active: 2026-03-18
- Python compatibility freeze date: 2026-05-31
- Target Python removal PR window: 2026-06-01 to 2026-06-15
- Target compatibility wrapper removal merge: 2026-06-30

## Execution Checklist

- [x] Switch primary docs to Go commands (`README.md`, `references/examples.md`, `references/ci-workflow.md`)
- [x] Set CI default validation path to Go-first checks
- [x] Keep Python compatibility checks in CI as secondary gate during migration window
- [x] Keep migration-era Python dispatcher wrapper with deprecation warning
- [x] Record rollback approach before Python removal
- [ ] Create removal PR deleting Python runtime implementations after freeze date
- [ ] Remove Python compatibility CI job after removal PR lands
- [ ] Archive migration notes after Python paths are fully removed

## Rollback Notes

If Go runtime regressions appear before Python removal:

1. Re-enable Python-first invocation guidance in README and references.
2. Keep Go checks running, but mark Python compatibility path as primary until parity is restored.
3. Re-run parity suite from `references/integration-cutover.md` before attempting cutover again.

