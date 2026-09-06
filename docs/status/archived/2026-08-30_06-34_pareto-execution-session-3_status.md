# Status: Pareto Plan Execution (Session 3) — A1+A2 Done, A3 Interrupted

**When:** 2026-08-30 06:34 CEST · **Session window:** 2026-08-29 ~20:45 → 2026-08-30 ~06:35
**Baseline:** `0c4cfe43c` (pareto plan commit, master, pushed) · **Now:** `b2930ff1f` (3 commits ahead, NOT pushed)
**Plan being executed:** `docs/planning/2026-08-29_20-33_pareto-v4-closeout-and-v5-train-plan.md` (27 medium tasks A1–D9)
**Working tree:** clean except untracked `metaengine/dgraphengine/graph_parity_debug_test.go` (A3 debug probe, see §d) and the foreign `t/tasks.buf` (now gitignored).

## Commits this session (3, per-task, no mega-commits)

| Commit      | Content                                                                                                     |
| ----------- | ----------------------------------------------------------------------------------------------------------- |
| `8c1009983` | test(a1): duplication-gate annotations (6 files) + snapshot `TransformedStore` floor tests (new file, +238) |
| `16e2d4c31` | fix(a1): verify-fallout sweep — 15 files of lint/dead-code fixes across 9 modules + `.gitignore` for `t/`   |
| `b2930ff1f` | docs(a2): projectionhost doc-lie purge (Start godoc, doc.go, README, advanced.md)                           |

---

## a) FULLY DONE

1. **A1 duplication gate** — ran `check-duplication` for the first time on the two change waves; 4 new clone groups found and triaged as intentional (mutex guards, per-facet DQL queries, fold prologues); annotated live with `//art-dupl:accept` per AGENTS.md #14. Gate now: 0 new clones (baseline 111 groups).
2. **A1 coverage gate** — first run exposed `snapshot` at 74.8% vs 91.9% documented: `transform_store.go` (encrypted-snapshot feature, commit `31d4a4638`) shipped with ZERO tests. Wrote `snapshot/transform_store_test.go`: ctor validation (nil store / nil transforms ×4), protect/restore round trip with inner-store prefix assertion + routing-metadata survival, LoadAtVersion hit+miss, and family classification of every failure mode (Rejection/Infrastructure/Corruption). Snapshot now 92.6–93.2%, gate GREEN (all 11 modules within ±2%).
3. **A1 check-arch gate** — GREEN, no breaches.
4. **A1 full `#verify` battery** — eventually ALL GREEN: build ✓, vet ✓, test ✓ (~120 pkgs), race ✓, lint ✓ (76 modules, 0 issues), check-arch ✓, check-depguard ✓, check-docserver-css ✓, check-duplication ✓, check-coverage ✓, api-stability ✓ (4328 exports), doc-check ✓ (1154 refs), verify-docs ✓. Took 5 attempts (see §d for why).
5. **Lint fallout fixed (9 modules)** — dead `schemaMu` variable in catalog (dead code removed), unchecked type assertion in `snapshot/read_pressure.go` `evictOldest` (now tolerates a foreign LRU value instead of panicking), missing `skipped` field in `SQLiteDeadLetterStore` ctor, `up`→`upcaster` varname in `schema/registry.go`, stale `nolint:gosec` removed in pebble, `contextcheck` justified in host Stop, `perfsprint` in dgraph transaction.go, `errcheck` rows.Close in pgengine vector.go, exhaustruct/wsl in commandlifecycle, godox "bug" wording, nlreturn, embeddedstructfieldcheck (6 fixtures — nolint with reason, the embedded-then-regular layout IS the case under test).
6. **B8.4 `t/tasks.buf` origin — SOLVED** — the 1 MiB all-zeros file was committed by accident in `2b602b55e` (2026-08-08), deleted in `0a85573d2` (2026-08-16), and an EXTERNAL task tool ("t") recreates it whenever it runs with this repo as cwd. Fixed with a documented `/t/` .gitignore entry (file left untouched on disk per prior instruction).
7. **A2 doc-lie purge** — all four sites fixed: `host.go` Start godoc ("fresh Host" lie → same-host Stop→Start rebuild + counter-reset note); `doc.go` DLQ admission (family-gated: only Rejection/Corruption park, retryable → loud WorkerFailed) + Reset ordering rationale (checkpoint BEFORE projection reset, crash-safety argument); `README.md` DLQ bullet + Reset section (+ForceStop/CheckStaleness contracts); `advanced.md` lifecycle steps 5–7 rewritten (family-gated error handling, poison admission, Stop→Start rebuild). doc-check GREEN after.
8. **Environment discoveries locked in** — `/tmp/.Trash-1000` held 37 G (tmpfs!) — emptied, now 33 G free; golangci-lint caches corrupted by earlier ENOSPC events (phantom "undefined: json" / "no such file" typecheck storms) cured by wiping GOLANGCI_LINT_CACHE and moving GOCACHE to disk (`/home/lars/projects/.gocache-disk`); verify scratch (GOTMPDIR/TMPDIR) must live on disk, not the shared 48 G tmpfs.

## b) PARTIALLY DONE

1. **A3 dgraph graph-parity** — prep done: root-cause hypotheses formed (off-by-one `@recurse` depth semantics vs `extractNeighborIDs` descent; `loop:false` + bidirectional-edge interactions), failing fixtures understood (`GraphDepthBound`: chain A→B→C→D→E depth2 expect {B,C} got {B}; `GraphDepth3Diamond`), `graph.go:176-268` read, reference DQL dumped into a debug probe test (`graph_parity_debug_test.go`, compiles, vetted). **BLOCKED mid-flight:** the ephemeral-Dgraph debug run launched ~22:30 hung overnight; its log vanished with the /tmp cleanup; the nix app + Dgraph Zero/Alpha survived 8 h in a wedged state (Alpha loop-failing on its own gRPC port). Tree killed + datadir removed at 06:35. The debug test file remains for the next run. NOT started: actual response-shape capture, fix, matrix green.
2. **C3 relational one-tx-per-event** — VERIFIED ALREADY-IMPLEMENTED, not yet closed in TODO_LIST: `RelationalProjection.Handle` (projection.go:135-151) already wraps every event in ONE tx (BeginTx → handler → Commit, deferred Rollback), `sqlSink` has exactly one construction site (inside that tx), and `tx_test.go` covers raw-SQL commit/rollback semantics. The 2026-08-16 TODO line L55 is stale. Needs: check the checkbox with evidence + drop the 90-min C3 task from the plan.
3. **C10 macOS claim honesty** — verified the script header already carries an honest "static review done, NOT exercised on real Mac hardware" note (2026-08-16); README/CONTRIBUTING scanned for overclaims (none found — no marketing of macOS support). Remaining: CI-matrix note + closing/editing TODO L266 wording.
4. **C4 durability audit (data gathered)** — per-engine matrix: only dgraph/duckdb/mysql/turso call `RejectDurabilityTier`; badger/bbolt/iroh/pebble/pg/sqlite NEITHER reject nor report tiers (silent-ignore = "durability lie" per the module's own doc). No ADR written yet, no engine gaps filled yet.
5. **C9 scans (read-only parts done)** — `BuildWhereClause` consumers enumerated (storage/relational ×2, storage/view ×2 — all internal to the v5-deletion set, so D5.3 delete is safe); `normalizeAny` (quic/latency.go:84) located for C1.1 table tests; pool eviction (`pool.go:94`) and dedup ring (`transport.go:105`, capacity 10000) located for C1.2–C1.4.
6. **B4 wave-1 CHANGELOG backfill** — research done: wave-1 (`ce98b2dda`) exported symbols enumerated (`WithVersionCacheCapacity`, `deriver.WithMaxDepth`/`AsHandler` options, `kv.Cache.Invalidate/InvalidateAll`, `Host.ForceStop`, `snapshot.WithReadTrackingLimit`, `sql.ValidateJournalIdentifiers`, `schema.ErrInvalidUpcastResult`) plus the behavioral changes. Not written into CHANGELOG yet; symbols gate not run.

## c) NOT STARTED (from the plan, in order)

1. **A3 fix itself** (see §b.1) — the 1%-tier gate "honest GREEN" still has 2 red parity tests.
2. **B1 depguard restore** — old allow list extracted from `git show 38afb6d2e^:.golangci.yml`; current nesting measured (items land at 24 spaces under `linters.settings`); `check-depguard.sh` awk needs the indentation-tolerant rewrite (hard-codes 12-space items). No config changes made yet.
3. **B2 behavioral tests** (dgraph read-your-writes + caller-retry-after-abort + MariaDB DESC twin-column sort).
4. **B3 PG live-tests** (VectorCount/Collections + projectionhost pg suites via `nix run .#integration-pg`).
5. **B5 AGENTS.md memory sweep** (quic replace-strip, depguard state, TEST_ARGS passthrough, twin columns, readTx routing, /mnt→/tmp→disk cache chain, tmpfs-trash trap).
6. **B6 skill recipes** (RunInTx, VectorCounter/Doctor, KeysetPositionQueryChecked migration note, projectionhost contracts — partially delivered already by A2's advanced.md rewrite).
7. **B7 tag-wave prep** (pin sweep, vulncheck, GOWORK=off matrix, quic replace-strip dry-run).
8. **B8 remainder** (ErrWorkerFailed sentinel, WorkerState counter-reset doc — partially covered by A2, boundedMap approximation comment, TODO checkbox reconciliation).
9. **C1 iroh QUIC hardening** — all 6 subtasks.
10. **C2 bench day** (ReadCosts for badger/bbolt/pebble — confirmed they declare NO ReadCosts today; Turso CTE probe).
11. **C4** ADR + engine gap-filling (see §b.4).
12. **C5 routing/lifecycle** — concretely identified: `routingSignature` (store_routing.go:116) fingerprints ONLY NetworkRTT — misses NsForRead (used by checkQueryRouting) and plan version (a Replan with unchanged RTTs serves stale cached diags); `Calibration` setters are unsynchronized (data race with concurrent Profile()); hysteresis gates suggestions only, not Replan assignments; over-declared Supports has no plan-time diagnostic.
13. **C6 verify-standalone + CI leg.**
14. **C7 release automation** (GH Releases script, retract-and-republish + pre-tag checklist, replace-drop sweep, indirect-dep consolidation).
15. **C9 design notes** into V5-OUTLINE (E1/E7/E8/E11/E13/E15).
16. **D1/D2/D3 planned tables** for pgengine/mysqlengine (reference impl read: `sqliteengine/planned.go` — ApplyLayout/ApplyLayoutPlan/registerLayout/execPlannedSet/mapGetPlanned/mapUpdatePlanned + `PlansColumnCompatible`; pg has only `ApplyLayout` at pushdown.go:107, no ApplyLayoutPlan; mysql has pushdown.go only).
17. **D4/D5/D6 v5 deletion waves** (all three; confirmed TODO intent is deletion on master, tags gated).
18. **D7 v5 migration guide + cut checklist.**
19. **D8 ClaimingTimerStore** (scheduling/sqlstore has no SKIP LOCKED today).
20. **D9 user-blocked** (billing, root, macOS, external tags) — untouched, awaiting input.

## d) TOTALLY FUCKED UP

1. **The overnight A3 debug run wedged for 8 hours.** The ephemeral-Dgraph nix app launched at ~22:30 never completed the debug test and never tore down; Zero/Alpha sat spinning (Alpha erroring on its own gRPC port every second) until I killed the tree at 06:35. Cost: the whole night. Likely cause chain: `go test` compiled against the brand-new disk-backed GOCACHE (first compile of dgraphengine+deps there), while simultaneously /tmp cleanup deleted `/tmp/dbg-dgraph.log` — the run's stdout — and possibly evicted build scratch (GOTMPDIR pointed at disk, but `go` still uses os.TempDir for some link steps unless TMPDIR is set; the earlier ENOSPC crashes also poisoned caches). Root behavioral failure: **I launched a long unattended job at end-of-session without a watchdog** — the same class as the orphaned-QEMU gotcha already documented in AGENTS.md, now proven for Dgraph too.
2. **Three wasted full-verify attempts on tmpfs exhaustion.** Attempts 1–3 (and the build-only repro) died with "no space left on device" in the linker/cgo — 48 G shared tmpfs at 99% while I fed it MORE (GOTMPDIR=/tmp/go-build-tmp made benchkit/pebble tests write multi-GB scratch into the same tmpfs). The fix (move GOCACHE+GOTMPDIR+TMPDIR to the 723 G disk, empty the 37 G `/tmp/.Trash-1000`) was found only after burning ~2 hours.
3. **ENOSPC-corrupted caches produced ghost lint failures.** After the disk-full crashes, golangci-lint reported ~34 phantom typecheck errors in cmd/cqrs-lint ("undefined: json", unreadable module files) that do NOT exist — stale export-data in GOLANGCI_LINT_CACHE/GOCACHE. I nearly started "fixing" nonexistent broken imports. Cured by cache wipe, not code edits. Lesson recorded: after ANY ENOSPC crash, wipe caches before trusting diagnostics.
4. **`go clean -cache` mid-session nuked a 12 G warm cache for nothing.** I cleaned it to free tmpfs, then verify4 still ENOSPC'd anyway — and verify5 had to recompile the entire 82-module workspace from scratch. The correct order was: free the trash first (37 G), THEN decide about caches.
5. **My own `sed` fix introduced a compile break.** The `up`→`upcaster` rename in `schema/registry.go` was applied with a partial sed (missed `up.SourceVersion()` on line 48 and the append on line 53), breaking the schema package — which then cascaded phantom "undefined: schema" failures into event's lint run. Caught by lint, fixed properly, but it's exactly the "edit without re-reading the whole symbol" failure the workflow warns about.
6. **t/tasks.buf sat unexplained through two sessions** — the answer (git history: added `2b602b55e`, deleted `0a85573d2`, resurrected by an external `t` tool) took one git command; B8.4 should have been done sessions ago instead of carrying "origin unknown" anxiety.

## e) WHAT WE SHOULD IMPROVE

1. **Never launch an unattended job without a completion check + timeout** — every nix-app run needs a bounded `TEST_TIMEOUT`, a log path OUTSIDE /tmp, and a poll before session end. A hung integration run must be treated like the orphaned-QEMU class: kill tree + remove state dir + record.
2. **Move ALL build scratch off the shared tmpfs permanently** — record in AGENTS.md: `GOCACHE=/home/lars/projects/.gocache-disk`, `GOLANGCI_LINT_CACHE=/home/lars/projects/.golangci-disk`, `GOTMPDIR=TMPDIR=/home/lars/projects/.gotmp`, keep GOMODCACHE wherever there's >3 G. /tmp is 22-user shared and has a 37 G trash sink on the same mount.
3. **After any ENOSPC, wipe caches before diagnosing** — corrupted golangci/go caches fabricate believable phantom errors ("undefined: json" in a file that compiles fine).
4. **The verify chain takes 5+ attempts when run cold** — worth a `#verify-fast` variant that skips race on the ~10 slowest modules, or documenting a "phased verify" (lint+gates first, then test, then race overnight).
5. **check-coverage.sh's nix wrapper runs without the cache env and reports 0.0% DRIFT for everything** — it misleads (looks like real drift, is env failure). Make the app export the env itself or fail loudly when `go test` output is empty.
6. **Coverage gate gap found and closed but instructive:** a whole feature file (`transform_store.go`) shipped untested because the gate had never run. Any new module/file needs the gate run in the SAME session, not "eventually".
7. **Deletion waves D4–D6 need a consumer-impact note per deletion** (guardrail #3) — draft those notes BEFORE deleting, not after.
8. **The Dgraph parity fix should reuse the debug probe as a permanent regression test** (dump-and-compare response shape at depth 2/3) rather than deleting it after the fix.
9. **Two status updates to the user during an all-night run** would have surfaced the wedged A3 job hours earlier.

## f) NEXT 50 (ordered by the plan's tiers; ~first 15 are the critical path)

**1% — finish honest GREEN**

1. Re-run A3 debug probe with log on disk + TEST_TIMEOUT=120; capture RAW @recurse JSON at depth 2/3 (file ready: `graph_parity_debug_test.go`).
2. Fix `graph.go` depth semantics (recurse query or `extractNeighborIDs` descent) per captured evidence.
3. Green `TestDgraphADTMatrix/GraphDepthBound` + `GraphDepth3Diamond` on ephemeral Dgraph; then FULL dgraphengine suite.
4. Convert debug probe into a permanent shape-pinning test; delete the temp file; commit A3.
5. Re-run the remaining verify gates (dup/coverage/arch/api/doc) — only test+race should need re-running.

**4% — enforcement + honesty**
6. B1: rewrite `check-depguard.sh` awk to indentation-tolerant (relative-indent capture).
7. B1: restore depguard settings block from `38afb6d2e^` at current 24-space nesting; drop from `disable`.
8. B1: run golangci; triage surfaced violations; add genuinely-new deps (dgo, iroh-ffi, etc.) to allow list; commit.
9. B2.1: dgraph in-tx read-your-writes test (MapSet→MapGet inside RunInTx).
10. B2.2: dgraph caller-retry-after-abort test (error → rollback → second RunInTx commits; catches stuck activeTxn).
11. B2.3: MariaDB DESC twin-column sort + keyset pagination live test (start userspace MariaDB first).
12. B3: ephemeral PG; run projectionhost pg_integration + pg_testcontainer suites; add pg VectorCount/Collections live tests; fix fallout.
13. B4: write wave-1 Added/Changed/Fixed into CHANGELOG [Unreleased] (symbols enumerated in §b.6); run check-changelog-symbols; doc-check.
14. B5: AGENTS.md sweep — dgraph orphan-cleanup gotcha, disk-cache env chain, tmpfs-trash trap, TEST_ARGS passthrough, gcn_ twin columns, readTx routing, quic replace pending-strip, depguard restored state; TODO reconciliation (L66, L49 tail, L437 verify+close).
15. B6: recipes 2.24 RunInTx, 2.25 VectorCounter/Doctor vector-sizes; faq.md KeysetPositionQueryChecked note; advanced.md already fixed by A2.
16. B7: enumerate changed modules vs last tags; pin-bump sweep; GOWORK=off build matrix; `#vulncheck`; tag-release.sh dry-run incl. quic replace strip; refresh cqrs-lint taskmanager golden if version set changes.
17. B8.1: `ErrWorkerFailed` sentinel + staleness rewire (additive; api-stability regen).
18. B8.2/B8.3: WorkerState counter-reset doc (done in A2 — verify and close); boundedMap `stale` negative-dip comment; catalog multi-embed conflict note.
19. B8.5: close L49 checkbox; commit batch.

**20% — features/automation**
20. C1.1: `normalizeAny` table tests (uint64/int64/slices/maps/MaxInt64 edge).
21. C1.2: dedup.Ring >10K eviction regression via transport markSeen (internal test).
22. C1.3: pooled-stream eviction error-injection test (frame-write error → evict → next send reopens).
23. C1.4: 1K-op pooled stress test.
24. C1.5: framing constants dedup + `WithStreamPooling` README row; CGo quic suite run.
25. C2.1–C2.3: quiet-window calibration benches badger/bbolt/pebble; wire 4-field ReadCosts constants; delete the SA1019 exclusion; regression ×3.
26. C2.4: Turso CTE-probe test mirroring sqliteengine.
27. C2.5: calibration-vs-baseline run; update BENCHMARKS.md.
28. C3: close TODO L55 as already-implemented with file:line evidence (work done, bookkeeping left).
29. C4.1: write durability-mapping ADR (per-engine tier→setting table incl. the 6 silent-ignore engines).
30. C4.2: add RejectDurabilityTier or real tier support to badger/bbolt/iroh/pebble/pg/sqlite factories.
31. C4.3: verify Doctor/Introspection surface EffectiveDurability; fill gaps.
32. C5.1: add mutex to Calibration (setters + ApplyCalibration + LiveLatency); race test; watch copylocks.
33. C5.2: routingSignature += plan version + NsForRead per read pattern.
34. C5.3: gate Replan assignments with hysteresis/minDelta; oscillation test.
35. C5.4: plan-time diagnostic for over-declared Supports + routing penalty.
36. C6: `#verify-standalone` flake app (GOWORK=off per-module build+test) + CI leg for leaf modules.
37. C7.1: `scripts/create-github-releases.sh` (gh CLI, changelog extract per tag).
38. C7.2/C7.3: CONTRIBUTING retract-and-republish + pre-tag checklist.
39. C7.4/C7.5: post-tag replace-drop sweep + indirect-dep consolidation.
40. C9.1–C9.5: BuildWhereClause note (consumers all internal — safe to delete in D5.3), snapshot wire-tag design note, transport deprecated-consumer scan, E-item design notes → V5-OUTLINE.
41. C10: CI-matrix note in ephemeral-pg.sh docs + close/adjust TODO L266.

**80% — v5 train**
42. D1: pgengine planned.go (port sqliteengine/planned.go: ApplyLayoutPlan + typed writes + reads + information_schema evolution + PlansColumnCompatible guard); live PG matrix + mis-type regression test.
43. D2: mysqlengine planned.go (MariaDB `ADD COLUMN IF NOT EXISTS` vs MySQL divergence); live MariaDB matrix.
44. D3: planned filter/sort pushdown both dialects; EXPLAIN index-usage proofs; injection adversarial tests; cross-engine parity vs sqlite fixtures.
45. D4: v5 deletion wave A — stack.Materialize, Bundle+8 presets, RunProjections; rebalance testModules/api list; consumer-impact notes; GOWORK=off matrix.
46. D5: v5 wave B — storage view+relational tiers, graph.GraphProjection, BuildWhereClause (post C9.1), ADR-0126 shells.
47. D6: v5 wave C — transport/http+grpc modules, tombstone metadata API (listing bridge stays), NewStreamRef breaking validation, snapshot wire tags, E-items.
48. D7: full v5 migration guide from V5-OUTLINE + cut checklist + post-landing sweep + dry-run cut on worktree.
49. D8: ClaimingTimerStore (SKIP LOCKED schema + lease) + two-Scheduler no-double-fire + lease-expiry tests (PG live).
50. D9: present the user-question list (below) and execute whichever blocks lift.

## g) QUESTIONS FOR YOU (cannot self-answer)

1. **DLQ semantics ratification (carried over, still unanswered):** the fail-loudly flip is now shipped AND documented (A2) — non-retryable poison parks in the DLQ, retryable failures restart and end in WorkerFailed when the budget is spent. Keep as-is, or do you want an opt-in flag preserving the old park-everything behavior for consumers who prefer silent-quarantine?
2. **depguard (carried over, now actionable):** plan default is restore-and-triage (B1). Confirm, or do you prefer ratifying its disabled state with an ADR instead (the allow list is ~90 entries of maintenance burden)?
3. **Overnight jobs:** may I run long verification/integration batteries unattended with an auto-kill watchdog (e.g. 30 min hard timeout, log to /home), or do you want every long run gated on you being around? The A3 stall cost a full night and I'd rather set the policy once.
