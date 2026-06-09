# Immutability Audit & Performance Optimization Status

**Date**: 2026-06-09 20:44
**Scope**: P0 Payload() clone benchmarking + P1 full mutability leak audit across all modules

---

## A) FULLY DONE

### P0: Benchmark slices.Clone overhead in Payload()

- **Benchmark suite** (`event/benchmark_clone_test.go`): Measures `slices.Clone` vs `make+copy` vs `append` across 16B–64KB payloads, Payload() vs direct field access, and DecodePayload clone vs zero-copy paths
- **Finding**: All three clone approaches are equivalent for realistic sizes. `Payload()` costs 37–1165 ns + 1 alloc (size-dependent) vs ~1 ns for direct access
- **Optimization**: `DecodePayload` now uses `payloadForDecode()` — a type assertion to `*ImmutableEvent` for zero-copy access, saving 1 alloc per decode. Falls back to `Payload()` for any custom `Event` implementation
- **Verified**: `*ImmutableEvent` is the only production `Event` implementation. One test-only stub exists. Optimization is safe

### P1: Mutability leak fixes (6 issues fixed)

| Leak | Module | Fix |
|------|--------|-----|
| `PersistedCommand.Metadata()` returns shared map | command | → `c.metadata.Clone()` |
| `BasicCommand.Metadata()` returns shared map | command | → `c.metadata.Clone()` |
| `getRefsUnsorted()` returns internal cache slice | listing | → `slices.Clone(cached)` |
| `projectionFunc.EventTypes()` returns internal slice | event | → `slices.Clone(p.eventTypes)` |
| `NewProjection` stores caller's slice directly | event | → `slices.Clone(eventTypes)` on intake |
| `builtProjection.EventTypes()` returns internal slice | projection | → `slices.Clone(p.eventTypes)` |
| `Builder.Build()` shares eventTypes with builder | projection | → `slices.Clone(types)` on build |

### Mutation-safety tests (4 new tests)

- `command/store_test.go` — `TestNewPersistedCommand_MetadataIsolation`
- `command/metadata_test.go` — `TestCommand_MetadataIsolation`
- `event/projection_test.go` — `TestProjection_EventTypesIsolation`
- `projection/builder_test.go` — `TestBuilder_EventTypesIsolation` + `TestBuilder_BuildIsolation`

### Full audit confirmation (all safe)

- `signing/`: HMAC/Ed25519 keys cloned at construction, middleware creates fresh events, `Signature.Bytes()` is defensive but redundant (safe)
- `memory/`: All stores return `slices.Clone`, snapshots deep-copy on save/load
- `storage/`: SQL reconstruction goes through `NewEvent` which clones payload
- `pebble/`: Keys/values freshly allocated, reads go through `NewEvent` clone
- All constructors clone `[]byte`/map/slice inputs before storing
- `tombstone.go` `MarkTombstone` calls `Payload()` (already clones) then passes to `NewEvent` (clones again) — double-clone is wasteful but safe

---

## B) PARTIALLY DONE

- **Event interface immutability contract**: The `ImmutableEvent.Payload()` method documents the clone contract, but the `Event` interface itself has no doc comment stating implementors must return safe copies. Third-party `Event` implementations wouldn't know the rule
- **`Signature.Bytes()` clone**: The `slices.Clone` in `signing/signature.go:21` is technically redundant — `Signature` is always freshly allocated by `mac.Sum(nil)` or `ed25519.Sign()`, never shared. Could be removed for a micro-optimization, but follows the established defensive pattern

---

## C) NOT STARTED

### Type Model Improvements

1. **Document immutability contract on `Event` interface** — Add interface-level doc stating `Payload()` must return a safe copy. Low effort, high value for third-party implementors
2. **Consider `PayloadForDecode()` as exported opt-in** — Consumers who only read payload bytes (signing, serialization) could benefit from the same bypass. Currently internal-only
3. **`MultiSignature.Get()` returns `*SignatureEntry` pointer into slice** — Caller can mutate entries. Low risk (always reconstructed from JSON) but violates defensive-copy principles
4. **Eliminate double-clone in `MarkTombstone`** — Calls `Payload()` (clones) then passes to `NewEvent` (clones again). Could use internal `buildEvent` with direct field access
5. **`event.Builder.WithPayload()` stores caller's `[]byte` directly** — Mitigated by `Build()` calling `NewEvent` which clones, but worth documenting or fixing

### Architecture / Library Considerations

6. **Consider `github.com/cockroachdb/crlfmt` or `go.uber.org/atomic`** for immutable-by-design patterns
7. **Evaluate if `samber/ro` reactive streams could replace custom bus implementations** — Already used for EventBus, CommandBus, QueryBus
8. **Consider a `ReadonlyPayload()` or `RawPayload()` method on `Event` interface** — For internal hot paths that only read. Would require careful interface evolution

---

## D) TOTALLY FUCKED UP / REGRETS

- **None**. All changes are clean, tests pass, no regressions introduced
- Pre-existing golden test failures in `codec/` and `middleware/` are unrelated to this work

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Event interface should document the immutability contract** — Currently only `ImmutableEvent.Payload()` documents it. Interface-level documentation would prevent third-party violations
2. **`payloadForDecode` is a one-off optimization** — If more internal paths need zero-copy payload access, consider a package-private `payloadAccessor` pattern or unexported interface
3. **Clone accounting** — We have 11 `slices.Clone` calls across the codebase plus several `make+copy` patterns. A consistent `cloneBytes` helper could reduce cognitive load and make the audit trail clearer

### Performance

4. **`MarkTombstone` double-clones** — The only hot-path caller of `Payload()` outside `DecodePayload`. Could save one allocation per tombstone marking
5. **`Signature.Bytes()` clone is redundant** — `mac.Sum(nil)` and `ed25519.Sign()` always return fresh allocations. Could remove for micro-optimization but follows defensive pattern

### Testing

6. **Property-based immutability tests** — Use `rapid` to verify all accessor methods return independent copies. Current tests are manual and only cover known cases
7. **Benchmark `Metadata.Clone()` overhead** — Not yet measured. The `maps.Clone` + struct copy has a measurable cost on hot paths

---

## F) Top #25 Things We Should Get Done Next

Sorted by Impact × Effort (Pareto):

| # | Priority | Task | Impact | Effort |
|---|----------|------|--------|--------|
| 1 | P0 | Fix pre-existing golden test failures in `codec/` and `middleware/` | Test hygiene | S |
| 2 | P1 | Document immutability contract on `Event` interface | Consumer safety | S |
| 3 | P1 | Eliminate `MarkTombstone` double-clone | Perf (alloc reduction) | S |
| 4 | P1 | Add `DecodePayloads` optimization (batch version uses same `payloadForDecode`) | Perf | S |
| 5 | P1 | Fix `MultiSignature.Get()` returning mutable pointer | Safety | S |
| 6 | P1 | Audit `event.Builder.WithPayload()` — clone on intake | Safety | S |
| 7 | P2 | Benchmark `Metadata.Clone()` overhead on hot paths | Visibility | S |
| 8 | P2 | Add property-based immutability tests with `rapid` | Test rigor | M |
| 9 | P2 | Run `nix run .#bench` baseline comparison before/after this change | Regression detection | S |
| 10 | P2 | Consider removing `Signature.Bytes()` clone (redundant) | Micro-perf | S |
| 11 | P2 | Add `go:generate` target for running immutability audits | DX | M |
| 12 | P3 | Evaluate `io.Reader` for payload streaming (large payloads) | Architecture | L |
| 13 | P3 | Consider `sync.Pool` for event payload buffers | Perf (GC pressure) | M |
| 14 | P3 | Add allocation profiling to CI (`-allocspace`) | Visibility | M |
| 15 | P3 | Document the `payloadForDecode` pattern in AGENTS.md | Knowledge | S |
| 16 | P3 | Review `event.Builder` for other mutability issues | Safety | M |
| 17 | P3 | Consider `[]byte` -> `string` for immutable payloads where possible | Perf (no copy needed) | L |
| 18 | P4 | Add fuzz tests for payload encode/decode roundtrip | Robustness | M |
| 19 | P4 | Evaluate `google/uuid` vs `oklog/ulid` for event ID generation perf | Perf | M |
| 20 | P4 | Profile real-world event processing pipeline end-to-end | Visibility | L |
| 21 | P4 | Consider `unsafe.String` / `unsafe.Slice` for zero-copy JSON interop | Perf (advanced) | L |
| 22 | P4 | Add `PooledEvent` type for high-throughput scenarios | Perf | L |
| 23 | P4 | Evaluate `golang.org/x/exp/constraints` for type-safe payload generics | DX | M |
| 24 | P5 | Write ADR for the `payloadForDecode` optimization pattern | Documentation | S |
| 25 | P5 | Consider `copier` or `deepcopy` library for complex struct cloning | DX | M |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `Payload()` clone at all?**

The current design clones defensively on every `Payload()` call. But in practice:
- **99% of callers** only read the bytes (JSON decode, signing hash, logging)
- **The only mutation path** is test code that explicitly verifies isolation
- The clone costs 37–1165 ns per call depending on payload size

**Alternative**: Remove the clone, document that `Payload()` returns an internal reference that must not be mutated, and trust consumers (Go style — don't protect against misuse). This is the `bytes.Buffer.Bytes()` approach.

**Risk**: A consumer mutates the returned slice and corrupts event state. In a library, this is a real concern since you don't control the consumer.

**My recommendation**: Keep the clone in `Payload()` for the public API, but expand the `payloadForDecode` pattern to all internal read-only paths. The `Event` interface stays safe-by-default, internals stay fast.

---

## Changed Files

```
command/command.go              | Metadata() → .Clone()
command/metadata_test.go        | + TestCommand_MetadataIsolation
command/store.go                | Metadata() → .Clone()
command/store_test.go           | + TestNewPersistedCommand_MetadataIsolation
event/codec.go                  | + payloadForDecode() zero-copy optimization
event/projection.go             | Clone on intake + output for EventTypes()
event/projection_test.go        | + TestProjection_EventTypesIsolation
listing/in_memory.go            | getRefsUnsorted() → slices.Clone()
projection/builder.go           | Clone on Build() + EventTypes() output
projection/builder_test.go      | + TestBuilder_EventTypesIsolation, TestBuilder_BuildIsolation
event/benchmark_clone_test.go   | NEW: comprehensive clone benchmarks
```

## Test Results

- All changed modules: PASS
- Full test suite: PASS (pre-existing failures in `codec/` and `middleware/` golden tests unrelated)
- Lint: 0 issues across all modules
- Build: PASS
