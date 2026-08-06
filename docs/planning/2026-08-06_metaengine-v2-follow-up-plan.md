# Metaengine v2 — Follow-Up Plan: Wiring, Polish, Production Readiness

**Date:** 2026-08-06
**Status:** Planning
**Predecessor:** `docs/status/2026-08-06_23-05_metaengine-v2-all-phases-complete.md`
**Verification basis:** All claims below verified by reading actual source code, running builds, and running tests in this session.

---

## What's Actually Broken vs What Works

### Working (verified green this session)

| Component                   | Evidence                                                                                                   |
| --------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `record/` module            | Builds, 5 tests pass. `Record`, `CommonMetadata`, `StreamRef` types are clean.                             |
| `metaengine/record_fold.go` | `OnRecord`/`OnRecordTyped`/`ApplyRecord`/`RecordAwareFold` — 3 tests pass.                                 |
| `metaengine/auto_fold.go`   | `AutoInsert`/`AutoDelete`/`AutoUpdate`/`AutoCRUD` — 6 tests pass. Reflection-based, pre-computed mappings. |
| `metaengine/sqliteengine/`  | Full 18-file module. Extracted from core. Builds + tests green.                                            |
| `metaengine/graphadapter/`  | Wraps `graph.MemoryDriver` as Engine. Tested.                                                              |
| `metaengine/badgerengine/`  | All 8 core ADTs pass `adttest.RunMatrix`. Implements `Calibratable`.                                       |
| Tombstone deprecation       | 7 symbols marked `// Deprecated:` + migration guide. Functional, not removed.                              |
| ADRs 0111–0119              | All written with rationale.                                                                                |

### Broken / Incomplete (verified this session)

| Gap                                                                                        | Severity       | Evidence                                                                                                                                |
| ------------------------------------------------------------------------------------------ | -------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| **`record/` is orphaned** — `event.Event` and `command.Command` do NOT reference it at all | CRITICAL       | Zero references to `record/` in `event/` or `command/`. Not in their go.mod files.                                                      |
| **`projectionadapter.Handle()` calls `store.Apply()`, NOT `store.ApplyRecord()`**          | CRITICAL       | `adapter.go:129` — `OnRecord` folds get zero-valued Records through the real pipeline. Dead code in production.                         |
| **`auto_fold.go` creates plain folds, not Record-aware**                                   | HIGH           | `AutoInsert` etc. use `insertFold` directly but never set `recordSetter`. No metadata access in auto-generated folds.                   |
| **3 design docs are stale**                                                                | HIGH           | project-definition, design, assumptions — all describe pre-record `any`-blob architecture. Addendums exist but body text is misleading. |
| **AGENTS.md missing `record/` module**                                                     | MEDIUM         | Not in modules list, not in module tree. AI sessions can't discover it.                                                                 |
| **No end-to-end integration test**                                                         | MEDIUM         | No test proves: event → projectionhost → projectionadapter → metaengine Store with Record context. Pieces exist, never proven together. |
| **No example uses new APIs**                                                               | MEDIUM         | Zero references to `record/`, `AutoInsert`, `OnRecord` in `example/`.                                                                   |
| **Badger calibration is guesswork**                                                        | LOW            | Constants copied from Pebble. `Calibratable` is implemented but never benchmarked.                                                      |
| **Dgraph engine is paper-only**                                                            | LOW (deferred) | ADR-0119 only. No code. Acceptable.                                                                                                     |
| **Naming-convention inference missing**                                                    | LOW            | `AutoCRUD` requires explicit C/U/D type params. No `*Created`/`*Deleted` suffix matching.                                               |

---

## Pareto Analysis

### The 1% that delivers 51%

**Wire `record/` into the real pipeline via adapter functions + projectionadapter fix.**

The `record/` module, `OnRecord` folds, and `ApplyRecord` are ALL dead code in production right now because `projectionadapter.Handle()` calls `store.Apply()` (not `ApplyRecord`), and nothing converts `event.Event` → `record.Record`. Fixing this two-part gap unlocks the entire ES-native vision with zero breaking changes.

### The 4% that delivers 64%

**Adapter wiring + AGENTS.md update + design doc refresh.**

After the code fix, updating the documentation ensures the next session and consumers can actually discover and use the new APIs.

### The 20% that delivers 80%

**Adapter + docs + integration test + auto-fold Record awareness.**

This proves the pipeline works end-to-end and makes auto-generated folds metadata-aware.

---

## Verschlimmbessern Risk Assessment

| Work Item                              | Risk                                                   | Mitigation                                                                            |
| -------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| `event.AsRecord()` adapter             | LOW — additive function, no existing API touched       | Pure conversion function, no embedding change                                         |
| `projectionadapter` ApplyRecord switch | MEDIUM — changes hot path of every projection          | Keep `Apply()` as fallback when no Record-aware folds exist; add `ApplyRecord()` path |
| `auto_fold` Record awareness           | LOW — additive `recordSetter` on existing fold structs | Set `recordSetter` in AutoInsert/AutoUpdate; auto-delete doesn't need it              |
| Design doc updates                     | LOW — docs only                                        | Review before rewriting; don't delete old content, add "Current Architecture" section |
| AGENTS.md update                       | LOW — docs only                                        | Add `record/` to modules list + tree comments                                         |
| Badger calibration                     | LOW — benchmark only                                   | Run `CalibrateEngine`, update constants                                               |

**Golden rule:** Every task leaves the build green. Run `go build ./...` + `go test` after each logical change.

---

## Task Table (sorted by impact × customer-value ÷ effort)

> All tasks are ≤12 minutes. Dependencies are noted in the "Dep" column.
> "PF" = phase, "Imp" = impact (1-5), "CV" = customer value (1-5), "Eff" = effort (1-5, lower = easier).

### Phase A: Wire `record/` Into the Real Pipeline (THE critical gap)

| ID  | Task                                                                                                                                                                      | PF  | Imp | CV  | Eff | Dep    | Est   |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ------ | ----- |
| A1  | Add `record/v4` to `event/go.mod` (`go get`)                                                                                                                              | A   | 5   | 5   | 1   | —      | 2min  |
| A2  | Write `event.AsRecord(evt Event) record.Record` — maps Type, Payload, StreamID (via StreamRef), StreamType, Version, MetaData (CorrelationID, CausationID, SchemaVersion) | A   | 5   | 5   | 2   | A1     | 10min |
| A3  | Write `event_asrecord_test.go` — verify all fields map correctly, zero-value fallbacks, round-trip consistency                                                            | A   | 5   | 5   | 1   | A2     | 10min |
| A4  | Add `record/v4` to `metaengine/projectionadapter/go.mod` (`go get`)                                                                                                       | A   | 5   | 5   | 1   | —      | 2min  |
| A5  | Change `projectionadapter.Handle()`: build `record.Record` via `event.AsRecord(evt)`, then call `store.ApplyRecord()` instead of `store.Apply()`                          | A   | 5   | 5   | 2   | A2, A4 | 10min |
| A6  | Write `projectionadapter/adapter_record_test.go` — verify OnRecord folds receive real StreamID/Version/MetaData through adapter                                           | A   | 5   | 5   | 2   | A5     | 12min |
| A7  | Verify build + test: `go build ./metaengine/projectionadapter/... && go test ./metaengine/projectionadapter/...`                                                          | A   | 5   | 5   | 1   | A6     | 5min  |

### Phase B: Auto-Fold Record Awareness

| ID  | Task                                                                                                                       | PF  | Imp | CV  | Eff | Dep | Est   |
| --- | -------------------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | --- | ----- |
| B1  | Set `recordSetter` on `insertFold` in `AutoInsert` — capture a `*record.Record` pointer, stamp metadata fields into result | B   | 4   | 4   | 2   | A2  | 10min |
| B2  | Set `recordSetter` on `updateFold` in `AutoUpdate` — same pattern                                                          | B   | 4   | 4   | 1   | B1  | 5min  |
| B3  | Write `auto_fold_record_test.go` — verify auto-generated folds can access `rec.MetaData.ServerStoredAt` etc.               | B   | 4   | 4   | 2   | B2  | 10min |
| B4  | Verify build + test: `go test ./metaengine/ -run Auto`                                                                     | B   | 4   | 4   | 1   | B3  | 3min  |

### Phase C: Documentation (kill the misinformation)

| ID  | Task                                                                                                                                          | PF  | Imp | CV  | Eff | Dep   | Est   |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ----- | ----- |
| C1  | Add `record/` to AGENTS.md modules row (Quick Reference table)                                                                                | C   | 3   | 4   | 1   | —     | 5min  |
| C2  | Add `record/` entry to AGENTS.md module structure tree with description                                                                       | C   | 3   | 4   | 1   | C1    | 5min  |
| C3  | Add `metaengine/sqliteengine/`, `metaengine/badgerengine/`, `metaengine/graphadapter/` to AGENTS.md modules list                              | C   | 3   | 4   | 1   | —     | 5min  |
| C4  | Update `docs/planning/meta-engine-project-definition.md` — add "Current Architecture (v2)" section referencing record/, OnRecord, ApplyRecord | C   | 4   | 3   | 2   | —     | 12min |
| C5  | Update `docs/planning/meta-engine-design.md` — add "Current Architecture (v2)" section, note record/ as foundation                            | C   | 4   | 3   | 2   | —     | 12min |
| C6  | Update `docs/planning/meta-engine-assumptions-and-query-planning.md` — add v2 section noting Record-typed folds                               | C   | 3   | 3   | 2   | —     | 10min |
| C7  | Run `cmd/doc-check` to verify all Go import paths in updated docs are valid                                                                   | C   | 3   | 3   | 1   | C4-C6 | 5min  |

### Phase D: Integration Test (prove the pipeline works end-to-end)

| ID  | Task                                                                                                                                                                         | PF  | Imp | CV  | Eff | Dep    | Est   |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ------ | ----- |
| D1  | Write `metaengine/projectionadapter/integration_test.go` — full pipeline: MemoryStore → Plan → Adapter → Handle(event) → ExecuteTyped query → verify result + Record context | D   | 5   | 5   | 3   | A6     | 12min |
| D2  | Add test case: OnRecord fold + projectionadapter.Handle() → verify fold receives real StreamID from event.Event                                                              | D   | 5   | 5   | 2   | D1     | 10min |
| D3  | Add test case: AutoInsert + projectionadapter.Handle() → verify auto-fold works through the pipeline                                                                         | D   | 4   | 4   | 2   | D1, B3 | 10min |
| D4  | Verify: `go test ./metaengine/projectionadapter/... -count=1 -v`                                                                                                             | D   | 4   | 4   | 1   | D3     | 3min  |

### Phase E: API Stability + Module Registration

| ID  | Task                                                                                         | PF  | Imp | CV  | Eff | Dep    | Est  |
| --- | -------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ------ | ---- |
| E1  | Add `record/` to `cmd/api-stability/main.go` modules list                                    | E   | 4   | 3   | 1   | —      | 3min |
| E2  | Regenerate api-stability golden: `cd cmd/api-stability && GOWORK=off go run main.go -update` | E   | 4   | 3   | 1   | E1, A2 | 5min |
| E3  | Verify `TestEveryGoModDirIsInModulesList` passes                                             | E   | 4   | 3   | 1   | E1     | 3min |

### Phase F: Badger Calibration (production readiness)

| ID  | Task                                                                                                              | PF  | Imp | CV  | Eff | Dep | Est   |
| --- | ----------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | --- | ----- |
| F1  | Write `badgerengine/calibration_bench_test.go` — BenchmarkSet/Get/CounterIncrement patterns (mirror pebbleengine) | F   | 3   | 3   | 2   | —   | 12min |
| F2  | Run `go test -bench=. ./metaengine/badgerengine/ -count=5` and record results                                     | F   | 3   | 3   | 1   | F1  | 10min |
| F3  | Update `BadgerNsPerOp`/`BadgerNsPerRead`/`BadgerNsPerWrite` constants with measured values                        | F   | 3   | 3   | 1   | F2  | 5min  |
| F4  | Update ADR-0118 with measured calibration results                                                                 | F   | 2   | 2   | 1   | F2  | 5min  |

### Phase G: Naming-Convention Inference (ADR-0116 Layer 1 enhancement)

| ID  | Task                                                                                                                                 | PF  | Imp | CV  | Eff | Dep | Est   |
| --- | ------------------------------------------------------------------------------------------------------------------------------------ | --- | --- | --- | --- | --- | ----- |
| G1  | Write `metaengine/auto_naming.go` — `AutoCRUDByConvention[R]()` that scans event types for `*Created`/`*Updated`/`*Deleted` suffixes | G   | 3   | 4   | 3   | B4  | 12min |
| G2  | Write `auto_naming_test.go` — verify suffix matching, ambiguous-case error, no-suffix fallback                                       | G   | 3   | 4   | 2   | G1  | 12min |
| G3  | Verify build + test                                                                                                                  | G   | 3   | 4   | 1   | G2  | 3min  |

### Phase H: Example App (consumer-facing proof)

| ID  | Task                                                                                                            | PF  | Imp | CV  | Eff | Dep    | Est   |
| --- | --------------------------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ------ | ----- |
| H1  | Create `example/metaengine-quickstart/` module skeleton (go.mod, main.go)                                       | H   | 3   | 5   | 2   | D4     | 10min |
| H2  | Write main.go: define 3 event types + 1 view type, use `AutoCRUD`, wire through projectionadapter, query result | H   | 3   | 5   | 3   | H1, D4 | 12min |
| H3  | Add to go.work + api-stability modules list                                                                     | H   | 2   | 3   | 1   | H1     | 5min  |
| H4  | Verify build: `go build ./example/metaengine-quickstart/...`                                                    | H   | 2   | 3   | 1   | H2     | 3min  |

### Phase I: Deferred / Future (not actionable now)

| ID  | Task                                                                                         | PF  | Imp | CV  | Eff | Dep   | Est      |
| --- | -------------------------------------------------------------------------------------------- | --- | --- | --- | --- | ----- | -------- |
| I1  | Dgraph engine implementation (needs running cluster) — ADR-0119 design complete              | I   | 2   | 2   | 5   | infra | DEFERRED |
| I2  | Tombstone v5 removal (deprecated in v4, remove in v5)                                        | I   | 2   | 2   | 4   | v5    | DEFERRED |
| I3  | `record.FromCommand()` adapter (mirror of FromEvent, for command lifecycle streams ADR-0117) | I   | 3   | 3   | 2   | A2    | 10min    |

---

## Execution Order (dependency-respecting)

```
Phase A (A1→A7) — THE critical path, unlocks everything
    ├── Phase B (B1→B4) — auto-fold Record awareness
    ├── Phase D (D1→D4) — integration test
    │     └── Phase H (H1→H4) — example app
    └── Phase E (E1→E3) — api-stability

Phase C (C1→C7) — documentation (parallel with A)

Phase F (F1→F4) — calibration (parallel, independent)
Phase G (G1→G3) — naming inference (after B)

Phase I — deferred items
```

**Estimated total (excluding deferred):** ~3.5 hours
**Critical path (Phase A → D):** ~75 minutes
