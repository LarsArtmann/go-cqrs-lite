# Status Report: P0 Execution Session — Red Gates Closed, Wave 1 Shipped

> **Written:** 2026-10-25 03:14 CET
> **Session:** 2026-10-24 ~19:30 → 21:30 CET (execution window; report written later)
> **Plan executed:** [`docs/planning/2026-10-24_19-19_SUPERB-repo-scale-pareto-red-gates-to-v5-wave.md`](../planning/2026-10-24_19-19_SUPERB-repo-scale-pareto-red-gates-to-v5-wave.md) (P0 = T01–T05, plus opportunistic T06/T07/T08/T21.3–4)
> **Verdict:** **HEAD IS GREEN.** Composed `#verify-fast` EXIT=0 (293 test packages, 88/88 modules lint-clean, duplication gate 0 new, doc-check 1356 refs). The two known-red gates that poisoned every wave are closed at root cause.

---

## a) FULLY DONE (verified, receipts cut)

| # | What | Evidence / Receipt |
|---|------|--------------------|
| T01 | **SQL planned-path suite RED fixed at root cause.** The 2026-10-23 all-TEXT layout switch (d24cbdc35) removed name-heuristic column types but left 7 tests living in the typed world, AND `PlannedColumnTypeCompatible` did not know `BIGINT`/`DOUBLE`/`FLOAT` (they fell into the permissive default, so the build-time mistype pin could never fire). Fixed: compat table treats INTEGER/BIGINT/REAL/DOUBLE/FLOAT as numeric accepting all Go int/uint/float widths (new pins in `errors_planned_test.go`); the red tests converted to `BuildLayoutPlanFromType` typed plans. **Scope correction: FIVE pg tests were red, not one** (RoundTrip, MisTypedExtract, PushdownScan_Keyset, MapUpdate, Explain_IndexUsageProofs) — the 10-23 TODO row undercounted. | pgengine FULL integration suite green ×2 vs ephemeral PG (`PG_MODULES=metaengine/pgengine nix run .#integration-pg`); mysql planned 10/10 PASS verbose, no skips, vs live userspace MariaDB on :33061; mysqlengine full module green. CHANGELOG Changed entry; TODO row closed with receipt. |
| T02/T03 | **check-duplication gate GREEN (0 new groups, baseline 189).** The plan's "~180 twins missing directives" was stale — the parallel session had already annotated them; only 7 residue groups existed. All resolved with `// art-dupl:accept` directives (cross-module SQL engine twins, command/event adapter prologue, mirror Escape/Unescape, trivial test sugar). NO baseline insert needed. Plus one self-inflicted group later (see e) fixed the same way. | `nix run .#check-duplication` → "No new clones detected" twice (before and after Wave 1). |
| T04 | **12-09 plan Wave 1 (value-integrity) executed.** `keycodec.EncodeJSONErr` (Rejection `keycodec.encode_value`) added; lenient `EncodeJSON` doc-split into key-path vs value-path guidance (no deprecation, per spec); ALL 21 LSM production write sites converted (pebble ×7, bbolt ×7, badger ×7 — map/multimap/log/stream/vector-metadata writes + MapUpdate); the `encodeJSON = keycodec.EncodeJSON` aliases deleted from all three engine var blocks; `adttest.AssertValueEncodeRejection` harness (family + canonical code, rerun-safe names) wired into all three engine suites; sqliteengine deliberately NOT wired (keeps its stronger dedicated specs with its own code — divergence noted as v5-envelope territory). Empirically verified `encoding/json/v2` returns ERRORS on cyclic structures (not a stack-overflow class) — the cyclic case is genuinely covered. | All four module suites green (metaengine 39s, pebble, bbolt `SOAK_SKIP_BOLT=1` 6.3s, badger); keycodec ×5; taxonomy gate green after adding `metaengine/keycodec` to GATED_MODULES (row `keycodec.encode_value` at error-taxonomy.md:768); api golden 6305 (+2 exports); 12-09 §11 log amended (the plan's own execution log, per the verschlimmbessern guard). |
| T05 | **Composed proof: `nix run .#verify-fast` EXIT=0** — 293 test packages ok, 88/88 modules lint-clean, 0 new clones, WAL raw-byte pins ×2 green, doc-check 1356 references valid. Took three runs (see d/e for the two chase-downs). | `/tmp/verify-fast4.log`: "✅ All fast verification checks passed". |
| T06 | **Fleet sqlstore scan + census refresh.** Exactly ONE direct fleet consumer of `scheduling/sqlstore`: **ChastityAPI** `cmd/server/main.go` (`NewSQLiteStore[UnlockTimerFired]`). **G2 ANSWERED BY EVIDENCE: delete-at-zero does NOT fire** — the module dies at v5.0.0 per its V007 sentence; ChastityAPI migrates via `scheduling/engine.NewTimerStore` in the v5 wave. This closes the "kill-now deprecation was issued WITHOUT the fleet census" honesty gap from the previous session. | `docs/evidence/fleet-census-2026-10-24_20-47.json` (53 distinct submodules required by the fleet); receipt folded into the TODO sqlstore row + AGENTS contract-30 ledger. |
| T07 | **kv package-doc contract-30 note** (the doc-note path, safe regardless of G1): durable scope = typed layer over engines + caches + MemStore; legacy adapters are the only durable implementors and are v5-removed; "do not add new satellite-SQL kv backends"; G1 verdict stays owner-gated. AGENTS contract-30 ledger ticked. | kv/doc.go; kv module build + tests green. |
| T08 | **E023 self-lint + triage + pin.** First pass: 7 in-repo findings → `benchkit/`, `cmd/`, `idempotency/sqlstore/` added to the exemption table (tooling harness + deprecated-SQL classes; exemption-table test extended incl. negative pins for middleware/projectionhost); `middleware/deadletter_sql.go` + `projectionhost/sqlite_dlq*.go` deliberately stay flagged — 3 true-positive advisories = live contract-30 debt the rule exists to keep visible. DiscordSync consumer run fires on its real satellite packages (internal/db ×N, metrics_db) — the field signal works. New pin: `TestShouldExitWithError_InfoAdvisoryNeverGates` (info/warning never gate exit; error does). | cqrs-lint full suite green (19 pkgs ok); rescan shows only the 3 intended findings. |
| T21.3/T21.4 | **AGENTS procedure checklists added**: "Deprecate a Module" (markers→V007-table-same-edit→docs trail→golden→SA1019 scoping→census-gated deletion) and "Add a Lint Rule" (detector→register→catalog→RULES.md→count pins→golden→suite) — the D1/D2 lessons made permanent. | AGENTS.md Procedures section; doc-check green. |
| Side-find | **QEMU hostfwd port bug root-caused and fixed.** `vm-mysql.sh`'s default port 33070 sits INSIDE `ip_local_port_range` (32768–60999); the node-exporter scrape pair on :9100 holds a long-lived connection with SOURCE port 33070, so QEMU's hostfwd bind failed at every boot → instant QMP `ConnectionResetError` → misleading "Driver exited unexpectedly", no QEMU error text anywhere. Diagnosis trick: same driver boots fine on a different port — swap ONE variable. Default moved to 23306 (below the range). | scripts/vm-mysql.sh + gotchas-testing entry (diagnosis recipe: `ss -tn | grep :<port>` — the squatter is ESTABLISHED, not LISTENING). |

## b) PARTIALLY DONE / with caveats

1. **The 12-09 plan's Wave 2–4 remain** (by design — this session's mandate was Wave 1 = its T5–T7; T1–T3 were pre-shipped). Wave 2 "prove the claims" is now LARGELY satisfied by my composed runs, but the plan's own wave structure (replay pins per engine, envelope design T19) is still open there.
2. **Authored-commit discipline: PARTIAL.** One authored commit landed (`7396d262d`); the auto-commit daemon absorbed the rest of the phase edits into `chore:` commits despite phase-boundary attempts (one commit literally lost the race: `fatal: cannot lock ref 'HEAD'`). Authorship is preserved in the plan logs' execution entries instead. Not a work defect, but the session's commits are mostly `chore:`-shaped.
3. **MySQL verification vehicle:** verified via the userspace MariaDB on :33061 (documented fast vehicle), NOT via the CI-parity QEMU leg — the QEMU run was killed once the userspace run proved the suite. The port fix itself was proven by a successful boot+ready on 23306, but no full VM suite ran to completion.

## c) NOT STARTED (plan items still pending — none were forgotten, all remain rowed)

- T09–T11 (core/v5 audit + cqrs-htmx migration legs 1/2) — the v5 wave-1 migration machine.
- T12/T13 (go-appkit migration, cqrs-upgrade v5 codemod).
- T14–T16 (V007 deletion waves, zero-consumer sweep).
- T17 (A9 keycodec-v2 LSM dual-read windows — ⛔ G3+G5).
- T18 (branded-`.String()` lint rule), T19 (cqrs-lint extraction — ⛔ G4), T20 (engine observability demo).
- T21.1/T21.2 (harvest both 10-24 status reports; module-count split-brain fix) — only .3/.4 shipped this session.
- T22–T27 (P3 tail).
- **G1 kv verdict** — still the open owner question (doc-note bought time, decision not made).

## d) TOTALLY FUCKED UP (caught and fixed in-session — full honesty)

1. **Directive inside a raw-string literal.** My first art-dupl annotation on the two `typed_wal.go` twins used the TRAILING-comment form on a line ending with a backtick (`const stmt = ` // art-dupl:accept …`) — the text landed INSIDE the SQL string, silently corrupting both engines' backfill statements. Caught immediately by review-before-commit, fixed to a comment-above line. The trailing form is only safe on lines that don't open a raw string.
2. **Flaky pin written against an unspecified guarantee.** My first `EncodeJSONErr` test pinned map-literal output order (`{"n":3,"status":"open"}`); json/v2 does NOT sort map keys (an EXISTING gotcha I had not cross-checked) — it failed on first run and, worse, my "fix" pinned insertion order, which is ALSO not a guarantee. Corrected to a struct-literal pin (declaration order is deterministic). Lesson applied: before writing a byte-pin, grep the gotchas file for the involved guarantee.
3. **Lint findings shipped in my own first composed run.** verify-fast run #2 failed on 5 findings, three of them mine-by-lineage: E023's two lookup tables (gochecknoglobals — the read-only-table nolint precedent existed, I skipped checking lint for the rule I inherited), a revive unused-param in my new harness, and a prealloc/gofumpt pair in my new test. The exhaustruct pair was on foreign-committed demo literals (fixed with scoped nolints + rationale since the fields are omitempty-optional). Also: my first nolint said `//nolint:exhaustruct` but the ENABLED linter is `exhaustruct_v5` — the suppression silently didn't apply until corrected.
4. **Self-inflicted clone group.** The MapUpdate conversions minted ONE new duplication group (badger↔pebble prologue twins) that only surfaced in the composed run. Fixed with directives; gate back to 0. (Working as designed — but it means my T02/T03 "done" claim needed a second pass after T04.)
5. **Red herring chase: "deadlock" that was a soak test.** bbolt suite FAIL at 600s looked like a deadlock I introduced; it was the documented 509–1145s soak that naive sweeps hit without `SOAK_SKIP_BOLT=1` (the gotcha existed; I hadn't read the testing gotchas file before running the suite).
6. **Three failed QEMU runs before the one-variable experiment.** I burned ~15 minutes rerunning the VM "hoping it was transient" before designing the isolate-one-variable test (port) that found the squatter in minutes. The manual-boot-works signal was there after run 2.

## e) WHAT WE SHOULD IMPROVE (process, durable)

1. **Read the relevant gotchas file BEFORE first contact with a subsystem** (soak skips, json/v2 order, port classes). This session paid three avoidable taxes (d4, d5, and part of d2) for skipping that step. The gotchas exist precisely for this.
2. **Never write byte-level pins without checking the encoding contract first** — map-key order, field order, codec version. Struct literals or decode-and-DeepEqual.
3. **art-dupl directive placement rule needs a gotcha row**: trailing comments are UNSAFE on lines that OPEN a raw string/struct/composite literal region where the directive text would change semantics; also nolint directives must name the ENABLED linter id (`exhaustruct_v5`, not `exhaustruct`). Both bit me in one evening.
4. **Per-change lint before the composed gate**: run golangci on touched modules immediately after edits (the `#lint` app made this cheap — I deferred and paid a composed-run cycle).
5. **The one-variable experiment is the first tool, not the last** — QEMU diagnosis went 3 failed reruns → 1 controlled comparison → root cause. Encode that instinct: when a repro is flaky-by-environment, change exactly one thing.
6. **`vm-mysql.sh`-class scripts should self-verify their hostfwd port is outside `ip_local_port_range` at start** (a 3-line guard would have saved the whole class); same check could be added to the other vm-*.sh scripts.
7. **Session commit authorship vs the daemon**: phase-boundary commits raced the daemon twice. Better: make the LAST edit of each phase the plan-log entry (which I did) and accept `chore:` absorption for code, or commit with `--no-verify` off-hours. Worth a policy decision.
8. **verify-fast under parallel load can flake systemscenario SSE** (1 observed failure, ×3 standalone green, clean composed run green). If it recurs, route that suite through the quiet-window machinery or bump its await budget — do NOT chase it as a regression without a standalone repro.

## f) NEXT UP TO 50 (from the plan's remaining rows + this session's residues, roughly Pareto-ordered)

**P0 residue / immediately unblocked:**
1. T09: core/v5 completion audit vs ADR-0152 package list (gap table with file:line evidence).
2. T10: cqrs-htmx leg 1 — import-surface switch of the biggest trains to core/v5 (73% of fleet edges).
3. T11: cqrs-htmx leg 2 — remaining trains + adminui/dashboardui/datastar + e2e green.
4. T12: go-appkit onto core/v5 (8 prod files).
5. T13: cqrs-upgrade v5 codemod target + fixture test.
6. T14: V007 deletion wave 1 (stack presets, storage/view, storage/relational; registry cleanup in all 8 gates).
7. T15: sqlstore deletion prep — with G2 now answered, wire the ChastityAPI migration into the v5 wave plan (its `NewSQLiteStore[UnlockTimerFired]` → `scheduling/engine.NewTimerStore`).
8. T16: census-driven kill/integrate sweep (`census-fleet.sh --zero-consumers`, fork+delete playbook for the condemned).
9. G1: kv verdict (owner) — then either the engine-Map adapter or cache-only demotion.
10. T21.1: harvest BOTH 10-24 status reports (+ this one) into TODO_LIST (docs-health HARVEST mode).
11. T21.2: module-count split-brain fix (101 vs 98 vs 96/97 — one gate-derived source).

**Value-integrity continuation (12-09 plan):**
12. Its Wave 2 residue: per-engine replay-pin receipts in its §11 log (my composed runs cover most; formalize).
13. Its Wave 3: envelope design spike → decision (stamped engine-value envelope, ADR-0066 reversal).
14. Its Wave 4: ⛔ owner-gated envelope implementation wave.
15. The graph encoders leg (TODO row now says 3/4 fixed): `mysqlengine/duckdbengine/pgengine/graph.go` + bigtableengine — decide the "keys stringified, values JSON" rule and migrate together.
16. Turso delegation rejection test (12-09 T3 — never confirmed done).

**Metaengine / engine quality:**
17. T17: A9 keycodec-v2 LSM dual-read windows (⛔ G3 + G5) — pebble reference first.
18. T24: compound-cursor issuance protocol (engines fill NextCursor).
19. T25: calibration campaign quiet-window legs + doctor golden regen.
20. Vector-metadata rejection parity check: does duckdbengine's vector metadata path also need EncodeJSONErr? (I did pebble/bbolt/badger; duck's path was not in the 21 — verify it has no silent-stringify equivalent.)
21. sqlengine (`storage/sql`, `stack/*` presets) value paths — same silent-stringify audit class as T04 (pre-v5 deletion they may not matter; confirm and note).
22. E023 scorecard wiring decision (⛔ G6-adjacent; TODO follow-up).
23. T18: branded-`.String()` warn-first lint rule (the 7+-fix bug class).
24. T20: engine-path observability demo replacing scheduler-otel-status.
25. `vm-pg.sh`/other vm-*.sh: add the ephemeral-range port guard (improvement e6).

**Hygiene / docs truth:**
26. CHANGELOG: cut the `[Unreleased]` block into the next tag wave when the parallel session is ready (my 3 entries ride along).
27. recipes.md §2.26 deprecation note (sqlstore recipe — TODO row exists).
28. ROADMAP wave prose + FEATURES rows for Wave-1 value-integrity (FEATURES row not updated for EncodeJSONErr — minor gap, see f-47).
29. module-map row check: keycodec gained a production export — confirm map prose still accurate.
30. Dead-letter noise: the systemscenario SSE flake investigation note (from this session) into gotchas-testing if it recurs.
31. Companion repos: schedule ChastityAPI's sqlstore migration ticket so the v5 deletion isn't blocked later (fleet coordination).
32. cqrs-lint: consider a self-lint CI leg (this repo) with the 3 true-positives baseline-pinned so NEW satellite SQL in-repo gets caught.
33. art-dupl: add the raw-string directive-placement trap + nolint-name-must-match-enabled-linter to gotchas-tooling-build.
34. AGENTS: the "Deprecate a Module" checklist should link the new census-evidence step I added to the sqlstore row (cross-link TODO_LIST).
35. T27 ritual: verify-chain composed proof after EVERY P1/P2 task (per-wave, not per-session).

**Lower tail (P3, from the plan):**
36. T22: `TimeAdvancesTo(t)` sugar; scenario `testing.TB` refactor; observational-equivalence port.
37. T23: EventCatalog 5.2 custom exports + mesh-demo flows pilot + warn-budget.
38. T26: bank-sync M13, DiscordSync workaround-branch drop, companion commit-history hygiene, ROADMAP prose, recipes §2.26 note.
39. Turso-go #9391 follow-up; EventCatalog licence shift (upstream asks).
40. goleak/coverage re-baselines (coverage gate will drift after the new tests — run `#check-coverage` at next wave closeout).
41. `#test-all-backends` full sweep once at the next quiet window (my changes were verified per-engine + verify-fast, but the full matrix incl. Dgraph/Redis/turso legs hasn't run since Wave 1).
42. Bench gate: `benchmark-regression.sh` after the 21-site encoder swap (MapSet now allocates an error check — regression check is cheap insurance).
43. Load sweep (`#load-sweep`) — timing paths touched (engine write paths) per AGENTS quick reference.
44. G5: deployed-DB prefixed-key census (sticky-key fallback vs re-key) — feeds A9.
45. G7: metaengine-vs-ADR-0152 ruling (dedicated project vs inside v5 core).
46. cqrs-lint extraction dry-run prep (⛔ G4 first): subtree measure + module_catalog drift-test replacement design.
47. FEATURES.md: add the value-integrity mechanism row (EncodeJSONErr + harness) — inventory gap from this session.
48. The 12-09 plan §1.2 "COOKBOOK stored values" — confirm the LSM half is now documented (the doc's SQL half predates; LSM rejection deserves a sentence).
49. ChastityAPI: quick E023 run + scorecard there (first consumer-field data point beyond DiscordSync).
50. G2 follow-through: record the ChastityAPI-finding in the ADR-0142 amendment's verdict table (the TODO + AGENTS ledger have it; the ADR table itself says "under evaluation"-era wording for kv only — sqlstore row should cite the census).

## g) QUESTIONS FOR THE OWNER (cannot be figured out from the repo)

1. **G1 — kv's durable future:** engine-Map adapter (`kv.Store` over `metaengine.MapBackend`, one durable runtime) vs cache-only demotion (durable typed KV = metaengine collections directly)? The doc-note ships regardless; the decision gates whether I build the adapter or write the demotion ADR. (DiscordSync/cqrs-htmx/Code-To-CV-Agent prod usage of TypedStore/ViewQuery makes (a) the lower-friction path in my read.)
2. **Release timing for the green HEAD:** the [Unreleased] block now carries the planned-path fix + the LSM silent-stringify behavior change (writes that used to "succeed" with garbage now Rejection-fail). Do you want a tag wave cut NOW off the green gates, or held until cqrs-htmx leg 1 lands so the fleet migrates once? (Behavior-change semantics say: cut soon, before more consumers absorb the lenient behavior.)
3. **verify-fast exclusivity vs the parallel session:** I ran composed gates while another session was active (one likely-load flake observed). Is #verify/#verify-fast now "mine to run freely," or should I serialize through the verify-chain quiet-window ritual whenever `git log` shows foreign commits younger than ~1h?

---

### Session integrity notes

- Foreign work respected throughout: `example/mesh-demo/merge_test.go` and in-flight catalog edits were never touched until daemon-committed; the exhaustruct fix on `catalog/demo.go` was applied only after verifying it was committed (17:47) not dirty.
- The stale gopls `typed_wal.go:197` error that persisted all session turned out to be a REAL compile break fixed by the parallel session at 18:50 (per its CHANGELOG entry) — my diagnostics view was stale, not the tree.
- The throwaway pre-session worktree (`/tmp/cqrs-pre`) used to prove the SSE flake was foreign-or-environmental was removed; `git worktree list` shows only pre-existing entries.
