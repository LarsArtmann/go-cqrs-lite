# ADR-0026: Experimental Features Behind Build Tags

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-19   |
| Status  | Accepted     |
| Decider | Lars Artmann |

## Context

Several experimental features are being explored:
- **jsonv2 codec** — Go's upcoming JSON v2 (encoding/json/v2)
- **Arena allocation** — Go experiment for bulk memory allocation
- **SIMD operations** — Vectorized crypto/encoding
- **WASM target** — Browser/edge compilation

These depend on unstable Go features or draft APIs that change frequently.

## Decision

**Experimental features live behind build tags and in dedicated modules.**
They are NOT compiled into default builds.

### Build Tag Convention

```go
//go:build goexperiment.jsonv2
//go:build goexperiment.arenas
//go:build goexperiment.simd
```

Each experiment is:
1. A separate file (`codec_jsonv2.go`, `event_arena.go`) with the build tag
2. Documented in `docs/EXPERIMENTAL_BUILD_TAGS.md`
3. Tested only when the experiment is enabled
4. Not part of the public API surface until stabilized

### WASM Compilation

The following modules compile to WASM (`GOOS=js GOARCH=wasm`):
- `id/` — branded IDs, ULID generation
- `codec/` — JSON, CBOR encoding
- `dispatcher/` — generic dispatch
- `event/` — event creation, metadata
- `command/` — command types
- `query/` — query types

**Known limitation**: `decider/` does NOT compile to WASM because OTel SDK
(transitive dependency) imports `os/user`, which is not available in js/wasm.
This will be resolved when OTel SDK adds WASM support or when the decider
module makes OTel fully optional.

## Consequences

- **+** Default builds are stable and dependency-free
- **+** Experiments can be tested without affecting production
- **+** Clear separation between stable and experimental code
- **-** Experiment code may rot between Go releases
- **-** Users must opt-in explicitly via build tags

## References

- `docs/EXPERIMENTAL_BUILD_TAGS.md` — build tag documentation
- `wasm/main.go` — WASM verification binary
- [Go experiments](https://tip.golang.org/doc/go1.26#experiments)
