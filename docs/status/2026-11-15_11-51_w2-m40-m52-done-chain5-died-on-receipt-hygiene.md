# W2 Session Report — M40 Complete, M52 Determinism Wave Done, Parallel-Session Gate Debt Cleared, Chain #5 Died on Receipt Hygiene (2026-11-15, 08:49–11:51 CET)

Session context: resumed the standing "WHOLE TODO LIST" order over plan
`docs/planning/2026-11-13_21-12_SUPERB-truth-green-release-train-v5-runway-pareto-plan.md`
after the W1 report (M39 cleared, M40 two-thirds, chain #5 armed 07:36). A
parallel session was concurrently active the whole time (tagged
`metaengine/v4 v4.25.0`, commit ddcdfcce5, mid-session; later seen editing
`versions.json`, `README.md`, `modules.md`, `system/v4/raw_query_decoder_test.go`).

## a) FULLY DONE (this session, all verified by scoped suites + gates)

1. **Dead-man check (resume step 1)** — chain #5 confirmed alive (pid 1089341),
   riding the storm; tracked to its eventual outcome (see d).
2. **M40 :839 — projectionhost checkpoint cadence flake — FIXED + PINNED.**
   - Mechanism ESTABLISHED (the row said "no mechanism established"): `shouldFlushCheckpoint`
     compares wall-clock `time.Since(cpLastSave)` on the worker goroutine; under scheduler
     starvation the second live event's handling trails the first flush by >interval, so the
     interval threshold legitimately fires the extra save (saves=4). Scheduling-dependent test,
     not a cadence bug.
   - Fix: clock seam `projectionhost.WithClock(now func() time.Time)` (ADR-0153 D4 additive
     seam, same shape as `scheduling.WithClock`); worker reads `ProcessedAt`/interval/
     `cpLastSave` through it. Verified the catch-up drain never sets `cpLastSave` (saves via
     `w.cpStore.Save` directly), preserving the zero-value first-flush premise.
   - Rewritten test: manual clock advanced only by the test; pins the documented "at least d
     elapsed" boundary EXACTLY (advance == 120ms ⇒ flush); post-Stop no-pending count asserted;
     the 150ms wall-clock sleep REMOVED (sleeps baseline shrank 182→181).
   - Mutation-verified twice: first attempt (130ms advance) was too weak — both `>=` and `>`
     pass; corrected to exact-boundary 120ms → green on `>=`, RED on `>` mutation, restored.
   - Same-edit discipline: api golden (+1 `projectionhost.WithClock`), CHANGELOG Added row,
     readmodels.md cadence-testing sentence, doc-check green (1391 refs), changelog-symbols
     green, cadence suite 5×-race + full package -race green (10.0s). TODO_LIST row closed
     with full lineage receipt.
3. **M40 :1764 — dgraph+redis shuffle watch — OBSERVED-OUT with evidence.**
   ~9-month window since the 02-16 row; ZERO order-dependent dgraph/redis seed failures ever
   recorded (TODO_LIST carries none; `build/shuffle-seeds.log` none). Fresh verdicts: CI runs
   43688584591 + 43686251073 both `Dgraph Integration (ephemeral)=success` +
   `Redis Broker Integration (ephemeral)=success`; weekly Dgraph soaks green 3× (11-01, 11-08,
   11-15). Recording mechanism institutionalized (seed logs + `shuffle-seeds-redis` artifact +
   `#rerun-shuffle-seed`), so the standing row is unnecessary. GitHub's jobs API was flaky
   (persistent 502s on specific run IDs) — worked around via `gh run view --json jobs` on the
   healthy path.
4. **Parallel-session v4.25.0 gate debt — ALL CLEARED (found by running MY cheap gates before
   the chain could burn hours on them).** Their commit ddcdfcce5 shipped tagged code with:
   - api-golden drift: 7 missing lines (TrimPlannedBefore family, PlannedTableSweeper,
     AutoCRUDTyped/AutoInsertTyped/AutoUpdateTyped/AutoDeleteTyped) — regenerated.
   - NEW test sleep in `metaengine/v4/auto_typed_test.go` — annotated
     `//go-sleep:accept` (legitimate poll-interval spacing in a deadline-bounded retry loop).
   - NEW file-size offender `metaengine/sqliteengine/v4/planned.go` (381 lines) — split
     cohesively: layout registration (constructor/ApplyLayout/registerLayout/ApplyLayoutPlan)
     → new `planned_layout.go` (114 lines), helpers stay (276 lines). Build+vet green;
     `runTxReadModifyWrite` consumers unaffected (same package).
   - package-surface growth `metaengine/v4` 851→859 symbols / 174→176 files — reviewed (coherent
     typed-CRUD twins + sweeper interface, tested, part of a tagged release) and re-pinned with
     the gate's own `--update-baseline` after review.
   - Verified green after repairs: file-size, sleeps, package-surface, untested-exports,
     engine-capabilities, error-taxonomy, gofmt (`nix fmt` 0 changed), doc-check.
5. **M52 — json/v2 determinism wave tail — DONE + MUTATION-VERIFIED (Critical milestone,
   deps M32 satisfied; executed early because it was load-independent).**
   - Discovery: the wave was ~80% already landed (T05, 11-09 era): `keycodec.EncodeJSON`/
     `EncodeJSONErr`, `sqliteengine.encodeKey`/`encodeValue`, `EncodeStreamValueErr` all carry
     `json.Deterministic(true)`; both TODO rows were stale-premise-heavy.
   - The real tail — NINE remaining plain-marshal sites fixed: **`PlanFingerprint` (correctness
     class: hashed nondeterministic bytes into a "stable for structurally identical plans"
     SHA-256 contract — latent today, the Serializable* shapes carry no map fields; prophylactic
     now and test-guarded)**, `ExportStore` rows (backup byte-identity), `VectorMetadataArg`
     (stored metadata, contract-27 byte-identical storage), `SaveManifest`, SSE wire payloads ×3,
     `InspectJSON`, trace lines. `reify`/`reifyReflect` deliberately left plain (transient
     in-process round-trips, bytes never persisted).
   - New engine-level KEYS pin: `adttest.AssertMapKeyIdentity` — Set→Get (split read),
     Set→Set-overwrite (uniqueness), Delete (orphan) with equal-but-differently-built map keys;
     wired on pebble, bbolt, badger, sqlite (the keys half needs only MapBackend, so it RUNS on
     badger/bbolt where the value-bytes half skips on no-RawValueReader).
   - New fingerprint pin: `TestPlanFingerprint_StructurallyIdenticalPlansAgree` (20× rebuild).
   - Mutation-verified: stripping `json.Deterministic(true)` from `keycodec.EncodeJSON` fails
     the engine pin immediately (pebble -count=5); restored; metaengine root + keycodec +
     4 engine suites green 3×.
   - Same-edit: api golden (+1 `adttest.AssertMapKeyIdentity`), CHANGELOG row (changelog-symbols
     green, 25 citations), both TODO rows closed with receipts.
6. **Composed-gate truth (partial, see d)** — chain #5's full run PROVED green: build, vet,
   test, race, lint, and every ratchet gate it traversed — including file-size (my split),
   package-surface (my re-pin), sleeps, coverage floors (pkg/rules/version 89.5 held;
   metaengine/v4 floor 70.5 held under the parallel session's new code), cycles, api-golden,
   md-go, engine-capabilities, error-taxonomy. Only the LAST leg failed, on my receipts.

## b) PARTIALLY DONE

- **M34 composed-verify receipt**: the composed run EXECUTED end-to-end (entered 10:11 CET at
  load1 2.30 after ~35 storm-riding refusals) but exited rc=1 at the doc-assertions leg, so no
  green receipt yet. One 4-row TODO_LIST hygiene fix away from a legitimate green re-run.
- **M35 release wave**: artifacts inventoried (manifest `build/wave-r11-13.manifest` ~29
  trains, `build/f221-changed-set-tests.sh` per-module runner); manifest's metaengine line is
  stale (says v4.24.0; v4.25.0 tagged mid-session; next would be v4.26.0 carrying my M52 +
  post-4.25 bits); projectionhost needs a NEW row (WithClock export). NOT executed — the
  parallel session was seen editing versions.json/README/modules.md (release-manifest shape)
  and executing test sweeps; ownership collision unresolved (question 3).
- **Chain protocol**: chain #5 dead (max retries used on a REAL failure, correctly not
  retried). Re-arm decision pending (question 2).

## c) NOT STARTED (gated on chain-green / M34 / M35, or owner decisions)

- Quiet-window `check-coverage.sh --update` (pins the 3 STALE improvements; F250a guards
  decreases) — note load1 5.16/load5 3.27 AT session end: a quiet window exists NOW if the
  ~1h re-measure is wanted immediately.
- M35 full wave: changed-set tests, batch-release, versions manifest, release-scripts,
  CHANGELOG train note, pre-push battery (#check-generated-artifacts, #erraudit through the
  new app — never yet exercised, #check-flake-lock-coherence --baseline origin/master).
- M36 D-sheet v2 fold (D16/D17/Q3 + 16 more owner-gated rows routed).
- W2–W4 per plan: M41 v5 deletion program (15 rows), M42 v5 cut + migration docs, M43 core/v5
  completion + codemod, M44 metaengine promotion (T27 evidence table, 16 stamps),
  M45 keycodec-v2 dual-read migration, M46/M47 testing-rigor programs, M48 cqrs-lint pack,
  M49 BDD pack, M50 catalog truth, M51 substrate tails, M53 quiet-window campaign,
  M54 fleet census, M55 docs/quality tails, M56 DuckLake gate/spike, M57 upstream matrix.
- Sunday 2026-11-22 nightly re-observation (first clean load-sweep post-+x-fix).

## d) TOTALLY FUCKED UP (honest ledger)

- **Chain #5 died on MY OWN receipts — the worst own-goal class.** The TODO_LIST hygiene gate
  failed 4 rows, all mine: :962 (checkpoint flake receipt — "[x]" with a 2026-10-23 date >
  14d old), :1284 (json/v2 values row, dated 2026-10-25), :1308 (the adjacent "Closed
  2026-10-25" block), :1882 (dgraph+redis row — "[x]" without a YYYY-MM-DD date on the
  checkbox line). I closed rows by flipping `[ ]`→`[x]` + appending receipts, WITHOUT reading
  the header policy (done rows: re-date or delete; >14d-old dates must be deleted). The
  composed gate burned ~1.5h and died at the last leg because of it. Fix is 4 small row edits,
  but the chain is spent.
- **First mutation test too weak** (M40): advanced the fake clock 130ms to test a `>=`/`>`
  boundary — both operators pass at 130ms; corrected to exact 120ms boundary. Caught by
  re-reading, not by the first red.
- **Sloppy no-op perl**: ran a substitution that replaced text with itself across 4 engine
  test files — content unchanged but mtimes bumped, triggering edit-tool mtime races. Should
  have used the edit tool from the start.
- **Near-miss carried, not caused**: chain #5 was armed (by W1 me) over a tree that already
   carried the parallel session's v4.25.0 gate debt (sleeps/file-size/package-surface/api-golden).
   If the storm had broken at 08:20 instead of 10:11, the chain would have died ~2h earlier on
   THEIR debt. I found it by luck of running my cheap gates first. Lesson: run the cheap gate
   battery BEFORE arming any chain, over the whole tree, not just the changed set.

## e) WHAT WE SHOULD IMPROVE (process, from this run)

1. **Receipt-policy check belongs in the same edit as the receipt**: a 5-second
   `bash scripts/check-todo-list-hygiene.sh` after closing rows would have caught all 4
   violations before the chain ever ran. Add to the close-out ritual.
2. **Pre-arm gate battery**: extend verify-chain.sh (or the pre-arm preflight row :321 ritual)
   with the cheap ratchet gates (sleeps, file-size, package-surface, untested-exports,
   api-golden drift) so a chain never launches over parallel-session debt. They run in
   ~2 min combined vs 1.5h of composed gate.
3. **Parallel-session gate debt has no owner signal**: their tagged release left 4 gates red
   and only an accident surfaced it. Consider: release ritual (tag-release/batch-release)
   should run the cheap-gate battery as a pre-tag step — it already runs verify=ok builds;
   cheap gates are faster than that.
4. **Todo row dating**: the hygiene gate's policy (delete >14d done rows) fights the
   docs-health lineage convention (preserve receipts). The two should be reconciled by the
   owner or the rows should carry their RESOLUTION date, not the ORIGINAL finding date, in the
   checkbox position the gate parses. (This is what my 4 rows got wrong.)
5. **Mutation discipline**: boundary pins need the EXACT boundary value, not a
   comfortably-beyond-it value. Write the mutation FIRST, then choose the pin value that
   separates the operator/branch under test.

## f) NEXT — up to 50, in execution order

1. Fix the 4 TODO_LIST hygiene rows (re-date receipts to 2026-11-15 or delete per policy).
2. Re-run `bash scripts/check-todo-list-hygiene.sh` + `check-todo-stale-premises.sh` green.
3. Re-arm chain #6 (load is QUIET right now: 5.16/3.27 — the gate could run immediately).
4. On chain #6 green: M34 dated composed-verify receipt (close :338 T25 + :519 composed-proof
   row; F214 note-only — push is owner-gated).
5. `nix run .#check-lint-config` (flake.nix was edited in W1 — never yet re-verified).
6. Phase-boundary authored commit (2026-11-03 ruling) covering the M40+M52+debt-repair wave.
7. Quiet-window `bash scripts/check-coverage.sh --update` (~1h; pins the 3 STALE
   improvements; F250a guards decreases) — window may be NOW.
8. Coordinate M35 ownership with the parallel session (see questions).
9. M35: re-derive the wave manifest (metaengine/v4 → v4.26.0 post-v4.25.0; NEW projectionhost
   row for WithClock; re-read tag state — tree moved).
10. M35: `build/f221-changed-set-tests.sh` over the updated manifest (~10 min warm).
11. M35: `./scripts/batch-release.sh --from-manifest build/wave-r11-13.manifest` (collect
    refusals = pass B evidence).
12. M35: `bash scripts/check-versions-manifest.sh --update`.
13. M35: `nix run .#check-release-scripts`.
14. M35: CHANGELOG release-train note.
15. M35 pre-push battery: full `#check-generated-artifacts`.
16. M35 pre-push battery: full `#erraudit` sweep (never yet run through the new app).
17. M35 pre-push battery: `#check-flake-lock-coherence --baseline origin/master`.
18. M36: D-sheet v2 (fold D16/D17/Q3 + route the 19 owner-gated rows; one-pager).
19. M44 (T27): per-module evidence table for all 16 engine-family stamps.
20. M44: stamp flips → FEATURES/SKILL/matrix alignment → api-stability regen (17 doc.go files).
21. M44: `memoryengine` extraction (:207/:2371).
22. M44: seven-package carve per ADR-0162 (:2363).
23. M41: `stack/*` deletion wave w/ census (T10).
24. M41: read-tier deletions (Materialize, RelationalProjection+view, GraphProjection,
    tombstone metadata API, ADR-0126 shells, BuildWhereClause, NewStreamRef validation) (T11).
25. M41: Phase A deriver ABSORB row; wire-vocabulary tail; ghost-module tail (10 more rows).
26. M45 (A9): pebble keycodec-v2 reference leg (alias block, getAny fallback, dual scans,
    seq seeding, reset ranges, sort-index grammar).
27. M45: badger/bbolt twins; legacy storage/pebble + storage/bbolt keys + bbolt journal idx
    rekey; README migration notes; per-engine conformance + v1-fixture acceptance.
28. M46: mutation-verify gate (tool pick + 3 pilots + CI ratchet).
29. M46: synctest wave 1.
30. M46: planner MBT via rapid (T18, :331).
31. M46: error-taxonomy behavior tests + deprecation pins (T16 tail).
32. M47: differential engine harness (T19).
33. M47: tail-latency matrix; heap/goroutine/disk budgets; crash-consistency;
    broker/DLQ/outbox matrix; context-cancellation write-path; companion-auth tail.
34. M52 residue check: fold the 11-09 system-level Deterministic fix into the test matrix
    (verify no system-layer marshal site remains plain) — small, rides M46/M47 test waves.
35. M48: cqrs-lint 3.7× analysis regression profile + bench-pin; #79 typed-load fixture;
    #71 ScanWithMeta; #83 baseline hardening; Doctor-JSON pre-merge semantics;
    --show-suppressed parseability; F024/F025/F005/B005/F032/T009 quality pack (19 rows).
36. M49: cqrs-htmx sqlite-lifecycle harness; TimeAdvancesTo sugar; observational-equivalence
    port; scenario TB refactor; version-carrying Given; Registry↔DomainConfig bridge;
    graph ADT recipe; journal dump⇄load round-trip (12 rows).
37. M50: catalog hub asks; warn-budget hardening; config-with-changelogs golden; upstream
    EventCatalog issues; AsyncAPI response schemas + bindings/security; COOKBOOK md-go gate
    (16 rows).
38. M51: metaengine substrate tails — stamped envelope spike; codec scope ruling; ingress
    marker unification; EncodedPayload ctor; #72 caps cluster; idempotency-retirement ADR
    (:1341 — HOT, the ADR ritual is documented in the row) (21 rows).
39. M53: SearchQuery count=5 re-run + dgraph re-anchor; MySQL-VM shuffled replay; hardened
    vm-mysql leg; bigtable soak; snapshot-migration mysql leg (7 rows; needs quiet windows).
40. M54: fleet census cross-check; systemscenario diffusion 10→15; untagged-API gate T26;
    system pin sweep (9+ rows).
41. M55: 350-line split program; API-golden tiering; duplication/naming hygiene; 190-group
    disposition; templ split-brain note (19 rows).
42. M56: DuckLake gate/spike only — external-claim pinning, vendored-extension LOAD+ATTACH
    spike, ADR, go/no-go (4 rows).
43. M57: upstream/blocked matrix — Turso matview runbook + filings; tursogo segfault filing;
    md-go upstream; cmdguard proposals; command-sourcing ADR (25 rows).
44. Re-observe Sunday 2026-11-22 nightly log for the first CLEAN load-sweep post-+x-fix.
45. Sweep the R11-13 manifest's ~20-row untagged backlog (:331-class) during M35.
46. Row the W1-era M34/M35 receipts + this session's into FEATURES.md if feature-visible
    (WithClock is consumer-facing; M52 is behavior-visible — CHANGELOG rows already landed).
47. Verify the parallel session's v4.25.0 coherence fully at M35 time (versions manifest,
    README manifest — they were mid-edit when last seen).
48. erraudit: the 255 error findings from the deep-dive — triage wave (rides M39's :1655-class
    watch or its own session).
49. Consider a docs-health HARVEST pass at session end (docs/status reports → TODO_LIST rows).
50. Final: status report + README index row (this file) — then hold D16/D17/Q3.

## g) QUESTIONS (cannot be resolved from the repo)

1. **D16 (push)**: once chain #6 is green and the M35 wave is tagged, may I push master +
   tags? Push has stayed blocked all session without an explicit ask; the origin-green F214
   receipt also needs a push to ever close.
2. **Composed-proof policy under contention**: chain #5 burned its patience window partly
   because the PARALLEL session's test sweeps held load1 above the ceiling 10 for hours. When
   chain #6 hits the same contention, should I (a) keep re-arming quiet-window chains
   (honest load context, unbounded wall time), or (b) switch to `#verify-serial`
   (VERIFY_FORCE=1, serial packages, 6h bound — composed but under forced load)? The tier
   exists exactly for this and has never been exercised.
3. **M35 ownership**: the parallel session is editing `versions.json`/`README.md`/`modules.md`
   (release-manifest surface) and running module test sweeps. Do they own the R11-13 release
   wave, or do I take it after chain #6 green? Two agents tagging the same wave is the
   collision I cannot deconflict myself.

— Session ended per interrupt: REPORT WRITTEN, WAITING FOR INSTRUCTIONS.
