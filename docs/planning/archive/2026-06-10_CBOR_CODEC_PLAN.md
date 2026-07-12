# CBOR Codec Implementation Plan

**Goal:** Implement Option B (all-in CBOR) for go-cqrs-lite's `codec/` module.

**Constraint:** Each task ≤ 12 minutes. Sorted by importance/impact/effort/customer-value.

## Task Table

| #   | Task                                                                   | Impact               | Effort | Files                                                                  | Dependencies |
| --- | ---------------------------------------------------------------------- | -------------------- | ------ | ---------------------------------------------------------------------- | ------------ |
| 1   | Add `fxamacker/cbor` dep to `codec/go.mod`                             | 🔴 Blocker           | 5min   | `codec/go.mod`, `codec/go.sum`                                         | None         |
| 2   | Add `EncodingCBOR` constant to `codec/codec.go`                        | 🔴 Blocker           | 3min   | `codec/codec.go`                                                       | #1           |
| 3   | Implement `CBORCodec` struct in `codec/cbor.go`                        | 🔴 Core feature      | 10min  | `codec/cbor.go` (new)                                                  | #1, #2       |
| 4   | Unit tests for `CBORCodec` in `codec/codec_test.go`                    | 🔴 Quality gate      | 10min  | `codec/codec_test.go`                                                  | #3           |
| 5   | Fuzz test for CBOR round-trip in `codec/codec_fuzz_test.go`            | 🔴 Safety            | 8min   | `codec/codec_fuzz_test.go`                                             | #3           |
| 6   | Benchmarks: CBORCodec vs JSONCodec in `codec/benchmark_test.go`        | 🟡 Validation        | 8min   | `codec/benchmark_test.go`                                              | #3           |
| 7   | Golden test + fixture for CBOR encoding                                | 🟡 Regression safety | 8min   | `codec/golden_test.go`, `codec/testdata/golden/cbor_encode.cbor` (new) | #3           |
| 8   | Verify `event/` module works with CBOR (encoding match, DecodePayload) | 🔴 Integration       | 10min  | No code changes — run existing tests with CBORCodec                    | #3           |
| 9   | Add CBOR-specific test in `event/codec_test.go`                        | 🟡 Coverage          | 8min   | `event/codec_test.go`                                                  | #3           |
| 10  | Verify Pebble store works with CBOR payloads                           | 🔴 Integration       | 10min  | No code changes — run existing tests                                   | #3           |
| 11  | Verify SQL store works with CBOR payloads                              | 🔴 Integration       | 10min  | No code changes — run existing tests                                   | #3           |
| 12  | Verify signing works with deterministic CBOR                           | 🔴 Critical property | 10min  | No code changes — run existing tests                                   | #3           |
| 13  | Verify encryption codec wrapper works with CBORCodec                   | 🟡 Integration       | 8min   | No code changes — run existing tests                                   | #3           |
| 14  | Update `codec/doc.go` with CBORCodec documentation                     | 🟢 DX                | 5min   | `codec/doc.go`                                                         | #3           |
| 15  | Add CBOR variant to event clone benchmarks                             | 🟡 Perf data         | 8min   | `event/benchmark_clone_test.go`                                        | #3           |
| 16  | Add CBOR variant to integration realistic benchmarks                   | 🟡 Perf data         | 10min  | `integration/realistic_bench_test.go`                                  | #3           |
| 17  | Run full test suite (`nix run .#test`) — verify no regressions         | 🔴 Final gate        | 5min   | None                                                                   | All above    |
| 18  | Run lint (`nix run .#lint`) — fix any issues                           | 🟡 Quality           | 5min   | None                                                                   | #17          |
| 19  | Update `codec/README.md` with CBOR section                             | 🟢 DX                | 5min   | `codec/README.md`                                                      | #3           |
| 20  | Update project `AGENTS.md` with CBOR codec info                        | 🟢 Memory            | 5min   | `AGENTS.md`                                                            | #17          |

## Phase Breakdown

### Phase 1: Core Implementation (#1–#3) — ~18min

The library itself. Everything else depends on this.

### Phase 2: Testing & Safety (#4–#7) — ~34min

Unit tests, fuzz tests, benchmarks, golden tests.

### Phase 3: Integration Verification (#8–#13) — ~58min

Run existing test suites to verify zero breaking changes. Most tasks are "run tests, verify pass" — no code changes needed because the architecture is codec-agnostic by design.

### Phase 4: Documentation & Polish (#14–#20) — ~43min

Docs, benchmarks, lint, final gate.

## Key Insight: Zero Breaking Changes

The architecture is already codec-agnostic. The `Codec` interface, `Encoding` field, `WithCodec()` option, and all storage layers (Pebble, SQL, memory) work with any `Codec` implementation. CBORCodec is a **pure addition** — no existing code changes behavior.

Only `codec/cbor.go` is a new source file. Everything else is tests, docs, and dependency management.
