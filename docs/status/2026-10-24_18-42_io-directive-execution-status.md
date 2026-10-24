# Status Report: All-I/O-Through-Metaengine Directive — Execution Session

> **Date:** 2026-10-24 18:42 CEST (Saturday)
> **Session scope:** owner directive "ALL I/O goes solely through metaengine, with the A/B gate" — encoding, evaluation, and execution of the four verdicts (timers kill-now, queue A-verdict, kv open, lint rule approved).
> **Preceded by:** `docs/status/2026-10-24_16-56_extraction-candidates-session-status.md` (extraction analysis — separate question, same day)
> **Branch:** `master`, tree CLEAN (auto-commit daemon absorbed all session work into `chore:` commits, `89afcfcbd` → `7ff0fe120`)
> **Code changed by this session:** YES (deprecation markers + new lint rule + gate wiring)
> **Format override:** Markdown per explicit user instruction (HTML is the skill default); flagged, not propagated.

---

## Executive summary

The directive is now **law in the system** (AGENTS.md contract 30 + SKILL.md I/O rule + ADR-0142 amendment with the A/B gate and the verdict ledger). All four owner verdicts were executed same-session: `scheduling/sqlstore` is deprecated with 21 per-symbol notices and a full docs trail; the queue A-verdict plus revisit trigger is recorded; the cqrs-lint **E023 `satellite-owned-sql`** rule is implemented, wired into every gate, and the full 215-rule suite is green; the kv origin research was delivered and the decision is parked with the owner. One honest scar: I wired the deprecation markers **after** the V007 drift gate caught them missing — the checklist existed and I didn't use it first.

Confidence: **high on what shipped** (every gate that failed during the session is now green). **Medium on blind spots**: no fleet scan for `scheduling/sqlstore` consumers preceded the kill-now deprecation, and E023 has never been run against this repo itself.

---

## a) FULLY DONE

| # | Deliverable | Evidence |
|---|---|---|
| A1 | **Directive encoded as contract 30** in AGENTS.md: the rule (metaengine = the ONLY storage layer), the A/B gate ((A) good-reason check → prefer no feature over substrate forks; (B) design-with-owner, ONE runtime per storage class), and the dated verdict ledger. | AGENTS.md item 30; doc-check green (1356 refs). |
| A2 | **SKILL.md I/O rule** (consumer-facing hard-rule section): all durable I/O through metaengine, A/B gate pointer to ADR-0142. Matches the established `../../../docs/adr/...` link convention. | `.agents/skills/go-cqrs-lite/SKILL.md`; doc-check green. |
| A3 | **ADR-0142 amendment (2026-10-24)**: scope widened writes → ALL data I/O; A/B gate formalized; **verdicts table** (timers / queue / kv / enforcement); ledger of engine-ridden vs open items. | `docs/adr/0142-universal-storage-substrate.md`; `check-adr-numbering` OK (159 ADRs). |
| A4 | **Verified I/O ledger before deciding anything**: journal/projections/snapshots/checkpoints/dedup already ride engines — including the surprise finding that checkpoints are ALREADY engine-default (`resolveCheckpointStore`: custom > engine Map collection > memory fallback; `system/checkpoint_engine.go`). | Session evidence log (report §Appendix). |
| A5 | **Timers verdict executed (kill-now)**: all 21 package-level symbols in `scheduling/sqlstore` carry `// Deprecated:` notices → `scheduling/engine.NewTimerStore` over a `metaengine.DueClaimer` engine; deleted at v5.0.0. Files: `store.go`, `claiming.go`, `claim_metrics.go`, `dialect.go` (package doc + every type/ctor/var/const). | gopls renders the notices live; `GOWORK=off go test` green (7.5s). |
| A6 | **No lint-config debt created**: same-package test usage of own deprecated symbols is SA1019-safe (verified against the `idempotency/sqlstore` precedent — its tests use deprecated ctors with no exclusion), so `.golangci.yml` and its hash golden are untouched. | Session verification. |
| A7 | **Exempt demo handled**: `example/scheduler-otel-status` keeps compiling + testing green with a banner explaining it demos the deprecated `ClaimMetrics` hooks until the engine-path twin replaces it. | Build + `go test -short` green. |
| A8 | **Full docs trail for the deprecation**: CHANGELOG `[Unreleased]` Deprecated entry (changelog-symbols gate green), FEATURES.md (feature row + maturity row), module-map row, skill `references/modules.md` row, ADR verdicts, TODO_LIST rows (incl. T19–T21 cross-link). | All gates re-run green. |
| A9 | **Queue A-verdict recorded**: task tables stay engine-owned (engines ARE drivers; ClaimDue predicate richer than `DueClaimer`); revisit trigger rowed — reopen DueClaimer-v2 ONLY when a second consumer needs priority-ordered due-claims on engine collections. | ADR-0142 verdicts + TODO_LIST. |
| A10 | **cqrs-lint E023 `satellite-owned-sql` implemented** (approved verdict): advisory info/high; flags `database/sql` + first-party SQL-driver imports in packages that also import go-cqrs-lite; engine-tier prefix exemptions (metaengine/, claiming/, queue/{sqlite,postgres,mysql}, storage/, stack/, scheduling/sqlstore, testutil/, example/); test files exempt; package-level cross-file mixing signal grouped by PkgPath. | `e023.go` + 7 tests (fire/split/library-only/sql-only/test-file/exemption table). |
| A11 | **E023 wired into every companion gate**: register.go, catalog (E023 entry), RULES.md regenerated, README 214→215 rules (architecture 22→23), detector-count meta-test 214→215, V007 whole-module table gained `scheduling/sqlstore`, api-stability golden regenerated (6303 exports). | **Full cqrs-lint suite green** (two consecutive runs, zero FAIL). |
| A12 | **kv origin research delivered to the owner** (their explicit question): ADR-0022 (2026-06-16) — kv.Store born as a minimal byte-key KV meta-interface for event-store backend interchangeability (ordered iteration + atomic batches + byte keys; no Go meta-API had all three; born with zero consumers); ADR-0032 (2026-06-20) merged readmodel/ into it (TypedStore/ViewStore/Cache). The raw idea's job is now metaengine's job. Recommendation given: demote kv to Cache + in-memory rather than build an engine adapter that resurrects the sentenced v1 read-model tier. Decision left to owner as asked. | ADR-0022/0032 read; recommendation in session final message; ledger marked OPEN. |

---

## b) PARTIALLY DONE

| # | Item | What is missing |
|---|---|---|
| B1 | E023 verification breadth | Rule tested via unit fixtures + full lint-module suite; **never executed against this repo or a real consumer** — unknown what it fires on in systemtest/systemscenario/doc-check (info severity, so likely harmless, but unverified). |
| B2 | Engine-path observability demo | Rowed in TODO_LIST (replace `scheduler-otel-status` before v5); not started. |
| B3 | recipes.md §2.26 | Still teaches `scheduling/sqlstore` as the multi-instance-timers recipe; deprecation note/lead-with-engine deferred (rowed). |
| B4 | kv decision memo | Origin + recommendation delivered verbally in-session; the ADR verdicts table row still says "under evaluation" pending the owner's ruling. |
| B5 | Directive enforcement depth | E023 covers the satellite-SQL class; the I/O rule's other failure modes (hand-rolled file persistence, non-engine KV wiring in new code) have no mechanical detector. |

---

## c) NOT STARTED

| # | Item | Why it matters |
|---|---|---|
| C1 | Fleet scan for `scheduling/sqlstore` consumers (`~/projects` + census) | **Should have preceded kill-now.** In-repo consumers were checked (example + doc-check strings only); fleet consumers (go-appkit, cqrs-htmx, apps) were NOT. If a fleet app pins sqlstore, the deprecation is still correct (nothing breaks in v4) but the migration note needs their names. |
| C2 | kv verdict implementation (whichever option the owner picks) | Blocked on owner decision (Q1 below). |
| C3 | DueClaimer-v2 design | Gated by the revisit trigger — deliberately not started. |
| C4 | Harvest of the FIRST report's 50 items (extraction session) into TODO_LIST | F22 from the 16-56 report is still unexecuted; both same-day reports now carry unhavested (f)-sections. |
| C5 | E023 scorecard wiring (if it should grade consumers' ModernityGrade) | Idea noted in TODO_LIST follow-up line; not started. |
| C6 | `nix run .#verify` / `#verify-fast` full pass | Module-level tests + gates all green, but the composed verify (build+vet+test+race+lint+doc-check for the whole workspace) was not run this session (load-gated, heavy). |

---

## d) TOTALLY FUCKED UP

Honest list — every item was caught and fixed in-session, but each one is a process failure worth naming:

| # | Failure | Severity | What happened → fix |
|---|---|---|---|
| D1 | **Deprecated markers written before their gate-wiring** | High | 21 `// Deprecated:` markers landed without a V007 table entry. The drift gate (`TestV007_TablesCoverAllV5DeprecationMarkers`) caught them — and its existence was documented in a TODO_LIST note I had READ that same session (M23 receipt). The checklist existed; I used the test-failure loop instead of the checklist. Fix: added `scheduling/sqlstore` to `deprecatedV5Modules` (which also gives consumers a v5-removal warning with the replacement pointer — a benefit I only got by accident). |
| D2 | **Rule companion-pins discovered by failure** | Medium | Adding E023 tripped THREE separate pins I did not pre-enumerate: detector-count meta-test (214→215), RULES.md freshness, README rule count. The repo's ADR-0128 removal checklist has an obvious add-rule inverse; I never looked for one. Three fix cycles that one checklist read would have prevented. |
| D3 | **`rg -r` flag trap hit AGAIN — twice** | Medium | The host hook warns that `rg -rn` looks like replace-display and blocks it; I triggered it twice this session (the hook now says "third strike of this exact trap on this host"). Sloppy repetition of a known-local failure mode. Correct form is `rg -n` (recursion is default). |
| D4 | **Wrong import path in the new test file — twice** | Low | `cqrs-lite` vs `cqrs-lint` typo survived two edits (one of my "fixes" re-introduced it). Caught by the compiler; still two wasted cycles. |
| D5 | **Assumed `BuildContextFromSource` leaves `Pkg` nil** | Low | I read the first 25 lines of `test_helpers.go`; the Pkg fabrication is at line 38 — each file gets a UNIQUE PkgPath. Cost: one debug cycle and a test rewrite (pin both files to one PkgPath). Reading the whole helper first was the fix. |
| D6 | **kill-now without the fleet census** | Medium | The repo's own law (AGENTS: fleet evidence IS the kill/integrate signal) demands the census before deletion-class decisions. I checked in-repo consumers only. The deprecation is reversible and v4-safe, which is why this is not worse — but the sequence was wrong. |

**Did I lie?** No. Every claim in the final summary has a green gate behind it. But D1/D2 mean my "wired into every gate" statement was only true AFTER the session's last test run — the honest phrasing is "the gates taught me the wiring."

---

## e) WHAT WE SHOULD IMPROVE

**1. What did you forget?**
The fleet census before kill-now (D6); the V007 table (D1); the rule companion-pins (D2); harvesting the first report's (f)-section; running E023 against the repo itself.

**2. What is something stupid that we do anyway?**
Deprecation and rule-adding are both multi-gate rituals whose checklists live in scattered ADR notes and test pins instead of one procedure file. This session paid that tax twice (D1, D2). Also: 21 hand-written near-identical deprecation notices — a candidate for a tiny generator or a documented snippet set.

**3. What could you have done better?**
Executed D1/D2 as checklist-first, failure-never. Read whole helper files before coding against their contracts. Run the one-command consumer scan (`rg` over `~/projects`) BEFORE deprecating anything.

**4. What could you still improve?**
Turn the two rituals into checklists in AGENTS.md §Procedures: "Deprecate a module" (markers → V007 table → CHANGELOG → FEATURES → module-map → modules.md → example banners → changelog-symbols gate) and "Add a lint rule" (rule → register → catalog → RULES.md regen → README count → detector-count test → api golden → V007 if deprecation-adjacent). One evening of writing, repaid every future wave.

**5. Did you lie to me?** No — see §d closing.

**6. How can we be less stupid?**
Make gates fail WITH the checklist: the V007 drift failure message and the RULES.md staleness message already do this well ("regenerate: ..."); the detector-count test could name the companion files too. Also: when a host hook blocks a command pattern twice, write the correct form into memory instead of re-offending (D3 is now recorded here as that memory).

**7. Ghost systems?** None created. E023 is registered, cataloged, tested, and documented — fully wired. The `IsEngineTierPackageForTest` idea from my first test draft was replaced by an internal test (no dead export shipped).

**8. Scope creep?** No: every change traces to one of the four verdicts. The E023 implementation was in-scope (approved via Q4).

**9. Did we remove something useful?** No removals; deprecation only, with the exempt demo preserved.

**10. Split brains created or found?**
Created: none. Found: the deprecation ritual knowledge itself is split across ADR-0128 (removal), ADR-0158 (deprecation precedent), V007 tables, and test pins — the §e.4 checklist consolidation kills that split brain.

**11. Tests?**
E023: 7 focused tests (two fire cases incl. cross-file package mixing, three silence cases, exemption table) + full module suite green twice. sqlstore: suite green post-markers (7.5s). Example: green. api-stability: `TestEvery` green. Not run: race, composed `#verify`, real-repo E023 execution (B1/C6).

**Architectural note worth keeping:** the session's best finding is that the I/O ledger was already 5/8 green before the directive landed — ADR-0142's 2026-09 execution had silently finished checkpoints too. Directives that arrive as "make X true" should start with a measured ledger; this one did (after the user's questions forced it), and it changed the verdicts (checkpoints needed nothing; the "flip T20" framing was stale).

---

## f) 50 things we should get done next

Types: **DECIDE** (owner gate), **VERIFY**, **BUILD**, **DOC**. Sorted roughly by impact/effort.

| # | Item | Type | Impact | Effort |
|---|---|---|---|---|
| F01 | Owner rules on kv: demote to Cache+memory (recommended) vs engine Map adapter; record verdict in ADR-0142 + contract 30 ledger | DECIDE | H | S |
| F02 | Fleet scan for `scheduling/sqlstore` consumers; name them in the CHANGELOG migration note if any | VERIFY | H | S |
| F03 | Run E023 against this repo (self-lint) and one fleet consumer; triage what fires | VERIFY | H | S |
| F04 | Write AGENTS.md §Procedures "Depprecate a module" checklist (from D1: markers → V007 → CHANGELOG → FEATURES → module-map → modules.md → banners → gates) | DOC | H | S |
| F05 | Write AGENTS.md §Procedures "Add a lint rule" checklist (from D2: rule → register → catalog → RULES.md → README count → detector-count test → api golden) | DOC | H | S |
| F06 | Run `nix run .#verify` (or `#verify-fast`) over the session's change set | VERIFY | H | M |
| F07 | Harvest THIS report's (f)-items + the 16-56 report's 50 items into TODO_LIST (docs-health HARVEST; two reports now entombed) | DOC | H | M |
| F08 | Engine-path observability demo (scheduling/engine + engine instrumentation) replacing scheduler-otel-status | BUILD | M | M |
| F09 | recipes.md §2.26: deprecation note or full conversion to scheduling/engine (keep compile-harness catalog green) | DOC | M | S |
| F10 | E023 follow-up: decide scorecard/ModernityGrade wiring (should satellite SQL grade consumers?) | DECIDE | M | S |
| F11 | Detector-count meta-test failure message: name the companion pins (README count, RULES.md regen) so D2 self-documents | BUILD | M | S |
| F12 | sqlstore deprecation: verify cqrs-upgrade's deprecation report picks the module up (V007-driven) in a consumer fixture | VERIFY | M | S |
| F13 | Check whether self-lint (CI) covers `example/` — if yes, V007 will flag scheduler-otel-status's own sqlstore import; add waiver or accept info noise | VERIFY | M | S |
| F14 | kv: if demote-verdict — mark `kv.TypedStore`/`ViewStore`/byte-`Store` deprecated-for-durable-use in docs; keep Cache/MemStore as the v5 surface proposal | DOC | M | S |
| F15 | kv: if adapter-verdict — implement `kv.Store` over `metaengine.MapBackend` behind conformance tests (SnapshotAdapter precedent) | BUILD | M | M |
| F16 | Add E023 to the E-rule docs table in skill references (advanced.md or modules.md rule list) if one exists | DOC | L | S |
| F17 | Update the 16-56 extraction report's queue verdict cross-reference (queue A-verdict now recorded; extraction question unaffected) | DOC | L | S |
| F18 | Consider a deprecation-notice generator (21 near-identical notices were hand-written; snippet or `go generate` helper) | BUILD | L | M |
| F19 | Run the fleet census (`scripts/census-fleet.sh`) and refresh `docs/evidence/` — also answers the extraction session's Q-gap | VERIFY | H | M |
| F20 | V5 deletion wave prep: sqlstore joins T19–T21; confirm the deletion checklist row includes example retirement + recipes purge | DOC | M | S |
| F21 | `scheduling/engine`: promote out of 🧪 Experimental in FEATURES once the fleet migration lands (it is now THE timer path) | DOC | M | S |
| F22 | systemtest/systemscenario: confirm no satellite SQL lurks (E023 self-lint covers; see F03) | VERIFY | M | S |
| F23 | Teach E023 the watermill/transport exemption explicitly if brokers ever import database/sql (currently only path-prefix exempt) | BUILD | L | S |
| F24 | Add E023 mutation test (mutant: drop the mixing signal → fire/silence tests must discriminate) if the mutant_discrimination suite doesn't already cover it | BUILD | M | S |
| F25 | CHANGELOG: add E023 to the cqrs-lint section (215 rules) as an Added entry if the wave hasn't already | DOC | M | S |
| F26 | Contract 30 ledger: tick `kv` once F01 lands; keep the ledger the single live view | DOC | L | S |
| F27 | ADR-0142: when T19–T21 executes, link the deletion receipts back to the 2026-10-24 verdicts table | DOC | L | S |
| F28 | Consider flagging `os.WriteFile`/file-persistence in library modules under contract 30 (second E-rule class beyond SQL) — design with owner first per gate B | DECIDE | M | S |
| F29 | cqrs-lint README: add E023 to the curated highlights table (architecture section) | DOC | L | S |
| F30 | Verify E023 behaves under `--strict` (info severity must never gate; write the pin test if absent) | BUILD | M | S |
| F31 | scheduling/engine docs: add a migration table sqlstore→engine (API mapping incl. ClaimMetrics → engine instrumentation delta) | DOC | M | M |
| F32 | example/taskmanager: confirm its queue usage renders correctly under E023 (queue engines exempt; the example itself is exempt-prefix) | VERIFY | L | S |
| F33 | Fold the "pgengine/typed_wal.go gopls error" (foreign, persisted all session) into the concurrent session's owning report — not this one's to fix, but someone should triage | DOC | L | S |
| F34 | Record the `rg -n` (never `-rn`) host rule in docs/agents/gotchas-tooling-build.md if absent (third strike happened here) | DOC | M | S |
| F35 | Time-box check: are the two same-day status reports index-linked in the docs/status live index? Update if one exists | DOC | L | S |
| F36–F50 | Reserved directive follow-through (parked, LOW until F01 rules): kv implementation either way; DueClaimer-v2 only on revisit trigger; E023 scorecard; verify-chain run; tag-wave entry for cqrs-lint v4.15.0 (215 rules) when the wave cuts; consumer-facing migration guide section for timers; FEATURES maturity matrix sync; module-map census note for the E023 files; ROADMAP extraction section cross-link to the 16-56 report; final full-report HARVEST closure. | MIXED | L | — |

---

## g) Questions I cannot answer from the repository (max 3)

**Q1. kv verdict?** Demote to Cache + in-memory (my recommendation — the raw idea's job belongs to metaengine now, and an engine adapter would resurrect the v1 read-model tier ADR-0123 sentenced), or build the one engine Map adapter (`kv.Store` over `metaengine.MapBackend`, SnapshotAdapter precedent) to keep durable typed-KV as a first-class surface?

**Q2. E023 severity for the fleet?** It ships as advisory `info`. Should it stay advisory (non-CQRS tables may legitimately exist in consumer apps), or be promoted to `warning` in a coming wave so the scorecard/strict modes surface it harder?

**Q3. sqlstore deletion timing?** v5.0.0 as pinned, or — if F02's fleet scan finds zero consumers — invoke the delete-at-zero policy (ADR-0158 precedent killed `idempotency/kvstore` mid-v4) and delete `scheduling/sqlstore` at the next wave instead?

---

## Appendix: session evidence log

1. Ledger verification greps: `system/` sibling imports (only `snapshot/v4`), `resolveCheckpointStore` (custom > engine > memory), `SnapshotAdapter` over `SnapshotBackend`, `kv.Store` implementors (legacy-only), `claiming` consumers (sqlstore = the one non-engine consumer).
2. ADR-0142 full read incl. amendment + capability matrix; ADR-0134 (claim tokens); TODO_LIST T19–T21/M23/M24 notes.
3. Deprecation edits: `scheduling/sqlstore/{store,claiming,claim_metrics,dialect}.go` (21 symbols + package doc); `example/scheduler-otel-status/main.go` banner.
4. E023: `e023.go` (+PkgPath-grouping fix), `e023_test.go` (5 tests), `e023_internal_test.go` (exemption table), register.go, catalog_architecture.go, RULES.md regen, README count, meta_test 215, v007_tables `scheduling/sqlstore`, api_surface.txt regen (6303).
5. Gates run green: doc-check (1356 refs, twice), check-adr-numbering (159), check-changelog-symbols, sqlstore suite, example suite, full cqrs-lint suite (twice, final zero-FAIL), api-stability TestEvery.
6. Gates that failed mid-session and were fixed: V007 drift (D1), RULES.md freshness + README count + detector count (D2).

## Closing note

Section (f) is NOT yet in TODO_LIST (only the directive's own five rows are). F07 exists to close that loop for BOTH of today's reports; until it runs, this file and the 16-56 report are the only records.
