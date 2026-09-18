# Status Report: Native Temporal Versioned Cells (ADR-0141) — Metaengine Deep Dive

> **Session:** 2026-09-18 ~08:00–14:07 · **Scope:** point-in-time versioning for metaengine, BigTable-aligned
> **Format note:** written as `.md` per explicit user request (skill default is HTML dashboard — overridden).
> Point-in-time snapshot; verify claims against source before acting (repo policy §50).

## Context

Started from the question "how advanced is the metaengine's native point-in-time versioning?"
(answer: experimental, memory-only, wall-clock stamps, dead `AsOfSignal`). The user commissioned a
full deep dive to make it **superb and native**, explicitly BigTable-aligned. Executed as:
research → ADR-0141 → core API → memory engine → SQLite engine → plan rule → bigtableengine module.

---

## a) FULLY DONE (verified green at time of writing)

1. **Research — BigTable Go SDK verified against real source** (v1.57.0 in module cache):
   `bttest.NewServer` in-process fake (no emulator binary), `Mutation.Set(family, col, ts Timestamp, []byte)`,
   `TimestampRangeFilterMicros`, `LatestNFilter`, `ChainFilters`, `MaxVersionsPolicy`/`MaxAgePolicy`,
   `TableConf.ColumnFamilies`, **ms-only granularity**, **SDK truncates both range bounds to ms**,
   **counter cells are 8-byte big-endian int64** (both bttest-verified — each cost a live debug round).
2. **ADR-0141** `docs/adr/0141-native-temporal-versioned-cells.md` — the temporal cell contract
   (as-of = latest `ts <= T`, tombstone = nil/empty, same-ts last-writer-wins, out-of-order legal,
   retention never prunes newest). Registered in `docs/adr/README.md` + `docs/README.md`.
3. **Core temporal API v2** (metaengine module, committed with golden regen — commit `9b3c22dbf`):
   - `VersionedWriter` (`MapSetAt`/`MapDeleteAt`, native-key), `VersionedUpdater` (`MapUpdateAt`,
     atomic timestamped RMW), `CellHistoryReader` (`MapHistory`), `CellVersion`, `RetentionPolicy`,
     `CellTimestamp(rec)` = Stored → Received → Created → wall-clock, `CellVersioningToggle`,
     exported `EngineVersionsCells`.
   - Fold path (insert/update/remove) stamps cells with **event time**; update folds prefer
     `VersionedUpdater` (single write-shaped call — preserves ADR-0137 catch-up semantics).
   - **AsOf input-field routing made real**: `AsOf time.Time` field on a point-lookup input →
     `VersionedStorage` read; zero = latest; non-versioned engine → loud `ErrUnsupportedADT`.
     `AsOfSignal` doc now tells the truth. AsOf joined `Limit`/`After`/`Depth` as a meta field
     (excluded from filter inference and key extraction).
4. **Memory engine v2**: sorted version chains (out-of-order-safe insert), retention trim
   (`MaxVersions`/`MaxAge`, newest never pruned — off-by-one found and fixed by test),
   `MapUpdateAt`, `MapHistory`, latest-view synced from chain tail. `NewMemoryEngineWithVersioning`
   now variadic (`...VersioningOption`, `WithRetention`).
5. **`adttest.AssertTemporalConformance`** — one cross-engine contract harness (as-of resolution,
   tombstones, out-of-order, same-ts LWW, history ranges, latest-view consistency, MapUpdateAt),
   pinned by memory + sqlite + bigtable. Same pattern as the vector-dimension contract.
6. **SQLite versioned cells**: `WithCellVersioning`, `meta_cell_versions(collection, key, ts, value)`
   history table (NULL = tombstone, PK collision = REPLACE), retention trim SQL, atomic
   `MapUpdateAt` (RunInTx), planned collections fail loudly (`ErrUnsupportedADT`), plain
   `MapSet`/`MapDelete` record wall-clock history when enabled, reset clears the table.
   Full sqliteengine suite green.
7. **Planner temporal rule** (`temporal-asof`): WARN diagnostic when an AsOf-declaring query is
   assigned to a non-versioned engine ("honest degradation" surfaced at plan time). Tested.
8. **bigtableengine module** — the flagship: engine + native versioned map cells
   (row `collection\x00key`, family `cqrs`, ms-truncated cell timestamps), empty-value tombstones,
   as-of reads (chain order: family → timestamp-range → LatestN — order matters, found by debug),
   history ranges, counters via `ReadModifyWrite` increments + prefix-scan `CounterGet`,
   `ResetEngine` (full table wipe via ApplyBulk), `HealthCheck`, GCPolicy option, driver
   registration (`"bigtable"`, DSN `project/instance/table`), **all tests green against bttest**
   including conformance, ms-collapse pin, tombstone→rebirth, reset, counters, DSN validation.
9. **Module registration**: flake.nix `testModules` + `cmd/api-stability` modules slice + go.work
   (but see d) — and depguard allow-list restored (but see d).

Test evidence this session (all with `-tags "goexperiment.jsonv2"`, GOWORK=off):
metaengine full suite `ok` (~35 s), adttest `ok`, sqliteengine full `ok`, bigtableengine full `ok`,
projectionadapter `ok`, system + bench build `ok`.

## b) PARTIALLY DONE

1. **Commit hygiene**: authored commits exist (`9b3c22dbf` core+golden, plus a sqlite+planner commit
   attempt), but the auto-commit daemon raced repeatedly — several logical phases landed as
   `chore: auto-commit` instead of authored history (including most of phase 1 and the sqlite phase).
2. **Registration for bigtableengine**: registered in 3 of 4 required places; `.golangci.yml`
   depguard entries NOT landed (edit lost to daemon file-race twice).

## c) NOT STARTED

1. **api-stability golden regen for bigtableengine** — module added to the meta-test slice but
   `--update` not re-run: `docs/api_surface.txt` contains 0 bigtableengine exports →
   `TestEvery`/pre-commit gate **will fail** until regenerated.
2. **Docs & gates phase**: root CHANGELOG entries (with `pkg.Symbol` citations), FEATURES.md rows
   (VersionedStorage row update + bigtableengine row + maturity), `docs/agents/module-map.md` row,
   planning-doc §3 implementation-status addendum (`meta-engine-layered-architecture.md`),
   skill references (recipes §temporal, modules.md, advanced.md), `nix run .#verify` /
   `#verify-fast`, `#check-arch` (dep budget for the new module), `#check-lint-config`,
   `#check-duplication`, doc-check run, cqrs-lint n/a.
3. **README.md for bigtableengine** — written as package doc; module README file not created
   (every other engine module has one).
4. **Todo**: `BigtableEngineProfile` calibration constants are priors (2 ms RTT), not measured.

## d) TOTALLY FUCKED UP (fix before ANY further work)

1. **go.work lists `./metaengine/bigtableengine` TWICE** (lines 41 and 54 — one line from the
   daemon's absorb, one from my registration edit). Every workspace-mode go command now fails:
   `go.work:54: path ... appears multiple times in workspace`. GOWORK=off still works.
   Fix: delete one line, then `go work sync`.
2. **`.golangci.yml` was mutilated by the auto-commit daemon at 11:11** (commit `c56d219a6` removed
   the entire 89-line depguard allow-list — a race with my background commit's `nix fmt` hook).
   I restored depguard from the parent commit, but my bigtable depguard entries
   (`cloud.google.com/go/bigtable`, `google.golang.org/api`, `google.golang.org/grpc`) were then
   lost to ANOTHER daemon file-race — current file has depguard restored but WITHOUT the three
   entries → `nix run .#lint` will reject bigtableengine imports. Also: this incident is a data-loss
   near-miss for a CI-critical file; the daemon's destructive absorb deserves investigation.
3. **Working tree is "clean" only because the daemon absorbed everything** — including the
   half-finished registration state above. Nothing is lost, but nothing is gated either: no
   verification gate has run since the bigtableengine module landed.

## e) WHAT WE SHOULD IMPROVE (process + design, this session's lessons)

1. **Run `api-stability --update` in the SAME edit as any API change** — I did this for core
   (pre-commit forced it) but forgot it for the new module; the golden is now stale.
2. **The auto-commit daemon is hazardous for config files**: it committed a HALF-WRITTEN
   `.golangci.yml` (data loss) and races explicit commits + edits (mtime churn broke 4+ edits this
   session). Mitigations to consider: daemon ignore-list for `.golangci.yml`/`go.work`, or
   commit-then-verify locks. At minimum: re-read any config file immediately before editing it.
3. **Edit-tool discipline**: two of my failures were self-inflicted (edits after daemon rewrites
   without re-View; one duplicate registration edit creating the go.work double-entry).
4. **bttest ≠ BigTable**: three real semantic gaps found only by empirical probing (ms-bound
   truncation, filter chain order, binary counters). Any future BigTable work should keep the
   probe-test pattern; consider pinning those three facts in the module README.
5. **Design wins worth keeping**: the `CellVersioningToggle` gate (versioning-disabled engines stay
   on plain MapBackend path — preserves wrapper interception AND catch-up fault models);
   `VersionedUpdater` as a separate optional interface (atomic RMW without growing shipped ones).

## f) Up to 50 things to do next (priority order)

1. Fix go.work duplicate `bigtableengine` entry; `go work sync`.
2. Re-add the 3 depguard entries to `.golangci.yml` (re-View first; daemon race).
3. `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` + `TestEvery`.
4. Run `nix run .#lint` — expect/fix bigtableengine lint findings (funlen 30, wrapcheck, etc.).
5. Run `nix run .#check-arch` — dep budget for bigtableengine (grpc/api are test-scope; verify).
6. Run `nix run .#check-lint-config` (depguard verify).
7. Run `nix run .#check-duplication` (register.go clone annotated; verify no new clones).
8. Run `nix run .#check-file-size` — all new files < 350 lines, functions < 30.
9. Write `metaengine/bigtableengine/README.md` (scope, emulator/GC notes, ms-granularity caveat).
10. CHANGELOG `[Unreleased]` Added entries citing `metaengine.VersionedWriter`,
    `metaengine.VersionedUpdater`, `metaengine.CellHistoryReader`, `metaengine.RetentionPolicy`,
    `metaengine.CellTimestamp`, `metaengine.EngineVersionsCells`, `metaengine.WithRetention`,
    `sqliteengine.WithCellVersioning`, `bigtableengine.New` (gate: check-changelog-symbols).
11. FEATURES.md: update VersionedStorage row (3 engines + planner routing); add bigtableengine row
    (🧪); add temporal rule row.
12. `docs/agents/module-map.md`: bigtableengine row.
13. Planning-doc §3 addendum: banner + per-section DONE/DIFFERENT status (point-in-time policy).
14. Skill refs: recipes.md new temporal section; modules.md bigtableengine row; advanced.md as-of
    section; core.md §3 mention AsOf meta-field.
15. Run doc-check gate over SKILL.md + references.
16. `nix run .#verify-fast` then `nix run .#verify` (exclusive).
17. `nix run .#verify-ci` (per-module GOWORK=off matrix incl. new module).
18. Go-mod pin sweep for bigtableengine (sibling-replace rules, gotchas-module-management).
19. Conformance-runner wiring: add bigtableengine to any cross-engine matrix runner that enumerates
    engines (bench/sqlite_factory-style) where appropriate — else document exclusion.
20. Calibrate bigtable priors (NsPerOp/RTT) or mark explicitly UNCALIBRATED in profile comment.
21. Consider `Watcher`/SSE interaction note for versioned engines (notifyLive unchanged — verify).
22. Property-based (rapid) temporal test for memory chains (out-of-order stamps) — engine-level.
23. sqlite versioned-cells soak/restart test (history table survives restart).
24. bigtableengine restart-safety test (two engines over one bttest server).
25. Add `MapUpdateAt` to bigtableengine via optimistic ReadRow+Apply (document non-atomic) or skip
    (fold-lock serialization makes it optional) — decide + document.
26. Retention on bigtableengine: client-side MaxAge trim via `DeleteTimestampRange` option (GC
    policy covers MaxVersions natively) — or document GC-policy-only.
27. TODO_LIST.md harvest of this report's (f) items (docs-health HARVEST).
28. Investigate the daemon's `.golangci.yml` data-loss mechanics; propose guard (owner decision).
29. `ExecuteAsOf` + AsOf-input doc examples into METAENGINE_DOMAIN_LANGUAGE.md update.
30. cqrs-docs: DOMain language VersionedStorage row already exists; update with new capabilities.
31. Consider pebble/bbolt versioned cells (natural `DeleteTimestampRange`-style prefixes exist) —
    scope decision for next wave.
32. Consider `SystemTimestamp` policy option (strict event-time vs write-time) for engines where
    wall-clock fallback is undesirable.
33. Race-detector run over bigtableengine tests (`-race`, gRPC fake).
34. Soak env var run per docs/agents/gotchas-testing.md for the new module.
35. Bench: `BenchmarkCalibration_Bigtable_*` stubs (prior constants) — optional, post-calibration.
36. AGENTS.md gotchas: add "bttest vs BigTable: ms bounds, chain order, binary counters" note.
37. AGENTS.md module count 91 → 92 check (`find . -name go.mod | wc -l`).
38. Docs: SKILL.md read-model tier table — add versioned engines column note.
39. Reconcile `docs/planning/event-query-model.md` §temporal-roles with shipped AsOf routing
    (addendum, not rewrite).
40. Review `metaengine/memory_versioned.go` old wall-clock path: MapSet on versioned engine still
    stamps wall-clock (documented) — consider naming clarity (`recordVersionAt(now)`).
41. Cross-engine fuzz: reuse `fuzz_test.go` pattern for MapSetAt/MapGetAsOf (memory vs sqlite).
42. Dgraph/PG/MySQL engines: temporal capability gap note in their READMEs (not supported yet).
43. `system.AdapterCore` — verify AsOf routing works through `system.New` compositions (blast radius).
44. projectionadapter: verify event stamps survive `ApplyRecord` → folds on versioned engines
    (integration test exists at Store level; adapter-level test would pin the CQRS path).
45. CHANGELOG policy: no per-module changelogs (contract 20) — ensure only root edited.
46. Update `docs/METAENGINE_DOMAIN_LANGUAGE.md` AsOfSignal row (now documentation-type).
47. Verify `nix run .#test` (workspace mode) passes after go.work fix — it currently CANNOT run.
48. Consider engine-pool/failover interaction: reroute onto versioned engine mid-flight (documented
    loud-fail; maybe planner-aware reroute preference later).
49. api-stability `TestEvery` will also demand metaengine adttest golden symbols — run and fix.
50. Final `git log` review: squash-annotate the chore-absorbed phases if authored history matters
    for release notes (owner decision; do NOT rewrite without approval).

## g) Questions I cannot answer myself

1. **Daemon data loss:** the auto-commit daemon destroyed the `.golangci.yml` depguard section once
   this session (commit `c56d219a6`) and keeps racing edits/commits. Do you want me to (a) leave the
   daemon as-is and adapt, (b) add specific config files to an ignore mechanism (which — is there
   one?), or (c) is this worth pausing the daemon during multi-file registration work? Your call —
   it's your tooling.
2. **bigtableengine scope for v4.x:** ship as-is (Map + Counters + temporal, bttest-tested, 🧪 in
   FEATURES) or hold the module back from the api-stability golden/CHANGELOG until you've run it
   against a REAL BigTable instance (I have no GCP credentials here — real-service validation is
   unverifiable from this machine)?
3. **Commit history:** several logical phases were absorbed into `chore: auto-commit` by daemon
   races. For the release-notes narrative, do you want authored commits reconstructed where
   feasible (new commits, no history rewrite), or is chore-absorbed history acceptable this cycle?

---

_Report written 2026-09-18 14:07. Point-in-time snapshot — re-verify before acting._
