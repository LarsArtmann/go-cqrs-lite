# Status Report — Micro-batch M27.16 closure + self-review — 2026-09-11 03:50 CEST

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** §f items 1-10 routed into TODO_LIST: encoded-apply audit + entry-point conformance sweep (🔥), ClaimMetrics docs/pin tail, calibration provenance + quiet-window re-run, FEATURES row (done by this pass), JSON marshal pin. §c2 Dgraph provenance, §b3 quiet-window count=5, and §e improvements (fold-dispatch table test, load gate, Doctor per-entry-point counters) are folded into the routed items. §f11-50 remain ROADMAP-grade brainstorm fuel.
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.

> Point-in-time snapshot. Session scope: the four micro items around archived
> 22-33/04-35 (ClaimMetrics surfacing, SearchQuery calibration fold, Demote
> record-context verification, enginetest fakes contract note) — the M27.16
> line from `docs/planning/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`.
>
> Format note: `.md` per explicit user instruction (status-report skill
> default is styled HTML; the override is intentional, one-off, and NOT
> propagated into the skill).

## Stat cards

| Metric                                    | Value                                    |
| ----------------------------------------- | ---------------------------------------- |
| TODO items closed this session            | 4 / 4 attempted                          |
| Real bugs found by "verify-only" tasks    | 1 (Demote mirror record context)         |
| Same-class gaps discovered in self-review | 1 (encoded-apply path, NOT fixed)        |
| Suites green                              | metaengine, adttest, scheduling/sqlstore |
| Lint (scoped golangci, both modules)      | 0 issues (after 2 fix rounds)            |
| `-race` on touched paths                  | green (run post-hoc during this review)  |
| Lies caught in my own artifacts           | 1 (calibration load wording — fixed)     |

---

## a) FULLY DONE

1. **enginetest fakes contract note** — "Contract for FAKE engines" paragraph
   added to `adttest.RunCapabilityConformance`
   (`metaengine/adttest/conformance.go`): fake engines must satisfy
   `engineServesADTNatively` for every ADT declared with native complexity —
   implement the backend interface or declare it in `DegradedADTs` — with
   the honestMapMixin/nativeMapEngine precedents named. Comment-only; no
   behavior change.
2. **Demote catch-up Record-context completeness** — the "verify" TODO found
   a REAL gap: the re-routed leg (`applyReplay`) honored `EventInput.Record`,
   but the mirror leg (`replayToShadow`, `metaengine/demote.go`) always
   synthesized a Type-only record, so OnRecord folds on the demoted engine
   rebuilt state with empty StreamID/Version while the EventLog held full
   context. Fixed to pass the recorded record through (synthesize only for
   legacy log entries), pinned by
   `TestDemoteEngine_RecordContextReplay`
   (`metaengine/demote_record_context_test.go`, new file — demote_test.go
   sat at the 349/350-line cap). The pin was proven to bite: temporary
   restore of the old behavior → FAIL with
   `mirror catch-up fed partial record context: {TaskID:t1 StreamID: Version:0}`;
   fix restored → green. Race-clean.
3. **ClaimMetrics surfacing** — `ClaimingTimerStore` now maintains claim
   counters itself (atomics: batches incl. empty polls as heartbeat, timers,
   renewed, renew-rejected) on every committed `Due` poll and `RenewLease`,
   and exposes a JSON-ready snapshot via `Metrics()`
   (`scheduling/sqlstore/claim_metrics.go`, `ClaimMetricsSnapshot`).
   `WithClaimMetrics` hooks unchanged for external exporters. Pinned by
   `TestClaimingSQLite_MetricsSnapshot`; nil-safety test untouched and green.
   API golden regenerated (`Metrics` + `ClaimMetricsSnapshot` in
   `docs/api_surface.txt`, `cmd/api-stability` meta-tests green); CHANGELOG
   `[Unreleased]` Added entry passes `check-changelog-symbols` (19 citations
   verified). Scoped golangci-lint 0 issues after fixing tagliatelle
   (camelCase JSON tags) and exhaustruct (explicit zero-value counter init).
4. **SearchQuery calibration fold** — ran
   `BenchmarkCalibration_DgraphSearchQuery` LIVE on ephemeral nixpkgs
   Dgraph (count=3, benchtime=20x; deliberately waited ~90 min for a compile
   storm to drain: load 29.5→4.9 before starting). Results:
   ~838µs / ~3.21ms / ~13.05ms (discard-cold medians) at 100/1K/10K docs;
   marginal slope ~1_093 ns/row at scale — server-side `anyofterms` is the
   CHEAPER read vs client-filtered MapScan (~2.2-2.7µs/row). Constants
   UNCHANGED (`NsPerScan=2_200` mid-band prices ReadFullTextSearch today;
   overstates ~2x at 10K results, understates ~1.5x at 1K — both
   conservative planner directions; dedicated search-cost field deferred).
   Folded into `docs/benchmarks/calibration-2026-08-30.md` as "Dgraph
   SearchQuery baseline (2026-09-11)" + "ADTMap complexity decision
   (2026-09-07)" sections (the O1 one-RPC rationale, the ~10x overstatement
   it fixed, the pin in `TestRealProfile_ReadCostsPinned`, and the explicit
   deferral of Set/Multimap/Log/StreamLog).
5. **Bookkeeping** — all four TODO_LIST entries annotated `[x] … DONE
   2026-09-11` with closure evidence; `nix fmt` clean on all touched files;
   full module suites green (metaengine 23.7s, adttest, scheduling/sqlstore);
   vet green; changelog symbol gate green.

## b) PARTIALLY DONE

1. **SearchQuery → cost model**: the bench is folded as a BASELINE record,
   but nothing consumes it yet — no dedicated search read-cost field, no
   drift-script coverage (live-DSN engines are hand-anchored only), and the
   2_200 constant knowingly mis-prices search in both directions. Documented
   as deferred; the precision decision is open.
2. **Dgraph version provenance**: the doc now says "25.4.0 (version per the
   09-01/09-06 campaign records; binary identity not re-verified this run)"
   — I cited the version from prior doc entries and failed to verify the
   actual binary (`nix eval` returned nothing; store-path glob empty after
   the run). Low risk (flake.lock pins nixpkgs; same derivation era as the
   09-06 runs) but it is an UNVERIFIED claim in a measurement record.
3. **Load-turbulence honesty**: medians are tight (±6%), but a fresh host
   compile storm was ramping during the final 10K count (03:36:44 trace:
   1-min 18.5 > 5-min 13.9). The original wording claimed the storm started
   "minutes AFTER the run" — stronger than provable; corrected this session
   to name the 13_741_630 outlier as the load signature. A quiet-window
   count=5 re-run would still be the rigorous close.
4. **Commit shaping**: the auto-commit daemon absorbed all session work into
   `chore: auto-commit` heuristic blob commits (working tree is clean).
   Nothing is pushed; nothing is semantically shaped — fine for the daemon
   workflow, hostile to review/release-train archaeology.
5. **cmd/cqrs-lint fmt drift**: `nix fmt` reformatted 3 files I never
   touched (`a020_a021_a022_a023.go`, `a030.go`, `d011_test.go`) — another
   session's work left unformatted. I kept the mechanical wraps (CI fmt gate
   requires a clean tree) but did not investigate who/why they were left
   dirty.

## c) NOT STARTED

1. **encoded-apply record-context audit** — self-review found
   `metaengine/encoded.go:49` feeds OnRecord folds
   `record.Record{Type: eventType}` on the encoded-apply path, OUTSIDE
   `applyWithRecord`, so it neither carries a real record nor feeds the
   Doctor synthetic-apply advisory. Same bug class as the Demote gap;
   pre-existing; untouched this session.
2. **FEATURES.md / scheduling/sqlstore README** — the new public
   `Metrics()`/`ClaimMetricsSnapshot` surface is in CHANGELOG + api golden
   but not in the feature inventory or the module README's claiming
   documentation. (Skill references verified clean: neither SKILL.md nor
   `references/*.md` ever mentioned ClaimMetrics, so no reference update was
   owed.)
3. **`-race` full suites** — only touched-path subsets raced this session;
   the repo gate is `#verify`, which was not run (load + exclusivity).
4. **Other verify gates** — `check-duplication`, `check-arch`,
   `check-coverage`, `vulncheck`, `doc-check` all unrun (no new deps, no
   skill-doc changes; low expected risk, unverified).
5. **HARVEST** — this report's section (f) is not yet routed into
   TODO_LIST/ROADMAP per the docs-health loop; only the four DONE items were
   annotated.
6. **Report commit** — the status-report skill says to commit the report;
   the standing rule is no commit without an explicit user instruction, so
   this file lands uncommitted (daemon will absorb it).

## d) TOTALLY FUCKED UP

Nothing shipped this session is broken — both new behaviors are tested,
linted, race-checked, and honestly documented. The brutally honest list of
what still stinks:

1. **The Demote bug existed at all** — store.go's own `EventInput` comment
   claimed "the replay paths (Backfill, Verify, DemoteEngine catch-up) hand
   Record-aware projections the original StreamID/Version" while the mirror
   leg silently dropped it. Doc-comment-as-contract drifted from code, and
   nothing caught it until a TODO said "go verify". The same pattern
   (per-entry-point record-context guarantees) has NO systematic test today —
   each path is pinned by an ad-hoc spot test, and encoded.go proves the
   sweep is incomplete.
2. **Calibration provenance is still soft** — a measurement record in this
   repo now contains an unverified engine-version citation and a
   load-ramping window, because the protocol has no mechanical provenance
   step (no "record the nix store path + binary version + load trace" gate)
   and no script-side load guard. On a 24-28-user shared host, every future
   live-DSN calibration inherits this weakness.
3. **Session history is unreviewable** — everything landed as daemon blob
   commits. If anyone ever asks "what exactly changed for M27.16?", git log
   answers with `chore: auto-commit N changed file(s)`.

## e) WHAT WE SHOULD IMPROVE

Answers to the self-review questions, straight:

- **What did I forget?** (a) Verify the Dgraph binary version at bench time —
  a 2-minute check skipped, then cited secondhand. (b) FEATURES.md/README
  rows for the new public surface. (c) The `applyFold` call-site sweep — I
  greped 5 call sites when fixing Demote, fixed 1, and did not audit the
  encoded.go one until this review.
- **What is stupid that we do anyway?** (a) "Discard-cold median" of count=3
  = median of TWO values — statistically thin for a baseline doc. (b)
  Calibration protocol relies on prose ("don't calibrate during storms") on
  a 28-user host instead of a script-side load gate. (c) The auto-commit
  daemon destroys commit semantics. (d) `ClaimMetrics` (hooks) vs
  `ClaimMetricsSnapshot` (counters) — justified split, but the names are one
  brain-fart away from confusion in review threads.
- **Could I have done it better?** Yes: bench FIRST in a verified-quiet
  window with count=5, THEN write all docs in one pass (I wrote the doc mid
  afternoon-load and had to self-correct the load claim). Also: run the
  same-class sweep immediately after finding the Demote gap, not in a later
  self-review.
- **What can we still improve?** (1) A single conformance-style test that
  walks EVERY fold-dispatch entry point (Apply, ApplyBatch, ApplyRecord,
  Backfill/replayShadows, Verify, DemoteEngine×2, encoded-apply, live
  replicator) and asserts the record-context contract + advisory counting in
  one table — kill the spot-test whack-a-mole. (2) Script-side load gate +
  provenance line (store path, `dgraph version`, uptime samples) as
  calibration protocol items. (3) Doctor section listing synthetic-record
  feeds PER entry point, so the encoded path becomes visible the day it is
  audited.
- **Did I lie to you?** One soft claim, now corrected: the calibration doc's
  "storm started minutes AFTER the run". The trace supports "was ramping
  around the final count". Everything else in the session claims maps to a
  command output shown above (bench lines, test FAIL/PASS, lint 0 issues,
  gate output).
- **Ghost systems?** `Metrics()` is deliberately consumer-facing with zero
  in-repo callers — that is the expected library state (the TODO itself
  complained the hooks had "zero consumers"; the built-in counters ARE the
  surfacing). Not a ghost: tested, golden'd, changelogged.
- **Scope creep?** Resisted twice: no dedicated search cost field (would
  have meant ADR + planner + pins + drift script), no Metrics() wiring into
  metaengine Doctor (wrong dependency direction — scheduling must not be
  imported by metaengine; the "Doctor or status endpoint" phrasing in the
  TODO was structurally ambiguous and I chose the store-side surface).

## f) Up to 50 things we should get done next

Brainstorm, not commitment — HARVEST fuel (docs-health applies routing
rigor; most of 30+ are ROADMAP-grade).

Session follow-ups (highest confidence, smallest effort):

1. Audit + decide the encoded-apply path (`metaengine/encoded.go:49`):
   carry real record context if the API can, or document + advisory-count
   the synthetic feed.
2. Build the entry-point record-context conformance sweep (one test, table
   of all dispatch paths: record source + advisory counted?).
3. Add `Metrics()` to `scheduling/sqlstore/README.md` claiming docs.
4. Add `ClaimMetricsSnapshot`/`Metrics()` row to FEATURES.md.
5. JSON marshal pin test for `ClaimMetricsSnapshot` (tag stability is now
   public API).
6. HARVEST this report's (f) into TODO_LIST/ROADMAP.
7. Re-anchor the calibration doc's SearchQuery provenance: record the actual
   nix store path + `dgraph version` output.
8. Quiet-window count=5 SearchQuery re-run; supersede today's table if
   medians move >5%.
9. Consider a MySQL/Postgres live-window `Metrics()` integration test
   (snapshot currently pinned on SQLite only).
10. Annotate the pareto plan doc: M27.16 (Demote/ClaimMetrics/SearchQuery/
    enginetest) fully closed 2026-09-11.

Calibration / metaengine program:

11. Script-side load gate for calibration runners (assert load < N before
    benching; abort loudly otherwise).
12. Protocol item 6: provenance line format (store path, binary version,
    uptime samples) for every baseline entry.
13. Dedicated search read-cost decision: field vs reuse of NsPerScan (needs
    a product call — see question g1).
14. ADTMap O1 wave: per-ADT benches for Set/Multimap/Log/StreamLog one-RPC
    reassessment (unblocks the BLOCKED TODO_LIST Q1).
15. Re-anchor ALL dgraph constants in one quiet window (current doc mixes
    09-01, 09-06, and 09-11 runs).
16. Bump remote-engine calibration campaigns to count=5 (median-of-2 today).
17. Doctor: per-entry-point synthetic-record feed counters.
18. Full `-race` pass over metaengine + scheduling/sqlstore (subsets only
    raced so far).
19. Commit raw bench outputs (docs/benchmarks/raw/) alongside the baseline
    doc so medians are reproducible.
20. `ClaimMetricsSnapshot`: consider a process-start timestamp so consumers
    can compute rates across restarts (counters are process-local).

Testing hardening:

21. Race-stress test: concurrent `Due` pollers vs `Metrics()` reader.
22. Test that `MarkFired`/`Schedule` deliberately do NOT touch claim
    counters (pin the counter scope).
23. Metaengine: pin that `replayShadows` (Backfill) synthesizes ONLY for
    legacy entries (today only Demote's leg is pinned end-to-end).
24. Property test: counters never exceed committed polls under races.
25. Fuzz/decode: `decodeDueTimer` corrupt-payload path interacting with
    counters (claimed-batches counts, timers don't).

Scheduling/sqlstore roadmap (pre-existing, adjacent):

26. RenewLease ownership/claim tokens (code comment defers it; current
    semantics extend whichever live claim exists).
27. Wire a worked `Metrics()` → `/status` example (example/ module or README
    snippet) so the surface has one visible consumer pattern.
28. otel wiring example for the hooks (the doc comment sketches
    otel.Int64Counter; make it runnable somewhere).
29. Scheduler + claiming-store + Metrics end-to-end example test.
30. Document counter reset-on-restart semantics in README (operators will
    ask why numbers dropped).

Repo hygiene:

31. Investigate who left the 3 cmd/cqrs-lint files unformatted; if a session
    is still active, coordinate.
32. Auto-commit daemon: define a pre-tag "shape the history" procedure (blob
    commits → semantic commits) for the release train.
33. Full `nix run .#verify` on a quiet window to stamp the batch end-to-end.
34. `check-duplication` + `check-arch` + `check-coverage` run for the batch.
35. Decide `ClaimMetricsSnapshot` naming confidence (vs `ClaimMetrics`) —
    cheap now, breaking later.

Carried-forward context glimpsed this session (not re-researched; owners
know the details):

36. TODO_LIST Q1 (BLOCKED): dgraph one-RPC scope authorization — needs you.
37. TODO_LIST Q2 (BLOCKED): CapabilityGaps reach into Doctor — needs you.
38. Turso sync/embedded-replica BYOK decision (BLOCKED on user need).
39. Upstream turso-go issues: 3 defects, verify-before-filing gate.
40. Turso IVM defect A/C: re-check on next upstream release tag.
41. GitHub Actions billing fix (paid CI jobs failing since ~07-17; user
    action).
42. v5 train: encryption-at-rest ADR (planning doc lists it).
43. v5 train: deletion waves per ADR-0114 (tombstone/removal work).
44. v5 train: stack preset removals (ADR-0123).
45. sqliteengine `EngineResetter` implementation (ADR-0136 follow-up listed
    in AGENTS; persistent engines still warn-only).
46. Demote fold-write failover to quarantined engines (ADR-0137 follow-up
    named in AGENTS internal contracts).
47. Revisit `metadata.TombstoneMark` → pure domain-event branching in
    `OnUpdate` (AGENTS contract 11, partial implementation).
48. calibration-drift redesign (named in the pareto plan W3; the drift
    script still hand-anchors live-DSN engines).
49. Doc-pass: whether scheduling claiming deserves a recipes.md section
    (references currently silent on ClaimMetrics entirely).
50. Close the loop on this report: after HARVEST, archive per docs-health
    conventions so the next status pass can ANNOTATE rather than re-audit.

## g) Questions I cannot figure out myself

1. **Search pricing precision** — ship a dedicated per-row search read-cost
   field (ADR + ReadCosts change + planner routing + pins + drift entry)
   now, or keep `NsPerScan=2_200` as the knowingly-mid-band proxy until a
   real consumer needs the precision? I lean "wait", but it is a
   product-precision tradeoff, not an engineering one.
2. **dgraph one-RPC wave (TODO_LIST Q1)** — authorize flipping
   Set/Multimap/Log/StreamLog to O1 in one wave (with per-ADT benches), or
   keep the incremental per-ADT path? This gates whether I build 4 bench
   suites or 1.
3. **Calibration load policy on this shared host** — is load ≤5 an
   acceptable ceiling for live-DSN baseline entries (today's SearchQuery
   table stands), or should the protocol require ≤3 and re-run today's
   numbers in a stricter window?

---

_Generated 2026-09-11 03:50 CEST. Waiting for instructions._
