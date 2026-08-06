# Status Report: bbolt Backend + MySQL Factory + KV Store Evaluation

**Date:** 2026-08-06 14:43
**Session focus:** cqrs-bench backend coverage — "why not tested?" + adding new KV stores
**Previous session:** SQLite CGo benchmark fairness (`2026-08-06_14-06_sqlite-cgo-bench-fairness.md`)

---

## Executive Summary

The user asked: (1) why does cqrs-bench not test duckdb, mysql, postgres, turso, graph?
(2) should we add other KV stores (badger, dgraph, valkey, prometheus, cayley, bbolt)?

We researched all 6 proposed stores, evaluated fit, implemented **bbolt** (the clear winner —
pure Go B+tree, different engine family from Pebble's LSM), added **MySQL** as a factory gap fix,
and wired both into cqrs-bench. The bbolt implementation is functional (7 tests pass with -race,
benchmarks run successfully) but has **significant completeness gaps** compared to the Pebble
module it was modeled after.

**Files created:** 16 new Go files (1720 lines), 2 new go.mod modules
**Files modified:** factory.go, flags.go, main.go, go.work (4 files)

---

## a) FULLY DONE

### MySQL factory gap fix

- [x] `cmd/cqrs-bench/factory.go` — added `mysql`/`maria`/`mariadb` case. Uses `mysql.New(dsn)`.
- [x] Updated help text, flag descriptions, and error messages to include mysql.
- [x] Build passes (CGO_ENABLED=0 and =1).

### KV store evaluation matrix (research)

- [x] **bbolt** — IMPLEMENTED (see below). Pure Go B+tree, embedded, active (v1.5.0 Jun 2026).
- [x] **badger** — Evaluated and SKIPPED. LSM + value log (WiscKey), active (v4.9.6). Redundant
      with Pebble (both LSM family). Would not test a different engine characteristic.
- [x] **dgraph** — Evaluated and SKIPPED. Distributed graph DB (requires Zero + Alpha cluster).
      Uses Badger internally. Wrong data model, wrong deployment model.
- [x] **valkey** — Evaluated and SKIPPED. Network server (C, RESP protocol). Not embeddable.
      Would require a running server process — not comparable to embedded stores.
- [x] **prometheus** — Evaluated and SKIPPED. Time-series DB storing float64 metrics only.
      "Not designed for use as a library." Cannot store arbitrary byte payloads.
- [x] **cayleygraph** — Evaluated and SKIPPED. Dormant (last commit Jul 2024, pre-1.0).
      RDF quad model. Multiple storage backends (Bolt, LevelDB, SQLite) — would test the
      backend, not the engine.

### bbolt storage module (`storage/bbolt/`) — functional core

- [x] `doc.go` — package documentation
- [x] `errors.go` — sentinel errors (ErrNilDatabase, ErrVersionMismatch, etc.)
- [x] `base.go` — storeBase struct, bucket names, createBuckets
- [x] `store.go` — EventStore (Save, AppendBatch, key helpers, version check). Single-writer
      model means NO per-stream locking needed (unlike Pebble's 256-shard mutex).
- [x] `load.go` — Load, LoadFromVersion, LoadToVersion, LoadToTimestamp
- [x] `journal.go` — ReadAll, ReadFrom (implements Journal + SeekableJournal)
- [x] `serialization.go` — CBOR serialization/deserialization (mirrors pebble pattern)
- [x] `checkpoint.go` — CheckpointStore (Save, Load)
- [x] `snapshot.go` — SnapshotStore (Save, Load, LoadAtVersion, Delete)
- [x] `kv_adapter.go` — KVAdapter implementing kv.Store (Get, Set, Delete, Has, Batch, NewIterator, SetIfAbsent)
- [x] `kv_iterator.go` — bboltIterator + bboltBatch + emptyIterator
- [x] `otel.go` — tracer + span helpers (startStreamSpan, startReadSpan, startProjectionSpan)
- [x] `backend.go` — Backend facade (Open, NewBackend, EventStore/SnapshotStore/CheckpointStore/ReadModels accessors)
- [x] `store_test.go` — 7 smoke tests (all pass with -race): SaveLoad, VersionConflict, JournalReadAll, Checkpoint, Snapshot, KVAdapter

### bbolt stack preset (`stack/bbolt/`)

- [x] `preset.go` — New(path) returns fully-wired stack.Bundle with EventStore, SnapshotStore,
      CheckpointStore, ReadModels, EventBus, capabilities, closer.

### cqrs-bench wiring

- [x] `factory.go` — `bbolt`/`bolt` case added (temp dir, disk path tracking)
- [x] `flags.go` — Backend help updated, Backends default changed to `memory,sqlite,bbolt,pebble`,
      Dir help updated to include bbolt.
- [x] `main.go` — Backends section updated with bbolt description.
- [x] Error message updated to list all backends.

### Verification

- [x] Full workspace builds: `go build ./...` (CGO_ENABLED=0 AND =1)
- [x] bbolt tests pass: 7/7 with `-race` (1.019s)
- [x] cqrs-bench tests pass: `go test ./cmd/cqrs-bench/...` (6.398s)
- [x] Benchmark runs successfully: 5-backend comparison (memory, sqlite, sqlite-cgo, bbolt, pebble)
- [x] go.work updated with both new modules

### Benchmark results (small profile, 10K events, 5 backends)

| Backend    | Write P50 | Load P50   | Write Amp | Heap       | Disk       |
| ---------- | --------- | ---------- | --------- | ---------- | ---------- |
| memory     | 620ns     | 660ns      | -         | 27 MiB     | 0 B        |
| pebble     | 16.9µs    | 103.2µs    | 10.2x     | 54 MiB     | 25 MiB     |
| **bbolt**  | **41µs**  | **94.6µs** | **32.8x** | **18 MiB** | **80 MiB** |
| sqlite-cgo | 243µs     | 351µs      | 11.3x     | 55 MiB     | 28 MiB     |
| sqlite     | 410µs     | 429µs      | 11.3x     | 53 MiB     | 28 MiB     |

Key finding: bbolt wins on loads (B+tree point reads) but loses on writes (single-writer
serialization + full page rewrites). The B+tree vs LSM tradeoff is now empirically visible.

---

## b) PARTIALLY DONE

### bbolt module completeness vs Pebble reference

The bbolt module was modeled after storage/pebble but is **missing several interfaces and
features** that Pebble implements. It implements the minimum viable event store but is NOT
at feature parity:

- [ ] **No CommandStore** — Pebble has CommandStore (Save, Load, CommandJournal, SeekableCommandJournal).
      bbolt Backend has no CommandStore() accessor. The stack preset does not wire one.
- [ ] **No QueryStore** — Pebble has QueryStore (SaveQuery, LoadQueries, QueryJournal, SeekableQueryJournal).
      bbolt Backend has no QueryStore() accessor.
- [ ] **No streaming iterators** — Pebble implements `event.StreamingSource` + `event.StreamingJournal`
      (LoadStream, LoadStreamFromVersion, ReadStream, ReadStreamFrom). bbolt does NOT implement
      these. The stack.Bundle will have nil StreamingSource/StreamingJournal fields.
- [ ] **No BackwardsSource** — Pebble does not implement this either, so parity is maintained.
- [ ] **No `WithDurability` option** — bbolt always uses `DurabilityStrict` (fsync per write).
      There is no `WithDurability(stack.DurabilityRelaxed)` equivalent (bbolt has no `NoSync`
      option exposed — `bolt.Options` has a `NoSync` field but we don't expose it).
- [ ] **No `WithAsyncWrites` option** — Pebble has `WithAsyncWrites()` StoreOption. bbolt has
      a `StoreOption` type but it's unused (the variadic param is `_ ...StoreOption`).
- [ ] **No `GracefulClose` on Backend** — implemented but NOT wired through the stack preset
      as a distinct method (the preset just registers `stack.WithCloser(backend)`).

### OTel spans created but NOT used

- [ ] `otel.go` defines `startStreamSpan`, `startReadSpan`, `startProjectionSpan` — but NONE
      of the store methods (Save, Load, ReadAll, etc.) actually call them. The span helpers
      are dead code. Pebble instruments every method; bbolt instruments none.

### Tests are smoke tests only

- [ ] 7 basic round-trip tests. NO contract test suite run (Pebble uses `eventtest.TestStore*`
      helpers — bbolt does not import or run these).
- [ ] No test for `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`.
- [ ] No test for `ReadFrom` (seekable journal with afterEventID skip).
- [ ] No test for `AppendBatch`.
- [ ] No test for `SetIfAbsent` (conditional writer).
- [ ] No test for KV iterator (NewIterator with prefix).
- [ ] No test for KV batch (Batch().Set/Delete/Commit).
- [ ] No test for snapshot `LoadAtVersion` or `Delete`.
- [ ] No concurrency test (multiple goroutines writing different streams).
- [ ] No golden/snapshot test.

### go.mod files need proper versioning

- [ ] `storage/bbolt/go.mod` — depends on `github.com/larsartmann/go-cqrs-lite/event/v4@v4.2.0`
      etc. but the module itself is NOT tagged. Consumers cannot `go get` it.
- [ ] `stack/bbolt/go.mod` — same. Cannot resolve `storage/bbolt/v4` outside workspace mode.
- [ ] Neither module has been added to `cmd/api-stability/main.go` modules list (required by
      `TestEveryGoModDirIsInModulesList`).

---

## c) NOT STARTED

### Documentation

- [ ] No README.md in `storage/bbolt/` or `stack/bbolt/` (all other storage/stack modules have one)
- [ ] AGENTS.md not updated — module list does not include `storage/bbolt` or `stack/bbolt`
- [ ] Monorepo structure tree in AGENTS.md not updated
- [ ] Module count in AGENTS.md says "69 go.mod files" — now 71
- [ ] `docs/api_surface.txt` not regenerated with bbolt exports
- [ ] No doc-check verification (Go import paths in any new markdown)

### CI/gate

- [ ] `nix run .#verify` NOT run (takes 3-4 min — the "stale GREEN" anti-pattern from AGENTS.md)
- [ ] `nix run .#lint` NOT run — bbolt code has not been linted
- [ ] `nix fmt` NOT run — code may not be gofumpt-formatted
- [ ] `nix run .#check-layers` NOT run — dependency budget not verified
- [ ] `nix run .#check-duplication` NOT run — duplication golden not updated
- [ ] `nix run .#check-coverage` NOT run

### cqrs-bench default compare

- [ ] Default compare backends changed to `memory,sqlite,bbolt,pebble` but NOT tested with
      `turso` (which works offline and could be included).

---

## d) TOTALLY FUCKED UP

### OTel dead code

The `otel.go` file defines 3 span helper functions (`startStreamSpan`, `startReadSpan`,
`startProjectionSpan`) and a `tracer()` function. **NONE of them are called anywhere.** The
`store.go` Save method has `_ context.Context` (underscore — context is discarded). Every
Load/ReadAll/ReadFrom method also discards context. This means:

1. OTel tracing is completely non-functional for bbolt
2. The span helpers are dead code that will be flagged by linters
3. Context cancellation is ignored — a long-running bbolt transaction cannot be cancelled

This is the biggest quality gap. Pebble instruments every single method with spans. The bbolt
code copies the span helper file but never wires it in.

### Unused StoreOption type

`store.go` defines `type StoreOption func(*EventStore)` and `NewStore` accepts `_ ...StoreOption`,
but there are NO StoreOption functions defined and the variadic is ignored. This is API surface
that lies about configurability.

### Error wrapping inconsistency

- `store.go` Save wraps errors with `errorfamily.WrapConflict` and `errorfamily.WrapCorruption`
  inside the `db.Update` callback — correct.
- But `db.Update` itself returns the callback error directly. If bbolt fails to begin/commit
  the transaction (disk full, file locked), that error is returned **unwrapped**. Pebble wraps
  these with `errorfamily.WrapInfrastructure`.
- `load.go` returns `err` directly from `db.View` without wrapping — should be
  `WrapInfrastructure`.

### `hasPrefix` reimplements `bytes.HasPrefix`

`load.go` defines a custom `hasPrefix(key, prefix []byte) bool` function that does exactly what
`bytes.HasPrefix` does. Should use stdlib.

### `bytesLastIndex` was reimplemented then removed

Initially wrote a custom `bytesLastIndex` function, then noticed it was just `bytes.LastIndexByte`
and removed it. The initial mistake indicates not checking stdlib first.

### Compare default backends change is a breaking default

Changed `CompareFlags.Backends` default from `memory,sqlite,pebble` to `memory,sqlite,bbolt,pebble`.
This means anyone running `cqrs-bench compare` without `--backends` now gets bbolt too. This
could surprise users who have CI parsing the output. Should have been opt-in first.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Run the contract test suite** — `eventtest.TestStore*` helpers exist specifically for this.
   bbolt should run the same contract suite as Pebble, SQLite, Memory, Turso. This would catch
   edge cases the 7 smoke tests miss.
2. **Add streaming iterators** — `event.StreamingSource` and `event.StreamingJournal` are
   critical for projectionhost and large datasets. bbolt's `View()` transaction + cursor
   can implement these efficiently.
3. **Wire OTel spans** — every public method should create a span. The helpers exist; they
   just need to be called.
4. **Add CommandStore + QueryStore** — for full CQRS stack parity. These use the same
   bucket-per-store pattern.
5. **Expose `NoSync` / `NoFreelistSync`** as durability options — bbolt has these in
   `bolt.Options` but the preset hardcodes `DurabilityStrict`.

### Code quality

6. **Use `bytes.HasPrefix` instead of custom `hasPrefix`** — stdlib.
7. **Wrap all `db.Update`/`db.View` return errors** with `errorfamily.WrapInfrastructure`.
8. **Remove unused `StoreOption` type** or add actual options.
9. **Format with `nix fmt`** before committing.
10. **Run `nix run .#lint`** — likely has findings (unused code, error wrapping, etc.)

### Testing

11. **Add contract tests** — import `eventtest` and run the full suite.
12. **Add KV contract tests** — `kv/viewstoretest` has a contract suite for kv.Store.
13. **Add concurrency tests** — multiple goroutines writing different streams to verify
    the single-writer model doesn't deadlock.
14. **Add the bbolt module to the stack contract test** — `stack/contracttest/` runs
    the same suite against every preset.

### Benchmarking

15. **Add bbolt to `stack/bench/`** — the bench module has benchmark suites for each preset.
16. **Profile the write path** — bbolt's 41µs write P50 is 2.4x slower than Pebble. Is it
    the fsync? The B+tree page split? The single-writer lock contention?
17. **Test with `NoSync`** — how fast is bbolt without fsync? This would isolate the
    fsync cost from the B+tree cost.
18. **Test with larger batch sizes** — bbolt's B+tree may benefit more from batching than LSM.

---

## f) Up to 50 Things to Do Next

### Critical (blocks correctness or CI)

1. Add `storage/bbolt` and `stack/bbolt` to `cmd/api-stability/main.go` modules list
2. Regenerate `docs/api_surface.txt` with bbolt exports
3. Run `nix fmt` on all new files
4. Run `nix run .#lint` and fix findings
5. Run `nix run .#verify` — the ONLY source of truth for build/lint/test status
6. Fix OTel dead code — either wire spans into every method or remove `otel.go`
7. Wrap `db.Update`/`db.View` errors with `WrapInfrastructure`
8. Replace custom `hasPrefix` with `bytes.HasPrefix`
9. Remove unused `StoreOption` type or add real options
10. Tag both new modules (`storage/bbolt/v4.x.y`, `stack/bbolt/v4.x.y`)

### High priority (feature parity)

11. Add CommandStore to bbolt Backend
12. Add QueryStore to bbolt Backend
13. Wire CommandStore + QueryStore in stack/bbolt preset
14. Implement `event.StreamingSource` (LoadStream, LoadStreamFromVersion)
15. Implement `event.StreamingJournal` (ReadStream, ReadStreamFrom)
16. Run eventtest contract suite against bbolt EventStore
17. Run kv/viewstoretest contract suite against bbolt KVAdapter
18. Add bbolt to stack/contracttest suite
19. Add tests for LoadFromVersion, LoadToVersion, LoadToTimestamp
20. Add tests for ReadFrom with afterEventID skip
21. Add tests for AppendBatch
22. Add tests for SetIfAbsent
23. Add tests for KV iterator with prefix
24. Add tests for KV batch commit
25. Add tests for snapshot LoadAtVersion + Delete
26. Add concurrency test (parallel stream writes)

### Medium priority (polish + DX)

27. Write README.md for `storage/bbolt/`
28. Write README.md for `stack/bbolt/`
29. Update AGENTS.md module list with both new modules
30. Update AGENTS.md monorepo structure tree
31. Update module count (69 → 71)
32. Add bbolt to the seven-tier model in AGENTS.md
33. Add bbolt to cqrs-lint module catalog
34. Expose `bolt.Options{NoSync: true}` as `WithDurability(DurabilityRelaxed)`
35. Expose `bolt.Options{NoFreelistSync: true}` as an option
36. Add `WithBatchOptions` or similar for tuning
37. Revert the default compare backends change (make bbolt opt-in, not default)
38. Add `bbolt` to `cqrs-lint doctor` feature profile detection

### Lower priority (nice to have)

39. Add bbolt to `stack/bench/` benchmark suite
40. Profile write path (fsync vs B+tree vs lock contention)
41. Benchmark with `NoSync` to isolate fsync cost
42. Benchmark with larger batch sizes
43. Consider bbolt `Batch()` API (coalesces concurrent writes) vs `Update()` (serialized)
44. Add BadgerDB as a second KV store (LSM + value log) — now Pebble + bbolt + Badger
    would cover 3 engine families
45. Consider adding `minio/minio` (S3-compatible object store) for cold event archives
46. Document the B+tree vs LSM tradeoff in a benchmark README
47. Add a `--engines` flag to cqrs-bench that groups backends by engine family
48. Consider Iroh as a CRDT-native store (already have irohengine in metaengine)
49. Update the SKILL.md routing table to mention bbolt as an option

---

## g) Questions I Cannot Answer Myself

### 1. Should bbolt be a first-class module (full feature parity with Pebble) or a benchmark-only curiosity?

The bbolt implementation is currently a "minimum viable event store" — enough to benchmark but
missing CommandStore, QueryStore, streaming iterators, and contract tests. Pebble took months
to reach its current state. Should I invest in full parity, or is bbolt's value primarily as a
B+tree comparison point in benchmarks? If the former, it needs contract tests, streaming, and
the full store set. If the latter, the current state may be sufficient.

### 2. Should we add BadgerDB despite the LSM redundancy with Pebble?

Badger uses a WiscKey architecture (LSM tree + separate value log) that is architecturally
distinct from Pebble's conventional LSM. It could test the "value log separation" hypothesis
(keys in LSM, values on disk). But it's still fundamentally LSM-family. Is that enough
differentiation to justify another ~1700 lines of code?

### 3. Should the default `compare` backends list be reverted to exclude bbolt?

I changed the default from `memory,sqlite,pebble` to `memory,sqlite,bbolt,pebble`. This makes
bbolt appear in every default comparison, which could break CI pipelines that parse the output
table (new row, different column widths). Should I revert to opt-in (`--backends memory,sqlite,bbolt,pebble`)
until bbolt reaches feature parity and gets tagged?
