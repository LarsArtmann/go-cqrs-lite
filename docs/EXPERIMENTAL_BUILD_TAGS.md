# Experimental Build Tags

> Go build tags that unlock experimental stdlib features. Applied automatically by `flake.nix` and CI.

## Active Tags

| Tag                   | Required For         | Description                                                                                                |
| --------------------- | -------------------- | ---------------------------------------------------------------------------------------------------------- |
| `goexperiment.jsonv2` | ~25 production files | Enables `encoding/json/v2` (fully adopted). Required until Go graduates json/v2 from experimental (1.27+). |

## Usage

```bash
# Build with jsonv2 (required for all builds in Go 1.26)
go build -tags goexperiment.jsonv2 ./...

# Test
go test -tags goexperiment.jsonv2 ./...

# Or use the flake, which applies the tag automatically
nix run .#build
nix run .#test
```

## Removed Tags

| Tag                                 | Why Removed                                                                      |
| ----------------------------------- | -------------------------------------------------------------------------------- |
| `goexperiment.arenas`               | Arena stub had zero consumers, no tests, no real GC benefit. Deleted 2026-07-11. |
| `goexperiment.simd`                 | Gated zero files. Removed earlier.                                               |
| `goexperiment.runtimesecret`        | Gated zero files. Removed earlier.                                               |
| `goexperiment.goroutineleakprofile` | Gated zero files. Removed earlier.                                               |

## Stability Policy

The `goexperiment.jsonv2` tag is required for the project to compile. It is not
opt-in — all builds, tests, and CI pipelines apply it. When Go stabilizes json/v2
(expected Go 1.27+), the tag can be removed with zero code changes.
