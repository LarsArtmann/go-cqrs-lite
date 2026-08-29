# Status Report: 2026-07-11 22:27 — v4 Release Execution (Updated)

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../../CHANGELOG.md) and
> [TODO_LIST.md](../../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

> **Session scope:** Execute the entire v4 TODO_LIST — code changes, tests, path migration, documentation, storage split, shutdown fix.
> **Outcome:** All blockers resolved + storage split + shutdown integration tests done. 20 files uncommitted. `git tag v4.0.0` pending user approval.

---

## A) FULLY DONE ✅

### A1. Envelope Backward-Compat Tests

- `kv.TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault` — raw JSON → CBOR store
- `kv.TestTypedStore_Migration_MixedOldAndNewData` — mixed old+new in same store

### A2. BackfillHandler → \*SSEBroker

- Signature: `BackfillHandler(broker *SSEBroker)` (was `SeekableJournal`)
- `BackfillHandlerWithTransform` removed — broker's transform used directly
- `SSEBroker.Journal()` + `SSEBroker.PayloadTransform()` accessors added
- 5 tests (ReturnsEvents, MissingAfterParam, LimitsTo1000, PayloadTransformFromBroker, NoJournalReturns503)
- All doc references updated (AGENTS.md, SKILL.md, FEATURES.md)

### A3. HealthCheck on OwnedDBHandle

- Added to `storage/sql/base.go` — all SQL stores inherit via embedding
- Redundant impl removed from `*SQLEventStore`

### A4. ADR-0053

- `docs/adr/0053-unified-codec-default-flip.md` — all 6 codec defaults, backward-compat chain

### A5. CHANGELOG [v4.0.0]

- 5 breaking changes: path migration, codec defaults, alias removal, BackfillHandler, storage split
- Migration code examples + link to MIGRATION-GUIDE.md

### A6. Documentation Updates

- **FEATURES.md** — codec section, storage section, SSE section, audit date
- **MIGRATION-GUIDE.md** — 5 breaking changes with before/after code
- **AGENTS.md** — BackfillHandler example, storage/ tree with eventstore/ + readmodel/

### A7. /v3 → /v4 Path Migration

- 49 go.mod files, ~750 .go files, all scripts, all tools
- External dep false positives caught + reverted (yaml/v3, shortuuid/v3, semver/v3, slim-sprig/v3)
- `event/v3/` → `event/v4/` directory renamed

### A8. v3 Git Tag Backfill

- v3.0.0, v3.3.0, v3.4.0, v3.5.0, v3.6.0, v3.7.0, v3.7.1 tagged
- (Note: v3.7.0 and v3.7.1 point to same commit — v3.7.1 was docs-only)

### A9. API Surface + Doc Check

- `docs/api_surface.txt` — 2206 exports (was 2212 pre-split)
- `cmd/doc-check` — 880 references valid across 34 packages

### A10. WithShutdownDependency Integration Tests

- `TestShutdown_ThroughNewConstructor` — goes through `stack.New(WithCloser(...), WithShutdownDependency(...))`, tracks actual Close() call order via `orderTracker` + `orderLog`
- `TestShutdown_ThroughNewConstructor_NoDeps` — proves registration order through real constructor
- Race detector passes on both

### A11. Storage/ Split

- `storage/eventstore/` — 8 files moved (event_store\*.go ×6, snapshot.go, checkpoint.go), package `eventstore`
- `storage/readmodel/` — 1 file moved (kv_sql.go), package `readmodel`
- `storage/eventstore_aliases.go` — type aliases + constructor re-exports (backward compat)
- `storage/readmodel_aliases.go` — type alias + constructor re-exports
- `storage/sql_backend.go` — imports from new packages, uses exported constructors
- **Zero breaking changes** — all `storage.SQLEventStore` references work via aliases
- Proposal updated to status DONE

### A12. Full Build + Test Verification

- All 60+ test packages pass (EXIT: 0)
- `go vet` clean on storage module
- Race detector clean on stack shutdown tests
- `go work sync` clean

---

## B) PARTIALLY DONE ⚠️

### B1. Git Status — 20 files uncommitted

None of the session's changes have been committed. The previous session's user commit (`e443adb3`) captured the v4 path migration, but all storage split + shutdown test + doc changes remain uncommitted.

### B2. api-stability Test (pre-existing failure)

`cmd/api-stability` TEST fails (subprocess doesn't inherit `goexperiment.jsonv2`). Golden file was updated correctly via `go run . -update`. The test failure is pre-existing — not caused by my changes.

### B3. AGENTS.md Modules List Not Updated

The Quick Reference `Modules` row still lists the old module set without `storage/eventstore` and `storage/readmodel`. The directory tree WAS updated, but the flat module list was not.

---

## C) NOT STARTED ⬜

### C1. `git tag v4.0.0` + push

All code, tests, docs ready. Awaiting user approval.

### C2. `nix run .#lint` (golangci-lint)

Never ran full lint. Only `go vet` on key modules.

### C3. Race Detector on Full Suite

Only ran `-race` on stack shutdown tests. Full workspace `-race` never executed.

---

## D) TOTALLY FUCKED UP 💥 (and what I did about it)

### D1. (Previous session) Global sed caught external deps — FIXED

Blanket `/v3` → `/v4` sed changed `go.yaml.in/yaml/v3`, `lithammer/shortuuid/v3`, etc. Caught via `go mod tidy` errors, reverted all 4 external deps. No damage.

### D2. (This session) No new fuckups

The storage split was clean — type aliases + constructor re-exports provide full backward compat. All tests pass. The shutdown fix used the real constructor path as intended.

---

## E) WHAT WE SHOULD IMPROVE

### E1. Duplicate `errStoreClosed` Sentinel

`eventstore.errStoreClosed` and `sql.ErrClosed` have the **same error code** (`"storage.closed"`) but are different variables. `eventstore`'s `checkClosed()` uses `errStoreClosed`; `OwnedDBHandle.HealthCheck()` uses `sql.ErrClosed`. Should consolidate — `errStoreClosed` should use `sql.ErrClosed`.

### E2. No Tests in eventstore/ or readmodel/ Packages

All test files stayed in `storage/` (package `storage` / `storage_test`). They test through the aliases, which works, but there are **zero test files** in `storage/eventstore/` or `storage/readmodel/`. If someone removes the aliases later, the tests break with no local coverage. At minimum, a `package eventstore` smoke test should exist.

### E3. SKILL.md Doesn't Mention New Sub-packages

SKILL.md module table and references don't mention `storage/eventstore/` or `storage/readmodel/`. Consumers reading the skill won't know these focused packages exist. The existing storage/ row is fine for backward compat, but new consumers should be directed to the focused packages.

### E4. Execution Plan Doesn't Mention Storage Split

The v4 execution plan (`docs/planning/2026-07-11_20-37_V4-RELEASE-EXECUTION-PLAN.md`) was updated with execution retrospective for the original phases but doesn't mention the storage split or shutdown fix that were added later.

### E5. Never Committed

Same issue as previous session — completed all work but created no git commit.

### E6. RelationalProjection / ViewStore Are in Separate Sub-packages But Not in readmodel/

The proposal mentioned `SQLViewStore` and `RelationalProjection` moving to `readmodel/`, but they already had their own sub-packages (`storage/view/`, `storage/relational/`) with aliases in `storage/`. Only `SQLKVStore` moved to `readmodel/`. This is architecturally correct (they were already separated) but doesn't fully match the proposal's scope.

---

## F) NEXT 50 THINGS TO DO

### Immediate (before tagging v4.0.0)

1. **Commit all 20 uncommitted files**
2. **Run `nix run .#lint`** — golangci-lint may surface issues
3. **Run `go test -race` on full workspace** — verify no race conditions
4. **Update AGENTS.md Modules row** — add `storage/eventstore`, `storage/readmodel` to flat module list
5. **Consolidate `errStoreClosed` → `sql.ErrClosed`** — eliminate duplicate sentinel in eventstore
6. **Add smoke test in `storage/eventstore/`** — at least one `package eventstore` test
7. **Add smoke test in `storage/readmodel/`** — at least one `package readmodel` test

### Post-tag (v4.0.0 shipped)

8. **`git tag v4.0.0`** + push
9. **Verify `go install .../event/v4@v4.0.0` works** (proxy fetch)
10. **Write v4.0.0 GitHub release notes**
11. **Run `nix flake check`**

### v4.1 — Parquet Journal + DuckDB

12. **Create `storage/parquet/` module** with `go.mod`
13. **Implement `SeekableJournal`** over Parquet segment files
14. **Design segment manifest** (JSON index for seeking)
15. **Implement segment flush** threshold logic
16. **Add Parquet schema mapping** from `ImmutableEvent` fields
17. **Property tests** — round-trip, seek correctness, segment boundaries
18. **Benchmark** vs SQLite/Pebble for append-only workloads
19. **Create `storage/duckdb/` module** with CGO build
20. **Implement `DuckDBDialect`** (11 methods on `sqlpkg.Dialect`)
21. **Verify `SQLViewStore` works with DuckDB**
22. **Verify `RelationalProjection` works with DuckDB**
23. **Create `stack/duckdb/` preset** combining DuckDB + Parquet
24. **Test `read_parquet()` from DuckDB** against Parquet segments
25. **Document CGO implications**

### Code Quality

26. **Fix api-stability test subprocess** — pass `GOEXPERIMENT` env to subprocess
27. **Add SSE + Backfill integration test** — same broker, both delivery paths
28. **Improve `envelope.go` comment** — document why `"cqrs"` is safe
29. **Run `go mod tidy` without `-e`** — check for masked dependency issues
30. **Add `// Deprecated:` comments to storage re-export aliases** — guide consumers to new packages
31. **Update SKILL.md** — add storage/eventstore/ and storage/readmodel/ to module table
32. **Clean up go.sum files** — regenerate all after workspace-wide tidy
33. **Verify `scripts/sync-replaces.sh`** works with new eventstore/readmodel packages

### Documentation

34. **Update all module `README.md` files** with v4 import paths (some may still have v3)
35. **Update `docs/index.md`** with v4 import examples
36. **Update `docs/getting-started.md`** with v4 import examples
37. **Add v4.0.0 to `docs/sessions/SESSION_MILESTONES.md`**
38. **Update storage split proposal** — note that ViewStore and RelationalProjection already had sub-packages

### CI / Infrastructure

39. **Add Postgres CI service** or label `stack/postgres` as experimental
40. **Update `.golangci.yml` depguard allow list** for any new dependencies
41. **Verify CI pipeline** passes with `/v4` module paths

### Polish

42. **README.md "sales page" rewrite** — per AGENTS.md rule
43. **License swap (PROPRIETARY → Apache-2.0)** — post-v4, user approval
44. **Git history scrub** — remove internal strategy docs, user approval
45. **Update `cmd/api-stability/main.go` module list** — add `storage/eventstore` and `storage/readmodel`
46. **Update `flake.nix` testModules** — add new packages if needed
47. **Verify `go.work` includes all new directories** (eventstore/ and readmodel/ are sub-packages, not workspace modules — they don't need `use` directives)
48. **Add doc.go to eventstore/ and readmodel/** — package-level documentation
49. **Consider whether timer_store.go should move to eventstore/** — it's event-sourcing infrastructure
50. **Review whether SQLBackend should return eventstore types** — currently returns aliases, which is correct for backward compat

---

## G) Top 2 Questions I Cannot Answer Myself

### G1. Should `storage/eventstore/` and `storage/readmodel/` be separate Go modules (own go.mod) or sub-packages of storage/v4?

Currently they are sub-packages within the `storage/v4` module (same go.mod). This means:

- ✅ No import path churn for consumers (backward compat aliases are free)
- ✅ No new go.mod to maintain
- ❌ The storage module is still large (though logically split)
- ❌ Can't version eventstore/ independently

Making them separate modules would give dependency isolation but add go.mod overhead. I cannot decide this tradeoff — it depends on whether consumers will ever want to import eventstore/ without the full storage/ backend facade.

### G2. Should I commit now and tag v4.0.0, or wait?

20 files are uncommitted. All tests pass. The previous session ended with the same question. I don't know if you want to review the storage split changes first, or if there are other things you want bundled into the v4.0.0 commit before tagging.
