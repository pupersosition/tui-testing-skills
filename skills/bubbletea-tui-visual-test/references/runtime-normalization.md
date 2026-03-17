# Runtime Normalization

Deterministic visual checks require explicit runtime pinning. Provide these values in `open.params`.

## Required Open Parameters

- `cols`: fixed terminal width (recommended `80`)
- `rows`: fixed terminal height (recommended `24`)
- `locale`: fixed locale (recommended `C.UTF-8`)
- `theme`: fixed UI theme label (recommended `light`)
- `color_mode`: one of `16`, `256`, `truecolor` (recommended `256` for CI portability)

## Recommended Environment Overrides

Set these in `open.params.env`:

- `TERM=xterm-256color`
- `LANG=C.UTF-8`
- `LC_ALL=C.UTF-8`
- `TZ=UTC`
- `NO_COLOR=`

If your app supports custom deterministic flags, pass them here too (for example animation-off toggles).

## Metadata Rules

Snapshot metadata should include:

- `cols`, `rows`, `locale`, `theme`, `color_mode`
- Renderer name and version
- OS and architecture
- Skill command version (`1.0.0`)

Baseline comparisons are valid only when metadata fields match expected baseline metadata.

## Stability Checklist

1. Run in a clean workspace and isolated run directory.
2. Avoid concurrent writes to the same artifact path.
3. Keep fixture/test data stable across runs.
4. Keep font/renderer stack fixed in CI images.
