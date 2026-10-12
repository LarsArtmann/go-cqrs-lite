# Status: N05 completed (doctor-audit bug fixed end-to-end) + M20 verify blockers fixed

**Session end:** 2026-10-12 17:15 CEST
**Branch:** master, tree clean, HEAD `23423746b`. **Origin moved externally during the session**: a push landed
origin/master at `7fe3d8c04` (the daemon commit carrying the scheduling clock fix) — only the authored M20
receipt `23423746b` is unpushed. A concurrent session is active in this repo (109-file wave `f2c892b5b` at
11:38 + a 151-line status report of its own in `a31671316`); its in-flight lint debt is the one thing keeping
repo-wide verify red.
**Scope:** continuation of the 12:26 report under the re-invoked execution mandate. The doctor stale-audit
bug that report found is now ROOT-CAUSED, FIXED, GATED, and DEPLOYED to the consumers that use the PATH
binary — plus the M20 quiet-window campaign ran to an honest receipt (two gate blockers fixed, lint leg
blocked by the other session's files).

---

## a) FULLY DONE

1. **cqrs-lint doctor-audit bug root-caused.** `cqrs-lint doctor` passed the configured path (typically `.`)
   into `analyzer.BuildContext` verbatim while the lint run absolutizes first (main.go:214-219). With a
   relative root, `DetectFeaturesPerModule` produced RELATIVE profile keys (`api`, `domain`, ...), and
   `ProfileForFile`'s longest-prefix match — which compares against go/packages' always-ABSOLUTE finding
   positions — silently missed for every file, falling back to the PRIMARY module's profile. On
   browser-history the api module's detected `HasServer=true` was lost (root module says false), so
   B029/B030 never fired in the doctor audit pipeline and two WORKING suppressions classified STALE.
   Evidence: env-gated dumps at `applyConfigOverrides` entry (detected HasServer=TRUE in both paths) vs
   `ProfileForFile` result inside `computeSuppressionAudit` (false in doctor) — the flip localizes to
   BuildContext's key derivation, not to `applyConfigOverrides` (idempotent, verified field-by-field) or
   `IsLibrarySelfLint` (position-based, false in both).
2. **Fix landed at the source.** `BuildContext` canonicalizes its root to an absolute path before any
   downstream derivation (`cmd/cqrs-lint/pkg/analyzer/loader.go:125`), so profile keys are absolute
   regardless of caller; `relDisplay`-style consumers get correct relative rendering as a bonus.
3. **Second latent bug fixed in the same stroke.** `AuditSuppressions` (the doctor-path classifier) used a
   hand-rolled line±1 matcher without the suppression filter's blank-skip walk; it now classifies through
   the SAME per-file matcher (`matchedSuppressionsForFile`) as the filter and the lint-run stale gate, so
   `cqrs-lint` and `cqrs-lint doctor` agree BY CONSTRUCTION on which directives are honored
   (`cmd/cqrs-lint/pkg/suppression/stale_audit.go`).
4. **Parity regression tests pinned.** New tests: blank-line-separated directive must classify ACTIVE
   (the exact false-positive shape), adjacent-above ACTIVE, code-between STALE, no-finding STALE, plus a
   cross-check that `AuditSuppressions` STALE verdicts ≡ `DetectStaleSuppressions` output. Multi-module
   integration tests updated to the absolute-key contract. Full cqrs-lint module suite green
   (`GOWORK=off go test ./...`, zero FAIL lines).
5. **End-to-end consumer verification.** browser-history's `nix run .#check-cqrs-lint` exits 0 —
   `✓ cqrs-lint passed clean (0 unsuppressed findings, 0 stale suppressions)` — with the `:159` B029/B030
   directives intact; the audit reports 29 suppressions, 0 stale. The gate's ground-truth env was
   replicated exactly first (`nix develop -c bash -c 'unset GOWORK; export GOEXPERIMENT=jsonv2 ...'`,
   validated by matching the gate's own lint-stage counts: 119 files, 28 suppressed).
6. **Global binary deployed.** `~/go/bin/cqrs-lint` rebuilt from the fixed tree (twice this session: the
   P015 roll this morning, then this fix). browser-history + cqrs-htmx (PATH-binary consumers) get the fix
   immediately; inbox-clean (workspace-linked) on next run.
7. **M20 blocker #1 fixed: api-stability ReadCosts roster drift.** `TestEngineProfilesSetReadCosts` still
   scanned `metaengine/engine.go` for sqliteengine's ReadCosts block after the profile factory was
   ratcheted into `engine_sqlite_profile.go`; roster now follows the extraction (same as the pg/duckdb
   `profile.go` rows). Test green; committed (daemon `0b432e1a4`).
8. **M20 blocker #2 fixed: scheduling clock-test data race.** The frozen-clock test advanced its injected
   clock by plain assignment while the scheduler tick goroutine read it through the `WithClock` closure —
   one DATA RACE failing five scheduler tests under `-race` at gate load. Fixed via `atomic.Value`; the
   test is green 3× under `-race -count=3` and the full scheduling suite is green under `-race`
   (committed, daemon `7fe3d8c04` — which origin now carries).
9. **Gotchas recorded** (`docs/agents/gotchas-tooling-build.md`, new "Fleet-roll traps" section): the
   stale-FOD trap (rev+old-hash silently serves old content; distinct-wrong placeholder dance; verify the
   built binary registers the new rule), the same-binary path-divergence debugging lesson (instrument the
   SHARED function's entry in both callers before theorizing about shared logic), and the env-replica
   recipe (validate against gate-visible counts before trusting any diagnostic data from a replica).
10. **TODO_LIST receipts harvested**: N05 fleet-roll receipt (fix-unreleased pin hazard, cqrs-htmx triage
    scope, per-repo delivery verdicts, binary-roll recipe) + M20 quiet-window verify receipt (two fixes,
    blocked lint leg, working invocation, the nonexistent-flake-app trap). 12:26 status report annotated
    RESOLVED inline (docs-health ANNOTATE discipline).
11. **doc-check green**: 1,284 references valid across 57 packages.
12. **This report** + index row (see §f item on HARVEST).

## b) PARTIALLY DONE

1. **M20 quiet-window verify — changed-set green, repo-wide blocked.**
   - Works: build ✓, vet ✓, full test+race sweep over the session's changed modules ✓ (cmd/cqrs-lint 51s
     green in-window, cmd/api-stability green post-roster-fix, scheduling green post-race-fix), doc-check ✓
     standalone, all inside genuine quiet windows (wrapper opened at load 3.62/9.37).
   - Remaining: the LINT leg reports pre-existing `gocognit` findings in `watermill/command_bus_test.go:264`,
     `watermill/event_bus_test.go:183`, `systemtest/time_fidelity_test.go:81` (+8 modules: schema,
     cmd/cqrs-upgrade, metaengine + 4 engines, core/v5, systemscenario, systemtest) — all from the
     concurrent session's `f2c892b5b` wave, none from this session's files. Blocked until that session's
     triage lands (or a follow-up restructures the three named test files). Effort to finish: S-M for the
     three named files, unknown for the other 8 module verdicts (likely the same wave).
2. **N05 fleet roll — fix unreleased.** PATH-binary and workspace-linked consumers have the fix; pbx-lens
   pins the v4.16.0 TAG (pre-fix source) and stays two fixes behind (P015 it has; doctor-parity it lacks).
   Blocker: release mechanics are a user decision (patch tag now vs next wave). Effort: S (tag + re-pin +
   smoke) once decided.
3. **cqrs-htmx gate triage — scoped, not executed.** 11/13 gate units green under the rolled binary. The
   root unit carries 28 findings (4×E021 schema declarations, 3×P009 JSON-CBOR []byte payloads, 4×A032,
   2×B027, 2×A013, 10 singletons) and systemadapter carries 2 (V003: systemscenario 3 minors behind;
   E014: projectionhost without a drain/sync/flush call). All are adoption-coaching / version-policy
   decisions parked behind §g Q2 — suppressing coaching rules wholesale without that decision would falsify
   the adoption signal. Also recorded: the gate MUST run via the flake app (its devShell sets GOWORK=off —
   a bare invocation reports a false "none import go-cqrs-lite").
4. **InboxClean receipt.** Its flake path fix is pushed (prior session) and it is workspace-linked, but this
   session never re-ran its gate to prove the fixed binary flows through. S effort, next time its gate runs.

## c) NOT STARTED

1. **M05 fleet adoption legs** (bank-sync state-shape patch F24, DiscordSync `map[any]any` workaround drop
   F25, cqrs-htmx UpcasterRegistry collapse onto `UpcastChain()` F26, graph-native standalone check F27) —
   gated on §g Q2 (roll scope) since three of four legs rewrite fleet repos whose linter/dependency
   versions are entangled with the same decision.
2. **M09–M11 v5 deletion slices + M27 cut assembly** — plan-sequenced after M05/M20; nothing written.
3. **Entangled fleet rolls** (bank-sync / DiscordSync / SystemNix linter + dependency version bumps) —
   frozen pending §g Q2.
4. **Push to origin** — 1 authored commit (`23423746b`) local; never push without explicit authorization.
5. **`--show-suppressed` structured-format parseability fix** (TODO_LIST P015-harvest item; untouched).
6. **M20 ratified design follow-ups T11** (EngineConfig.SingleWriter lease, AggregateOn first cut, routing
   v1) — pre-v5 W2 work, untouched this session.
7. **SKILL.md / references doctor-audit text audit** — NOT done this session (see §d item 5: I flagged it
   only while writing this report).

## d) TOTALLY FUCKED UP

1. **I repeated the archaeology-before-instrumentation failure the prior session already paid for.** The
   12:26 report's lesson was literally "instrument-before-archaeology", yet I spent ~6 tool rounds on
   static analysis (applyConfigOverrides idempotency, IsLibrarySelfLint, preset resolution, filter
   semantics) before adding the debug dump — which then cracked the case in ONE round. Mitigation now
   recorded as a gotcha; the generalizable version (instrument the shared function's ENTRY in both
   callers) is a `references/lessons.md` candidate.
2. **My first debug dump produced a false alarm I almost chased.** The initial probe indexed
   `actx.FeatureProfiles` with an ABSOLUTE dir against RELATIVE keys and printed a zero-value profile —
   read naively, that says "the map is empty", a wrong root cause. Caught only by improving the dump
   (print keys + use the real `ProfileForFile`) instead of reasoning from the first reading. Lesson: a
   diagnostic whose lookup can MISS silently must print its lookup key form, not just its result.
3. **The M20 window hunt burned ~1.5 hours on avoidable mistakes**: ceiling 6 (unreachable on a 25-31-user
   shared box — the default is 5, I picked 6), a flag typo (`--max-wait` vs `--ceiling/--deadline`), and
   launching the campaign WITHOUT pre-flighting the two modules most likely to fail (scheduling had a
   red-suite history per the 2026-10-06 lesson; api-stability is a meta-test magnet). Each verify attempt
   is 10-40 minutes; two attempts died on bugs a 60-second targeted pre-run would have caught. Layered
   gates made it worse: my outer wrapper opened at <14 while verify's OWN gate refuses at load5 ≥ 10 —
   three attempts burned as refusal loops.
4. **The gate's own retry hint names a flake app that does not exist** (`nix run .#can-run-composed-gate` —
   "does not provide attribute"). I followed the hint and lost a cycle. Severity: low but corrosive — the
   documented recovery path for a refused gate is dead. Workaround recorded in the M20 receipt
   (`quiet-window-run --ceiling 10` replicates the precondition; the real `scripts/can-run-composed-gate.sh`
   is one-shot only).
5. **I did NOT check whether the skill references document the doctor-audit semantics I changed.**
   `.agents/skills/go-cqrs-lite/references/` may still describe `--audit-suppressions`/
   `--fail-on-stale-suppressions` behavior that this fix alters (matcher parity, absolute-key contract).
   doc-check passes (it validates SYMBOL references, not behavioral prose), so a behavioral lie could be
   sitting there undetected. Needs a targeted read of faq.md/core.md doctor sections — flagged in §f.
6. **Commit/daemon races hit three times** (index.lock once, two nothing-to-commit absorptions). Known
   pattern, known cure (check `git status` immediately before committing; commit at phase boundaries) —
   I still lost two authored-commit opportunities; the fix content is fine (daemon commits), attribution
   is what was lost.

## e) WHAT WE SHOULD IMPROVE

1. **Instrument-first as a HARD rule** for divergence bugs: when two code paths disagree, the first move is
   an env-gated dump at the shared function's entry in BOTH callers — before any theory. Impact: this
   session's ~6 wasted rounds; the prior session wasted ~6 on the same pattern. Candidate for
   `references/lessons.md` (cross-project).
2. **Env-replica validation gate**: any diagnostic data collected under a reproduced env is inadmissible
   until the replica reproduces the REAL environment's observable counts (files analyzed, findings
   suppressed). The prior session's "wrong findings universe" conclusion was itself suspect for exactly
   this reason; this session validated parity first and it paid off.
3. **Pre-flight heavy campaigns**: before a long quiet-window campaign, run the historically-red modules
   standalone (60s each). The M20 campaign would have been: pre-flight (2 fixes) → ONE window → green.
   Consider a `scripts/preflight-heavy-modules.sh` (scheduling, api-stability, bench-adjacent).
4. **The layered-load-gate story needs one number**: outer wrapper ceiling (mine: 14/10), verify's internal
   gate (10), bench gate (its own), calibration gate (its own). A shared constant or a table in
   gowork-modes.md/gotchas would stop every session re-deriving it by failure.
5. **Fleet pin-lag visibility**: PATH-binary repos get fixes instantly; source-pin repos (pbx-lens) lag a
   wave — silently. A "fleet linter version" column in the versions manifest (or the fleet delivery map as
   a maintained doc) would make the lag a checkable fact instead of a report paragraph.
6. **ProfileForFile hardening**: the silent fallback that hid this bug for a wave deserves a loud mode —
   when no profile key matches and the primary is used, WARN once (doctor renders it; lint can stay
   silent). Would have surfaced the relative-key class the day it shipped. (§f item.)
7. **Authored commits under the daemon**: the only reliable window is immediately-after-write with a
   pre-commit `git status` check. Build the habit or accept chore-attribution; both are fine, dithering
   isn't.
8. **BuildFlow pre-commit findings gate** still fails every authored commit on the same 10 pre-existing
   error findings (cqrs-lint 7, securitymd 2, go-structure-linter 1 — nightly-rot class) plus the known
   pseudo-version FP. It's report-only noise at this point; the batch fix (§f) or a waiver pass would
   restore signal.

## f) NEXT (priority-ordered, up to 50 — brainstorm for HARVEST; most are TODO_LIST/ROADMAP fuel)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Cut the next cmd/cqrs-lint tag wave carrying the doctor-parity fix (patch v4.16.1 or bundle) | High | S | Release |
| 2 | Re-pin pbx-lens to the fixed tag (hash dance: distinct-wrong placeholder, verify P015 + doctor parity) | High | S | Fleet |
| 3 | Answer §g Q2 (roll scope) — unblocks items 4-9 | High | — | Decision |
| 4 | M05 F24: apply bank-sync state-shape patch; build + tests | High | M | Fleet |
| 5 | M05 F25: drop DiscordSync `map[any]any` workaround branch | High | M | Fleet |
| 6 | M05 F26: collapse cqrs-htmx homegrown UpcasterRegistry onto `UpcastChain()` | High | M | Fleet |
| 7 | M05 F27: `check-example-standalone.sh --build` → 0 findings (graph-native) | Medium | S | Fleet |
| 8 | cqrs-htmx root-unit triage: 28 findings (E021 ×4, P009 ×3, A032 ×4, B027 ×2, A013 ×2, 10 singletons) — audit-or-suppress with recorded reasons | High | L | Fleet |
| 9 | cqrs-htmx systemadapter: V003 (systemscenario bump) + E014 (projection drain) decisions | Medium | M | Fleet |
| 10 | Repo-wide green verify once the concurrent session's gocognit triage lands (3 named files + 8 modules) | High | M | Quality |
| 11 | Audit `.agents/skills/go-cqrs-lite/references/` (faq.md, core.md doctor sections) for doctor-audit semantics this fix changed; update + doc-check | High | S | Documentation |
| 12 | Fix the verify gate's retry hint: add a `.#can-run-composed-gate` app or rewrite the hint to the script | Medium | S | Tooling |
| 13 | Harden `ProfileForFile`: WARN (doctor-rendered) when no key matches and the primary profile is used | High | S | Bug-prevention |
| 14 | Pre-flight script for heavy campaigns (scheduling, api-stability, timing suites) before quiet-window runs | Medium | S | Tooling |
| 15 | Decide push authorization for `23423746b` (+ the carried-over commit-reflex question) | Medium | — | Decision |
| 16 | `--show-suppressed` parseability: structured formats carry a `suppressed` array (or route text to stderr) + regression test | Medium | M | Bug |
| 17 | M09 deletion slice 1: `stack.Materialize`, ADR-0126 shells, `storage/sql.BuildWhereClause`, tombstone metadata API + golden regen | High | L | Feature |
| 18 | M10 deletion slice 2: `storage.RelationalProjection`+`view`, `graph.GraphProjection`, `stack.Bundle`+presets | High | L | Feature |
| 19 | M11 breakage pair: `record.NewStreamRef` empty-entityID validation; core/v5 `TombstoneFilter` | High | M | Feature |
| 20 | M20-T11(a): implement EngineConfig.SingleWriter advisory lease (ADR-0150 ratified) | High | L | Feature |
| 21 | M20-T11(b): AggregateOn first cut (AggregateSpec QueryOption + MatViewSpecReporter) | High | L | Feature |
| 22 | M20-T11(c): routing integration v1 (scalar-covered shapes price O(1)) | High | L | Feature |
| 23 | Turso strict DSN param policy implementation (RULED: reject unknown *key*/*encrypt* params) | Medium | M | Feature |
| 24 | Batch-fix the nightly-rot 10 error findings gating BuildFlow pre-commit (cqrs-lint 7, securitymd 2, go-structure-linter 1) | Medium | M | Quality |
| 25 | BuildFlow upstream filing #1: pseudo-version-hygiene FP on /v4 module paths (with the 2026-10-12 breakage receipt) | Medium | S | Upstream |
| 26 | BuildFlow upstream filing #2: go-work-paths /v4 mismatch warning class (90 paths of noise) | Medium | S | Upstream |
| 27 | Fleet linter-version visibility: add a per-repo cqrs-lint delivery/version table to the versions manifest or a maintained doc | Medium | M | Documentation |
| 28 | lessons.md candidate: "instrument shared-function entries first" + "validate env replicas against observable counts" (cross-project, via crush-config commit) | Medium | S | Process |
| 29 | Re-verify InboxClean's gate actually flows the fixed binary (workspace-linked; receipt pending) | Low | S | Fleet |
| 30 | Re-run browser-history's gate after item 1's tag lands, using the TAGGED binary (not the local build) | Medium | S | Verification |
| 31 | nsfw-classifier: confirm doctor parity fix behaves there too (its lock is current; one gate run) | Low | S | Verification |
| 32 | M06 repro-test legs (#62 sqlite time.Time filter vs RFC3339; #63) | High | M | Bug |
| 33 | Plan the next release wave: [Unreleased] now holds 6 Fixed entries (Pebble DESC, 2×MySQL, lint-config restore, api-stability golden, doctor-parity) + this session's 2 gate fixes | High | M | Release |
| 34 | doc-check ambiguous-alias advisories (`id`/`kv` core-v5 vs v4): scope the imports in readmodels.md:114 + recipes.md:69 | Low | S | Documentation |
| 35 | AGENTS.md 322 > BuildFlow's 220-line budget: trim or record a justified limit bump | Low | S | Documentation |
| 36 | lychee excludes for archived doc dirs (recurring warn every preflight) | Low | S | Tooling |
| 37 | Track GO-2026-6617 (stdlib HTTP/2 HPACK) — toolchain bump when the fixed Go lands in nixpkgs | Medium | S | Security |
| 38 | Make verify's internal load ceiling configurable (env knob) for shared-machine sessions; VERIFY_FORCE is too blunt | Low | S | Tooling |
| 39 | Record the metaengine/go.mod sqliteengine v4.5.3 pinned-require as a justified BuildFlow-doctor suppress (the FP fires every run) | Low | S | Tooling |
| 40 | M22 owner Q3 (TODO_LIST header carry-over) | Low | — | Decision |
| 41 | #26 retract-publish watch: close when the next stack/postgres tag lands | Low | S | Release |
| 42 | Stable output path for local cqrs-lint builds (/tmp binary vanished mid-session; use ~/go/bin directly) | Low | S | Process |
| 43 | Background-shell hygiene: kill completed jobs promptly (this session left several; the 50-job ceiling is real) | Low | S | Process |
| 44 | Consider a fleet-wide "gate env contract" note: every consumer gate's env (GOWORK state, GOEXPERIMENT, PATH source) documented next to its flake app | Medium | M | Documentation |
| 45 | Ratify the M20 shape this receipt proposes: changed-set trust green + repo-wide once the concurrent wave clears (replaces the old "quiet-window campaign" framing) | Medium | — | Decision |
| 46 | docs/status README index row for this report (done with the report; listed here for HARVEST completeness) | — | — | — |
| 47 | Whether E021/P009 coaching findings in fleet repos should auto-suggest in cqrs-upgrade's suggestions section (T25 pattern) — idea-level | Low | M | Idea |
| 48 | SystemNix: master-tracking input lock is stale (`7142e43de`) — roll when Q2 is answered | Medium | S | Fleet |
| 49 | bank-sync flake pin-policy comment says frozen rev — revisit under the same Q2 bundle | Medium | S | Fleet |
| 50 | The 12:26 report's §f items not superseded by this session (SystemNix/bank-sync/DiscordSync untouched; M09-M11/M27) — carried verbatim | — | — | — |

## g) QUESTIONS (cannot answer myself)

**Q1 — Fix-release mechanics.** Should the doctor-parity fix ship as a fast standalone `cmd/cqrs-lint/v4.16.1`
patch tag (pbx-lens re-pins immediately; browser-history/cqrs-htmx already have it via PATH), or bundle into
the next feature wave? I cannot decide because the wave cadence is yours: a standalone tag costs ~15 min
(tag + smoke + re-pin) but adds a version; bundling leaves pbx-lens on a doctor that false-flags working
suppressions until the next wave — tolerable only if that wave is imminent.

**Q2 — cqrs-htmx gate posture.** The failing root unit's 28 findings are mostly ADOPTION coaching (4×E021
"wrote schema declarations" is a real migration project; 3×P009 CBOR swaps are wire-format changes), not
quick suppressions. Do I (a) suppress batch with documented reasons to get the gate green today and file
the adoption work as M05 legs, (b) hold the gate red as an honest adoption signal until M05 lands, or (c)
split: suppress the singleton noise, leave E021/P009 red? And within the same decision: systemadapter's
V003 (systemscenario 3 minors behind) — bump NOW or fold into the entangled bank-sync/DiscordSync/SystemNix
roll bundle you were already asked about (the 12:26 report's Q2)?

**Q3 — Ownership of the verify-blocking lint debt.** Repo-wide verify is red solely on `gocognit` findings
in `watermill/command_bus_test.go`, `watermill/event_bus_test.go`, `systemtest/time_fidelity_test.go` (+8
modules) introduced by the concurrent session's 109-file wave (`f2c892b5b`, 11:38 today). Is that session
still active and planning to triage them (I stay off its files), or should a follow-up of mine restructure
those tests to unblock the gate? If mine: test-complexity refactors in another stream's fresh work carry a
real merge-conflict risk, so I want the all-clear before touching them.

---

*Report then WAIT. HARVEST note: §f items 1-16 and 24-28 are TODO_LIST-grade; the rest are ROADMAP fuel or
one-line receipts already harvested into the N05/M20 receipt blocks.*
