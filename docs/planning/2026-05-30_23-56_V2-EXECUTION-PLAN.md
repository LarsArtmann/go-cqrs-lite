# V2.0.0 Execution Plan — GET SHIT DONE

**Date:** 2026-05-30_23-56
**Total Issues:** 65 (12 P0 + 14 P1 + 13 P2 + 9 P3 + 7 P4 + 13 P5 — some P5 deferred to post-v2)
**Scope:** P0 + P1 + P2 + P4 fixes, selected P3/P5 if time permits

---

## Pareto Breakdown

### 1% → 51% Impact: Fix all P0 correctness bugs (12 items)

These undermine consumer trust. A library with data races, XSS, and immutability violations is not v2.0.0 material.

### 4% → 64% Impact: + P1 type safety & dead code (14 items)

Error taxonomy consistency, nil-check guards, file-size compliance, dead code removal.

### 20% → 80% Impact: + P2 duplication & naming (13 items)

Extract shared helpers, fix naming inconsistencies, reduce clone count by ~30 groups.

---

## Task Plan (20 tasks, 30-100 min each)

| #   | Task                                                                                | Priority | Est.  | Impact   | Modules                    |
| --- | ----------------------------------------------------------------------------------- | -------- | ----- | -------- | -------------------------- |
| T1  | Fix event.New() []byte immutability bug                                             | P0       | 15min | CRITICAL | event                      |
| T2  | Fix CatalogDispatcher data race (add mutex)                                         | P0       | 15min | CRITICAL | dispatcher                 |
| T3  | Fix XSS in catalog docserver html.go                                                | P0       | 15min | CRITICAL | catalog                    |
| T4  | Fix watermill double-close + MustParse panics                                       | P0       | 20min | CRITICAL | watermill                  |
| T5  | Fix memory checkpoint missing closed-check                                          | P0       | 10min | HIGH     | memory                     |
| T6  | Fix signing middleware panics (nil signer/verifier)                                 | P0       | 15min | HIGH     | signing                    |
| T7  | Fix cmd/api-stability wrong module paths                                            | P0       | 15min | MEDIUM   | cmd                        |
| T8  | Fix cmd/cqrs-gen query handler signature                                            | P0       | 20min | MEDIUM   | cmd                        |
| T9  | Fix example issues (storage go.mod, projection event type)                          | P0       | 10min | LOW      | examples                   |
| T10 | Split event.go, runner.go, pebble/store.go under 250 lines                          | P1       | 45min | HIGH     | event, projection, pebble  |
| T11 | Fix middleware error handling (validation, circuit breaker, OTel metrics)           | P1       | 30min | HIGH     | middleware                 |
| T12 | Fix nil-check panics (pebble logger, schema, decider)                               | P1       | 20min | MEDIUM   | pebble, schema, decider    |
| T13 | Remove dead code (otel, middleware constants) + fix memory bus double-wrap          | P1       | 15min | MEDIUM   | otel, middleware, memory   |
| T14 | Extract shared helpers (tombstone, signing extractOrPassThrough)                    | P2       | 20min | MEDIUM   | event, signing             |
| T15 | Parameterize middleware recovery + dispatch closed-check boilerplate                | P2       | 30min | MEDIUM   | middleware, command, query |
| T16 | Fix naming inconsistencies (ErrQueryNotSupported, ErrTypeAssertion, ParseUserAgent) | P2       | 15min | LOW      | query, command, event      |
| T17 | Add doc comments to id/command_id.go                                                | P2       | 5min  | LOW      | id                         |
| T18 | Fix example/user projection missing handlers + dead code cleanup                    | P4       | 20min | LOW      | examples                   |
| T19 | Fix example/todo stale README + dead CommandTypeError                               | P4       | 15min | LOW      | examples                   |
| T20 | Build + lint + test verification                                                    | P0       | 30min | CRITICAL | all                        |

**Total estimated time: ~8.5 hours**

---

## Micro-Task Breakdown (75 tasks, max 15 min each)

### Batch A: P0 — Critical Safety (T1-T9)

| #   | Micro-Task                                                          | File(s)                        | Est.  |
| --- | ------------------------------------------------------------------- | ------------------------------ | ----- |
| M1  | Clone []byte in event_new.go:66 case []byte                         | event/event_new.go             | 5min  |
| M2  | Clone json.RawMessage in event_new.go:68 case                       | event/event_new.go             | 5min  |
| M3  | Add mutex to CatalogDispatcher.RegisterHandlerMeta + CatalogEntries | dispatcher/dispatcher.go       | 15min |
| M4  | Add html/template escaping to scalarHTML in docserver               | catalog/docserver/html.go      | 10min |
| M5  | Add html/template escaping to asyncAPIHTML in docserver             | catalog/docserver/html.go      | 5min  |
| M6  | Add sync.Once to SubscriberAdapter.Close                            | watermill/subscriber.go        | 10min |
| M7  | Replace MustParse* with Parse* in buildMetadata                     | watermill/protocol.go          | 15min |
| M8  | Add CheckClosed to MemoryCheckpointStore.Load                       | memory/checkpoint.go           | 5min  |
| M9  | Add CheckClosed to MemoryCheckpointStore.Save                       | memory/checkpoint.go           | 5min  |
| M10 | Replace panic with error-return in SignMiddleware(nil)              | signing/middleware.go          | 5min  |
| M11 | Replace panic with error-return in VerifyMiddleware(nil)            | signing/middleware.go          | 5min  |
| M12 | Replace panic with error-return in RequireSignatureMiddleware(nil)  | signing/middleware.go          | 5min  |
| M13 | Replace panic with error-return in MultiSignMiddleware(nil)         | signing/multisig/middleware.go | 5min  |
| M14 | Replace panic with error-return in MultiVerifyMiddleware(nil)       | signing/multisig/middleware.go | 5min  |
| M15 | Replace panic with error-return in RequireMultiSig(nil)             | signing/multisig/middleware.go | 5min  |
| M16 | Fix module paths in cmd/api-stability (remove core/ prefix)         | cmd/api-stability/main.go      | 10min |
| M17 | Add go.mod for cmd/api-stability if missing                         | cmd/api-stability/             | 5min  |
| M18 | Fix cqrs-gen query handler to return (R, error) not error           | cmd/cqrs-gen/main.go           | 15min |
| M19 | Add replace directive for listing in example/storage/go.mod         | example/storage/go.mod         | 5min  |
| M20 | Fix ItemAdded → ItemRemoved in example/projection/main.go:137       | example/projection/main.go     | 5min  |

### Batch B: P1 — File Splits (T10)

| #   | Micro-Task                                                | File(s)     | Est.  |
| --- | --------------------------------------------------------- | ----------- | ----- |
| M21 | Extract NewEvent+Clone from event.go → event_construct.go | event/      | 15min |
| M22 | Extract replay logic from runner.go → runner_replay.go    | projection/ | 15min |
| M23 | Extract iteration from pebble/store.go → iteration.go     | pebble/     | 15min |

### Batch C: P1 — Error Handling (T11-T12)

| #   | Micro-Task                                                            | File(s)                       | Est.  |
| --- | --------------------------------------------------------------------- | ----------------------------- | ----- |
| M24 | Fix validation.go: preserve original error cause                      | middleware/validation.go      | 10min |
| M25 | Fix circuit_breaker.go: use event.WrapTransient instead of fmt.Errorf | middleware/circuit_breaker.go | 10min |
| M26 | Fix OTelMetricsRecorder.Observe: accept context parameter             | middleware/metrics_otel.go    | 15min |
| M27 | Fix pebble/helpers.go: nil-logger guard in logEventOperation          | pebble/helpers.go             | 10min |
| M28 | Add nil check to NewVersionedStore                                    | schema/versioned_source.go    | 5min  |
| M29 | Add nil check to NewUpcaster                                          | schema/upcaster.go            | 5min  |
| M30 | Log/trace snapshot errors in decider instead of discarding            | decider/decider.go            | 10min |
| M31 | Add validation for snapshot store+codec pair in NewRepository         | decider/decider.go            | 10min |

### Batch D: P1 — Dead Code + Memory Double Wrap (T13)

| #   | Micro-Task                                                           | File(s)                    | Est.  |
| --- | -------------------------------------------------------------------- | -------------------------- | ----- |
| M32 | Remove SpanFromContext from otel/spans.go                            | otel/spans.go              | 5min  |
| M33 | Remove ComponentTracer from otel/spans.go                            | otel/spans.go              | 5min  |
| M34 | Remove unused metricName\* constants from middleware/metrics_otel.go | middleware/metrics_otel.go | 5min  |
| M35 | Remove double error wrapping in memory/bus.go Publish                | memory/bus.go              | 10min |

### Batch E: P2 — Duplication (T14-T15)

| #   | Micro-Task                                                                  | File(s)                | Est.  |
| --- | --------------------------------------------------------------------------- | ---------------------- | ----- |
| M36 | Extract copyAndAnnotate helper in tombstone.go                              | event/tombstone.go     | 10min |
| M37 | Extract extractOrPassThrough to signing internal                            | signing/               | 10min |
| M38 | Refactor multisig/middleware.go to use shared extractOrPassThrough          | signing/multisig/      | 5min  |
| M39 | Parameterize recovery: extract generic Recovery(domain string)              | middleware/recovery.go | 15min |
| M40 | Parameterize dispatch closed-check: extract helper on DispatcherWithCatalog | command, query         | 15min |

### Batch F: P2 — Naming (T16-T17)

| #   | Micro-Task                                                       | File(s)           | Est. |
| --- | ---------------------------------------------------------------- | ----------------- | ---- |
| M41 | Rename ErrQueryNotSupported → ErrHandlerNotFound for consistency | query/errors.go   | 5min |
| M42 | Change ErrTypeAssertion from Corruption to Rejection             | command/errors.go | 5min |
| M43 | Rename ParseUserAgent → SanitizeUserAgent                        | event/types.go    | 5min |
| M44 | Add doc comments to id/command_id.go                             | id/command_id.go  | 5min |

### Batch G: P4 — Example Fixes (T18-T19)

| #   | Micro-Task                                            | File(s)                        | Est.  |
| --- | ----------------------------------------------------- | ------------------------------ | ----- |
| M45 | Add UserDeleted handler to example/user projection    | example/user/projection.go     | 10min |
| M46 | Add UserReborn handler to example/user projection     | example/user/projection.go     | 5min  |
| M47 | Remove dead CommandTypeError from example/todo        | example/todo/commands/mixin.go | 5min  |
| M48 | Fix example/todo README stale references              | example/todo/README.md         | 10min |
| M49 | Remove dead errUnexpectedQueryType from example/user  | example/user/handlers.go       | 5min  |
| M50 | Fix example/user catalog.go: use command payload type | example/user/catalog.go        | 10min |

### Batch H: Verification (T20)

| #   | Micro-Task                               | File(s) | Est.  |
| --- | ---------------------------------------- | ------- | ----- |
| M51 | Run full build                           | all     | 5min  |
| M52 | Run full lint                            | all     | 5min  |
| M53 | Run full test suite                      | all     | 10min |
| M54 | Run dupl to verify clone count reduction | all     | 5min  |
| M55 | Final git commit + verify clean state    | all     | 5min  |

---

## Execution Graph (Mermaid)

```mermaid
graph TD
    subgraph "P0 — Critical Safety"
        M1[M1: Clone []byte in event.New]
        M3[M3: Fix data race in CatalogDispatcher]
        M4[M4: Fix XSS in docserver]
        M6[M6: Fix watermill double-close]
        M7[M7: Fix MustParse panics]
        M8[M8: Fix checkpoint closed-check]
        M10[M10-15: Fix signing panics]
        M16[M16: Fix api-stability paths]
        M18[M18: Fix cqrs-gen query sig]
        M19[M19: Fix storage go.mod]
        M20[M20: Fix projection event type]
    end

    subgraph "P1 — File Splits"
        M21[M21: Split event.go]
        M22[M22: Split runner.go]
        M23[M23: Split pebble/store.go]
    end

    subgraph "P1 — Error Handling"
        M24[M24: Fix validation error]
        M25[M25: Fix circuit breaker errors]
        M26[M26: Fix OTel Observe ctx]
        M27[M27: Fix pebble nil logger]
        M28[M28: Fix schema nil checks]
        M30[M30: Log decider snapshot errors]
        M31[M31: Validate snapshot config]
    end

    subgraph "P1 — Dead Code"
        M32[M32: Remove dead otel funcs]
        M34[M34: Remove dead constants]
        M35[M35: Fix memory double wrap]
    end

    subgraph "P2 — Duplication"
        M36[M36: Extract tombstone helper]
        M37[M37: Share signing extract]
        M39[M39: Parameterize recovery]
        M40[M40: Extract dispatch helper]
    end

    subgraph "P2 — Naming"
        M41[M41: Fix query error name]
        M42[M42: Fix TypeAssertion family]
        M43[M43: Rename ParseUserAgent]
    end

    subgraph "P4 — Examples"
        M45[M45: Fix user projection]
        M47[M47: Remove dead code]
        M48[M48: Fix README]
    end

    subgraph "Verification"
        V1[Build + Lint + Test]
        V2[Final Commit]
    end

    M1 --> M21
    M3 --> M40
    M10 --> M37
    M21 --> M24
    M22 --> M36
    M24 --> M25 --> M26
    M32 --> M35
    M37 --> M39
    M41 --> M42 --> M43

    M20 --> V1
    M35 --> V1
    M43 --> V1
    M48 --> V1
    V1 --> V2

    style M1 fill:#ff4444,color:#fff
    style M3 fill:#ff4444,color:#fff
    style M4 fill:#ff4444,color:#fff
    style M6 fill:#ff4444,color:#fff
    style M10 fill:#ff4444,color:#fff
    style V1 fill:#22c55e,color:#fff
    style V2 fill:#22c55e,color:#fff
```

---

## Rules

1. **DO NOT BREAK BUILD** — run `nix run .#build` after every batch
2. **VERIFY** — run tests after each module's changes
3. **COMMIT** — commit after each batch with detailed message
4. **PARALLEL** — where possible, fix multiple files in same batch
