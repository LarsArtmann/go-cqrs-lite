# Status Report: TODO List Code-Verified Audit

**Date:** 2026-08-08 00:02
**Session scope:** Verify every TODO_LIST.md item against the actual codebase, mark stale `[ ]` items as `[x]` where the code proves they're done, correct inaccurate descriptions.
**Files changed:** `TODO_LIST.md` (10 edits via multiedit, 1 edit via edit for date)

---

## A) FULLY DONE (this session — 9 items newly marked `[x]`)

| # | Section                       | Item                                                | Evidence                                                                                                                                                                                                                                                                                                                                                         |
| - | ----------------------------- | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Metaengine v2 / Test coverage | Record-aware integration test through SQLite engine | `metaengine/sqliteengine/record_stamp_test.go:20` — `TestSQLite_RecordStamping` uses `AutoInsert` + `store.ApplyRecord` through the SQLite engine                                                                                                                                                                                                                |
| 2 | Metaengine v2 / Test coverage | Benchmark `ApplyRecord` overhead                    | `metaengine/projectionadapter/bench_test.go:26` — `BenchmarkHandle_ApplyRecord` + `BenchmarkHandle_AutoInsert` for before/after comparison                                                                                                                                                                                                                       |
| 3 | Metaengine v2 / Code quality  | `record.FromCommand()` adapter                      | Implemented as `command.AsRecord()` at `command/asrecord.go:34`, mirroring `event.AsRecord()` at `event/asrecord.go:41`. Lives in `command/` package following the same pattern as event side. Test at `command/asrecord_test.go`.                                                                                                                               |
| 4 | Irohengine                    | Evaluate `iroh-go` C binding stability              | ADR-0096 (`docs/adr/0096-iroh-distributed-engine-bridge-evaluation.md`) + design doc (`docs/planning/meta-engine-eventual-consistency-and-iroh.md`). Decision: short-term sidecar, long-term CGo FFI.                                                                                                                                                            |
| 5 | Irohengine                    | WriteOp.ID dedup ring                               | Both transports have bounded dedup sets: QUIC (`metaengine/irohengine/quic/stream.go:100`) and loopback (`metaengine/irohengine/loopback/conn.go:86`). Both reset at 10K entries. Previous TODO claim that "loopback does not" was stale.                                                                                                                        |
| 6 | Code Quality                  | Benchmark audit for 10 skipped modules              | All 10 now have benchmark test files: `codec/benchmark_test.go`, `command/benchmark_test.go`, `dispatcher/benchmark_test.go`, `query/benchmark_test.go`, `middleware/benchmark_test.go`, `snapshot/benchmark_test.go`, `listing/benchmark_test.go`, `watermill/benchmark_test.go`, `transport/http/sse_fanout_bench_test.go`, `storage/view/store_bench_test.go` |
| 7 | CI / Release                  | Add `go test` to CI for example/taskmanager         | `per-module-test` CI job (ci.yml:150-168) tests all discovered modules including example/taskmanager via `GOWORK=off go test -tags "goexperiment.jsonv2" ./... -count=1 -race`                                                                                                                                                                                   |
| 8 | Deferred Debt                 | Ghost bus removal (ADR-0028)                        | All three files deleted: `storage/memory/bus.go`, `storage/memory/command_bus.go`, `storage/pg_bus.go` — none exist on disk                                                                                                                                                                                                                                      |
| 9 | Deferred Debt                 | Metadata aliases completion (ADR-0031)              | Both `command.Metadata` (`command/metadata.go:23`) and `query.Metadata` (`query/query.go:48`) are standalone structs with own `Clone()`/`Merge()`/`WithCustom()`. Doc comments confirm: "It is a standalone struct (not a type alias)"                                                                                                                           |

**Plus 1 correction:** Updated cqrs-lint "Migrate global detectors" item from "~20 detectors" to "8 detector files" (6 in `adoption/`, 1 in `api/`) — the original count was overstated.

### Previously marked `[x]` (confirmed correct — 3 items from prior sessions)

| # | Section             | Item                    | Verified Evidence                                                                                                                                                 |
| - | ------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | System Package / P2 | koanf YAML config       | `system/config_loader.go:8-11` (koanf imports), `:59-62` (env merge), ADR-0105                                                                                    |
| 2 | System Package / P2 | DuckDB/PG Transactional | `metaengine/duckdbengine/transaction.go:37`, `metaengine/pgengine/transaction.go:66`, `enginetest/enginetest.go:419` (`RunTransactionalTest`)                     |
| 3 | System Package / P2 | Bus driver registry     | `system/driver_registry.go:77-87` (lookupBusDriver with RLock/RUnlock fix), `:151-154` (gochannel registered generically), `errors.go:20` (`ErrUnknownBusDriver`) |

### Also confirmed done (in prose, not checklist — 2 items)

| # | Item                                                                 | Evidence                                                                |
| - | -------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| 1 | `#verify-fast` as pre-merge CI gate                                  | Wired as `verify-fast-gate` at `ci.yml:128` (Declined/Rejected section) |
| 2 | `retry/` → `go-retry` + `idempotency/` → `go-idempotency` extraction | Both repos pushed with annotated tags (Deferred Debt section prose)     |

---

## B) PARTIALLY DONE (verified this session)

| # | Section    | Item                               | What's done                                                                                                                                                                                     | What remains                                                                                                                                                                                                            |
| - | ---------- | ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Irohengine | Non-CRDT op rejection on QUIC path | Architecturally done: `metaengine/irohengine/engine.go:169-180` — MapUpdate calls local engine directly, never publishes. Test exists at `replication_test.go:16-40` but uses InProcessNetwork. | No QUIC-specific test verifying MapUpdate non-replication over the QUIC transport                                                                                                                                       |
| 2 | cqrs-lint  | L1.5 domain severity calibration   | `DomainKind` enum (`feature_profile.go:174-180`) + `applyDomainBias` (`filters.go:276-297`) shipped. 7 tests in `domain_bias_test.go`.                                                          | `DomainSecurity` and `DomainInternal` bias paths untested. `applyDomainBias` only handles `DomainFinancial`. Broader testing against financial/security projects needed.                                                |
| 3 | Irohengine | WriteOp.ID dedup ring              | Both QUIC and loopback have dedup sets.                                                                                                                                                         | Neither uses a ring buffer — both use `map[string]struct{}` that resets (not evicts) at 10K entries. The TODO title says "ring" but implementation is a bounded set. This is a naming inaccuracy, not a functional gap. |
| 4 | Dedup      | Clone groups                       | Threshold 3 baseline driven to 0 (per prose). 45 entries in `.art-dupl-baseline.json` are the accepted/intentional clones.                                                                      | 92 threshold-2 groups remain for investigation. Some extractable (`capitalizeFirst`, `truncateString`, `isCBORData`, `recordErr`, `startStreamSpan`).                                                                   |

---

## C) NOT STARTED (verified this session — genuinely pending)

### Metaengine v2 — Publishability (3 items)

- **Tag `metaengine/bench/v4.0.0`** — no git tag exists
- **Tag `metaengine/pebbleengine/v4.0.0`** — no git tag exists
- **Tag drifted modules for GOWORK=off CI** — `retry/v4`, `middleware/v4`, `benchkit/v4`, `stack/*` have API changes since last tag

### Metaengine v2 — Test coverage (5 items)

- Record-aware integration test through Pebble engine — no `OnRecord`/`ApplyRecord` in pebbleengine tests
- Soak test with `AutoCRUDByConvention` — existing soak uses manual folds, not auto-projection
- Add `RunTransactionalTest` to sqliteengine/badgerengine — DuckDB + PG have it, SQLite + Badger do not
- Add concurrent `RunInTx` test — two goroutines, verify isolation
- Add `MultiAdd` + `LogAppend` transactional tests — Multimap and Log backends untested in transactions

### Metaengine v2 — Module health (3 items)

- Add `metaengine/keycodec`, `metaengine/enginetest`, `testutil/pgtestcontainer` to api-stability modules list
- Add same 3 modules to AGENTS.md module list
- Fix 16 COVERAGE GAPs in `check-module-layers.sh`

### System Package — P2 Hardening (6 items)

- `system/README.md` Quick Start doesn't compile
- Fix `cmd/doc-check` cmdguard arg-parsing
- Add `system.HealthCheck(ctx)` method
- Add `system.GracefulClose(ctx)`
- Add `system.ResetProjection(name)`
- Wire checkpoint store as configurable

### bbolt Storage Backend (5 items)

- Add CommandStore contract tests
- Add QueryStore contract tests
- Add same-stream concurrency contention test
- Add bbolt to `stack/bench/`
- Consider `WithBatchSize` option for `AppendBatch`

### Irohengine (4 items)

- QUIC transport integration with `adttest.RunMatrix` — only InProcessNetwork passes the matrix
- Non-CRDT op rejection on QUIC path (test gap, see Partially Done)
- Fix `TestQuicSetConvergence` flakiness
- Fix `TestLoad_ConcurrentLoadsCoalescedBySingleflight` flake

### cqrs-lint (9 items)

- Run cqrs-lint against real consumer projects
- Fix remaining false positives (C001, D012, C008)
- Triage remaining ~199 self-lint WARNING/INFO findings
- Missing regression tests (S006, A018, B004)
- Migrate 8 remaining global detectors to per-module evaluation
- Scorecard SARIF `logicalLocations`
- Deferred P-series rules (4 rules)
- L1.5 domain severity calibration (broader testing)
- ~14 remaining Pareto backlog items

### Code Quality (4 items)

- `metadata.CustomData[K]` immutability gap — `EnsureCustom()` still at `metadata/metadata.go:84`
- `query.WithCustomMetadata` missing — `command/metadata.go:94` has it, query does not
- Stale `metadata/README.md` — still documents `EnsureCustom` at line 54
- Fix `.golangci.yml` exclusion sprawl — ~30 blocks, ~50% undocumented

### Dedup (3 items)

- Investigate threshold-2 clone groups (92 remaining)
- Extract `renderTable(b, headers, rows)` helper
- `deferClose(closer)` helper — no helper exists, pattern repeated across 7 engines

### CI / Release / Infrastructure (4 items + 1 BLOCKED)

- [BLOCKED] Publish go-finding + go-must as tagged modules — `go-finding/pipeline` has zero pseudo-version (no tag)
- Pin GitHub Actions to commit SHAs — all actions use `@vN` tags, zero SHA pins
- Add self-lint to CI
- Add `--fail-on-stale-suppressions` CI gate

### Integration Test Infrastructure (11 items, M34-M48)

- All 11 items confirmed NOT STARTED. No macOS verification, no caching, no profiling, no nixos-container, no DuckDB/SQLite/Turso/Pebble VM tests, no multi-backend contract suite, no Redis/NATS, no aggregator script.

### Layer Enforcement (5 items)

- Rename `FOUR-TIER-MODEL.md` → `SEVEN-TIER-MODEL.md` — file still has old name
- Remove dead `EXCEPTIONS[storage]="listing"` — still at `scripts/check-module-layers.sh:107`
- Fix 16 COVERAGE GAPs
- Expand go-arch-lint to remaining modules — **DISCREPANCY: TODO says "only 6 modules have configs" but sub-agent found ZERO go-arch-lint config files in the entire repo. Either the configs were deleted, or the TODO is wrong.**
- Consider rewriting `check-module-layers.sh` as `cmd/check-layers` — still 330 lines of bash, no Go program

---

## D) TOTALLY FUCKED UP

Nothing is totally fucked up. The build compiles, the verify gate was reported GREEN by the prior session, and no items were found to be broken or counterproductive. The only "fuckup" is the stale TODO list itself — 9 items were done but never marked `[x]`, and at least 2 description inaccuracies persisted (`~20` detectors was actually `8`, loopback "does not" have dedup was wrong).

### Discrepancies found (not fuckups, but worth flagging)

1. **Dedup baseline count mismatch** — TODO prose says "baseline: 0 groups" at threshold 3, but `.art-dupl-baseline.json` contains 45 entries. These 45 are likely the accepted/intentional clones (the baseline records what's allowed, not what's remaining). The prose is technically correct (0 _new_ clone groups at threshold 3) but could be misread.

2. **go-arch-lint config count** — TODO says "only 6 modules have per-module go-arch-lint configs" but zero `.go-arch-lint.yml` files exist in the repo. Either the configs were removed in a recent session, or the TODO claim was never accurate. This needs investigation.

3. **`docs/api_surface.txt` modified** — Git status at conversation start showed this file as modified (`M docs/api_surface.txt`). I did not investigate what changed. If an auto-commit daemon updated it, it may or may not be intentional.

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements

1. **TODO audit cadence** — 9 of ~60 items were stale (done but unmarked). That's a 15% staleness rate. The TODO list should be audited against code at least once per sprint, not left to drift across sessions.

2. **CHANGELOG not updated for newly-verified-done items** — I marked 9 items as `[x]` in TODO_LIST.md but did NOT add CHANGELOG entries for them. The TODO_LIST.md header says "Completed work lives in CHANGELOG.md and is never duplicated here," implying done items should be moved to CHANGELOG and removed from TODO. I only marked them `[x]` — a future session should migrate them to CHANGELOG and remove the `[x]` lines.

3. **Auto-commit daemon created new TODO items I didn't verify** — The TODO_LIST.md was updated by the auto-commit daemon between my first and second read (header changed from "2026-08-07" to "2026-08-07 post publishability hardening"). New items appeared (e.g., "Tag `metaengine/bench/v4.0.0`", "Fix `TestQuicSetConvergence` flakiness"). I verified these but the experience shows the TODO list is a moving target.

4. **No `nix run .#verify` run this session** — I verified code state via grep/read but never ran the full build+test+lint gate. The AGENTS.md explicitly warns about the "stale GREEN" anti-pattern. While I didn't claim GREEN, I also can't confirm it.

5. **Sub-agent depth** — I used 4-6 parallel sub-agents per batch, each checking 3-4 items. This was efficient but some checks could have been deeper (e.g., the go-arch-lint discrepancy, the dedup baseline count).

6. **TODO descriptions should include verification method** — Items like "Evaluate iroh-go C binding stability" are tasks to DO (write an evaluation), not code to verify. I found the evaluation already existed as ADR-0096. But the distinction between "write a thing" and "verify a thing exists" should be clearer in the TODO format.

---

## F) Up to 50 things we should get done next

### High impact (publishability blockers)

1. Tag `metaengine/bench/v4.0.0`
2. Tag `metaengine/pebbleengine/v4.0.0`
3. Tag drifted modules (`retry/v4`, `middleware/v4`, `benchkit/v4`, `stack/*`)
4. Publish `go-finding/pipeline` as tagged module (unblocks BLOCKED item)
5. Add `metaengine/keycodec`, `metaengine/enginetest`, `testutil/pgtestcontainer` to api-stability modules list
6. Add same 3 modules to AGENTS.md module list
7. Fix 16 COVERAGE GAPs in `check-module-layers.sh`

### Metaengine v2 completeness

8. Record-aware integration test through Pebble engine
9. Soak test with `AutoCRUDByConvention` (100K events, auto-projection path)
10. Add `RunTransactionalTest` to sqliteengine tests
11. Add `RunTransactionalTest` to badgerengine tests
12. Add concurrent `RunInTx` test (two goroutines, verify isolation)
13. Add `MultiAdd` + `LogAppend` transactional tests
14. Projectionhost lifecycle test with Record-aware folds through Start/Stop

### System Package hardening

15. Fix `system/README.md` Quick Start (make it copy-pasteable)
16. Add `system.HealthCheck(ctx)` method
17. Add `system.GracefulClose(ctx)` (bounded Close with timeout)
18. Add `system.ResetProjection(name)` (delegate to projectionhost)
19. Wire checkpoint store as configurable (`WithCheckpointStore`)
20. Fix `cmd/doc-check` cmdguard arg-parsing

### bbolt completeness

21. Add CommandStore contract tests (Save/AppendBatch/Load/ReadAll/ReadFrom)
22. Add QueryStore contract tests
23. Add same-stream concurrency contention test (10 goroutines)
24. Add bbolt benchmarks to `stack/bench/`
25. Consider `WithBatchSize` option for `AppendBatch`

### Irohengine hardening

26. QUIC transport `adttest.RunMatrix` parity test
27. Non-CRDT op rejection test on QUIC path
28. Fix `TestQuicSetConvergence` flakiness
29. Fix `TestLoad_ConcurrentLoadsCoalescedBySingleflight` flake

### cqrs-lint trustworthiness

30. Run cqrs-lint against real consumer projects (highest-value non-coding task)
31. Fix C001 false positive (read-only bbolt transactions)
32. Fix D012 false positive (CLI tools excluded)
33. Fix C008 false positive (non-monetary floats)
34. Add regression tests for S006, A018, B004 fixes
35. Migrate 8 remaining global detectors to per-module evaluation
36. Implement Scorecard SARIF `logicalLocations`
37. Implement deferred P-series rules (4 rules)
38. Broader L1.5 domain severity testing (financial/security projects)
39. Triage ~199 self-lint WARNING/INFO findings
40. Add self-lint to CI
41. Add `--fail-on-stale-suppressions` CI gate

### Code quality & cleanup

42. Fix `metadata.CustomData[K]` immutability gap (decision: sweep or accept)
43. Add `query.WithCustomMetadata` (mirror command module)
44. Update stale `metadata/README.md` (EnsureCustom → WithCustom)
45. Fix `.golangci.yml` exclusion sprawl (document ~30 blocks)
46. Extract `renderTable(b, headers, rows)` helper in cqrs-lint explain.go
47. Extract `deferClose(closer)` helper for metaengine engines
48. Rename `FOUR-TIER-MODEL.md` → `SEVEN-TIER-MODEL.md`
49. Remove dead `EXCEPTIONS[storage]="listing"` from check-module-layers.sh
50. Pin GitHub Actions to commit SHAs (72+ unpinned)

---

## G) Questions I cannot answer myself

### 1. Should `[x]` items be removed from TODO_LIST.md and migrated to CHANGELOG.md?

The TODO header says "Completed work lives in CHANGELOG.md and is never duplicated here." But the 3 prior `[x]` items (koanf, Transactional, bus registry) were left in place by the prior session. Should I follow the same pattern (leave `[x]` with evidence) or migrate to CHANGELOG and remove from TODO? **The prior session's convention suggests leaving them, but the header text suggests removing them. Which is correct?**

### 2. The go-arch-lint discrepancy — should I investigate further?

TODO says "only 6 modules have per-module go-arch-lint configs" but zero `.go-arch-lint.yml` files exist in the repo. Either: (a) the configs were deleted in a recent session, (b) they use a different filename, or (c) the TODO was never accurate. **Should I dig into git history to find when they disappeared, or just correct the TODO to say "zero configs exist"?**

### 3. Should I run `nix run .#verify` now to confirm the build is GREEN?

The AGENTS.md warns about the "stale GREEN" anti-pattern. I verified code state via grep/read but never ran the full gate. The prior session claimed "Verify gate GREEN (all 17 steps pass)" in the TODO header. **Should I spend the 3-4 minutes to re-run `nix run .#verify` now, or is the prior session's claim still valid given no code was changed (only documentation)?**

---

## Session summary

- **Items verified:** ~60 (every checklist item in TODO_LIST.md)
- **Items newly marked `[x]`:** 9
- **Descriptions corrected:** 2 (detector count, dedup ring description)
- **Discrepancies found:** 3 (dedup baseline count, go-arch-lint configs, api_surface.txt modification)
- **Files changed:** 1 (`TODO_LIST.md`)
- **Build/lint run:** No (documentation-only session, no code changed)
- **Auto-commit daemon activity:** CHANGELOG.md and ROADMAP.md were modified by the daemon during this session (128+ and 45+ lines respectively). These changes are not mine.

---

## Resolution (2026-08-08)

This audit identified TODO_LIST structural decay (69 completed `[x]` items). The TODO_LIST was **fully rebuilt** later the same day: all 69 completed items deleted, 24 genuinely open items retained, new items harvested from 08-07/08-08 status reports. See current `TODO_LIST.md`.

The 3 discrepancies noted above:

- ~~Dedup baseline count~~ resolved — `.art-dupl-baseline.json` is current
- ~~go-arch-lint configs~~ resolved — repo uses `scripts/check-module-layers.sh` exclusively (zero `.go-arch-lint.yml` files)
- ~~api_surface.txt modification~~ resolved — golden regenerated to 3807+ exports
