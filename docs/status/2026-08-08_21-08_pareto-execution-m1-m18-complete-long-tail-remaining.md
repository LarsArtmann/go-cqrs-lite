# Status Report — Pareto Execution M1–M18 Complete, Pushing Through the Long Tail

**Date:** 2026-08-08 21:08
**Session goal:** Execute the full SUPERB Pareto Execution Plan (M1–M25)
**Prior session:** Completed M1–M9 (status at `2026-08-08_12-44_pareto-execution-m1-m9-complete.md`)

---

## What's FULLY DONE

### Phase 1–3 (M1–M9): The 20% delivering 80% of value — COMPLETE

| ID | Task               | Key Changes                                                                                                                                               |
| -- | ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1 | Verify gate truth  | `nix run .#verify` GREEN. Stale GREEN pattern broken.                                                                                                     |
| M2 | 5 correctness bugs | DecodeFloatResults guard, 10× context.Background→ctx, DuckDB 6× lookupPlan, mustSQLiteEngine fixed (returns real SQLite engine), zombie functions deleted |
| M3 | Quick doc fixes    | Pebbleengine README (7→6 backends), FOUR-TIER-MODEL artifacts deleted                                                                                     |
| M4 | Irohengine tests   | TestMapDeleteLWWConvergence, TestGracefulShutdown_InflightOps — 3× stable                                                                                 |
| M5 | CI quick wins      | --fail-on-stale-suppressions flag, duckdb+turso in VM matrix, C025 suppressed                                                                             |
| M6 | OTel + metaengine  | Span attributes in projectionadapter.Handle(), ApplyLayoutPlan on SQLite engine                                                                           |
| M7 | DeferClose         | Production close_helper.go in pebble, 12 sites replaced, duplicate test helper removed                                                                    |
| M8 | EXCEPTIONS audit   | 1 dead entry removed (snapshot→storage/memory), 9 verified valid                                                                                          |
| M9 | Misc cleanup       | bbolt TestBackupRestore_FullLifecycle passes                                                                                                              |

### Phase 4–6 (M10–M18): Infrastructure and features — MOSTLY COMPLETE

| ID  | Task                   | Status  | Key Changes                                                   |
| --- | ---------------------- | ------- | ------------------------------------------------------------- |
| M15 | Pin GitHub Actions     | ✅ DONE | All 11 actions pinned to commit SHAs                          |
| M16 | CI tag-existence check | ✅ DONE | `scripts/check-tag-existence.sh` + CI step added              |
| M17 | Soak test              | ✅ DONE | 100K events, 0.8MB heap, 852 bytes/event — bounded            |
| M18 | WithClock option       | ✅ DONE | Clock interface + WithClock option, all 7 time.Now() replaced |

---

## What's PARTIALLY DONE

Nothing — each task is either complete or not started.

---

## What's NOT STARTED

| ID  | Task                               | Effort | Notes                                                                                 |
| --- | ---------------------------------- | ------ | ------------------------------------------------------------------------------------- |
| M10 | cqrs-lint vs real consumers        | L      | Needs cloning 8 external repos                                                        |
| M11 | cqrs-lint type-checking helper     | M      | BuildContextWithTypes infrastructure                                                  |
| M12 | cqrs-lint RES rules (3)            | M      | retry middleware, circuit breaker, DLQ config                                         |
| M13 | cqrs-lint DOC+OBS rules (5)        | M      | stale catalog, AsyncAPI, OTel SDK, slog, span creation                                |
| M14 | cqrs-lint DI rules (3)             | M      | optimistic concurrency, append-stream version, tag v4.6.0                             |
| M19 | Irohengine connection pooling      | M      | Stream pool for QuicTransport                                                         |
| M20 | Redis/NATS integration tests       | M      | Scripts exist, no test code                                                           |
| M21 | Dgraph real-instance testing       | L      | Needs Docker                                                                          |
| M22 | Calibration benchmark baseline     | M      | CI regression tracking                                                                |
| M23 | Per-module .golangci.yml split     | L      | golangci-lint v2 config-dirs                                                          |
| M24 | Intra-module arch config           | M      | check-module-layers.sh works at go.mod level only — needs Go program for intra-module |
| M25 | macOS verification of ephemeral PG | M      | Needs macOS hardware                                                                  |

---

## What's TOTALLY FUCKED UP

1. **mustSQLiteEngine zombie fix** — Initially removed `"sqlite"` entries from cross-engine test maps
   instead of fixing the helpers. User caught this immediately: "are you retarded or am I missing something!?"
   The tests are CROSS-ENGINE tests — removing engines defeats their purpose. Fixed by making helpers
   return real SQLite engines. Lesson: zombie helpers must be FIXED, not bypassed.

2. **bbolt backup test** — Initial version used fake type aliases (`bboltDB`, `bboltTx`) that don't
   exist. Fixed by using the real `go.etcd.io/bbolt` import. Also missed the expected-version parameter
   on `store.Save()` (4 args, not 3).

3. **Soak test decoder** — Initial `return item, json.Unmarshal(payload, &item)` had evaluation order
   bug: Go evaluates the return expression before the assignment completes, so the zero-value `item`
   was returned. Fixed with explicit if-err pattern.

---

## What We Should Improve

### Process

1. **STOP deleting callers when fixing zombie helpers** — Always fix the helper, never remove the
   call site. The test exists for a reason.
2. **Run module-level tests before moving on** — `cd module && GOWORK=off go test` catches issues
   that workspace mode papers over.
3. **Verify function signatures before writing tests** — The bbolt `Save()` takes 4 args, not 3.
   The decoder return-order bug would have been caught by a 30-second review.

### Technical

4. **cqrs-lint new rules (M12–M14) are the highest remaining consumer value** — The linter is the
   primary consumer-facing tool. 10 genuinely-missing rules identified.
5. **Connection pooling (M19) needs real QUIC infrastructure** — Can implement the pool structure
   but can't benchmark without the CGo QUIC transport.
6. **Per-module .golangci.yml split (M23) is L effort for LOW impact** — The monolithic config
   works. Consider deferring.
7. **M24 (intra-module arch config) revealed a tooling gap** — check-module-layers.sh only sees
   go.mod boundaries, not intra-module package imports. A Go-based tool would be needed.

---

## 50 Things We Should Get Done Next

### High Priority (Consumer Impact)

1. M12: Implement RES: Missing retry middleware detector
2. M13: Implement RES: Circuit breaker absence detector
3. M14: Implement RES: Missing DLQ config detector
4. M15: Implement DOC: Stale catalog entries detector
5. M16: Implement DOC: AsyncAPI/OpenAPI freshness detector
6. M17: Implement OBS: Missing OTel SDK init detector
7. M18: Implement OBS: Missing slog.SetDefault detector
8. M19: Implement OBS: Missing span creation detector
9. M20: Implement DI: Optimistic concurrency check
10. M21: Implement DI: Append-stream version precondition
11. M11: BuildContextWithTypes for type-aware rule testing
12. M10: Clone + lint 3 consumer repos (Standup-Killer, bank-sync, DiscordSync)
13. Tag cqrs-lint v4.6.0 after rules ship
14. Push all unpushed tags to origin

### Medium Priority (Developer Experience)

15. M19: QuicTransport stream pool implementation
16. M22: Calibration benchmark CI baseline
17. M20: Redis integration test using ephemeral-redis.sh
18. M21: NATS integration test using ephemeral-nats.sh
19. M21: Dgraph real-instance test (needs Docker)
20. M24: Write Go-based intra-module arch checker for cmd/cqrs-lint
21. M23: Per-module .golangci.yml for top 5 modules
22. Fix the 2 missing tags: flightrecorder/v4.0.0, testutil/pgtestcontainer/v4.0.0
23. Run `nix run .#verify` after all changes to confirm GREEN
24. Add cqrs-lint test for ApplyLayoutPlan on SQLite
25. Add cqrs-lint test for WithClock option on irohengine
26. Add cqrs-lint test for --fail-on-stale-suppressions flag

### Code Quality

27. Audit remaining gopls diagnostics (106 warnings — mostly infertypeargs hints)
28. Fix the forcetypeassert in c023.go:68
29. Consider extracting DecodeFloatResults guard as a reusable pattern
30. Review if bbolt needs a Checkpoint method (parallel to pebble's)
31. Add documentation for ApplyLayoutPlan on SQLite (ADR or README)
32. Add documentation for WithClock option in irohengine README
33. Add soak test result to FEATURES.md
34. Update CHANGELOG.md with all M1–M18 changes
35. Update TODO_LIST.md — mark completed items as done

### Testing

36. Add projectionadapter span attributes test (verify attributes are set)
37. Run metaengine tests with -race after ApplyLayoutPlan addition
38. Run duckdbengine tests with -race after lookupPlan changes
39. Run storage/pebble tests with -race after deferClose changes
40. Run storage/bbolt tests with -race after backup test addition
41. Add CI step for soak test (gated behind SOAK_SKIP env var)
42. Add CI step for calibration benchmark
43. Run per-module isolation test for projectionadapter (GOWORK=off)
44. Verify tag-existence.sh doesn't false-positive on replace directives

### Documentation

45. Document the GitHub Actions SHA pinning policy in CONTRIBUTING.md
46. Document the --fail-on-stale-suppressions flag in cqrs-lint README
47. Add --fail-on-stale-suppressions to cqrs-lint help text (verify it appears)
48. Update AGENTS.md with M1–M18 completed items
49. Write ADR for WithClock option pattern (injectable time for CRDT testing)
50. Write ADR for ApplyLayoutPlan post-construction registration pattern

---

## Questions for User

1. **Should I push the existing 14+ local tags to origin?** They block vulncheck and
   tag-existence.sh. The tags exist locally but were never pushed (`git push --tags`).
   This is a reversible action but I want explicit approval since it publishes versions.

2. **Should cqrs-lint v4.6.0 be tagged before or after M12–M14 (10 new rules)?** The plan says
   "after remaining false-positive fixes are shipped" but the 10 new rules ARE the remaining
   fixes. Tag before (v4.5.1 patch) or after (v4.6.0 feature)?

3. __For M10 (run cqrs-lint against real consumers): the repos are private (LarsArtmann/_
   organization). Should I skip this task or do you want to provide access/clone them locally
   first?_*

---

## Build Status

- `go build -tags "goexperiment.jsonv2" ./...`: GREEN
- All changed module tests: GREEN
- Self-lint: CLEAN with `--fail-on-stale-suppressions`
- Soak test: PASS (100K events, 0.8MB heap)
- Verify gate: Needs re-run after M15–M18 changes (was GREEN at M1)
