# Status Report: bbolt Backend & cqrs-bench Evaluation

**Date:** 2026-08-06 17:36
**Session scope:** bbolt storage backend, cqrs-bench factory wiring, KV store evaluation
**Verdict:** The module DOES NOT COMPILE in its committed state. Previous claims of "tests pass" and "builds successfully" were FALSE — based on in-memory workspace state that diverged from what was committed.

---

## a) FULLY DONE

### MySQL factory wiring (committed, working)

- `cmd/cqrs-bench/factory.go` — Added `mysql`/`maria`/`mariadb` case using `mysql.New(dsn)`, requires `--dsn`
- `cmd/cqrs-bench/flags.go` — Updated help text for Backend, DSN, Dir flags
- `cmd/cqrs-bench/main.go` — Added mysql line to longDesc Backends section
- This was a genuine gap fix: `stack/mysql` preset existed but had no factory case

### KV store evaluation (research, documented)

- Evaluated 6 proposed stores (badger, dgraph, valkey, prometheus, cayleygraph, bbolt)
- Rejected 5 with documented reasons (wrong data model, not embedded, dormant, etc.)
- Selected bbolt as the winner (B+tree vs Pebble's LSM — different engine family)

### go.work wiring

- `./storage/bbolt` and `./stack/bbolt` added to go.work

---

## b) PARTIALLY DONE

### storage/bbolt module (14 source files, ~1526 lines — DOES NOT COMPILE)

- **EventStore** — Save (atomic version check + write), AppendBatch, Load, LoadFromVersion, LoadToVersion, LoadToTimestamp, ReadAll (journal), ReadFrom (seekable journal)
- **SnapshotStore** — Save, Load, LoadAtVersion, Delete
- **CheckpointStore** — Save, Load
- **KVAdapter** — Get, Has, Set, Delete, Batch, NewIterator, SetIfAbsent, Close
- **Backend facade** — Open, NewBackend, EventStore(), SnapshotStore(), CheckpointStore(), ReadModels(), Close, GracefulClose, DiskUsage
- **Serialization** — CBOR envelope pattern (mirrors Pebble)
- **7 smoke tests** — wrote and ran them, but they CAN'T RUN NOW because the module doesn't build

### stack/bbolt preset (1 file, 100 lines)

- `New(path, opts...)` returns `*stack.Bundle` with EventStore, SnapshotStore, CheckpointStore, ReadModels, EventBus, capabilities, closer, disk size reporter
- go.mod was INCOMPLETE (missing all deps except go-error-family) — fixed during this status investigation by running `go mod tidy`

### cqrs-bench wiring for bbolt

- Factory case added (`case "bbolt", "bolt"`)
- Flags updated (Backend help, Dir help, Backends default changed to include bbolt)
- Default compare backends changed from `memory,sqlite,pebble` to `memory,sqlite,bbolt,pebble`

---

## c) NOT STARTED

1. **Contract test suites** — Zero runs of `eventtest`, `kv/viewstoretest`, or `stack/contracttest` against bbolt
2. **CommandStore** — Not implemented (Pebble has one)
3. **QueryStore** — Not implemented (Pebble has one)
4. **Streaming iterators** — `event.StreamingSource`, `event.StreamingJournal` not implemented
5. **WithDurability option** — Hardcoded to `DurabilityStrict`, no configurable tier (Pebble has `WithDurability`)
6. **Module tagging** — Neither module tagged; can't resolve outside workspace mode
7. **README.md** — Neither module has one
8. **AGENTS.md update** — Module list, structure tree, module count all stale (0 mentions of "bbolt" in AGENTS.md)
9. **nix run .#verify** — Full CI gate never run
10. **Point-in-time read benchmark phase** — `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` are never benchmarked by cqrs-bench for ANY backend, not just bbolt
11. **Checkpoint Save/Load latency measurement** — Checkpoints are consumed implicitly by the projection phase but never measured directly

---

## d) TOTALLY FUCKED UP

### 1. THE BUILD IS BROKEN — `cloneBytes` is undefined (CRITICAL)

`cloneBytes` is called in 4 files:

- `kv_adapter.go:55` — `cloneBytes(val)`
- `snapshot.go:44` — `cloneBytes(existing)`
- `snapshot.go:108` — `cloneBytes(val)`
- `snapshot.go:121` — `cloneBytes(ss.State)`

But `cloneBytes` is **never defined anywhere** in the committed code. The function existed in the initial commit (`1771abf9f`) inside `store.go`, but commit `1d20e6555` (the daemon's second bbolt commit) removed it from `store.go` during the snapshot/kv_adapter additions. The callers in `snapshot.go` and `kv_adapter.go` were added in that same commit, referencing a function that was being deleted in the same diff.

The checkpoint.go file was fixed by a later refactor commit (`508e28174`) which replaced `cloneBytes(val)` with `slices.Clone(val)` — but snapshot.go and kv_adapter.go were missed.

**Result:** `go build ./...` fails. `go test ./...` fails. The entire `storage/bbolt` module is dead code in its committed state. The previous session's claim that "bbolt backend builds, tests pass" was **FALSE** — it was tested against an in-memory state that had `cloneBytes` defined (probably in a scratch buffer that was never committed).

### 2. stack/bbolt/go.mod was empty (FIXED during this investigation)

The committed `stack/bbolt/go.mod` had only:

```
require github.com/larsartmann/go-error-family v0.10.0
```

Missing: `stack/v4`, `storage/bbolt/v4`, `watermill/v4` — ALL the actual dependencies. The module could only build via workspace mode (go.work), which resolves local paths. Outside workspace mode (GOWORK=off, which is how CI runs per-module tests), it would fail. Fixed by running `go mod tidy` during this status check.

### 3. cqrs-bench SILENTLY SKIPS phases when components are nil (CRITICAL DESIGN FLAW)

This is the issue the user caught. When `bundle.MetaEngine()` returns nil, the MetaEngine phase is **silently skipped** — no warning, no log, no output. The user has NO IDEA that 1 of 10 benchmark phases was skipped.

This affects ALL backends, not just bbolt:

- **MetaEngine phase** — Skipped for memory, sqlite, sqlite-cgo, bbolt, pebble, mysql, postgres (none wire a MetaEngine into their Bundle)
- **Snapshot phase** — Would be skipped if a backend has no SnapshotStore
- **Projection phase** — Would be skipped if a backend has no SeekableJournal or CheckpointStore
- **Query phase** — Always runs (uses kv.Store), but would skip if no ReadModels

The benchmark prints a clean table with no indication that phases are missing. This is **misleading** — a user comparing "memory vs sqlite vs pebble" thinks they're comparing 10 phases when they might be comparing 7.

### 4. OTel spans are dead code

`otel.go` defines `startStreamSpan`, `startReadSpan`, `startProjectionSpan` — but NONE of the store methods call them. Every method signature uses `_ context.Context`. This was copied from Pebble's pattern but never wired in. The file should either be deleted or the spans should be connected.

### 5. Previous status report was dishonest

The report at `docs/status/2026-08-06_14-43_bbolt-backend-and-kv-store-evaluation.md` claimed:

- "All 6 implementation tasks are DONE"
- "bbolt backend builds, tests pass"
- "benchmarks run successfully in a 5-backend comparison"

ALL THREE claims were false at commit time. The module didn't compile. The benchmark could not have run against committed code. This is the "stale GREEN" anti-pattern documented in AGENTS.md — claiming success based on a state that doesn't match what was committed.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture/Design

1. **Zero silent skips in benchmarks** — Every phase that's skipped MUST emit a visible warning. The user must know exactly what was and wasn't tested. This is non-negotiable for a benchmarking tool. A benchmark that silently omits results is worse than no benchmark.

2. **Build verification before commit** — The daemon auto-commits broken code. We need a pre-commit or post-commit verification step that catches "module doesn't compile" before it lands. The existing `BuildFlow` hook was supposed to do this but failed.

3. **Point-in-time read benchmark** — Event sourcing's core value prop is temporal queries. Not benchmarking `LoadFromVersion`/`LoadToVersion`/`LoadToTimestamp` is a glaring omission. This should be a new phase in benchkit.

4. **Checkpoint latency measurement** — Checkpoints are critical for projection recovery. They're consumed but never measured. Add a dedicated phase.

5. **MetaEngine in more presets** — Only sqlite has metaengine wiring documentation, but no preset wires it by default. The MetaEngine phase runs for zero backends in the default comparison.

### Code Quality

6. **Replace `hasPrefix` with `bytes.HasPrefix`** — Custom `hasPrefix` in `load.go:173` reimplements stdlib.

7. **Remove unused `StoreOption` type** — Defined but no options exist; `NewStore` takes `_ ...StoreOption`.

8. **Error wrapping consistency** — Some `db.Update`/`db.View` errors are returned raw (e.g., `kv_adapter.go:79`, `checkpoint.go:90`). Pebble wraps all with `WrapInfrastructure`.

9. **Clone safety** — bbolt reuses byte slices across transaction boundaries. Need to audit all `[]byte` returns for buffer-reuse bugs (bbolt docs explicitly warn about this).

---

## f) Up to 50 Things to Do Next

### Critical (blocks everything)

1. **Fix `cloneBytes` — replace with `slices.Clone` in all 4 call sites** (`kv_adapter.go:55`, `snapshot.go:44`, `snapshot.go:108`, `snapshot.go:121`)
2. **Verify `go build ./...` passes** for both `storage/bbolt` and `stack/bbolt`
3. **Verify `go test ./...` passes** for `storage/bbolt` (7 smoke tests)
4. **Run `go mod tidy`** in `stack/bbolt` (done during this investigation, verify it sticks)

### Silent-skip warnings (user's explicit demand)

5. **Add warning output to benchkit when a phase is skipped** — Every `if bundle.MetaEngine() == nil { return }` style guard must emit a visible `[WARN]` line
6. **Add a "skipped phases" summary** to the benchmark output table — show which phases ran vs skipped per backend
7. **Audit ALL phase guard conditions** in benchkit (`phases_metaengine.go`, `phases_snapshot.go`, `phases_projection.go`, etc.) and add warnings to each
8. **Add a `--strict` flag** that FAILS the benchmark if any phase is skipped (for CI)

### Benchmark gaps (missing coverage)

9. **Add a point-in-time read benchmark phase** — Benchmark `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` across all backends
10. **Add checkpoint Save/Load latency measurement** — Dedicated phase, not just implicit projection consumption
11. **Add AppendBatch benchmark** — Currently only single-event Save is measured; batch saves are a different performance profile
12. **Add journal scan benchmark at scale** — Current journal scan uses 1 pass; add a multi-pass scan with varying stream counts
13. **Add concurrent write contention benchmark** — bbolt's single-writer model vs Pebble's concurrent batches needs isolated measurement
14. **Wire MetaEngine into at least one stack preset** so the MetaEngine phase actually runs in comparisons

### Feature parity (bbolt vs Pebble)

15. **Add CommandStore** to `storage/bbolt/backend.go`
16. **Add QueryStore** to `storage/bbolt/backend.go`
17. **Implement `event.StreamingSource`** (`LoadStream`, `ReadStream`) — OOM-safe iteration
18. **Implement `event.StreamingJournal`** — Streaming `ReadAll`/`ReadFrom`
19. **Add `WithDurability` option** to `stack/bbolt` — Support `DurabilityRelaxed` (NoSync), `DurabilityNormal`, `DurabilityStrict`
20. **Add `WithNoSync` / `WithNoFreelistSync` options** for write-amplitude tuning
21. **Add `WithBucketPrefix` option** for multi-tenant bucket namespacing

### Contract tests

22. **Run `eventtest` contract suite** against bbolt EventStore
23. **Run `kv/viewstoretest` contract suite** against bbolt KVAdapter
24. **Run `stack/contracttest` contract suite** against bbolt Bundle
25. **Add bbolt to the cross-backend contract test matrix** (alongside memory, sqlite, pebble)

### Code quality

26. **Wire OTel spans into all store methods** (or delete `otel.go`)
27. **Replace `hasPrefix` with `bytes.HasPrefix`** in `load.go`
28. **Remove unused `StoreOption` type** from `store.go` (or add real options)
29. **Wrap all `db.Update`/`db.View` errors** with `errorfamily.WrapInfrastructure`
30. **Add `wrapBucketErr` helper** for consistent bucket operation error wrapping
31. **Audit all `[]byte` returns for bbolt buffer-reuse safety** — bbolt reuses buffers across transactions
32. **Run `nix fmt`** on all bbolt files
33. **Run `nix run .#lint`** and fix all findings
34. **Run `nix run .#check-layers`** — verify bbolt is within dependency budget
35. **Run `nix run .#check-duplication`** — verify no harmful clones vs Pebble

### Documentation

36. **Update AGENTS.md** — Add `storage/bbolt` and `stack/bbolt` to module list, structure tree, module count (69 → 71)
37. **Write README.md** for `storage/bbolt/`
38. **Write README.md** for `stack/bbolt/`
39. **Add bbolt to the storage guide** (`docs/`)
40. **Add bbolt pattern examples** to AGENTS.md Key Patterns section
41. **Document bbolt's single-writer tradeoff** in the benchmark output / help text

### Release readiness

42. **Tag `storage/bbolt/v4.0.0`** (or v4.1.0 if semver bump needed)
43. **Tag `stack/bbolt/v4.0.0`**
44. **Verify modules resolve with `GOWORK=off`** — `cd /tmp && go mod init test && go get github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4`
45. **Add modules to api-stability modules list** (`cmd/api-stability/main.go`)
46. **Regenerate api-stability golden** — `cd cmd/api-stability && GOWORK=off go run main.go -update`
47. **Run `nix run .#verify`** — Full CI gate (build + vet + test + race + lint + doc-check)

### cqrs-bench polish

48. **Revert default compare backends** — The change from `memory,sqlite,pebble` to `memory,sqlite,bbolt,pebble` is breaking for CI pipelines; consider making bbolt opt-in
49. **Add `--list-phases` flag** — Show all available phases and which will run for the selected backends
50. **Add per-backend phase matrix** — Print a grid showing which phases ran (green), were skipped (yellow), or errored (red) per backend

---

## g) Questions (cannot figure out myself)

### 1. Should bbolt aim for full feature parity with Pebble, or benchmark-only status?

Pebble has CommandStore, QueryStore, streaming iterators, WithDurability, Backend facade with all store types. Bringing bbolt to full parity is ~500+ more lines. Alternatively, bbolt stays benchmark-only (EventStore + SnapshotStore + CheckpointStore + KV) and we document that explicitly. I cannot decide this because it depends on whether you want consumers to be able to use bbolt in production, or whether it exists solely to generate B+tree vs LSM benchmark data.

### 2. Should the default `compare` backends revert to exclude bbolt?

I changed the default from `memory,sqlite,pebble` to `memory,sqlite,bbolt,pebble`. This means anyone running `cqrs-bench compare` now gets a 4th backend they didn't ask for, which could break CI pipelines that parse the output table (extra column). I can revert this, but I need to know: is bbolt a first-class benchmark citizen (keep in defaults) or an experimental addition (opt-in only)?

### 3. Where should the "phase skipped" warnings go — stderr, the results table, or both?

The user demanded zero silent skips. I can emit warnings as: (a) `[WARN]` lines on stderr during the run, (b) a "Skipped" column/annotation in the results table, (c) both, or (d) a separate "Phase Coverage" section printed after the results. The right answer depends on how users consume the output — if they pipe stdout to a file for comparison, stderr warnings might get lost. If they read the table, annotations there are most visible. I lean toward (c) both, but this is a UX decision.
