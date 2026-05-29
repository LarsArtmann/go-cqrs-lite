# Session 126 — Comprehensive Execution Plan

**Created:** 2026-05-29 03:39 CEST
**Test Status:** 29/29 packages pass, clean tree, all pushed to origin

---

## A. FULLY DONE (Shipped This Session)

| #   | Task                                                              | Commit           |
| --- | ----------------------------------------------------------------- | ---------------- |
| 1   | OTel tracing for storage Save/Load/Outbox/Checkpoint/Snapshot     | `2d6d421`        |
| 2   | OTel tracing for projection replay                                | `70af96a`        |
| 3   | OTel tracing for decider.Execute                                  | `424ce7d`        |
| 4   | CI benchmark job with regression detection                        | `424ce7d`        |
| 5   | Benchmark baseline file                                           | `424ce7d`        |
| 6   | BDD tests for Version, SchemaVersion, CheckVersionConflict        | `eeb356b`        |
| 7   | Stream cursor pagination test                                     | `eb239ee`        |
| 8   | TombstonePolicy.String + AggregateStatus JSON marshaling          | `64bafde`        |
| 9   | Export ParseSchemaVersion                                         | `b7e2071`        |
| 10  | Extract validateTablePrefix helper                                | `888cda3`        |
| 11  | Pebble Delete method removal                                      | `70af96a`        |
| 12  | example/user smoke test + README overhaul                         | `424ce7d`        |
| 13  | Sink/Source ADR (docs/adr/0006)                                   | `afc4c99`        |
| 14  | signing/v1.0.0 tag pushed                                         | `signing/v1.0.0` |
| 15  | Golden test protection (.prettierignore)                          | prior session    |
| 16  | All go.mod/go.sum tidy across 16 modules                          | `5d020be`        |
| 17  | FEATURES.md: added Stream + OTel sections                         | `424ce7d`        |
| 18  | AGENTS.md: Sink/Source split, tombstone policy, OTel module graph | `9792b5a`        |

---

## B. PARTIALLY DONE (Needs Completion)

| #   | Task                            | What's Done                                                          | What's Left                                                                                 |
| --- | ------------------------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| 1   | Storage sqlite test duplication | Helpers extracted to `sqlite_integration_helpers_test.go`            | File has 11 duplicate symbol declarations with `sqlite_integration_test.go` — need to merge |
| 2   | BDD test coverage expansion     | command, event, query, middleware, decider, saga, stream, projection | stream integration tests, storage BDD, saga persistent store tests                          |

---

## C. NOT STARTED — Sorted by Impact × Value ÷ Effort

Each task is scoped to ≤12 minutes.

### Legend

- **Impact**: 🔴 Critical / 🟡 High / 🔵 Medium / ⚪ Low
- **Effort**: S (≤5 min) / M (5-12 min) / L (>12 min, split further)
- **Value**: Consumer-facing library quality

| #                                  | Task                                                                                                               | Impact | Effort | Value                                                                                         |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------ | ------ | ------ | --------------------------------------------------------------------------------------------- |
| **ERROR QUALITY**                  |                                                                                                                    |        |        |
| 1                                  | Classify signing/event.go errors (2 `fmt.Errorf` → `event.WrapRejection`)                                          | 🟡     | S      | Signing errors lose family classification; breaks `IsRetryable` for consumers                 |
| 2                                  | Classify signing/signer.go errors (2 calls)                                                                        | 🟡     | S      | Same — signer construction errors unclassified                                                |
| 3                                  | Classify signing/hmac.go errors (1 call)                                                                           | 🟡     | S      | HMAC init error unclassified                                                                  |
| 4                                  | Classify signing/ed25519.go errors (3 calls)                                                                       | 🟡     | S      | Ed25519 key validation errors unclassified                                                    |
| 5                                  | Classify signing/middleware.go errors (8 calls)                                                                    | 🟡     | M      | Sign/verify middleware errors unclassified — consumers can't distinguish retryable from fatal |
| 6                                  | Classify signing/multisig.go errors (6 calls)                                                                      | 🟡     | M      | Multi-signer construction errors unclassified                                                 |
| 7                                  | Classify signing/multisig_extract.go errors (5 calls)                                                              | 🟡     | S      | Signature extraction errors unclassified                                                      |
| 8                                  | Classify signing/multisig_middleware.go errors (12 calls)                                                          | 🟡     | M      | Multi-sig middleware errors — largest file, highest count                                     |
| 9                                  | Classify stream/ errors (10 `fmt.Errorf` → `fmt.Errorf` with `%w` or classified)                                   | 🔵     | S      | stream doesn't import core/event — use `%w` wrapping of sentinels instead                     |
| 10                                 | Classify middleware/ errors (4 `fmt.Errorf` in validation.go, recovery.go)                                         | 🔵     | S      | Recovery errors should be classified as Transient                                             |
| **COMPILATION BLOCKERS**           |                                                                                                                    |        |        |
| 11                                 | Fix storage sqlite test duplication — merge `sqlite_integration_helpers_test.go` into `sqlite_integration_test.go` | 🔴     | M      | **11 duplicate symbols** — tests won't compile when both files present                        |
| **INTERFACE SAFETY**               |                                                                                                                    |        |        |
| 12                                 | Add `var _ saga.Store = (*MemoryStore)(nil)`                                                                       | 🔵     | S      | Compile-time safety for saga store                                                            |
| 13                                 | Add `var _ AggregateReader = (*InMemoryAggregateReader)(nil)`                                                      | 🔵     | S      | Compile-time safety for stream reader                                                         |
| 14                                 | Add `var _ AggregateReader = (*SQLAggregateReader)(nil)`                                                           | 🔵     | S      | Compile-time safety for SQL reader                                                            |
| 15                                 | Add `var _ event.Bus = (*MemoryBus)(nil)`                                                                          | 🔵     | S      | Compile-time safety for event bus                                                             |
| 16                                 | Add `var _ event.Store = (*MemoryStore)(nil)` + Journal + SeekableJournal                                          | 🔵     | S      | Already has StreamLoader, missing Store/Journal/SeekableJournal                               |
| **TEST QUALITY**                   |                                                                                                                    |        |        |
| 17                                 | Add saga persistent store integration test (SQLite)                                                                | 🟡     | M      | saga module has no storage tests — only MemoryStore                                           |
| 18                                 | Add stream integration test (SQLite)                                                                               | 🟡     | M      | stream module has unit tests but no integration tests                                         |
| 19                                 | Split `storage/sqlite_integration_test.go` (663L → 2 files)                                                        | 🔵     | M      | Largest test file, 89% over 350L limit                                                        |
| 20                                 | Split `core/event/outbox_publisher_test.go` (617L → 2 files)                                                       | 🔵     | M      | 76% over limit                                                                                |
| 21                                 | Split `catalog/schema_test.go` (604L → 2 files)                                                                    | 🔵     | M      | 73% over limit                                                                                |
| 22                                 | Split `storage/event_store_load_test.go` (576L → 2 files)                                                          | 🔵     | M      | 65% over limit                                                                                |
| 23                                 | Split `memory/store_test.go` (545L → 2 files)                                                                      | 🔵     | M      | 56% over limit                                                                                |
| 24                                 | Add fuzz tests for event.NewEvent                                                                                  | 🔵     | M      | Resilience testing for event creation                                                         |
| 25                                 | Add fuzz tests for id.ParseAggregateID                                                                             | 🔵     | S      | Resilience testing for ID parsing                                                             |
| **DEPRECATION CLEANUP**            |                                                                                                                    |        |        |
| 26                                 | Remove `event.GlobalLoader` assertions from memory/store.go                                                        | 🔵     | S      | Deprecated → Journal; remove backward-compat assertion                                        |
| 27                                 | Remove `event.PositionalLoader` assertions from memory/store.go                                                    | 🔵     | S      | Deprecated → SeekableJournal                                                                  |
| 28                                 | Remove `event.GlobalLoader` assertions from storage/event_store.go                                                 | 🔵     | S      | Same                                                                                          |
| 29                                 | Remove `event.PositionalLoader` assertions from storage/event_store.go                                             | 🔵     | S      | Same                                                                                          |
| 30                                 | Remove `event.TransactionalStore` assertion from storage/transactional_store.go                                    | 🔵     | S      | Deprecated → TransactionalSink                                                                |
| 31                                 | Remove `event.TransactionalStore` assertion from storage/sql_backend.go                                            | 🔵     | S      | Same                                                                                          |
| **TODO_LIST.MD ITEMS (Unblocked)** |                                                                                                                    |        |        |
| 32                                 | Add ProcessedAt to CheckpointStore (store EventID+time, not just EventID)                                          | 🟡     | M      | Consumers can't track when checkpoints were processed                                         |
| 33                                 | Add event.Context propagation through NewEvent/PublishChanges                                                      | 🔵     | L      | Thread ctx into event lifecycle — needs design first                                          |
| 34                                 | Wire example/user/ to catalog-aware event constructors                                                             | 🔵     | M      | Example doesn't show full catalog integration                                                 |
| 35                                 | Add background polling for InMemoryRunner (currently push-only)                                                    | 🟡     | L      | Projections need pull-model for journal stores                                                |
| 36                                 | Add projection parallel processing (goroutine pool)                                                                | 🔵     | L      | Performance improvement for multi-projection replay                                           |
| 37                                 | Add WithAsyncWrites() for PebbleEventStore                                                                         | 🔵     | M      | Pebble async write performance                                                                |
| 38                                 | Parallelize CI matrix (one job per module)                                                                         | 🔵     | M      | CI speed — currently runs all modules in one job                                              |
| 39                                 | Benchmark storage backends (PG vs SQLite vs Pebble)                                                                | 🔵     | L      | Performance guidance for consumers                                                            |
| 40                                 | Enforce 350-line limit on test files via pre-commit hook                                                           | 🔵     | M      | Prevents test file bloat                                                                      |
| **DOCS & DX**                      |                                                                                                                    |        |        |
| 41                                 | Add stream module README.md                                                                                        | 🔵     | S      | No module-level docs for stream/                                                              |
| 42                                 | Add saga module README.md                                                                                          | 🔵     | S      | No module-level docs for saga/                                                                |
| 43                                 | Add otel module README.md                                                                                          | 🔵     | S      | No module-level docs for otel/                                                                |
| 44                                 | Update TODO_LIST.md — mark completed items, remove done items                                                      | 🔵     | M      | Stale TODO list causes confusion                                                              |
| 45                                 | Update CHANGELOG.md with Session 126 work                                                                          | ⚪     | S      | Changelog maintenance                                                                         |

---

## D. BLOCKED (Requires External Action)

| #   | Task                                                    | Blocker                                     |
| --- | ------------------------------------------------------- | ------------------------------------------- |
| 1   | Remove `replace` directives from go.mod files           | Requires v1.0.0 tags pushed for ALL modules |
| 2   | Standardize integration/catalog/example go.mod versions | Requires tag push                           |
| 3   | Add PostgreSQL integration tests with testcontainers    | Requires Docker setup                       |
| 4   | Push release tags for remaining modules                 | Requires `git push --tags` per module       |
| 5   | Create go-branded-id v0.2.0                             | Different repo                              |
| 6   | Change LICENSE from proprietary to MIT/Apache-2.0       | Owner decision                              |
| 7   | Move example/todo to own repository                     | Requires manual repo creation               |

---

## E. FUTURE (Speculative / Far-term)

| #   | Task                                                           | Notes                |
| --- | -------------------------------------------------------------- | -------------------- |
| 1   | v2: Split query.Handler from `any` to generic                  | Breaking change      |
| 2   | v2: Split core/event god-package into sub-packages             | Breaking change      |
| 3   | v2: io.Closer removal from core interfaces                     | Breaking change      |
| 4   | v2: Add TransactionID branded type                             | Breaking change      |
| 5   | Middleware generic `Middleware[H]` to eliminate 3x boilerplate | API redesign         |
| 6   | Missing branded IDs (SagaID, TenantID, ProjectionID)           | New types, migration |
| 7   | Catalog diff/breaking-change detection tool                    | New tooling          |
| 8   | High-level test utilities (AggregateTester, ProjectionTester)  | New API surface      |
| 9   | Documentation site (Docusaurus/MkDocs)                         | Infra                |
| 10  | Bi-temporal support (ValidAt, WithValidAt)                     | New feature          |
| 11  | Distributed consensus (Raft/CRDT overlay)                      | Major feature        |
| 12  | Pull-before-push sync protocol                                 | Major feature        |
| 13  | Thin PostgreSQL store adapter (no Watermill)                   | New adapter          |
| 14  | Thin NATS bus adapter (no Watermill)                           | New adapter          |
| 15  | Absorb projection/ into core/event                             | Restructure          |

---

## F. TOTALLY FUCKED UP (Honest Assessment)

| #   | Issue                                                                                               | Severity    | Fix                                                                                                                                                                                 |
| --- | --------------------------------------------------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Storage sqlite test duplication** — 11 duplicate symbols across 2 test files                      | 🔴 Critical | Merge files; currently builds only because one is untracked. If both land in the same test binary, compilation fails.                                                               |
| 2   | **Signing error classification gap** — 36 bare `fmt.Errorf` calls in production code                | 🟡 High     | Signing errors lose error-family classification. `event.IsRetryable()` returns wrong results for signing failures. Consumers can't distinguish retryable from fatal signing errors. |
| 3   | **BuildFlow commit cycle** — pre-commit hook reformatting creates 3-5 extra amend rounds per commit | 🔵 Medium   | BuildFlow runs go-mod-tidy on ALL modules on every commit, creating cascading changes. Not broken, but costs 30-60s per commit.                                                     |
| 4   | **Test files over 350L limit** — 20 test files exceed the project's own 350-line rule               | 🔵 Medium   | Largest: `storage/sqlite_integration_test.go` (663L), `core/event/outbox_publisher_test.go` (617L). Rule exists but isn't enforced.                                                 |

---

## G. Top 25 Execution Order (Ready to Ship, ≤12 min each)

Sorted by: **Impact × Customer Value ÷ Effort** (highest first)

| Rank | #     | Task                                                 | Impact | Effort | Why This Order                                 |
| ---- | ----- | ---------------------------------------------------- | ------ | ------ | ---------------------------------------------- |
| 1    | 11    | Fix storage sqlite test duplication (merge files)    | 🔴     | M      | **Compilation blocker** — must fix first       |
| 2    | 1     | Classify signing/event.go (2 calls)                  | 🟡     | S      | Quick win, highest error-density file          |
| 3    | 2     | Classify signing/signer.go (2 calls)                 | 🟡     | S      | Quick win                                      |
| 4    | 3     | Classify signing/hmac.go (1 call)                    | 🟡     | S      | Quick win                                      |
| 5    | 4     | Classify signing/ed25519.go (3 calls)                | 🟡     | S      | Quick win                                      |
| 6    | 5     | Classify signing/middleware.go (8 calls)             | 🟡     | M      | High-impact — middleware is consumer-facing    |
| 7    | 6     | Classify signing/multisig.go (6 calls)               | 🟡     | M      | Multi-signer API surface                       |
| 8    | 7     | Classify signing/multisig_extract.go (5 calls)       | 🟡     | S      | Signature extraction                           |
| 9    | 8     | Classify signing/multisig_middleware.go (12 calls)   | 🟡     | M      | Largest file, completes signing classification |
| 10   | 12    | Add saga.MemoryStore compliance check                | 🔵     | S      | 1 line, compile-time safety                    |
| 11   | 13    | Add InMemoryAggregateReader compliance check         | 🔵     | S      | 1 line                                         |
| 12   | 14    | Add SQLAggregateReader compliance check              | 🔵     | S      | 1 line                                         |
| 13   | 15    | Add MemoryBus compliance check                       | 🔵     | S      | 1 line                                         |
| 14   | 16    | Add MemoryStore Store/Journal/SeekableJournal checks | 🔵     | S      | 3 lines                                        |
| 15   | 26-31 | Remove deprecated interface assertions (6 files)     | 🔵     | S      | Clean up 6 nolint comments                     |
| 16   | 32    | Add ProcessedAt to CheckpointStore                   | 🟡     | M      | Consumer-facing feature                        |
| 17   | 17    | Add saga persistent store test (SQLite)              | 🟡     | M      | First storage test for saga                    |
| 18   | 18    | Add stream integration test (SQLite)                 | 🟡     | M      | First integration test for stream              |
| 19   | 19    | Split storage/sqlite_integration_test.go (663L)      | 🔵     | M      | Largest test file                              |
| 20   | 20    | Split core/event/outbox_publisher_test.go (617L)     | 🔵     | M      | Second largest                                 |
| 21   | 24    | Add fuzz tests for event.NewEvent                    | 🔵     | M      | Resilience                                     |
| 22   | 9     | Classify stream/ errors (10 calls)                   | 🔵     | S      | Stream error quality                           |
| 23   | 10    | Classify middleware/ errors (4 calls)                | 🔵     | S      | Middleware error quality                       |
| 24   | 44    | Update TODO_LIST.md                                  | 🔵     | M      | Stale docs cause confusion                     |
| 25   | 40    | Enforce 350L test limit in pre-commit                | 🔵     | M      | Prevents future bloat                          |

---

## H. Top #1 Question I Cannot Figure Out Myself

**Signing error classification strategy: should signing errors use `event.WrapRejection` or `event.WrapInfrastructure`?**

- `signing/middleware.go` errors during sign/verify — these are **infrastructure failures** (crypto errors, nil signer) → `event.WrapInfrastructure`
- `signing/multisig.go` errors during multi-signer construction — **rejections** (not enough signatures, invalid threshold) → `event.WrapRejection`
- `signing/ed25519.go` errors — **infrastructure** (invalid key length, wrong curve) → `event.WrapInfrastructure`

**But:** signing is a standalone module that currently depends on `core/event` only for the `NewRejection` sentinel errors. Should it import `event.WrapInfrastructure` / `event.WrapRejection` / `event.WrapTransient` directly, or should signing define its own error wrappers that consumers can classify?

**My recommendation:** Use `event.WrapInfrastructure` for crypto/key errors and `event.WrapRejection` for validation errors (wrong number of signatures, unmet threshold). This matches the existing pattern in `signing/errors.go` which already uses `event.NewRejection`.
