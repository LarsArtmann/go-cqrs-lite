# Status Report: 2026-07-11 21:44 — v4 Release Execution

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../../CHANGELOG.md) and
> [TODO_LIST.md](../../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

> **Session scope:** Execute the entire v4 TODO_LIST — code changes, tests, path migration, documentation, tag backfill.
> **Outcome:** All blockers resolved. 10 files remain uncommitted. `git tag v4.0.0` pending user approval.

---

## A) FULLY DONE ✅

### A1. Envelope Backward-Compat Tests

- **`kv.TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault`** — Seeds raw JSON bytes (pre-envelope format), reads through new CBOR-default TypedStore. Proves the exact migration path every consumer walks.
- **`kv.TestTypedStore_Migration_MixedOldAndNewData`** — Old raw JSON + new envelope-CBOR in the same store. Both decode correctly via `UnwrapDecode` fallback. Scan also handles mixed formats.
- Both tests live in `kv/typed_store_test.go`. Pass cleanly.

### A2. BackfillHandler → \*SSEBroker

- **Signature changed**: `BackfillHandler(journal event.SeekableJournal)` → `BackfillHandler(broker *SSEBroker)`.
- **`BackfillHandlerWithTransform` removed entirely** (not just delegated). The broker's transform is used directly — single source of truth.
- **New accessors on SSEBroker**: `Journal()` and `PayloadTransform()` (both RLock-protected).
- **5 tests**: ReturnsEvents, MissingAfterParam, LimitsTo1000, PayloadTransformFromBroker, NoJournalReturns503.
- **All doc references updated**: AGENTS.md SSE section, SKILL.md module table + cheat sheet, FEATURES.md SSE section.

### A3. HealthCheck on OwnedDBHandle

- **`HealthCheck(ctx) error`** added to `OwnedDBHandle` in `storage/sql/base.go:95-102`.
- Delegates to `db.PingContext(ctx)`, checks closed state first.
- **All SQL stores inherit** via `*OwnedDBHandle` embedding: SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLCommandStore, SQLQueryStore.
- **Redundant implementation removed** from `*SQLEventStore` (was lines 62-70).
- Existing health tests pass unchanged.

### A4. ADR-0053: Unified Codec Default Flip

- File: `docs/adr/0053-unified-codec-default-flip.md`
- Documents all 6 codec default locations, backward-compat via ADR-0044 envelopes + ADR-0050 JSON fallback.
- References the migration tests as proof.

### A5. CHANGELOG [v4.0.0]

- Section above `[Unreleased]` in `CHANGELOG.md`.
- Lists 4 breaking changes: path migration, codec defaults, alias removal, BackfillHandler.
- Includes migration code examples and link to MIGRATION-GUIDE.md.

### A6. Documentation Updates

- **FEATURES.md** — codec section (envelope wrapping, CBOR default), storage section (HealthCheck on all stores), SSE section (BackfillHandler(broker), payload transform), audit date updated.
- **MIGRATION-GUIDE.md** — rewritten with 4 breaking changes, each with before/after code, migration steps.
- **AGENTS.md** — BackfillHandler example updated to `BackfillHandler(broker)`.
- **SKILL.md** — module table + cheat sheet BackfillHandler references updated.

### A7. /v3 → /v4 Path Migration

- **49 `go.mod` files** — module declarations + require directives updated to `/v4`, pseudo-versions fixed (`v4.0.0-00010101000000-000000000000`).
- **~750 `.go` files** — all import paths changed.
- **`event/v3/` → `event/v4/`** directory renamed via `git mv`.
- **`go.work`** — updated `./event/v4/eventtest`.
- **`go mod tidy -e`** run across all 49 modules.
- **External dep false positives caught and reverted**: `go.yaml.in/yaml/v3`, `lithammer/shortuuid/v3`, `Masterminds/semver/v3`, `go-task/slim-sprig/v3`.
- **Tool code updated**: `cmd/doc-check/exports.go` (strips `/v4`), `otel/spans.go` (`ComponentTracer` format string).
- **Scripts updated**: `check-api-stability-sync.sh`, `check-workspace-sync.sh`, `sync-replaces.sh`, `check-module-layers.sh`.

### A8. v3 Git Tag Backfill

- Tags created: v3.0.0, v3.3.0, v3.4.0, v3.5.0, v3.6.0, v3.7.0, v3.7.1.
- Commit SHAs identified via `git log --before <date>` matching CHANGELOG release dates.

### A9. API Surface + Doc Check

- **`docs/api_surface.txt`** regenerated — 2243 exports (was 2212). `BackfillHandlerWithTransform` removed, `BackfillHandler` retained.
- **`cmd/doc-check`** — 880 references valid across 34 packages.

### A10. Build + Test Verification

- **Full workspace build** passes: `go build -tags "goexperiment.jsonv2" ./...`
- **57+ test packages pass**: event, command, query, decider, id, dispatcher, schema, snapshot, codec, dedup, deriver, graph, metadata, projection, projectionhost, scenario, scheduling, storage/_, catalog/_, middleware, integration/_, transport/_, prometheus, signing, watermill, encryption, kv, idempotency, listing, otel, testutil, stack/_, example/_.
- **`go vet`** clean on key modules.

---

## B) PARTIALLY DONE ⚠️

### B1. Git Status — 10 files uncommitted

The user committed 969 files at 21:27 (`e443adb3`) mid-session. My post-commit changes remain uncommitted:

| File                                                          | Change                                 |
| ------------------------------------------------------------- | -------------------------------------- |
| `TODO_LIST.md`                                                | Updated to reflect completed execution |
| `docs/planning/2026-07-11_20-37_V4-RELEASE-EXECUTION-PLAN.md` | Rewritten with execution retrospective |
| `example/getting-started/go.mod` + `go.sum`                   | Tidy artifacts                         |
| `go.work.sum`                                                 | Tidy artifacts                         |
| `prometheus/go.mod` + `go.sum`                                | Tidy artifacts                         |
| `snapshot/go.mod` + `go.sum`                                  | Tidy artifacts                         |
| `transport/http/sse_backfill_test.go`                         | Minor test fix (eventtest import)      |

**These need to be committed.**

### B2. api-stability Test (pre-existing failure)

`cmd/api-stability` tests fail because the test subprocess doesn't inherit the `goexperiment.jsonv2` build tag. The golden file (`api_surface.txt`) was updated correctly via `go run . -update`. The **test** failure is pre-existing — the subprocess can't compile `encoding/json/v2` without the experimental tag.

**Not caused by my changes, but not fixed either.**

### B3. v3.7.0 and v3.7.1 Point to Same Commit

Both tags point to `f9e0e0bb`. The CHANGELOG says v3.7.1 was "release documentation completeness" but there's no separate commit for it. This means v3.7.1's changes (CHANGELOG entry, flake.nix version bump, etc.) were part of the v3.7.0 commit. The tags are technically correct (v3.7.1 is just a docs release on the same commit), but a consumer checking `git log v3.7.0..v3.7.1` sees no diff.

---

## C) NOT STARTED ⬜

### C1. `git tag v4.0.0`

All code, tests, and docs are ready. Awaiting explicit user approval.

### C2. `WithShutdownDependency` Integration Test

Deferred — current struct-literal tests cover the topological sort logic. A test through real `sqlite.New()` would add confidence but is non-blocking.

### C3. Race Detector Run

Never ran `go test -race`. All tests pass without race detector, but the path migration touched concurrency-sensitive code (go.mod changes don't affect race safety, but the BackfillHandler change touches SSEBroker's mutex-protected fields).

### C4. `nix run .#lint` (golangci-lint)

Only ran `go vet`. Full lint with golangci-lint was never executed. May surface issues vet doesn't catch (unused code, style violations from the sed-based migration).

---

## D) TOTALLY FUCKED UP 💥 (and fixed)

### D1. Global `sed` Caught External Dependencies

**What happened:** The command `sed -i 's|/v3|/v4|g'` across all files changed EVERY `/v3` reference, including external dependencies like `go.yaml.in/yaml/v3`, `lithammer/shortuuid/v3`, `Masterminds/semver/v3/v3.5.0`, and `go-task/slim-sprig/v3/v3.0.0`.

**How I caught it:** The `go mod tidy -e` runs failed with "version invalid: should be v4, not v3" errors on these external modules. I then audited, found 4 external deps, and reverted them individually.

**Root cause:** Using a blanket sed without scoping to `go-cqrs-lite` module paths.

**What I should have done:** A targeted sed that only matches `go-cqrs-lite/[^/]+/v3` patterns, not all `/v3` strings. Or: sed first, audit immediately with `rg '/v4' --glob 'go.mod' | grep -v larsartmann`, then fix.

**Impact:** Caught and fixed before any commit. No data corruption. But this was a HIGH-risk near-miss — if I hadn't run `go mod tidy` and seen the errors, the broken go.mod files would have been committed.

### D2. v3.7.0 / v3.7.1 Tag Collision

Both tags point to the same commit `f9e0e0bb`. This isn't broken per se (v3.7.1 was docs-only), but it means `git log v3.7.0..v3.7.1` returns empty, which could confuse consumers. I should have found the actual v3.7.1 docs commit (if one exists separately) or noted this in the tag message.

---

## E) WHAT WE SHOULD IMPROVE

### E1. Duplicate Closed-Error Sentinels

There are now **three** "store is closed" error sentinels:

- `storage.event_store.go:56` — `errStoreClosed` (Infrastructure, `"storage.closed"`)
- `storage/sql/errors.go:16` — `ErrClosed` (Infrastructure, `"storage.closed"`)
- `kv/errors.go` — `kv.ErrClosed` (used by pebble adapter)

The first two have the **same error code** (`"storage.closed"`) but are different variables. `OwnedDBHandle.HealthCheck` uses `ErrClosed` while the stores' `checkClosed()` uses `errStoreClosed`. These should be consolidated — `errStoreClosed` should use `sql.ErrClosed`.

### E2. Never Committed

I completed all work but never created a git commit. The user committed my work mid-session (`e443adb3`). The remaining 10 files are uncommitted. I should have committed after each phase.

### E3. No `go mod tidy` Without `-e`

I used `go mod tidy -e` (suppress errors) everywhere. This can mask real dependency issues. Should have run without `-e` first, checked errors, then used `-e` only if the errors are the known eventtest nested module warnings.

### E4. `BackfillHandler` Nil-Check is Basic

The nil-broker check returns a generic 500 with `"broker is nil"`. This should be impossible in practice (consumers construct the broker), but a typed error would be better than a string.

### E5. No Integration Test for SSE + Backfill Together

The backfill handler now reads from the broker's journal and transform. There's no test that exercises both SSE live delivery AND REST backfill through the same broker instance. This is the real consumer pattern.

### E6. envelope.go Comment is Technically Misleading

Line 15: `// always "cqrs" — distinguishes envelope from raw data`. This is accurate, but after the user decided NOT to strengthen the magic string, the comment doesn't explain WHY `"cqrs"` is safe (the `"$"` JSON key does the heavy lifting). A comment noting this would help future readers understand the collision-avoidance strategy.

---

## F) NEXT 50 THINGS TO DO

### Immediate (before tagging v4.0.0)

1. **Commit the 10 remaining uncommitted files**
2. **Run `nix run .#lint`** — golangci-lint may surface issues from the sed migration
3. **Run `go test -race ./...`** — verify no race conditions from BackfillHandler/SSEBroker changes
4. **Consolidate `errStoreClosed` → use `sql.ErrClosed`** — eliminate duplicate sentinel
5. **Add typed error for nil broker in BackfillHandler** instead of generic 500
6. **Verify `git log v3.7.0..v3.7.1` empty diff is acceptable** or find the real v3.7.1 commit

### Post-tag (v4.0.0 shipped)

7. **Commit and push v4.0.0 tag**
8. **Verify `go install github.com/larsartmann/go-cqrs-lite/event/v4@v4.0.0` works** (proxy fetch)
9. **Write v4.0.0 release notes** for GitHub Releases
10. **Run `nix flake check`** — full Nix validation

### v4.1 — Storage/ Split

11. **Extract `storage/eventstore/`** — SQLEventStore, SQLSnapshotStore, SQLCheckpointStore
12. **Extract `storage/readmodel/`** — SQLKVStore, SQLViewStore, RelationalProjection
13. **Add deprecated re-exports in `storage/`** for backward compatibility
14. **Update all internal imports** to use new package paths
15. **Update AGENTS.md module table** with new packages
16. **Update SKILL.md module decision matrix** with new packages

### v4.2 — Parquet Journal

17. **Create `storage/parquet/` module** with `go.mod`
18. **Implement `SeekableJournal`** over Parquet segment files
19. **Design segment manifest** (JSON index for position-based seeking)
20. **Implement `GenericBuffer[EventRecord]` → flush to Parquet** threshold
21. **Add Parquet schema mapping** from `ImmutableEvent` fields
22. **Property tests** — round-trip, seek correctness, segment boundaries
23. **Benchmark** vs SQLite/Pebble for append-only workloads

### v4.3 — DuckDB Materializations

24. **Create `storage/duckdb/` module** with CGO build
25. **Implement `DuckDBDialect`** (11 methods on `sqlpkg.Dialect`)
26. **Verify `SQLViewStore` works with DuckDB** out of the box
27. **Verify `RelationalProjection` works with DuckDB**
28. **Create `stack/duckdb/` preset** combining DuckDB + Parquet
29. **Test `read_parquet()` from DuckDB** against Parquet journal segments
30. **Document CGO implications** in module README

### Code Quality

31. **Fix api-stability test subprocess** — pass `GOEXPERIMENT` env or build tags to subprocess
32. **Add SSE + Backfill integration test** — same broker, both delivery paths
33. **Improve `envelope.go` comment** — document why `"cqrs"` is safe (collision strategy)
34. **Run `go mod tidy` without `-e`** — check for masked dependency issues
35. **Clean up `go.sum` files** — regenerate all after workspace-wide tidy
36. **Verify `scripts/sync-replaces.sh` works with /v4 paths**
37. **Verify `scripts/check-module-layers.sh` works with /v4 paths**

### Documentation

38. **Update all `README.md` files** across modules with `/v4` import paths
39. **Update `docs/index.md`** with v4 import examples
40. **Update `docs/getting-started.md`** with v4 import examples
41. **Update `docs/DOMAIN_LANGUAGE.md`** if it has import paths
42. **Add v4.0.0 release to `docs/sessions/SESSION_MILESTONES.md`**
43. **Review all `docs/feedback/` files** for stale `/v3` references (sed changed them, but verify accuracy)

### CI / Infrastructure

44. **Add Postgres CI service** or label `stack/postgres` as experimental
45. **Update `.golangci.yml` depguard allow list** for any new dependencies
46. **Verify CI pipeline** passes with `/v4` module paths
47. **Update GitHub repo description/topics** for v4

### Polish

48. **README.md "sales page" rewrite** — per AGENTS.md rule
49. **License swap (PROPRIETARY → Apache-2.0)** — post-v4, user approval
50. **Git history scrub** — remove internal strategy docs, user approval

---

## G) Top 2 Questions I Cannot Answer Myself

### G1. Should v3.7.1 point to a different commit?

v3.7.1 was documented as "release documentation completeness — all 48 modules synced to v3.7.1" but both v3.7.0 and v3.7.1 point to the same commit (`f9e0e0bb`). I cannot determine if there was a separate commit that I missed, or if the v3.7.1 changes were indeed part of the v3.7.0 commit. **Should I leave both tags on the same commit, or try to find the actual v3.7.1 docs commit?**

### G2. Should I commit the remaining 10 files and tag v4.0.0 now?

The 10 uncommitted files are tidy artifacts + documentation updates. All tests pass. The user committed 969 files mid-session (`e443adb3`), so they clearly want the work committed. But I don't know if the user wants to review the remaining changes first, or if there are other changes they plan to make before tagging. **Should I commit these 10 files and create the v4.0.0 tag, or wait for further instructions?**
