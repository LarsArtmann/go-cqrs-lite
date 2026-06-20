# Code Quality Scan & Full Code Review — 2026-06-01

## Executive Summary

| Metric                           | Result                |
| -------------------------------- | --------------------- |
| Build                            | ✅ Pass               |
| Lint (all 22 modules)            | ✅ 0 issues           |
| Tests (all packages)             | ✅ All pass           |
| Vet (workspace mode)             | ✅ 0 issues           |
| Format (treefmt)                 | ✅ Pass               |
| Duplication (dupl, threshold 50) | 27 clone groups found |
| Total production LOC             | 20,133                |
| Total test LOC                   | 36,171                |

---

## Pareto Analysis: Top Issues by Impact

### 🔴 1% → 51% Impact (Fix These First)

| #   | Issue                                                                          | Module                   | Impact                             | Effort |
| --- | ------------------------------------------------------------------------------ | ------------------------ | ---------------------------------- | ------ |
| 1   | **Middleware 3x duplication** (~500 lines)                                     | middleware               | Eliminate 500 lines of copy-paste  | 2h     |
| 2   | **Three `ErrHandlerNotFound` / `ErrDispatcherClosed` sentinels**               | dispatcher/command/query | Cross-module `errors.Is` is broken | 30min  |
| 3   | **`VersionedStore` exposes embedded `event.Store`** — callers bypass upcasting | schema                   | Data corruption risk               | 15min  |
| 4   | **`catalog/schema/reflect.go:ToAny` silently swallows errors**                 | catalog                  | Silent data loss                   | 15min  |
| 5   | **`watermill/protocol.go:messageToEvent` is 81 lines**                         | watermill                | Maintenance burden                 | 30min  |
| 6   | **Circuit breaker double-wraps errors**                                        | middleware               | Polluted error chains              | 15min  |

### 🟠 4% → 64% Impact

| #   | Issue                                                            | Module         | Impact                      | Effort |
| --- | ---------------------------------------------------------------- | -------------- | --------------------------- | ------ |
| 7   | **`command.Metadata` duplicates `event.Metadata`** (split brain) | command        | Maintenance drift           | 1h     |
| 8   | **`command.aggregate_ref.go` re-exports `event` types**          | command        | Module boundary violation   | 30min  |
| 9   | **Storage backend error duplication** (pebble vs sql errors)     | storage/pebble | Inconsistent error checking | 1h     |
| 10  | **`decider/load.go:opError` produces unclassified errors**       | decider        | Breaks error taxonomy       | 30min  |
| 11  | **`catalog/d2` `sanitizeID(GetID(...))` repeated 6×**            | catalog        | DRY violation               | 15min  |
| 12  | **Three `SchemaToAny` wrappers** across catalog packages         | catalog        | Split brain                 | 15min  |
| 13  | **`event/batch.go` duplicates marshal logic from `New`**         | event          | DRY violation               | 30min  |

### 🟡 20% → 80% Impact

| #   | Issue                                                                                 | Module        | Impact                  | Effort |
| --- | ------------------------------------------------------------------------------------- | ------------- | ----------------------- | ------ |
| 14  | **`command.Type` and `query.Type` bare strings** — no `Parse()`                       | command/query | Type safety             | 1h     |
| 15  | **`signing/HasSignature` swallows corruption errors**                                 | signing       | Silent failures         | 30min  |
| 16  | **`middleware/ErrCircuitBreakerOpen` bypasses error taxonomy**                        | middleware    | Inconsistent errors     | 15min  |
| 17  | **`watermill/protocol` silently drops malformed IDs**                                 | watermill     | Data loss               | 30min  |
| 18  | **`pebble/config.go:Backend` type is unused at runtime**                              | pebble        | Dead API surface        | 15min  |
| 19  | **`catalog/GetID` returns Name as fallback** (dishonest)                              | catalog       | Surprising behavior     | 15min  |
| 20  | **`event/reactive.go:FilterEventTypes` duplicates `newTypeSet`**                      | event         | DRY violation           | 10min  |
| 21  | **`event/Version.Sub` can produce negative versions**                                 | event         | Silent corruption       | 15min  |
| 22  | **`query/TypedHandler[T]` takes `Query` not `T`** — less safe than command equivalent | query         | Type safety gap         | 1h     |
| 23  | **`storage/sql_aggregate_reader` hardcodes `?` placeholders** (SQLite only)           | storage       | PostgreSQL incompatible | 1h     |
| 24  | **`projection/runner.go:replay` is 64 lines**                                         | projection    | Complexity              | 30min  |
| 25  | **`catalog/NewTestCreateOrderFlow` in production code**                               | catalog       | Test code in prod       | 10min  |

---

## Duplication Analysis (dupl -t 50)

27 clone groups found across production code.

**Top files by clone count:**

| File                                   | Clone Count | Root Cause                              |
| -------------------------------------- | ----------- | --------------------------------------- |
| `catalog/internal/cattest/builders.go` | 6           | Test builder boilerplate                |
| `middleware/metrics_otel.go`           | 5           | 3x command/event/query pattern          |
| `catalog/registry.go`                  | 4           | AddCommand/AddEvent/AddQuery repetition |
| `middleware/circuit_breaker.go`        | 3           | 3x command/event/query pattern          |
| `memory/store_load.go`                 | 3           | Load variants share structure           |
| `id/*.go` (7 files)                    | 7           | ID type boilerplate (unavoidable in Go) |

---

## Module-by-Module Findings

### event/ — 30 issues (0 critical, 7 medium, 21 low, 2 nit)

- Strong domain modeling, clean ISP, excellent error taxonomy
- Key: `batch.go` duplicates marshal logic, `Version.Sub` can go negative, `FilterEventTypes` DRY violation

### command/query/dispatcher/decider/id/ — 15 issues (4 high, 7 medium, 4 low)

- Key: `command.Metadata` split brain with `event.Metadata`, three `ErrHandlerNotFound` sentinels break cross-module `errors.Is`, `decider/load.go` unclassified errors

### schema/snapshot/codec/signing/otel/ — 36 issues (4 high, 17 medium, 15 low)

- Key: `VersionedStore` exposes embedded Store (bypass upcasting), `ToAny` swallows errors, signing middleware duplication

### middleware/ — 11 issues (2 critical, 4 moderate, 5 minor)

- Key: **~500 lines of 3x copy-paste** across command/event/query, circuit breaker double-wraps, `ErrCircuitBreakerOpen` bypasses taxonomy

### memory/storage/pebble/turso/ — 22 issues (0 critical, multiple medium)

- Key: Storage backend error duplication, `sql_aggregate_reader` SQLite-only, `aggregate_projection` uses `context.Background()`

### catalog/projection/listing/watermill/integration/cmd/ — 20 issues (5 high, 10 medium, 5 low)

- Key: `messageToEvent` 81 lines, three `SchemaToAny` wrappers, silently dropped IDs in watermill, `GetID` dishonest name

---

## Detailed Issue Registry

### Type Safety

| ID    | Severity | Location                         | Issue                                                        |
| ----- | -------- | -------------------------------- | ------------------------------------------------------------ |
| TS-01 | Medium   | `event/batch.go:17`              | `payloads []any` — generic `NewEvents[T]` could eliminate    |
| TS-02 | Medium   | `event/event_new.go:23`          | `payload any` — could narrow to constrained interface        |
| TS-03 | Low      | `event/replay.go:12`             | `bool` replay mode should be enum                            |
| TS-04 | Low      | `event/types.go:136`             | `Version.Sub` can produce negative versions                  |
| TS-05 | Low      | `event/tombstone.go:67`          | Stringly-typed boolean `"true"` comparison                   |
| TS-06 | Medium   | `command/command.go:12`          | `Type` is bare string, no `Parse()`                          |
| TS-07 | Medium   | `query/query.go:9`               | `Type` is bare string, no `Parse()`                          |
| TS-08 | Medium   | `codec/codec.go:14-15`           | `Codec` interface uses `any` for Encode/Decode               |
| TS-09 | High     | `codec/raw.go:6,13`              | Comment claims `json.RawMessage` support but it doesn't work |
| TS-10 | High     | `schema/versioned_source.go:12`  | Embedded `event.Store` allows bypassing upcasting            |
| TS-11 | Medium   | `signing/multisig/types.go:74`   | `MultiSignature` entries can be constructed invalid          |
| TS-12 | Medium   | `catalog/openapi/types.go:71-86` | Four `any` fields in schema types                            |
| TS-13 | Medium   | `pebble/config.go:10`            | `Backend` string type allows arbitrary values                |
| TS-14 | Medium   | `storage/sql/dialect.go:18-19`   | `FormatTime`/`ParseTime` use `any` with invisible coupling   |

### Error Handling

| ID    | Severity | Location                               | Issue                                                       |
| ----- | -------- | -------------------------------------- | ----------------------------------------------------------- |
| EH-01 | Medium   | `middleware/circuit_breaker.go:222`    | Double-wrapped error (allow + execute both wrap)            |
| EH-02 | Medium   | `middleware/circuit_breaker.go:243`    | `ErrCircuitBreakerOpen` uses bare `errors.New` not taxonomy |
| EH-03 | Medium   | `middleware/multisig/middleware.go:71` | Uses `fmt.Errorf` instead of `event.WrapInfrastructure`     |
| EH-04 | High     | `catalog/schema/reflect.go:44-57`      | `ToAny` silently swallows marshal errors                    |
| EH-05 | Medium   | `signing/event.go:88`                  | `HasSignature` swallows corruption errors                   |
| EH-06 | Medium   | `signing/multisig/extract.go:99`       | `HasMultiSignature` same issue                              |
| EH-07 | Medium   | `watermill/protocol.go:162-205`        | Silently drops malformed ID parse errors                    |
| EH-08 | High     | `decider/load.go:56-64`                | `opError` produces unclassified `fmt.Errorf` errors         |
| EH-09 | Medium   | `pebble/store.go:85-109`               | Double-wrapping in Save                                     |
| EH-10 | Medium   | `pebble/serialization.go:40`           | MarshalMetadataJSON error silently discarded                |
| EH-11 | Medium   | `storage/checkpoint.go:52-56`          | Uses `fmt.Errorf` instead of structured wrapping            |
| EH-12 | Medium   | `query/query.go:33`                    | Returns bare sentinel without context                       |

### Split Brains

| ID    | Severity | Location                                                    | Issue                                                     |
| ----- | -------- | ----------------------------------------------------------- | --------------------------------------------------------- |
| SB-01 | High     | `command/metadata.go:9`                                     | `command.Metadata` duplicates `event.Metadata` structure  |
| SB-02 | High     | `command/aggregate_ref.go:10-34`                            | Re-exports `event` types — module boundary violation      |
| SB-03 | High     | `dispatcher/errors.go` × 3                                  | Three separate `ErrHandlerNotFound` sentinels             |
| SB-04 | High     | `dispatcher/errors.go` × 3                                  | Three separate `ErrDispatcherClosed` sentinels            |
| SB-05 | Medium   | `pebble/errors.go:17` vs `storage/sql/errors.go:12`         | Duplicate error types across backends                     |
| SB-06 | Medium   | `catalog/schema.go` + asyncapi + openapi                    | Three `SchemaToAny` wrappers                              |
| SB-07 | Medium   | `signing/middleware.go` vs `signing/multisig/middleware.go` | Error code string literals duplicated                     |
| SB-08 | Medium   | `signing/event.go:39` vs `multisig/extract.go:105`          | `AttachSignature`/`attachMultiSignature` divergent clones |
| SB-09 | Low      | `event/event_construct.go:12` vs `event/event.go:156`       | Inconsistent clone style (`make+copy` vs `slices.Clone`)  |

### Code Duplication

| ID    | Severity | Location                                               | Issue                                                         |
| ----- | -------- | ------------------------------------------------------ | ------------------------------------------------------------- |
| CD-01 | Critical | `middleware/*.go` (10 files)                           | ~500 lines of 3x command/event/query duplication              |
| CD-02 | Medium   | `command/dispatcher.go:87` & `query/dispatcher.go:111` | `checkClosed` copy-pasted                                     |
| CD-03 | Medium   | `event/batch.go:40` vs `event/event_new.go:18`         | Marshal+create pattern duplicated                             |
| CD-04 | Medium   | `event/reactive.go:37`                                 | `FilterEventTypes` duplicates `newTypeSet`                    |
| CD-05 | Medium   | `schema/versioned_source.go:33-87`                     | 4 near-identical load methods                                 |
| CD-06 | Medium   | `signing/hmac.go:59` vs `signing/ed25519.go:78`        | Shared nil-check guard duplicated                             |
| CD-07 | Medium   | `signing/middleware.go` vs `multisig/middleware.go`    | Sign middleware near-identical loops                          |
| CD-08 | Medium   | `catalog/validate.go`                                  | `validateDomain`/`validateChannel` duplicate seen-map pattern |
| CD-09 | Medium   | `catalog/eventcatalog/exporter.go:28-91`               | 8 copy-paste entity loops                                     |
| CD-10 | Medium   | `catalog/d2/*.go`                                      | `sanitizeID(GetID(...))` repeated 6×                          |
| CD-11 | Low      | `storage/sqlite_helpers.go:56-57`                      | `ConfigureSQLitePool`/`ConfigureTursoPool` identical          |
| CD-12 | Low      | `catalog/eventcatalog/writer_frontmatter.go:41,63`     | `addObjectIDsListField`/`writeIDListField` are clones         |

### Naming Issues

| ID    | Severity | Location                          | Issue                                         |
| ----- | -------- | --------------------------------- | --------------------------------------------- |
| NM-01 | High     | `otel/logging.go:16`              | `TraceIDLogger` name/doc don't match behavior |
| NM-02 | Medium   | `catalog/types.go:153`            | `GetID` returns Name as fallback — dishonest  |
| NM-03 | Medium   | `signing/multisig/signer.go:134`  | `VerifyActor` should be standalone function   |
| NM-04 | Low      | `middleware/common.go`            | "common" is a vague dumping ground name       |
| NM-05 | Medium   | `catalog/registry_helpers.go:138` | `NewTestCreateOrderFlow` in production code   |

### Function Size Violations (>30 lines)

| ID    | Location                                              | Function                    | Lines |
| ----- | ----------------------------------------------------- | --------------------------- | ----- |
| FS-01 | `watermill/protocol.go:79`                            | `messageToEvent`            | 81    |
| FS-02 | `projection/runner.go:119`                            | `replay`                    | 64    |
| FS-03 | `catalog/eventcatalog/exporter.go:28`                 | `Export`                    | 63    |
| FS-04 | `storage/sql_aggregate_reader.go:47`                  | `ListWithStatus`            | ~112  |
| FS-05 | `storage/event_store.go:64`                           | `Save`                      | ~52   |
| FS-06 | `signing/multisig/middleware.go:150`                  | `RequireMultiSigMiddleware` | 55    |
| FS-07 | `catalog/schema/reflect.go:121`                       | `fieldToProperty`           | 40    |
| FS-08 | `catalog/eventcatalog/exporter_resources_extra.go:36` | `writeFlowSteps`            | 62    |

### Dead Code / Unused

| ID    | Location                                        | Issue                                                    |
| ----- | ----------------------------------------------- | -------------------------------------------------------- |
| DC-01 | `pebble/errors.go:12-15`                        | `ErrUnknownBackend` declared but never returned          |
| DC-02 | `pebble/config.go:59-69`                        | `Backend` field ignored at runtime                       |
| DC-03 | `pebble/config.go:71-90`                        | 20 lines of backward-compat aliases                      |
| DC-04 | `signing/multisig/errors.go:8`                  | `ErrNoVerifier` defined but never used                   |
| DC-05 | `catalog/eventcatalog/writer_frontmatter.go:63` | `writeIDListField` is a clone of `addObjectIDsListField` |
| DC-06 | `listing/in_memory.go:124-147`                  | `TombstoneInclude` case is unreachable (dead code)       |
| DC-07 | `middleware/circuit_breaker.go:97-98`           | `return nil` after exhaustive switch (dead code)         |
| DC-08 | `signing/multisig/extract.go:27-33`             | `ErrNoVerifier` not used; inline error created instead   |

---

## D2 Execution Graph

```d2
title: Code Quality Fix Priority

1_percent: {
  shape: rectangle
  label: "1% → 51% Impact\n6 tasks, ~4h total"

  middleware_dedup: "Middleware 3x dedup (~500 lines)"
  sentinels: "Consolidate ErrHandlerNotFound/ErrDispatcherClosed"
  versioned_store: "Unexport VersionedStore.Store"
  toany_swallow: "Fix ToAny silent error swallowing"
  message_to_event: "Decompose messageToEvent (81→20 lines)"
  cb_double_wrap: "Fix circuit breaker double-wrap"
}

four_percent: {
  shape: rectangle
  label: "4% → 64% Impact\n7 tasks, ~4h total"

  metadata_split: "Fix command.Metadata split brain"
  aggregate_ref: "Remove command aggregate_ref re-exports"
  storage_errors: "Consolidate storage backend errors"
  decider_errors: "Fix decider opError classification"
  d2_sanitize: "Extract displayID helper in catalog/d2"
  schema_to_any: "Consolidate SchemaToAny wrappers"
  batch_dry: "Fix event batch.go marshal duplication"
}

twenty_percent: {
  shape: rectangle
  label: "20% → 80% Impact\n12 tasks, ~6h total"

  type_parse: "Add Parse() to command/query Type"
  signing_errors: "Fix signing silent error swallowing"
  cb_taxonomy: "Fix ErrCircuitBreakerOpen taxonomy"
  watermill_ids: "Fix watermill silently dropped IDs"
  pebble_backend: "Remove unused Backend type/aliases"
  get_id: "Fix GetID dishonest fallback"
  filter_types: "Fix FilterEventTypes DRY violation"
  version_sub: "Add validation to Version.Sub"
  typed_handler: "Fix query TypedHandler input type"
  sql_reader: "Fix sql_aggregate_reader SQLite-only"
  replay_func: "Decompose projection replay (64 lines)"
  test_in_prod: "Move NewTestCreateOrderFlow to test file"
}

1_percent -> four_percent: "then"
four_percent -> twenty_percent: "then"
```

---

## Module Quality Scores

| Module      | Build | Lint | Tests | Type Safety | Error Handling | ISP | Naming     | Duplication     | Overall |
| ----------- | ----- | ---- | ----- | ----------- | -------------- | --- | ---------- | --------------- | ------- |
| event       | ✅    | ✅   | ✅    | B+          | A              | A   | A-         | B+              | **A-**  |
| command     | ✅    | ✅   | ✅    | B           | A              | A   | B+         | B               | **B+**  |
| query       | ✅    | ✅   | ✅    | B           | B+             | A   | A          | B               | **B+**  |
| decider     | ✅    | ✅   | ✅    | A           | B              | A   | A          | A               | **A-**  |
| id          | ✅    | ✅   | ✅    | A           | A              | A   | A          | B (boilerplate) | **A**   |
| dispatcher  | ✅    | ✅   | ✅    | A-          | A              | A   | A          | A               | **A**   |
| schema      | ✅    | ✅   | ✅    | B           | A              | A   | A          | B               | **B+**  |
| snapshot    | ✅    | ✅   | ✅    | B+          | A              | A-  | A          | A               | **A-**  |
| codec       | ✅    | ✅   | ✅    | B           | A              | A   | A          | A               | **A-**  |
| signing     | ✅    | ✅   | ✅    | B+          | B              | A   | A-         | B               | **B+**  |
| otel        | ✅    | ✅   | ✅    | A           | A              | A   | B (naming) | A               | **A-**  |
| middleware  | ✅    | ✅   | ✅    | B+          | B              | A   | A-         | **D**           | **B**   |
| memory      | ✅    | ✅   | ✅    | A           | B+             | A   | A          | A               | **A-**  |
| storage     | ✅    | ✅   | ✅    | B+          | B+             | B+  | B+         | B               | **B+**  |
| pebble      | ✅    | ✅   | ✅    | B+          | B              | A   | B+         | B               | **B+**  |
| turso       | ✅    | ✅   | ✅    | A           | A              | A   | A          | A               | **A**   |
| catalog     | ✅    | ✅   | ✅    | B           | B+             | A   | B+         | B               | **B+**  |
| projection  | ✅    | ✅   | ✅    | A           | A              | A   | A          | A               | **A-**  |
| listing     | ✅    | ✅   | ✅    | A           | A              | A   | A          | A               | **A**   |
| watermill   | ✅    | ✅   | ✅    | A           | B              | A   | A          | B               | **B+**  |
| integration | ✅    | ✅   | ✅    | A           | A              | A   | A          | A               | **A**   |

**Overall project grade: A-** — High quality with targeted areas for improvement. Middleware duplication is the single biggest drag on quality.
