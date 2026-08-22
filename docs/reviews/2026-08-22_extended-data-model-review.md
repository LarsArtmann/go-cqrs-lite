# Extended Data-Model Review — storage/*, system/, stack/, watermill/, middleware/

> **Date:** 2026-08-22 · **Plan task:** T21 (core data-model v5 execution plan)
> **Companion to:** `docs/reviews/2026-08-22_core-data-model-review.html`
> (which covered id/event/command/query/record/metadata/decider/snapshot).
> Same rigor applied to the remaining model surface. Every finding has a
> file:line verified at review time. Findings feed TODO_LIST and the
> [v5 deprecation sweep](../planning/v5-deprecation-sweep.md).

**Verdict:** the extended surface is materially healthier than the core was
pre-plan — no `any`-typed value fields anywhere in these modules, error
sentinels are real, capability composition is interface-driven. The
findings below are mostly *drift between backends that encode the same
concept differently* — the exact disease the core review called out, in its
milder storage-layer form.

## P1 — Cross-backend drift (Medium)

| # | Finding | Where | Fix |
|---|---------|-------|-----|
| E1 | **Event-envelope Encoding type drift**: event wire structs type `Encoding` as raw `string` while snapshot wire structs in the SAME packages use typed `record.Encoding` (added T17/T18) | `storage/pebble/serialization.go:133` vs `snapshot.go:263`; `storage/bbolt/serialization.go:102` vs `snapshot.go:188` | v5: type both as `record.Encoding`; conversion sites (`codec.Encoding(s.Encoding)`) disappear |
| E2 | **Snapshot identity tag drift** (known, now pinned): pebble `aggregate_id`/`aggregate_type` vs bbolt `stream_id`/`stream_type` for the same concept | pebble `snapshot.go:258`, bbolt `snapshot.go:183` | v5 rename per T18 audit (TODO_LIST §v5) |
| E3 | **Error-model drift**: bbolt command/query marshal+reconstruct paths use bare `fmt.Errorf` (no family/code) while pebble's equivalents are `errorfamily`-coded | bbolt `command_serialization.go:34,59`, `query_serialization.go:30,52` vs pebble counterparts | Adopt the pebble pattern in bbolt (S, additive) |
| E4 | **Sentinel name ↔ code mismatch** (aggregate vocabulary in codes, stream vocabulary in names) | `storage/sql/errors.go:22` (`ErrStreamTypeMismatch` → `storage.aggregate_type_mismatch`), pebble `errors.go:15` | Batch rename at v5 with sweep §4 |
| E5 | **Facade composition divergence** for the same concept: SQLBackend = lazy+borrowed, pebble/bbolt = eager+own-or-borrow, turso = pure alias, backuptest = interface | `storage/sql_backend.go:18`, `storage/pebble/backend.go:21`, `storage/bbolt/backend.go:43`, `storage/turso/backend.go:33` | Acceptable (backend ecosystems differ); document the capability matrix instead of forcing one shape |

## P2 — Config/type hygiene (Medium-Low)

| # | Finding | Where | Fix |
|---|---------|-------|-----|
| E6 | `middleware.Option` vs `middleware.BundleOption` — same shape, separate universes | `middleware/options.go:10`, `otel_bundle.go:43` | Merge or bridge at v5 |
| E7 | **RetryConfig name collision**: `watermill.RetryConfig` (MaxRetries/InitialInterval/MaxInterval, no Validate) vs `middleware.RetryConfig` (MaxAttempts/InitialDelay/MaxDelay/IsRetryable/OnDeadLetter, validated) — same name, incompatible fields | `watermill/middleware.go:22`, `middleware/middleware.go:15` | Rename watermill's (v5) or add Validate to match |
| E8 | Stringly closed sets: `MessageAdapter.Kind` + `DeadLetterEntry.Kind` (`"command"/"event"/"query"` as plain string), `SQLDeadLetterStore.dialect` free string | `middleware/generic.go:26-30`, `deadletter.go:22`, `deadletter_sql.go:47` | Typed `Kind` enum (v5, additive type now possible) |
| E9 | `turso.Policy` zero value compiles but mutators panic on nil maps (reads guarded, writes not) | `storage/turso/policy.go:59-77` | Guard writes like reads, or constructor-only (S) |
| E10 | `system.ShutdownDependency.Before/After` free strings; a typo'd engine name silently no-ops (falls back to creation order) | `system/config_types.go:94-97` | Validate names against Engines at System build (S, additive) |
| E11 | `system.AdapterCore.Encode func(T) string` — no error return; encode failures cannot surface | `system/adapter_core.go:37` | v5: `func(T) (string, error)` |
| E12 | Watermill envelope is 17 flattened string metadata keys with silent-drop on unknown keys; `processing_mode` round-trips typed enum through strings | `watermill/protocol.go:19-38`, `catchup_subscriber.go:255` | Acceptable wire compat constraint; document the key set as protocol version |
| E13 | `SQLTimerStore[P]` type param never appears in struct fields (payload type only used by methods) — shape lies about genericity | `storage/timer_store.go:26` | v5: methods-only generic design or document |
| E14 | Handle ownership asymmetry inside `storage/eventstore`: event store owns (closable), snapshot/checkpoint borrow (no-op Close) | `event_store.go:17` vs `snapshot.go:19`, `checkpoint.go:14` | Document ownership per store (S) |
| E15 | Seven distinct middleware signatures across the surface (3 structurally identical error-wrap aliases + query's `any`-return + 2 publish-side + watermill's router universe); no bridge between router and dispatcher middleware | see agent census: `middleware/generic.go:14-17`, `query/query.go:129`, `event/bus.go:56`, `command/bus.go:37`, `watermill/middleware.go:17` | v5 candidate: unify #1-#3 via generics; router universe stays foreign |

## P3 — Cosmetic / accepted

- `PebbleMetrics` mixed signedness (`int64` vs `uint64` fields) — mirrors pebble's own API; accept.
- bbolt/pebble `storeBase` same slot, opposite semantics (`batch` vs `syncWrites`) — names are honest locally; accept.
- `turso/indexing.Index.Where` meaningful only when `Partial` — bool+string pair; accept (SQL-shaped).
- `system.BusConfig.Mode` + `CacheConfig.Engine` dead fields with documented v5 removals — already tracked.

## Fixed during this review

- `storage/pebble/adapter.go:53` — doc claimed "Panics if db is nil" while the constructor returns `ErrNilDatabase`; lie removed.

## Capability matrix (verified assertions)

| Capability | memory | SQL eventstore | pebble | bbolt |
|---|---|---|---|---|
| MultiSink | ✓ (only) | ✗ | ✗ | ✗ |
| BackwardsSource | ✓ | ✓ | ✗ | ✗ |
| SeekableJournal | ✓ | ✓ | ✓ | ✓ |
| StreamingJournal/Source | ✓ | Journal only | ✓ | ✓ |
| EventByIDLoader | ✗ | ✓ (only) | ✗ | ✗ |
