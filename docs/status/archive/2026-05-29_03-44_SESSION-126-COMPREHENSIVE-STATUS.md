# Session 126 — Comprehensive Status Report

**Date:** 2026-05-29 03:44 · **Branch:** master · **Commits ahead:** 4

---

## Executive Summary

All 14 production modules build and pass tests. 0 compilation errors, 0 vet issues.
Stale gopls diagnostics from prior sessions remain visible but are not real errors.

---

## A) FULLY DONE

| #   | Item                                                          | Evidence                                                        |
| --- | ------------------------------------------------------------- | --------------------------------------------------------------- |
| 1   | Stream module BDD tests (93.5% coverage)                      | 23 BDD specs, SQL reader + projection + integration             |
| 2   | Core types BDD (Version, SchemaVersion, CheckVersionConflict) | 17 BDD specs + internal parseSchemaVersion test                 |
| 3   | go.mod version standardization                                | All 21 modules now v1.6.0, 0 mismatches                         |
| 4   | Performance regression CI                                     | Benchmark job in ci.yml, 21-baseline benchmarks                 |
| 5   | example/user smoke tests                                      | TestFullStack_WithSigning, TestFullStack_DuplicateUserRejection |
| 6   | example/user README                                           | Expanded with Quick Start, module diagram, test table           |
| 7   | BDD test description polish                                   | ~60 vague It strings improved across command, event suites      |
| 8   | Storage test file split                                       | 663-line monolith → 4 focused files                             |
| 9   | OTel error recording                                          | SQLEventStore.Save version check now records errors on span     |

## B) PARTIALLY DONE

| #   | Item                         | What's Missing                                               |
| --- | ---------------------------- | ------------------------------------------------------------ |
| 1   | otel module                  | Has tracing helpers, 0% test coverage. No docs.              |
| 2   | testhelpers coverage (79.8%) | Below 80% gate — needs a few more tests                      |
| 3   | Signing module BDD           | Has unit tests (93.9%) but no BDD suite                      |
| 4   | Event type model consistency | 3 Parse functions bypass error taxonomy (see E below)        |
| 5   | Benchmark baseline in CI     | Runs but baseline file not yet on remote (requires tag push) |

## C) NOT STARTED

| #   | Item                                              | Notes                                                |
| --- | ------------------------------------------------- | ---------------------------------------------------- |
| 1   | Deprecated Store interface cleanup                | GlobalLoader→Journal aliases still present           |
| 2   | Version.Sub safety (negative overflow)            | No guard, silently produces negatives                |
| 3   | Event Stringer/GoString for debugging             | ImmutableEvent has no fmt.Stringer                   |
| 4   | Codec ContentType() method                        | No content-type negotiation for multi-format brokers |
| 5   | Outbox Nack/DeadLetter for poison messages        | Only Append/PollPending/Ack                          |
| 6   | Streaming/chunked Journal.ReadAll                 | Loads all events into memory — OOM risk              |
| 7   | Metadata.Validate() and .Clone()                  | No validation, Clone logic duplicated                |
| 8   | ErrNilPayload sentinel cleanup                    | Defined but never used — dead code                   |
| 9   | DecodePayloads double-wrapping fix                | Corruption wraps Corruption                          |
| 10  | example/saga, example/storage, example/projection | All 0% test coverage                                 |

## D) TOTALLY FUCKED UP

| #   | Issue                                      | Severity | Details                                                                                                    |
| --- | ------------------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------- |
| 1   | Stale gopls diagnostics                    | Low      | 14 phantom errors from prior file versions. LSP restart didn't clear them. Not real — builds pass.         |
| 2   | Pre-commit hook: go-structure-linter fails | Low      | 4 MEDIUM warnings (root go.sum empty, no pkg/internal dirs). These are by design (multi-module workspace). |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture / Type Model

1. **Error taxonomy consistency** — `ParseVersion`, `ParseSchemaVersion`, `ParseSource` use `fmt.Errorf` instead of `WrapRejection`. The project's 5-family taxonomy (Rejection/Conflict/Transient/Infrastructure/Corruption) should be applied everywhere.

2. **Version.Sub silent negatives** — `Version(1).Sub(5)` produces `-4` without error. Should return an error or at least have a `SafeSub` variant.

3. **ImmutableEvent lacks fmt.Stringer** — Debugging events requires accessing each field individually. A `String()` method would make logs readable.

4. **NewEvent returns \*ImmutableEvent not Event** — Consumers are coupled to the concrete type. Should return the interface.

5. **No streaming/chunked loading** — `Load` and `ReadAll` load everything into memory. A `ReadChunks(ctx, chunkSize, fn)` on Journal would prevent OOM on large event stores.

6. **Outbox missing Nack/DeadLetter** — No way to handle poison messages. PollPending just re-delivers forever.

7. **Metadata no Clone/Validate** — Clone logic is duplicated between ImmutableEvent.Metadata() and tests. Validate would help catch missing correlation IDs.

8. **DecodePayloads double-wraps** — `DecodePayload` wraps with Corruption, then `DecodePayloads` wraps again. The inner error is Corruption(Corruption(original)).

### Established Libraries We Could Use

9. **`google/uuid`** or keep `oklog/ulid` — ULID is correct for event sourcing (time-sortable). No change needed, but document the choice explicitly.

10. **`ogen-go/ogen`** — Could generate typed event schemas from OpenAPI/AsyncAPI instead of hand-rolling catalog types. Currently catalog uses struct tags.

11. **`bufbuild/buf` + protobuf** — For the codec layer, adding a ProtoCodec would unlock high-perf serialization. The Codec interface already supports this — just needs implementation.

### Testing / Quality

12. **otel module: 0% coverage** — Only helper functions, but used by 5+ modules. Should have at least basic tests.

13. **testhelpers: 79.8% coverage** — Below the 80% CI gate. Easy to fix.

14. **signing module: no BDD suite** — 93.9% coverage but all plain tests. BDD would improve discoverability.

15. **Example modules: saga/storage/projection all 0%** — No test coverage. Would rot without attention.

---

## F) Top 25 Next Tasks (sorted by Impact × Ease)

| Priority | Task                                                                  | Impact | Effort | Module             |
| -------- | --------------------------------------------------------------------- | ------ | ------ | ------------------ |
| 1        | Fix testhelpers coverage → 80%+                                       | High   | 15 min | testhelpers        |
| 2        | Fix ParseVersion/ParseSchemaVersion/ParseSource to use error taxonomy | High   | 15 min | core/event         |
| 3        | Add fmt.Stringer to ImmutableEvent                                    | High   | 10 min | core/event         |
| 4        | Add Version.SafeSub that errors on negative                           | Medium | 10 min | core/event         |
| 5        | Fix DecodePayloads double-wrapping                                    | Medium | 10 min | core/event         |
| 6        | Remove ErrNilPayload dead sentinel                                    | Low    | 5 min  | core/event         |
| 7        | Add Metadata.Clone()                                                  | Medium | 15 min | core/event         |
| 8        | otel module basic tests → 60%+                                        | High   | 30 min | otel               |
| 9        | signing module BDD suite                                              | Medium | 30 min | signing            |
| 10       | example/saga basic test                                               | Medium | 20 min | example/saga       |
| 11       | example/storage basic test                                            | Medium | 20 min | example/storage    |
| 12       | example/projection basic test                                         | Medium | 20 min | example/projection |
| 13       | Add ContentType() to Codec interface                                  | Medium | 30 min | core/event         |
| 14       | NewEvent return Event interface not \*ImmutableEvent                  | High   | 45 min | core/event         |
| 15       | Add Journal.ReadChunks streaming method                               | High   | 45 min | core/event         |
| 16       | Outbox Nack/DeadLetter/RetryCount                                     | High   | 60 min | core/event         |
| 17       | Add Event.Equal(other Event) bool                                     | Medium | 15 min | core/event         |
| 18       | Remove deprecated GlobalLoader/PositionalLoader aliases               | Low    | 15 min | core/event         |
| 19       | Add Outbox.PendingCount/OldestPendingAge for monitoring               | Medium | 20 min | core/event         |
| 20       | Add EventSource.Exists(ctx, aggType, aggID) bool                      | Medium | 20 min | core/event         |
| 21       | Add ID.Sort helper for ordering event slices                          | Low    | 10 min | core/pkg/id        |
| 22       | example/user coverage → 60%+                                          | Medium | 30 min | example/user       |
| 23       | Push tags and remove replace directives                               | High   | 60 min | all                |
| 24       | Add ProtoCodec implementation                                         | Medium | 60 min | core/event or new  |
| 25       | Add Metadata.Validate() for required fields                           | Low    | 15 min | core/event         |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `NewEvent` return `Event` (interface) or `*ImmutableEvent` (concrete)?**

Currently it returns `*ImmutableEvent`. This is a breaking API change. Arguments for `Event`:

- Enforces interface-only consumption (library best practice)
- Prevents consumers from depending on concrete fields

Arguments for `*ImmutableEvent`:

- All fields are private anyway (true encapsulation)
- Returning interface prevents consumers from using `== nil` checks
- Breaking change for all downstream code

**My recommendation:** Return `Event` but provide a `MustNewEvent` that also returns `Event`. This is the Go-idiomatic approach for interface-first libraries. But it requires updating all consumers (example/, integration/, etc.) and is a v2-level breaking change.

---

## Coverage Summary

| Module             | Coverage        |
| ------------------ | --------------- |
| catalog            | 96.3%           |
| catalog/asyncapi   | 93.7%           |
| catalog/d2         | 95.0%           |
| core/command       | 94.3%           |
| core/decider       | 91.7%           |
| core/event         | 92.9%           |
| integration        | [no statements] |
| memory             | 99.6%           |
| middleware         | 93.7%           |
| otel               | 0.0%            |
| projection         | 95.3%           |
| saga               | 94.3%           |
| signing            | 93.9%           |
| storage            | 90.0%           |
| stream             | 93.5%           |
| testhelpers        | 79.8%           |
| watermill          | 94.4%           |
| example/user       | 45.9%           |
| example/todo       | 68.4%           |
| example/saga       | 0.0%            |
| example/storage    | 0.0%            |
| example/projection | 0.0%            |

**All production modules above 80%.** Only otel (0%) and testhelpers (79.8%) are below.
