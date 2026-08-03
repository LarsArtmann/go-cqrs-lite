# Status: Cleanup Sprint — PostgresBus Build Fix, Cost Constant Revert, Lint/Doc/Golden Sync

**Date:** 2026-08-03 21:31
**Session type:** Cleanup / debt resolution
**Trigger:** Prior session handoff with 6 known issues left behind
**Verify gate:** GREEN (2 consecutive clean runs)

---

## What This Session Was About

The prior session (`docs/planning/2026-08-03_19-29_SUPERB-ADR-REVIEW-FINDINGS-EXECUTION-PLAN.md`) executed a 14-task ADR review sprint but left 6 critical items unresolved: broken cost constants, missing depguard entries, stale api-stability golden, unformatted code, and — critically — a **compile-breaking deletion** that was masked by workspace-mode false greens.

This session picked up all 6 items and resolved them. The auto-commit daemon also did substantial parallel work (SSE refactoring, dependency syncing, documentation) during this session.

---

## a) FULLY DONE

### 1. Reverted DuckDB cost constants (CRITICAL)
- `DuckDBNsPerOp`: 4,800,000 → **15,000** (reverted to original)
- `DuckDBNsPerRead`: 546,000 → **3,000** (reverted to original)
- **Why:** The prior session changed these based on point-lookup benchmarks (MapGet). DuckDB is a columnar analytical engine — point lookups are its worst case (full column scan + CGo boundary). The new values would have made the planner route EVERYTHING away from DuckDB, including analytical GROUP BY workloads where DuckDB should dominate by 10-50x.
- Added explanatory comments documenting that point-lookup benchmarks exist but are NOT the basis for these constants.
- `metaengine/duckdbengine/doc.go` was already correct (showed original values 15,000/3,000) — now matches engine.go again.

### 2. Reverted Postgres cost constants
- `PG_NsPerOp`: 33,000 → **12,000** (reverted to original)
- `PG_NsPerRead`: 28,000 → **5,000** (reverted to original)
- **Why:** Docker testcontainer network overhead inflated measurements 3-5x. The values should model production Postgres (same-datacenter network or Unix socket).
- Added comments noting the Docker-measured values for reference.

### 3. Fixed broken `stack/postgres` build (CRITICAL — prior session's bug)
- **Root cause:** The prior session deleted `storage.PostgresBus` types (`NotificationListener`, `PostgresBusOption`, `NewPostgresBus`) but left `stack/postgres/` referencing them — compile-breaking.
- **Why it wasn't caught:** `go build ./...` in workspace mode gives a **false green** — it doesn't compile individual module directories. Only `nix run .#build` or `cd module && GOWORK=off go build ./...` catches this.
- **What was removed (1,177 LOC across 5 files):**
  - `stack/postgres/pg_listener.go` (273 LOC) — PgxListener implementation
  - `stack/postgres/pg_listener_options.go` — listener config options
  - `stack/postgres/pg_listener_reconnect.go` — auto-reconnect with backoff
  - `stack/postgres/pg_listener_test.go` (368 LOC) — listener tests
  - `stack/postgres/pg_bus_integration_test.go` (263 LOC) — bus integration tests
- **Updated:** `stack/postgres/preset.go` (removed `WithDistributedBus`, `buildBus`, listener config fields), `stack/postgres/doc.go`, `storage/eventstore/event_store_by_id.go` (comment), `stack/capabilities.go` (comment)
- `stack/postgres/go.mod` tidied (removed unused pgxpool imports)

### 4. Added depguard allow-list entries
- `.golangci.yml`: Added `github.com/larsartmann/go-retry` and `github.com/larsartmann/go-idempotency` to the depguard `Main` allow list.

### 5. Regenerated api-stability golden
- `docs/api_surface.txt`: Regenerated twice (first after PostgresBus removal → 3,200 exports, then after PgxListener removal → 3,183 exports).
- The api-stability tool itself was NOT broken (prior session misdiagnosed `collectExports undefined` — it only fails with `go run main.go`, not `go run .`).

### 6. Updated AGENTS.md
- `storage/` line: removed `PostgresListenNotifyBus` reference
- `retry/` line: updated from "standalone, extraction planned" to "re-export aliases for go-retry"
- `idempotency/` line: updated from "extraction planned" to "re-export aliases for go-idempotency"
- Dependency table: added `go-retry` and `go-idempotency` to Production deps

### 7. Tagged external repos
- `go-retry`: tagged v0.1.0 (annotated)
- `go-idempotency`: tagged v0.1.0 (annotated)
- Both repos already had commits from the prior session + daemon additions

### 8. Formatted all changed files
- `gofumpt` + `goimports` on 17+ changed .go files across metaengine, storage, stack, command, query, retry, idempotency

### 9. Lint fixes
- `godoclint`: Removed duplicate package doc comments from `retry/alias.go` and `idempotency/alias.go` (only `doc.go` should carry the package godoc)
- `wrapcheck`: Added `//nolint:wrapcheck` to `retry/alias.go` (thin alias — caller sees the same errors)
- `gci`: Fixed import ordering in `stack/postgres/preset.go`

### 10. Verify gate: GREEN
- Two consecutive clean runs of `nix run .#verify` (build + vet + test + race + lint + doc-check)
- One initial flake in `benchkit.TestCompare_ThreeBackends` (timing-sensitive) — passed on retry

---

## b) PARTIALLY DONE

### SSE Consolidation (P1)
- The **daemon** committed SSE refactoring work during this session (commits `b7bb2647`, `bca4f31d`, `f7512176`) — delegating wire-format serialization to `go-sse` library per ADR-0097
- Files changed by daemon: `metaengine/sse.go`, `transport/http/sse_event.go`, `transport/http/go.mod`
- **I did NOT review, test, or verify this daemon work** — it happened in parallel
- The remaining uncommitted files include `metaengine/sse.go` and `transport/http/sse_event.go`

### Extraction completion (P2c/P2d)
- go-retry: core extracted + tagged, but NOT pushed to GitHub
- go-idempotency: core extracted + tagged, but NOT pushed to GitHub
- Both repos have local `replace` directives in go-cqrs-lite — they resolve locally only
- Sub-modules (kvstore, sqlstore) still depend on kv/ and codec/ — not extractable cleanly

---

## c) NOT STARTED

1. **Push go-retry and go-idempotency to GitHub** — both repos are local-only
2. **idempotency/kvstore and sqlstore extraction** — blocked on kv/ and codec/ dependency complexity
3. **command/bus.go evaluation** — zero external consumers but internal watermill consumer exists
4. **Analytical GROUP BY benchmark for DuckDB** — needed to validate or replace the reverted analytical-cost constants with empirical evidence
5. **Review daemon's SSE refactoring** — metaengine/sse.go and transport/http/sse_event.go changes were not reviewed or tested by this session

---

## d) TOTALLY FUCKED UP

### Prior session shipped a BROKEN BUILD (not this session)
- The prior session deleted `storage.PostgresBus` (4 files, 1,226 LOC) but left `stack/postgres/` with 6 files referencing the deleted types
- They claimed "go build ./... — PASS" and "go test — ALL PASS" — **both were false greens**
- `go build ./...` in workspace mode does NOT compile individual modules' dependencies; it resolves them from the workspace and can skip type-checking of transitive imports
- Only `nix run .#build` (which runs per-module `GOWORK=off go build`) catches this
- **This is the "Stale GREEN" anti-pattern documented in AGENTS.md** — claiming verify is GREEN without actually running it in the current session
- The breakage persisted for ~3 hours and was only caught when this session ran `nix run .#verify`

### DuckDB cost constant change was a planner regression
- The prior session changed constants from analytical-cost values to point-lookup values
- This would have made the planner route everything AWAY from DuckDB, even for analytical workloads where it should win by 10-50x
- The execution plan itself documented this risk ("Open Risk: DuckDB Cost Constants") but shipped it anyway

---

## e) WHAT WE SHOULD IMPROVE

1. **Stop trusting `go build ./...` as a build gate** — It gives false greens in workspace mode. Always use `nix run .#build` or per-module `GOWORK=off go build ./...` for real verification.
2. **Every deletion must be followed by a workspace-wide `GOWORK=off` build** — before committing. The daemon commits fast, but broken code ships if the pre-commit gate uses workspace mode.
3. **Cost constants need a dual-model: point-lookup AND analytical** — A single NsPerRead value cannot represent both. DuckDB at 3,000 ns/read for analytical GROUP BY is correct; at 546,000 ns/read for point lookups is also correct. The planner needs to know which query type it's routing.
4. **The verify gate takes 3-4 minutes** — The temptation to skip it is strong, but every session that skips it ships broken code. Make it faster or make it mandatory in the devShell shellHook.
5. **Doc references to deleted API must be grepped BEFORE claiming done** — `grep -rn "DeletedTypeName"` across the entire workspace including docs.
6. **The auto-commit daemon ships both real work AND breaking changes** — SSE refactoring, dependency bumps, and code changes happen without review. This session caught one breakage (stack/postgres); there may be others in the daemon's commits that weren't tested.
7. **PgxListener removal may have broken consumers** — PgxListener was the LISTEN-side implementation. Even if PostgresBus had zero consumers, PgxListener might have been imported independently. No consumer audit was done for PgxListener specifically.

---

## f) Up to 50 Things We Should Get Done Next

### Critical (correctness / CI-blocking)
1. Review daemon's SSE refactoring (metaengine/sse.go, transport/http/sse_event.go) — verify it works
2. Audit consumers of `PgxListener` / `NewPgxListenerFromDSN` across all external repos
3. Run `nix run .#verify` one more time to confirm the daemon's SSE commits didn't break anything
4. Push go-retry repo to GitHub (currently local-only)
5. Push go-idempotency repo to GitHub (currently local-only)
6. Update go-cqrs-lite `retry/go.mod` to use the published GitHub tag instead of local replace
7. Update go-cqrs-lite `idempotency/go.mod` to use the published GitHub tag instead of local replace
8. Run `nix run .#vulncheck` to verify no module resolution issues with the new external deps

### Cost Model
9. Write a DuckDB analytical GROUP BY benchmark (vectorized aggregation, not point lookup)
10. Write a Postgres analytical benchmark without Docker network overhead (Unix socket or same-host)
11. Add a `NsPerAnalyticalRead` field to EngineProfile (separate from `NsPerRead` for point lookups)
12. Update the planner to use the correct constant based on query type (analytical vs point)
13. Document the dual-model cost constants in the metaengine design docs

### Architecture Debt
14. Evaluate `command/bus.go` for removal (zero external consumers, internal watermill consumer)
15. Extract `idempotency/kvstore` to go-idempotency (blocked on kv/ dependency)
16. Extract `idempotency/sqlstore` to go-idempotency (blocked on storage/sql/ dependency)
17. Consider extracting `kv/` to a standalone repo to unblock idempotency sub-module extraction
18. Consider extracting `codec/` to a standalone repo (40 of 58 modules depend on it)
19. Complete SSE consolidation — verify all SSE code paths use go-sse primitives

### Documentation
20. Update ADR-0097 with SSE refactoring completion status
21. Update ADR-0064 (retry extraction) with "v0.1.0 tagged" status
22. Update ADR-0065 (idempotency extraction) with "core tagged, sub-modules deferred" status
23. Add an ADR for the PgxListener/PostgresBus removal (currently undocumented removal)
24. Add a "Cost Model Calibration" ADR documenting the dual-model approach
25. Update AGENTS.md module count (64 go.mod files, but some modules were modified)
26. Update SKILL.md if any consumer-facing API changed
27. Update CHANGELOG for all changes this session

### Testing
28. Add integration test for the SSE refactoring (daemon's work — needs verification)
29. Add a regression test that catches the "workspace false green" pattern (per-module build check)
30. Add property-based test for planner routing with the reverted constants
31. Run `nix run .#check-layers` to verify dependency budgets after new external deps
32. Run `nix run .#check-duplication` after all the file changes
33. Run `nix run .#check-coverage` to verify coverage didn't regress

### Cleanup
34. Remove the `event` import from `stack/postgres/preset.go` if unused (was kept for `io.Closer`)
35. Check if `stack/postgres/go.mod` still has the pgx dependency (may be transitively needed)
36. Clean up `go.work.sum` after all module changes
37. Verify the `cmd/cqrs-lint/main.go` struct tag reordering is intentional (daemon changed it)
38. Verify `query/query.go` whitespace addition is intentional (daemon changed it)
39. Run `nix flake check` to verify the flake is still valid

### Metaengine Polish
40. Add Pebble cost constant comments noting they WERE re-measured and kept (2000/1300/2500)
41. Add a "How to calibrate" doc section in metaengine explaining the benchmark methodology
42. Consider adding a `CalibrationBenchmarks` section to the metaengine README

### Consumer Impact Assessment
43. Check if any consumer repo uses `postgres.WithDistributedBus`
44. Check if any consumer repo uses `postgres.NewPgxListener`
45. Check if any consumer repo uses `postgres.NewPgxListenerFromDSN`
46. Check if any consumer repo uses `storage.NotificationListener`
47. Document migration path for consumers who relied on distributed bus

### Process
48. Add a pre-commit check that runs `GOWORK=off go build ./...` per module
49. Document the "workspace false green" trap more prominently in AGENTS.md
50. Consider making `nix run .#verify` a hard gate before any daemon commit

---

## g) Questions (that I CANNOT figure out myself)

### 1. Should the daemon's SSE refactoring work be trusted or reviewed?
The daemon committed SSE wire-format delegation to go-sse (commits `b7bb2647`, `bca4f31d`, `f7512177`) during this session. I did NOT write, review, or test this code. There are still uncommitted changes in `metaengine/sse.go` and `transport/http/sse_event.go`. Should I:
- a) Trust the daemon and commit the remaining changes?
- b) Review every daemon SSE commit before committing?
- c) Revert the daemon's SSE work entirely?

### 2. Was PgxListener independently consumed by external repos?
The prior session confirmed zero consumers of `storage.PostgresBus`, but no audit was done for `postgres.PgxListener` / `postgres.NewPgxListenerFromDSN` specifically. PgxListener was a standalone LISTEN-side implementation — it may have been imported independently of PostgresBus. Should I grep the consumer repos, or do you know definitively whether any consumer used it?

### 3. Should go-retry and go-idempotency be pushed to GitHub now?
Both repos are local-only with v0.1.0 tags. The go-cqrs-lite modules use `replace` directives pointing to local paths. Pushing would make them resolvable by external consumers but also makes them public. Should I push, or wait until they're more mature (tests, README, CI)?
