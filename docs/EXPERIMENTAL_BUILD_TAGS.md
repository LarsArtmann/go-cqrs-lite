# Experimental Build Tags

> Go build tags that unlock experimental stdlib features. Applied automatically by `flake.nix` and CI.

## Active Tags

None. Go 1.27 graduated `encoding/json/v2`; the last active tag was retired in
the coordinated 2026-09-19 sweep (see Removed Tags).

## Usage

```bash
# Build (no tags needed on Go 1.27+)
go build ./...

# Test
go test ./...

# Or use the flake
nix run .#build
nix run .#test
```

## Removed Tags

| Tag                                 | Why Removed                                                                                                                                          |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `goexperiment.jsonv2`               | Go 1.27 graduated `encoding/json/v2` (2026-09-19); the tag/`GOEXPERIMENT=jsonv2` were no-ops and were removed from scripts, flake.nix, CI, and docs. |
| `goexperiment.arenas`               | Arena stub had zero consumers, no tests, no real GC benefit. Deleted 2026-07-11.                                                                     |
| `goexperiment.simd`                 | Gated zero files. Removed earlier.                                                                                                                   |
| `goexperiment.runtimesecret`        | Gated zero files. Removed earlier.                                                                                                                   |
| `goexperiment.goroutineleakprofile` | Gated zero files. Removed earlier.                                                                                                                   |

## Stability Policy

No experimental stdlib features are in use. If a future experiment is adopted,
document it here, wire it through `flake.nix` (`goTags`) and CI, and record the
removal criteria in this file — the jsonv2 lifecycle (experiment → adoption →
graduation sweep) is the template.
