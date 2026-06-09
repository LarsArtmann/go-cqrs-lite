# Immutability Audit — Phase 3: Cross-Module Zero-Copy

**Date**: 2026-06-09 23:15
**Scope**: Export PayloadReadOnly, eliminate 5 wasted clones across 4 modules, fix builder double-clone

---

## A) FULLY DONE

### Exported `PayloadReadOnly` (new public API)
- `event.PayloadReadOnly(evt Event) []byte` — zero-copy payload accessor for read-only paths
- For `*ImmutableEvent`: returns internal `payload` field directly (no clone)
- For custom `Event` implementations: falls back to `evt.Payload()` (safe copy)
- Documented contract: returned slice MUST NOT be mutated
- Unit tests: `TestPayloadReadOnly_ReturnsInternalReference`, `TestPayloadReadOnly_FallbackForCustomImplementation`

### Eliminated 5 wasted clones across 4 modules

| Module | File | Before | After | Save |
|--------|------|--------|-------|------|
| signing | `payload.go:28` | `evt.Payload()` → SHA-256 hash | `PayloadReadOnly(evt)` | 1 alloc/call |
| signing | `event.go:25` | `evt.Payload()` → `NewEvent` (double-clone) | `PayloadReadOnly(evt)` → `NewEvent` (single clone) | 1 alloc/call |
| pebble | `serialization.go:21` | `evt.Payload()` → `json.Marshal` | `PayloadReadOnly(evt)` | 1 alloc/call |
| storage/sql | `helpers.go:73` | `evt.Payload()` → `ExecContext` | `PayloadReadOnly(evt)` | 1 alloc/call |
| middleware | `sse.go:141` | `evt.Payload()` → `string()` → Fprintf | `PayloadReadOnly(evt)` | 1 alloc/call |

### Fixed builder double-clone
- `builder.Build()` now calls `buildEvent()` directly instead of `NewEvent()`
- `WithPayload()` already clones on intake; `buildEvent` skips re-validation/re-clone
- Saves 1 alloc per builder-built event

### Previous phases (cumulative)
- 12 mutability leaks sealed (P1 fixes from Phase 1+2)
- Golden tests fixed in codec/ and middleware/
- `encodingForCopy` preserves raw encoding field
- `MarkTombstone`/`MarkRebirth` bypass double-clone
- `WithCommandMetadata` clones on intake
- 6 `make+copy` patterns consolidated to `slices.Clone`
- 3 fuzz tests, 6 field-preservation/isolation tests
- ADR-0013, AGENTS.md patterns #14/#15

---

## B) PARTIALLY DONE
- None

---

## C) NOT STARTED
- Property-based tests with `rapid` for all accessor methods
- `sync.Pool` for event payload buffers
- `io.Reader` for payload streaming
- Allocation profiling in CI
- `NewSliceStream` slice cloning (decided: not needed, Event is immutable boundary)

---

## D) TOTALLY FUCKED UP / REGRETS
- Buildflow pre-commit hook still panics (unrelated to our changes)
- No other regrets

---

## E) WHAT WE SHOULD IMPROVE
1. Benchmark before/after for `PayloadReadOnly` — quantify the actual alloc reduction in signing/storage/pebble benchmarks
2. Consider if `watermill/protocol.go` could also benefit — currently needs clone (Watermill takes ownership)
3. ADR update: ADR-0013 should mention `PayloadReadOnly` as the cross-module solution

---

## F) Top #10 Next Items

| # | Priority | Task | Impact | Effort |
|---|----------|------|--------|--------|
| 1 | P1 | Update ADR-0013 to include PayloadReadOnly cross-module pattern | Documentation | S |
| 2 | P2 | Benchmark PayloadReadOnly in signing/storage/pebble hot paths | Visibility | S |
| 3 | P2 | Add PayloadReadOnly benchmark to event/benchmark_clone_test.go | Visibility | S |
| 4 | P2 | Property-based tests with rapid for all accessor methods | Test rigor | M |
| 5 | P3 | Allocation profiling in CI | Visibility | M |
| 6 | P3 | Fuzz tests for codec/ module | Robustness | S |
| 7 | P3 | Integration test: signing roundtrip through real store | Test coverage | S |
| 8 | P4 | sync.Pool for event payload buffers | Perf | M |
| 9 | P4 | io.Reader for payload streaming | Architecture | L |
| 10 | P5 | Write ADR for PayloadReadOnly API design decision | Documentation | S |

---

## G) Top #1 Question

None — all questions from previous phases resolved.

---

## Test Results
- All 39 test packages: PASS
- Lint: 0 issues across 22 modules
- Build: PASS
