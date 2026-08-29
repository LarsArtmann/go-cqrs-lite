# Status Report: Zero Silent Skips + bbolt Code Quality + Versioned Read Phase

**Date:** 2026-08-06 18:16
**Session scope:** Fixing broken bbolt build, eliminating ALL silent phase skips in cqrs-bench, bbolt code quality cleanup, adding versioned read benchmark phase
**Verdict:** The silent-skip problem is ELIMINATED. Every skipped phase now emits a visible warning. The bbolt module builds, tests pass, and the versioned read benchmark phase works. Remaining work is documentation, contract tests, and feature parity.

---

## a) FULLY DONE

### 1. Fixed broken bbolt build (CRITICAL — was broken since commit `1d20e6555`)

- `cloneBytes` was called in 4 files but NEVER defined in committed code
- Replaced all `cloneBytes(x)` calls with `slices.Clone(x)` in `kv_adapter.go`, `snapshot.go`
- Verified: `go build ./storage/bbolt/...` passes, `go test -race` passes (7/7 tests)

### 2. Eliminated ALL silent phase skips in benchkit (THE USER'S CORE DEMAND)

**13 nil-guard warnings wired across 8 phase files:**

| File                   | Guard                                    | Warning message                                                               |
| ---------------------- | ---------------------------------------- | ----------------------------------------------------------------------------- |
| `phases.go`            | `EventSink == nil`                       | `raw sink phase skipped: bundle has no EventSink`                             |
| `phases.go`            | `EventSource == nil`                     | `integrity check skipped: no EventSource or no streams written`               |
| `phases_read.go`       | `ReadModels == nil`                      | `read model phase skipped: bundle has no ReadModels (kv.Store)`               |
| `phases_query.go`      | `ReadModels == nil`                      | `query phase skipped: bundle has no ReadModels (kv.Store)`                    |
| `phases_projection.go` | `SeekableJournal/CheckpointStore == nil` | `projection phase skipped: bundle missing SeekableJournal or CheckpointStore` |
| `phases_snapshot.go`   | `EventStore() not ok`                    | `snapshot phase skipped: bundle has no EventStore (event.Store interface)`    |
| `phases_snapshot.go`   | `SnapshotStore == nil`                   | `snapshot phase: snapshot save/load skipped (no SnapshotStore)`               |
| `phases_metaengine.go` | `MetaEngine() == nil`                    | `metaengine phase skipped: bundle has no MetaEngine`                          |
| `phases_journey.go`    | `EventSink/ReadModels == nil`            | `journey phase skipped: bundle missing EventSink or ReadModels`               |
| `phases_mixed.go`      | `EventSink/EventSource == nil`           | `mixed workload phase skipped: bundle missing EventSink or EventSource`       |
| `phases_mixed.go`      | `len(refs) == 0`                         | `mixed workload phase: skipped (no streams written by prior phases)`          |
| `phases_durability.go` | no DiskSizer/DiskPath                    | `durability phase: disk size not measured (no DiskSizer and no DiskPath set)` |
| `phases_durability.go` | recovery no EventSource                  | `recovery phase: skipped (reopened bundle has no EventSource)`                |

**10 config-flag skips wired:** All `phaseSteps()` skip=true entries now record the phase name in `SkippedPhases`.

### 3. Warning visibility in ALL output formats

- **Text comparison table** (`PrintComparison`): Post-table "Phase Coverage / Warnings" section with per-backend `⚠ backend: message` lines
- **Markdown** (`PrintMarkdown`): Post-table `> ⚠ **backend**: message` blockquotes
- **Single-result report** (`PrintReport`): Dedicated "Skipped Phases" and "Warnings" sections at the bottom
- **JSON** (`WriteComparisonJSON`): `skippedPhases` and `warnings` fields in Result struct (auto-serialized)

### 4. Strict mode for CI gates

- Added `Strict bool` to `Config` — when true, any skipped phase causes `ErrStrictSkip` error
- Added `ErrStrictSkip` sentinel to `errors.go` (Rejection family)
- Added `--strict` flag to `cqrs-bench` CLI (wired into all 3 Config construction sites: run, compare, sweep)

### 5. Warning tests (4 tests, all pass with -race)

- `TestSkippedPhases_MetaEngineMissing` — verifies metaengine skip recorded for memory backend
- `TestSkippedPhases_ConfigFlags` — verifies config skip flags populate SkippedPhases
- `TestStrictMode_FailsOnSkip` — verifies Strict mode returns ErrStrictSkip
- `TestStrictMode_ConfigSkipAlsoFails` — verifies config skips also trigger ErrStrictSkip

### 6. bbolt code quality cleanup

- Replaced custom `hasPrefix` with `bytes.HasPrefix` in `load.go`, `kv_adapter.go`, `kv_iterator.go`
- Removed unused `StoreOption` type from `store.go`
- Deleted dead OTel code (`otel.go` — 41 lines of never-called span helpers)
- Wrapped ALL `db.View`/`db.Update` errors with `wrapBucketErr` across `load.go`, `journal.go`, `kv_adapter.go`, `snapshot.go`, `checkpoint.go`
- Ran `go mod tidy` on both `storage/bbolt` and `stack/bbolt` (removed unused otel dep, resolved missing deps)

### 7. Versioned read benchmark phase (NEW — addresses critical ES gap)

- New `phases_versioned.go` — benchmarks `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`
- Registered in `phaseSteps()` as "versioned read phase" (skipped when `SkipReads` is true)
- New Result fields: `LoadFromVersionLatency`, `LoadToVersionLatency`, `LoadToTimestampLatency`
- Renders in single-result report as "Versioned Reads" section
- Guarded with nil-check + skip warning (same pattern as all other phases)
- Verified working with live benchmark run: memory backend shows P50=230ns for LoadFromVersion

### 8. Reverted default compare backends

- Changed from `memory,sqlite,bbolt,pebble` back to `memory,sqlite,pebble` (bbolt is opt-in)

### 9. Formatted all modified files

- `gofumpt -w` + `goimports -w` on all bbolt, benchkit, and cqrs-bench files

---

## b) PARTIALLY DONE

### bbolt module (compiles, tests pass, but missing features vs Pebble)

- **EventStore** — works (Save, AppendBatch, Load, LoadFromVersion, LoadToVersion, LoadToTimestamp, ReadAll, ReadFrom)
- **SnapshotStore** — works (Save, Load, Delete)
- **CheckpointStore** — works (Save, Load)
- **KVAdapter** — works (Get, Has, Set, Delete, Batch, Iterator, SetIfAbsent)
- **Backend facade** — works (Open, NewBackend, Close, GracefulClose, DiskUsage)
- **stack/bbolt preset** — works (New, returns Bundle with all components)
- **go.mod** — tidied, deps resolved

### Benchmark warning system (works, but more phases to add)

- Versioned read phase added and working
- Checkpoint latency phase NOT yet added
- AppendBatch phase NOT yet added

---

## c) NOT STARTED

1. **AGENTS.md update** — 0 mentions of bbolt in module list, structure tree, module count
2. **api-stability modules list** — bbolt modules not registered
3. **api-stability golden regen** — not run
4. **README.md** for `storage/bbolt/` — not written
5. **README.md** for `stack/bbolt/` — not written
6. **Contract test suites** — eventtest, kv/viewstoretest not run against bbolt
7. **WithDurability** in stack/bbolt — hardcoded to DurabilityStrict
8. **CommandStore** in storage/bbolt — not implemented
9. **QueryStore** in storage/bbolt — not implemented
10. **Streaming iterators** — `event.StreamingSource`, `event.StreamingJournal` not implemented
11. **Module tagging** — neither module tagged, can't resolve outside workspace
12. **nix run .#verify** — full CI gate not run
13. **nix run .#lint** — lint findings not checked
14. **nix run .#check-layers** — dependency budget not verified
15. **nix run .#check-duplication** — duplication vs Pebble not checked
16. **--list-phases flag** — not implemented
17. **Phase coverage matrix** — grid view not implemented
18. **Checkpoint latency benchmark** — not added
19. **AppendBatch benchmark** — not added
20. **Byte-slice clone safety audit** — not done
21. **bbolt pattern examples** in AGENTS.md — not written
22. **go-output dep added to production deps table** — `go.etcd.io/bbolt` not in deps table

---

## d) TOTALLY FUCKED UP

### 1. Build was BROKEN when session started (FIXED)

`cloneBytes` was called 4 times across 2 files but never defined in committed code. The prior session claimed "builds, tests pass" — this was FALSE. The module had never compiled in its committed state. Fixed by replacing with `slices.Clone`.

### 2. Syntax errors during error wrapping (FIXED)

When wrapping `return events, err` with `wrapBucketErr(...)`, my edits accidentally consumed function doc comments (e.g., `// LoadFromVersion returns events...` became `} returns events...`). This broke 4 functions across `load.go`, `checkpoint.go`, `snapshot.go`. Fixed by restoring each comment with proper `//` prefix and closing braces.

### 3. stack/bbolt/go.mod was empty (FIXED)

The committed go.mod had only `go-error-family` as a dependency — missing `stack/v4`, `storage/bbolt/v4`, `watermill/v4`. Fixed by running `go mod tidy`.

### 4. Prior status report was dishonest (DOCUMENTED)

The report at `docs/status/2026-08-06_14-43_bbolt-backend-and-kv-store-evaluation.md` claimed all tasks done and tests passing. This was false at commit time. The current report documents the actual state.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Wire MetaEngine into at least one stack preset** so the MetaEngine phase actually runs in comparisons (currently skipped for ALL backends — only a consumer-provided option enables it)
2. **Add checkpoint Save/Load latency benchmark** — checkpoints are critical for projection recovery but only measured implicitly via the projection phase
3. **Add AppendBatch benchmark** — batch writes are a fundamentally different performance profile from single Save calls
4. **Add phase coverage matrix** to comparison output — a visual grid showing which phases ran (green) vs skipped (yellow) vs errored (red) per backend

### Process

5. **Run `go build` before every commit** — the daemon committed broken code. The BuildFlow hook was supposed to catch this but failed.
6. **Never claim GREEN without running verify** — prior session's stale GREEN claim masked a broken build for multiple commits.

---

## f) Up to 50 Things to Do Next

### Critical (blocks release)

1. Run `nix run .#lint` and fix findings in bbolt/benchkit files
2. Run `nix run .#check-layers` for bbolt dependency budget
3. Run `nix run .#check-duplication` for bbolt vs pebble clone detection
4. Run `nix run .#verify` (full CI gate)

### Documentation

5. Update AGENTS.md module list (add `storage/bbolt`, `stack/bbolt`)
6. Update AGENTS.md structure tree
7. Update AGENTS.md module count (69 → 71)
8. Add `go.etcd.io/bbolt` to AGENTS.md production deps table
9. Write README.md for `storage/bbolt/`
10. Write README.md for `stack/bbolt/`
11. Add bbolt pattern examples to AGENTS.md Key Patterns section
12. Add bbolt to the storage guide in `docs/`

### API stability

13. Add `storage/bbolt` and `stack/bbolt` to `cmd/api-stability/main.go` modules list
14. Regenerate api-stability golden file
15. Run `TestEveryGoModDirIsInModulesList` to verify

### Contract tests

16. Run eventtest contract suite against bbolt EventStore
17. Run kv/viewstoretest contract suite against bbolt KVAdapter
18. Run stack/contracttest contract suite against bbolt Bundle
19. Add bbolt to cross-backend contract test matrix

### Feature parity (bbolt vs Pebble)

20. Wire `WithDurability` into stack/bbolt preset
21. Implement CommandStore in storage/bbolt
22. Implement QueryStore in storage/bbolt
23. Implement `event.StreamingSource` (LoadStream, ReadStream)
24. Implement `event.StreamingJournal` (streaming ReadAll/ReadFrom)
25. Add `WithNoSync` / `WithNoFreelistSync` bbolt options
26. Add `WithBucketPrefix` for multi-tenant namespacing

### Benchmark gaps

27. Add checkpoint Save/Load latency benchmark phase
28. Add AppendBatch benchmark phase (batch write throughput)
29. Add phase coverage matrix to comparison output
30. Wire MetaEngine into at least one stack preset
31. Add `--list-phases` flag to cqrs-bench
32. Add concurrent write contention benchmark (isolated for bbolt single-writer)

### Code quality

33. Audit bbolt []byte returns for buffer-reuse safety (bbolt reuses buffers)
34. Verify error wrapping covers ALL db.Update/db.View paths (not just the outer returns)
35. Check file line counts (max 350 lines/file CI rule)

### Release readiness

36. Tag `storage/bbolt/v4.0.0`
37. Tag `stack/bbolt/v4.0.0`
38. Verify modules resolve with `GOWORK=off`
39. Verify bbolt builds with `CGO_ENABLED=0`
40. Run nix-based integration tests if applicable

### Benchkit polish

41. Add versioned read metrics to comparison table columns
42. Add versioned read metrics to markdown output
43. Add more warning tests (projection skip, snapshot skip, query skip, journey skip)
44. Add test for --strict flag via CLI
45. Add test for markdown warning rendering

### Documentation polish

46. Update benchkit README with new phases (versioned read) and warning system
47. Document the Strict flag in benchkit README
48. Document the SkippedPhases/Warnings fields in benchkit types
49. Add warning system to benchkit CHANGELOG
50. Update cqrs-bench help text to include --strict flag in usage examples

---

## g) Questions (cannot figure out myself)

### 1. Should bbolt be full feature parity with Pebble, or benchmark-only?

Pebble has CommandStore, QueryStore, streaming iterators, WithDurability. Bringing bbolt to full parity is ~500+ lines. The alternative is keeping bbolt as benchmark-only (EventStore + SnapshotStore + CheckpointStore + KV) with explicit documentation. I cannot decide because it depends on whether consumers should use bbolt in production, or whether it exists solely for B+tree vs LSM benchmark data.

### 2. Should versioned read metrics appear in the comparison table?

The comparison table has 13 columns already (WriteP50, WriteP99, LoadP50, LoadP99, ColdP50, GCMaxPau, TailR, A/op, WrtAmp, CoV%, Heap, Disk, Integrity). Adding 3 versioned read columns would make it 16 columns — potentially too wide for terminal display. Options: (a) add them, (b) show them only in single-result report (current), (c) add a `--verbose` flag that includes extra columns. This is a UX decision.

### 3. Should the cqrs-bench factory also add MetaEngine to presets that support it?

Currently no stack preset wires a MetaEngine by default. The MetaEngine phase is skipped for ALL backends in comparisons. We could add `stack.WithMetaEngine(sqliteMemoryEngine)` to the sqlite factory case so the MetaEngine phase actually runs. But this changes what the benchmark measures (adds metaengine overhead to the sqlite backend). This is a design decision about what "benchmarking a backend" means.
