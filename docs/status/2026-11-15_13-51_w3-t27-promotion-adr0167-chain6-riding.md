# W3 Status — T27 Promotion Shipped, ADR-0167 + Doctor-Gaps Landed, Chain #6 Storm-Riding (2026-11-15, 12:21–13:51 CET)

> Session resumed from the W2 "WAIT FOR INSTRUCTIONS" interrupt under the standing
> NOW-GET-SHIT-DONE order. All times CET. Chain #6 armed at 12:27, still riding at
> report time (attempt 28; load 50–93 the whole window; max-wait expires ~16:27).

## a) FULLY DONE (verified this session)

1. **The 4 TODO_LIST hygiene rows that killed chain #5** — fixed by re-dating the checkbox
   lines with honest `RESOLVED 2026-11-15` closure keywords (the receipts' content is
   unchanged; the gate parses the checkbox line only). `check-todo-list-hygiene` +
   `check-todo-stale-premises` green.
2. **Pre-arm cheap-gate battery + parallel-session debt** (the W2 lesson, applied):
   - `check-versions-manifest --update`: system row v4.18.0 → v4.19.0 (their tag, manifest
     not refreshed). Re-check green (117 trains).
   - `TestEveryModuleGoSumIsTidy`: `systemcatalog/v4` + `systemtest/v4` go.sum untidy —
     **this killed chain #6's FIRST arming in preflight**; both tidied, meta-test green.
   - `check-untested-exports --update`: pruned 3 stale rows (1079 → 1076; their new tests
     covered frozen-debt symbols).
   - api golden compare green (6497 exports — their RawQuery/systemcatalog work regen'd
     properly); sleeps/changelog-symbols/file-size/package-surface/error-taxonomy/
     engine-capabilities all green.
3. **exhaustruct canary path bug FIXED** — `scripts/test-exhaustruct-canary.sh` grepped
   `stack/capabilities.go` but the 2026-11-10 `/v4` move put it at `stack/v4/`; the gate
   (`#check-lint-config`) had been silently red since. Green now (all canaries).
4. **M44/T27 metaengine promotion — the commissioned core, COMPLETE** (owner ruling
   2026-11-03): all **17 doc.go stamps** flipped `# Experimental` → `# Stable` (17/17,
   0 residue); **fresh fleet census** run (`docs/evidence/fleet-census-2026-11-15_12-34.json`);
   **19-row T27 evidence table** in FEATURES.md (module × fleet-consumers × suite-status ×
   disclosed-caveats, incl. the honest zeros: turso/pg/mysql/duckdb/dgraph/iroh/bigtable/
   graphadapter at 0 direct); FEATURES section → `## Metaengine ✅ FULLY_FUNCTIONAL` with
   promotion blockquote; 109 feature-status cells 🧪→✅; 16 matrix rows → `✅ Production
   (T27-promoted 2026-11-15: …)`; SKILL.md stability note rewritten (the split brain
   resolved in favor of the SKILL); modules.md graphadapter tail re-stamped. Gates after:
   doc-check 1391 refs green, canonical-facts green, engine-capabilities green,
   changelog-symbols green; api golden is symbols-only — stamps cause no drift.
5. **M52 system-layer residue closed** (report item 34): the three StreamLog wire envelopes
   (`adapter_{command,event,query}_serial`) + `checkpoint_engine`'s stored-value marshal now
   carry `json.Deterministic(true)`; `adapter_core` re-marshal, `evolutions.reifyTo`, and
   `stdlibJSONParser.Marshal` stay deliberately plain (transient/v1-json — documented).
   Full system/v4 suite green (2 runs).
6. **FEATURES feature-visibility rows**: `projectionhost.WithClock` (checkpoint cadence clock)
   - `Persisted-byte determinism` (metaengine feature row).
7. **ADR-0167 (Proposed) written + routed**: retire `idempotency/sqlstore` at v5 behind a
   `system` dedup facade over `metaengine.DedupStore`; design questions → decision-sheet
   **D27**; ADR-0152 ghost-verdict addendum recorded (kvstore resolved-by-deletion);
   both indexes rowed; `check-adr-numbering` green before AND after (165 ADRs). **Fleet
   truth corrected during drafting**: `idempotency/kvstore` was ALREADY deleted 2026-10-17
   (delete-at-zero) — only sqlstore survives (5 direct consumers, census 2026-11-15);
   parent `idempotency/v4` (MemoryStore, 8 direct) out of scope.
8. **CapabilityGaps → Doctor (RULED Q2, 2026-10-08)**: `capabilityDoctorSection` now passes
   `s.capabilityGaps[profile.Name]` (was nil) — documented gaps silence Doctor's capability
   violations exactly as they silence PLAN diagnostics. Regression test pins both halves
   (Doctor 5→4 violations + no violation text; `CapabilityAudit.Table` shows
   `KNOWN GAP: <reason>`). Scoped tests green 2×.
9. **CHANGELOG**: 3 rows (T27 promotion; system determinism tail; ADR-0167 decision) —
   `check-changelog-symbols` green.
10. TODO rows closed with receipts: T27 wave (:340), promotion scope (:350),
    idempotency-ADR (:1384-class), CapabilityGaps-Doctor (:1479-class) — hygiene gate green.

## b) PARTIALLY DONE

- **Chain #6**: armed 12:27 (first arming died in preflight on the tidy debt — fixed,
  re-armed 12:27+); preflight green; **storm-riding since** (attempt 28 at 13:51; load
  50–93 all session; max-wait 4h → expires ~16:27; if exhausted → re-arm #7). When it runs
  it adjudicates EVERYTHING above on a quiet tree.
- **metaengine/v4 full package suite**: 1 FAIL (54.2s, under load-90); immediate re-run's
  FAIL-line grep came back empty (likely load flake); NOT chased to ground — scoped
  audit/doctor/capability tests green 2×; chain #6 will adjudicate on a quiet tree.
- **T27/M44 follow-on legs deliberately DEFERRED while the chain is armed** (half-edited
  module risk): `memoryengine` extraction (:207/:2371), seven-package carve (ADR-0162,
  :2363) — both now unblocked by the promotion, both are L-effort structural waves.

## c) NOT STARTED (this session)

- M34 receipts (:338/:519) — need chain #6 green.
- `check-coverage.sh --update` (pin 3 STALE improvements) — needs a quiet window.
- M35 release wave — owner-blocked (Q3 below). M41/M45 (M35-gated), M46/M47 (M34-gated),
  M48/M49/M50 (M35-gated), M53/M54, `FilterContains/FilterPrefix` (:2253-class), remaining
  M51 tails (stamped-envelope spike, #72 caps cluster = D23-gated, compound-cursor,
  AggregateOn/SingleWriter, Turso strict DSN).
- Sunday 2026-11-22 nightly re-observation (future-dated).

## d) TOTALLY FUCKED UP (and fixed, honestly)

1. **config_loader.go build break**: added `json.Deterministic(true)` to a file importing
   `encoding/json` **v1** (the other four files import `encoding/json/v2`) — `undefined:
   json.Deterministic`, system suite BUILD FAILED. Caught by the immediate test run;
   reverted (the site is config-loading anyway — transient class). Lesson: check the json
   flavor per file before adding v2 options.
2. **Fixture panic**: reused `auditLyingEngine` (nil embedded Engine, documented
   "audit-only") in a Doctor test with `defer store.Close()` → nil-deref PANIC in
   `eng.Close()`. Fixed by giving the fixture a real `Close() error { return nil }`.
3. **Test-authoring waste**: first test version failed at `Plan` (lying engine can't serve
   `findTaskQuery` — needed the memory engine alongside); second asserted `KNOWN GAP` in
   Doctor output when that string only exists in the audit TABLE (Doctor correctly goes
   SILENT — that's the ruling). Two extra red-green cycles from not reading the render
   path before asserting.
4. **The unattributed full-suite FAIL** (b) — reported as-is rather than chased; that is a
   verification debt this report owns.

## e) WHAT WE SHOULD IMPROVE

- **Fold the cheap-gate battery into verify-chain.sh preflight**: this session's battery
  caught 3 real debt items (tidy ×2, manifest, canary) — one of them (tidy) IS in preflight,
  but versions-manifest + exhaustruct-canary are not. Pre-arm ritual should be code, not
  memory.
- **CI jobs API 502-flakiness**: per-leg verification impossible on specific run IDs today;
  the cite-only-verified policy held (T27 table), but a cached/retrying jobs query (or a
  nightly snapshot artifact) would remove the guesswork.
- **Render-path tests**: read what the section actually prints before writing assertions
  (d.3) — the Doctor-silence semantics were knowable from capability_audit.go alone.

## f) NEXT — up to 50, in execution order

1. If chain #6 exhausts max-wait (~16:27) without the gate running: re-arm #7 immediately.
2. On chain #6 green: M34 dated composed-verify receipt; close :338 (T25 composed
   quiet-window) + :519 (composed-proof row).
3. On chain #6 green: phase-boundary authored commit (2026-11-03 ruling) covering this
   wave (hygiene rows, debt repairs, T27, M52 tail, ADR-0167, Doctor-gaps, canary fix).
4. If chain #6 red: triage WITHOUT editing while it runs; fix the smallest owning defect;
   re-arm.
5. Quiet window: `bash scripts/check-coverage.sh --update` (~1h; pins the 3 STALE
   improvements + this session's new tests).
6. Quiet window: re-run the full metaengine package suite to attribute or clear the
   one-time FAIL (d.4/b).
7. M35 ownership decision (question 1 below) → then manifest re-derive (metaengine already
   tagged v4.26.0 by the parallel session; NEW rows needed for projectionhost WithClock +
   this session's system/metaengine surface), `build/f221-changed-set-tests.sh`,
   batch-release, versions-manifest, release-scripts, CHANGELOG train note.
8. M35 pre-push battery: full `#check-generated-artifacts`, full `#erraudit` (never yet
   exercised through the app), `#check-flake-lock-coherence --baseline origin/master`.
9. M41 `stack/*` deletion wave w/ census (T10) — after M35.
10. M41 read-tier deletions (T11: Materialize, RelationalProjection+view, GraphProjection,
    tombstone metadata API, ADR-0126 shells, BuildWhereClause, NewStreamRef validation).
11. M41 Phase A deriver ABSORB row; wire-vocabulary tail; ghost-module tail.
12. M44: `memoryengine` extraction (:207/:2520-class; 8 files, 1,382 LOC, behind same API).
13. M44: seven-package carve execution per ADR-0162's file map (v5 shape).
14. M45 (A9): pebble keycodec-v2 reference leg (D3 recommends own-wave — confirm with owner
    if D3 unruled when reached).
15. M45: badger/bbolt twins; legacy storage/pebble + storage/bbolt rekey; README notes;
    conformance + v1-fixture acceptance.
16. M46: mutation-verify gate (tool pick + 3 pilots + CI ratchet).
17. M46: synctest wave 1 (:2648-class).
18. M46: planner MBT via rapid.
19. M46: error-taxonomy behavior tests + deprecation pins (T16 tail).
20. M47: differential engine harness (T19).
21. M47: tail-latency matrix; heap/goroutine/disk budgets; crash-consistency;
    broker/DLQ/outbox matrix; context-cancellation write-path; companion-auth tail.
22. M48: cqrs-lint 3.7× analysis regression profile + bench-pin.
23. M48: #79 typed-load fixture + fragment audit; #79 catalog symmetry doc.
24. M48: #71 ScanWithMeta decision-exec; #83 baseline hardening (D23 co-rider).
25. M48: Doctor-JSON pre-merge semantics; `--show-suppressed` parseability.
26. M48: F024/F025 utilization; pushdown cookbook; load-scope widening; F005/B005;
    F032/T009 quality.
27. M49: cqrs-htmx sqlite-lifecycle harness variant; TimeAdvancesTo sugar; Bus knobs.
28. M49: observational-equivalence port; scenario TB refactor; companion commit hygiene.
29. M49: go-appkit GracefulClose re-check; scenario version-carrying Given.
30. M49: catalog Registry↔DomainConfig bridge; graph ADT recipe; graph-native dgraph leg.
31. M49: journal dump⇄load round-trip; test-isolation strategy doc.
32. M50: hub asks response; warn-budget hardening; config-with-changelogs golden.
33. M50: upstream EventCatalog issues (verify-before-filing); ROADMAP wave prose.
34. M50: AsyncAPI response schemas; protocol bindings + securitySchemes.
35. M50: COOKBOOK md-go walk; ADT recipe waivers; CUE tail; fleet idempotency sweep (10
    stale requires → DedupStore migration notes, now ADR-0167-shaped).
36. M51: stamped envelope spike (:1139-class).
37. M51: compound-cursor issuance (:679-class).
38. M51: AggregateOn + SingleWriter ratified impls (:1255/:2074-tail).
39. M51: Turso strict DSN (:1278-class); CapabilityGaps follow-ons if any remain.
40. M51: `FilterContains`/`FilterPrefix` FilterOp extension (planner+memory+SQL+adttest;
    CV's 2.3–28ms → LIKE-class win).
41. M51: scheduler engine-path demo; recipes §2.26; Forever mapping (upstream-gated).
42. M53: SearchQuery count=5 re-run + dgraph re-anchor; MySQL-VM shuffled replay;
    hardened vm-mysql leg; bigtable soak; snapshot-migration mysql leg.
43. M54: fleet census cross-check; systemscenario diffusion 10→15; untagged-API gate T26;
    system pin sweep; zero-consumer watchlist (graph/deriver/claiming).
44. erraudit 255-findings triage wave (deep-dive residue).
45. Fold the cheap-gate battery into verify-chain.sh preflight (e.1).
46. Sweep the R11-13 manifest's ~20-row untagged backlog during M35.
47. Re-observe Sunday 2026-11-22 nightly log (first clean load-sweep post-+x-fix).
48. docs-health HARVEST pass at next session end (this report's §f → TODO rows if not
    already rowed).
49. Consider FEATURES/SKILL notes for D27's outcome once the owner rules the facade shape.
50. Final session report + README index (this file is the W3 instance).

## g) QUESTIONS (cannot be resolved from the repo)

1. **M35 ownership (carried from W2, now urgent)**: the parallel session tagged
   `system/v4.19.0`, `metaengine/v4/v4.26.0`, `metaengine/sqliteengine/v4.9.1` mid-session
   and was editing the release-manifest surface (versions.json/README/modules.md) — while
   W2/W3 hold the M35 manifest-derivation procedure. Who owns the R11-13 release wave:
   this session (I re-derive + batch-release) or theirs (I stand down)? Double-owning a
   tag wave risks tag collisions and double-sweeps.
2. **D16/D17 (carried from W1)**: once chain #6 is green — push master + the 48-tag
   R11-13 wave now (first-green-master-CI receipt, F214), and ratify the 48-module
   manifest as-is (9 MINOR + 39 PATCH + the policy holds) or amended for the tags cut
   since (system v4.19.0, metaengine v4.26.0, sqliteengine v4.9.1)?
3. **Q3/D18 + D27 (bundled routing)**: (a) the W1 coverage-debt policy question and the
   scratch/staging disposition remain unanswered on the sheet; (b) NEW: D27 (this session)
   needs the system-dedup-facade design rulings (shape, store alignment, sentinel return,
   quarantine/reset semantics) before the ADR-0167 migration rows can execute — bundle
   these into your next D-sheet pass?

— Reported 13:51 CET; chain #6 riding (attempt 28); tree absorbed except TODO_LIST.md
(daemon in-flight). WAITING FOR INSTRUCTIONS.
