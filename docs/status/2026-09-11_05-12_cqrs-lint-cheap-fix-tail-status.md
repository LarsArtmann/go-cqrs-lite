# Status Report — cqrs-lint audit cheap-fix + test-gap tail executed

> **RESOLVED-BY ROUTING (docs-health 6th pass, 2026-09-11):** §f rows 24/30/32/33 are struck inline (file-size ratchet shipped; `RULES.md` regenerated — 206 rules recounted; the reset ladder completed by the 05-40/05-51 sessions; §f harvested into TODO_LIST/ROADMAP with citations). Remaining open §f rows live in `TODO_LIST.md` / `ROADMAP.md` (notably the golden-profile harness idea). This snapshot is ARCHIVED; the living backlog is `TODO_LIST.md`.

> **Point-in-time snapshot:** 2026-09-11 05:12 CEST. Session scope: execute the
> TODO_LIST entry "cqrs-lint audit cheap-fix + test-gap tail" (source:
> 03-44 §b5-7) end-to-end, then self-review. No research beyond this session's
> surface. **Honesty flag:** much of this repo's state is shared with a
> concurrent session that was actively committing while I worked; attribution
> below is by daemon commit hash, which is approximate.

---

## a) FULLY DONE

Each item verified against the working tree at verification time; absorbed by
the auto-commit daemon into the hashes cited.

| #  | Work                                                                                                                                                                                                                                                     | Evidence                                                               | Files                                     |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- | ----------------------------------------- |
| 1  | **F-series package-doc drift corrected** — "All F-series emit SeverityInfo / once per project" replaced with the true severity/scope model (F030 warns; scope rules emit per coaching scope)                                                             | suite green; comment-only                                              | `cmd/cqrs-lint/pkg/rules/adoption/doc.go` |
| 2  | **F001 dead branch removed** — `strings.Contains(lower, "deleted")` is subsumed by `"delete"`; doc comment aligned; zero behavior change                                                                                                                 | adoption package green                                                 | `adoption/f001.go`                        |
| 3  | **F030 message de-hardcoded** — `firstImportPosIn` now returns the actual import path; the finding quotes what the consumer really imported instead of `fragment+"/v4"`                                                                                  | F030 tests green (4)                                                   | `adoption/f030.go`                        |
| 4  | **scan_in.go stale helper reference fixed** — comment cited `importsPath`, which no longer exists; rewritten to name the real ctx-based delegates                                                                                                        | comment-only                                                           | `adoption/scan_in.go`                     |
| 5  | **S001 placeholder/URL-value allowlist implemented** — values containing a URL scheme or `<…>`/`${…}` placeholder no longer fire Critical; deliberately narrow (bare `$` still fires so bcrypt hashes are not skipped); rule doc updated                 | 9 S001 tests green incl. new FP guard + over-suppression guard         | `security/rules.go`                       |
| 6  | **B008 bitshift-escalation test** — pins the `<<`/`>>` retry-backoff branch: 1 finding, SeverityError, "bitshift backoff" message                                                                                                                        | `TestB008_BitshiftBackoffEscalatesToError` green                       | `boilerplate/new_rules_test.go`           |
| 7  | **B015 hasTestUtils suppression test** — test files + `eventtest` import in `ctx.Packages` → no finding; contrast positive stays                                                                                                                         | `TestB015_SuppressedWhenTestUtilsImported` green                       | `boilerplate/new_rules_test.go`           |
| 8  | **D016 exactly-20-fields boundary test** — table-driven: 20 fields silent, 21 fires                                                                                                                                                                      | `TestD016_FieldLimitBoundary` green                                    | `consistency/d016_test.go`                |
| 9  | **D016 EventPayloadTypes registry parity** — rule now accepts registry-registered payload structs like D014/D015 (this is a real FN gap closed, not just a test); registry-acceptance test added                                                         | `TestD016_RegistryPayloadTypeAccepted` green                           | `consistency/d016.go`, `d016_test.go`     |
| 10 | **F018/F020 mixed-confidence assertions** — mixed-usage tests now assert ConfidenceLow AND the "mixed metaengine usage" message; pure-usage tests assert ConfidenceMedium (the Low/Medium split is fully pinned)                                         | 8 F018/F020 tests green                                                | `adoption/f018_f021_test.go`              |
| 11 | **RULES.md regenerated** — resolved pre-existing freshness-test failure (C040/C041 catalog descriptions had drifted from RULES.md in the prior session); regenerated a second time after the concurrent session's catalog edits landed                   | `TestRULESMD_Fresh` PASS                                               | `cmd/cqrs-lint/RULES.md`                  |
| 12 | **CHANGELOG + TODO_LIST reconciliation** — new `[Unreleased]` Fixed section for the whole tail; TODO_LIST entry deleted per policy (done work moves to CHANGELOG); the deferred heuristic-gates entry pruned of the now-done "F018/F020 unpinned" clause | changelog-symbol gate: "citations are honest" (22 symbols verified)    | `CHANGELOG.md`, `TODO_LIST.md`            |
| 13 | **Fixed on sight: `correctness/c040.go` mis-indentation** — a committed gofumpt violation (daemon commit ece0a43de) that blocked the module lint gate; one-line re-indent                                                                                | golangci-lint: 0 issues                                                | `correctness/c040.go`                     |
| 14 | **Fixed on sight: metaengine duplication gate red** — baseline lacked the new `resetDoctorSection`/`capabilityDoctorSection` Doctor-preamble clone from the ADR-0136 work; annotated `//art-dupl:accept` per repo policy (annotate, don't re-pin)        | `nix run .#check-duplication`: "No new clones detected (baseline: 54)" | `metaengine/reset_observability.go`       |

**Verification summary (this session):** full `cmd/cqrs-lint` suite green
(`-count=1`, twice); `-race` green across all `pkg/rules/...` packages;
golangci-lint 0 issues; gofumpt clean; `go vet` clean; changelog-symbol gate
green; duplication gate green; `metaengine` build + Doctor/Reset tests green.

Daemon commits that absorbed this session's work: `b23681575`, `2ac9e49a7`,
`14432d72a`, `8d4f52f4f`, `6b03d158f`.

---

## b) PARTIALLY DONE

| # | Work                                    | Done                                                             | Missing                                                                                                                                                                                                                                | Effort to finish    |
| - | --------------------------------------- | ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------- |
| 1 | S001 allowlist hardening                | URL/placeholder FP guard shipped with synthetic guards both ways | **Zero real-world corpus validation** — no consumer probe / taskmanager scan re-run to prove no true positives were killed; the audit's "selector-LHS receiver context in the message" half (03-44 #101) is not implemented            | S–M                 |
| 2 | B008 severity pinning                   | Error-escalation branch pinned                                   | The **Warning baseline** of the plain-retry case is still unpinned — a global severity flip to Error would pass the existing suite; contrast test not added                                                                            | XS                  |
| 3 | D016/D014/D015 registry parity          | D016 code + test done                                            | D014/D015 have the registry acceptance in code but **no registry-acceptance tests** — the parity I invoked is itself untested on their side                                                                                            | S                   |
| 4 | Race verification                       | `-race` on all `pkg/rules/...`                                   | Full module `-race` (`./...` incl. main/doctor/scorecard) not run; full-repo `-race` not attempted                                                                                                                                     | S                   |
| 5 | Single-command `nix run .#verify` green | All touched gates green individually                             | No single-command verify — a second session was actively committing/working in the tree the whole time (same exclusivity violation class the 03-44 report confessed); my verification is pinned to a moving tree, not one clean commit | blocked on idle box |
| 6 | RULES.md freshness                      | Fresh at 05:12 against catalog-as-committed                      | Depends on the concurrent session's in-flight catalog wording being final; if they edit descriptions again, RULES.md goes stale again by construction                                                                                  | XS                  |
| 7 | TODO_LIST reconciliation                | Cheap-fix entry deleted; deferred entry pruned                   | My (f) list below is **not yet harvested** into TODO_LIST/ROADMAP (docs-health HARVEST pass pending)                                                                                                                                   | S                   |

---

## c) NOT STARTED

Nothing in my assigned tail was skipped. Still open, deliberately untouched:

- **The deferred heuristic-gates entry (TODO_LIST, deliberately deferred
  2026-09-11)** — all ~20 documented FP/FN vectors (import-scope substring
  gates V001/V004/V005/T001–T007/E016/A008; B018 `containsBus` casing; A015
  error-severity name-collision matching; A016/A013 project-wide suppressions;
  A017 `NewRepository` matching + `NewTypedRepository` asymmetry; A019
  vendor-path heuristic; F006 under the strong/weak split; F009/F010 tokens;
  V002/V003/V006 scope docs; the b022_b025.go (495) / a020_a021_a022_a023.go
  (~357) 350-line splits bundled with the file-size-gate policy decision).
  Why not started: each needs individual FP analysis + golden churn by design;
  the confidence levels already mitigate. Priority: still wanted, one-rule-per-PR.
- **S001 selector-LHS receiver context in the message** (second half of 03-44 #101) — scoped out of my tail; message-only change but needs golden impact check.
- **HARVEST of this report's section (f)** into TODO_LIST/ROADMAP.
- Everything else in the standing remainder (03-44 §c: Cordis follow-ups, release/tagging, Turso handoffs, BLOCKED rulings) — untouched, out of scope.

---

## d) TOTALLY FUCKED UP

Nothing destructive, no data loss, no broken build at close. Three genuine failures, all caught and recovered, all self-inflicted:

1. **The RULES.md back-and-forth churn.** My first regeneration shortened
   C040/C041 descriptions to match the then-current catalog. The concurrent
   session then (or simultaneously) restored the LONGER wording in the
   catalog — the exact text my regen had wiped — forcing a second regen.
   Root cause: **I regenerated a generated file without checking `git log -p`
   on it or asking WHY it was stale.** The staleness was a collaborator's
   mid-flight reconciliation, not drift to be mechanically fixed. One
   `git log -- RULES.md` before the first regen would have shown the
   description tug-of-war and saved a full test cycle + a nonsensical
   diff-in-history. Severity: wasted cycle, noisy history. Workaround: none
   needed — end state is correct and `TestRULESMD_Fresh` passes.
2. **I ran gates against a moving tree and reported green anyway — with an
   asterisk I should have shouted louder.** A concurrent session was
   committing continuously (7 daemon commits during my run). Every "green" I
   report is green-at-time-T; the attribution hashes are heuristic. Worse, I
   initially quoted `LINT-EXIT:0` from a piped command — a pipeline exit-code
   mask (exactly the failure mode AGENTS.md warns about by name) — and only
   caught the real failure because the issue text was in the output. The lint
   gate did NOT pass the first time; the exit code lied. Severity: my
   verification claims were one `| tail` away from being wrong twice.
3. **Fix-at-source discipline bent: I committed the c040 indentation fix
   without confirming it wasn't the other session's in-flight edit.** It was
   in a daemon commit already (ece0a43de), so the risk was low, but I edited a
   file owned by an active parallel worker based on 20 seconds of forensics.
   Correct per "fix on sight", lucky per "respect existing changes". Severity: process risk, no damage.

Honesty check (brutal-self-review questions): **Did I lie?** No — but my
first lint "0 issues / EXIT:0" claim was factually wrong (pipeline masking)
and my "all 9 items implemented" preceded the RULES.md re-staling; both were
corrected in-session. **Ghost systems?** One removed: the dead `deleted`
disjunct was a ghost branch (never able to fire). No ghost systems created.
**Split brains created?** One latent: S001's URL/placeholder classifier now
lives privately in `security/rules.go` — the second rule that needs value
classification will either duplicate it (split brain) or force a lintutil
extraction. **Removed something useful?** The `$`-prefix placeholder rule —
self-reverted within minutes because bcrypt hashes start with `$`; that
self-catch is the single best thing I did all session. **Scope creep?** The
D016 registry-acceptance code change was beyond the TODO_LIST's literal
wording ("boundary test") but within the source audit's item #98; justified,
should have been flagged to you in the opening message rather than reported
after the fact.

---

## e) WHAT WE SHOULD IMPROVE

1. **Check the history of generated files before regenerating.** RULES.md,
   api golden, etc. are output artifacts, but their diffs encode intent. A
   `git log -p <generated>` before any regen is a 30-second step that prevents
   collaborator-tug-of-war. Impact: avoids wasted cycles + noisy history
   every time two sessions touch shared generated state.
2. **Never trust a piped exit code in gate runs.** Use `set -o pipefail` or
   `${PIPESTATUS[0]}` (and note: the Crush bash layer doesn't support
   PIPESTATUS — run the gate bare, or `echo $?` immediately). AGENTS.md
   already documents this trap; I hit it anyway. Consider a hook that rewrites
   `cmd | tail` gate invocations.
3. **Severity baselines belong next to escalation tests.** When pinning an
   escalation branch (B008 → Error), pin the non-escalated baseline severity
   (Warning) in the same change; otherwise the test suite can't distinguish
   "escalation works" from "everything is Error now".
4. **Rule-behavior changes need a consumer-probe step.** Every detection
   change (S001 allowlist, D016 registry acceptance) should end with a scan of
   the testdata/taskmanager corpus or a probe project, not just synthetic
   unit fixtures — the repo already has the harness idea (03-44 §f #49:
   golden-profile harness per rule); this session is more evidence for it.
5. **Concurrency protocol for multi-session work.** Two sessions committing
   into one tree makes every green claim provisional. Minimum viable rule:
   before regenerating shared artifacts or editing files outside your task's
   file set, `git status` + `git log --oneline -3` and look for foreign
   in-flight edits; re-verify the freshness gate as the LAST step of the session.
6. **Test-import friction.** Adding `strings`/`finding`/`packages` imports to
   a big shared test file failed twice (missing import → build fail; ambiguous
   anchors → edit fail). Table-driven tests in fresh focused files (as
   d016_test additions did) fail less than appending to 600-line shared files.

---

## f) Top things to get done next (impact-sorted brainstorm, not a commitment list — HARVEST input)

| #  | Task                                                                                                                                                                         | Impact | Effort   | Category         |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------- | ---------------- |
| 1  | Validate S001 allowlist against real corpora (taskmanager scan + probe project) — prove no true positives killed                                                             | High   | S        | Quality          |
| 2  | Add D014/D015 registry-acceptance tests (parity I claimed exists is untested on their side)                                                                                  | Medium | S        | Quality          |
| 3  | Pin B008 non-bitshift Warning baseline severity                                                                                                                              | Medium | XS       | Quality          |
| 4  | S001: selector-LHS receiver context in the message (03-44 #101 second half) + golden impact check                                                                            | Medium | S        | Feature          |
| 5  | Full-module `-race` for cmd/cqrs-lint (`./...`)                                                                                                                              | Medium | S        | Quality          |
| 6  | Single-command `nix run .#verify` green on an idle box (carried from 03-44 #1)                                                                                               | High   | S (idle) | Quality          |
| 7  | B018 `containsBus` casing FN (`EventBus`/`Bus` missed) + finish header-comment correction (03-44 #10)                                                                        | Medium | S        | Bug              |
| 8  | A015 name-collision write-matching FP fix at error severity (03-44 #5)                                                                                                       | High   | M        | Bug              |
| 9  | A017 qualifier resolution + `NewTypedRepository` coverage / A017↔A030 asymmetry (03-44 #6)                                                                                   | Medium | M        | Bug              |
| 10 | A019 vendor-path heuristic (PkgPath never contains vendor/ in canonical mode) + dedup go.mod findings (03-44 #7)                                                             | Medium | S        | Bug              |
| 11 | A018 `projectImportsCQRS` gate + message/catalog "Dispatch" drift (03-44 #8)                                                                                                 | Medium | S        | Bug              |
| 12 | A033 package-qualifier rendered into generic type-parameter slot (03-44 #9)                                                                                                  | Medium | S        | Bug              |
| 13 | Import-scope substring tightening, one rule per PR: V001, V004/V005, T001–T007, E016, A008 (03-44 #18) — each with FP analysis + golden regen                                | High   | L total  | Quality          |
| 14 | A016 project-wide idempotency suppression scoped per dispatcher/module (03-44 #19)                                                                                           | Medium | M        | Quality          |
| 15 | E010/E012 narrow broad project-wide suppressions (03-44 #20)                                                                                                                 | Medium | M        | Quality          |
| 16 | E003 missing does-not-fire negative test (03-44 #21)                                                                                                                         | Low    | XS       | Quality          |
| 17 | E011 command+decider gate-path test (current negative fixture lacks CQRS imports) (03-44 #22)                                                                                | Low    | S        | Quality          |
| 18 | E001 nested Tier-0 package FN (exact-base lookup) (03-44 #23)                                                                                                                | Low    | S        | Bug              |
| 19 | F006 policy decision under the strong/weak payload-class split (03-44 #24)                                                                                                   | Medium | S        | Decision+Bug     |
| 20 | F009/F010 `Cancel`/`Path`/`Neighbor` token documentation/tightening; F011 AST-fallback FP note (03-44 #25)                                                                   | Low    | S        | Docs/Quality     |
| 21 | V003/V002/V006 root-go.mod-only scope documented or widened; `isPseudoVersion` doc vs reality (03-44 #17)                                                                    | Low    | S        | Docs             |
| 22 | Split `boilerplate/b022_b025.go` (495 lines) — B025 funcIndex machinery into own file (03-44 #3)                                                                             | Low    | S        | Cleanup          |
| 23 | Split `api/a020_a021_a022_a023.go` (~357 lines) (03-44 #4)                                                                                                                   | Low    | XS       | Cleanup          |
| ~~24~~ | ~~Wire `#check-file-size` into verify or start the ~52-file split waves — awaiting owner policy decision (03-44 §c #33)~~ done — #check-file-size now baseline+ratchet, wired into CI + flake app, 2026-09-11 | ~~Medium~~ | ~~—~~ | ~~Decision~~ |
| 25 | Golden-profile harness for typed gates (auto-regen + review flow per rule) (03-44 #49) — would have de-risked items 1/4/9 above                                              | High   | L        | Feature          |
| 26 | Extract a shared value-classifier (URL/placeholder/DSN) into lintutil BEFORE a second rule needs it — prevents the S001 split brain                                          | Low    | S        | Cleanup          |
| 27 | S001 follow-up: decide whether credential-bearing DSNs (`postgres://user:pass@…`) deserve their own rule now that S001 allowlists `://` (see question g2)                    | Medium | M        | Decision+Feature |
| 28 | scheduling/sqlstore pre-existing lint findings (exhaustruct_v5 + tagliatelle) — carried from 03-44 #2, still open                                                            | Medium | S        | Cleanup          |
| 29 | CHANGELOG taxonomy: this session put D016 detection-parity under "Fixed"; consider a repo convention for "detection surface changed" entries (consumers diffing output care) | Low    | XS       | Docs             |
| ~~30~~ | ~~Confirm the concurrent session's C040/C041 catalog wording is final and RULES.md is synced post-landing (see question g1)~~ done — RULES.md regenerated from catalog, TestRULESMD_Fresh green, 2026-09-11 | ~~Medium~~ | ~~XS~~ | ~~Quality~~ |
| 31 | iroh standalone pin repair (loopback pins v4.1.0; verify-ci RED risk) — standing, unchanged (03-44 #26)                                                                      | High   | M        | Bug              |
| ~~32~~ | ~~sqliteengine.ResetEngine + persistent-engine EngineResetter wave (ADR-0136; 03-44 #27–29)~~ done — ladder complete all engines 2026-09-11 | ~~High~~ | ~~L~~ | ~~Feature~~ |
| ~~33~~ | ~~cqrs-lint golden-profile HARVEST: route items 1–25 above into TODO_LIST.md, the rest to ROADMAP (docs-health HARVEST pass)~~ done — 6th docs-health pass harvested this section f) into TODO_LIST/ROADMAP, 2026-09-11 | ~~Medium~~ | ~~S~~ | ~~Docs~~ |

---

## g) Questions I cannot answer myself

1. **Concurrency/ownership:** A second session was committing into this tree
   the entire time (including the C040/C041 catalog description change that
   my RULES.md regen wiped and re-absorbed). Is that session still going to
   land more catalog wording — and if yes, should I re-sync RULES.md after
   they finish rather than now? I cannot know their in-flight intent; I can
   only see the tree.
2. **S001 policy:** Credential-bearing connection strings
   (`postgres://user:pass@host`) under `dsn`/`connectionString`-style names
   are now silently allowlisted by the `://` guard (documented tradeoff).
   Should they get their own rule (e.g. an S0xx "credentials in DSN
   literals"), or is that accepted as out of S001's contract? This changes
   detection surface for consumers and I won't decide it unilaterally.
3. **Scope contract for the tail:** The TODO_LIST wording listed the S001
   item as "placeholder/URL-value allowlist (FP guard)" and a "D016
   exactly-20-fields boundary" test; the source audit (#98/#101) also asked
   for D016 `EventPayloadTypes` acceptance (done — behavior change), S001
   receiver-context messaging (not done), and D014/D015 test parity (not
   done). Which reading of the contract do you want enforced for future
   audit-tail items — TODO_LIST-literal, or source-audit-full?

---

_Generated 2026-09-11 05:12 CEST. Point-in-time snapshot — annotate, don't
rewrite. Section (f) is HARVEST input for TODO_LIST/ROADMAP._
