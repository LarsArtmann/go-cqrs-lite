# Status Report — Issue-#20 Closeout: Three Releases Live, Release Tooling Hardened

**Date:** 2026-09-11 01:47 CEST
**Session scope:** The 6-item tooling/release TODO batch (cqrs-bench stub, retract v4.8.0, tag-release.sh hardening, version-reporting unification, cqrs-upgrade CI dogfood, badger data-loss review) — executed end-to-end, tagged, pushed, and proxy-verified.
**HEAD at report time:** `e37642adf` "fix(release-tooling): close issue-#20 class end-to-end" (committed 01:38:34, tree clean).
**Verification state:** All three new tags are PUSHED and the proxy serves them; `@latest` resolution verified for all three acceptance criteria (see Appendix A).

**Format note:** the status-report skill's canonical output is a styled HTML dashboard; the explicit `.md` instruction (and the repo's `docs/status/*.md` convention) wins here.

---

## a) FULLY DONE

Each item is verifiably complete with evidence.

1. **`cmd/cqrs-bench` dead-path deprecation stub — shipped, pushed, live.**
   One-off detached-worktree commit (never merged), annotated tag
   `cmd/cqrs-bench/v0.1.1`: suffix-less go.mod, zero requires, loud-failure
   stub pointing at the `/v4` install command. Evidence: stub built
   standalone (`GOWORK=off go build`, vet OK, exit 1 with correct message);
   post-push `go install …/cmd/cqrs-bench@latest` now downloads **v0.1.1**
   and prints "cqrs-bench has moved." — the ancient v0.1.0 binary is no
   longer what the dead path serves.

2. **`cmd/cqrs-lint/v4.8.0` retracted for real — `v4.10.1` cut, pushed, live.**
   The retract directive had been sitting inert on master (a retract only
   exists for consumers once a tag carries it). Cut `v4.10.1` from a
   detached worktree via `tag-release.sh` (const bumped to 4.10.1 inside
   the tag, standalone build verified, `retract v4.8.0` confirmed in the
   tagged go.mod). Evidence post-push: proxy serves v4.10.1 (attempt 1),
   clean-dir `go install …/cmd/cqrs-lint/v4@v4.10.1` succeeds, `--help`
   exits 0, and `go list -m …@latest` returns **v4.10.1** — v4.8.0 is
   skipped by resolution.

3. **`metaengine/badgerengine` v4.0.0–v4.1.0 retracted (restart-seeding data
   loss) — `v4.2.1` cut, pushed, live.** Retract block with reason comment
   (survives tidy; pkg.go.dev will display it). Evidence: proxy serves
   v4.2.1; `go list -m …@latest` returns **v4.2.1** (retracted line
   skipped). No code changes vs v4.2.0 (go.mod/go.sum only).

4. **Badger data-loss retrospective written.** ADR-0118 incident addendum:
   window bounded (v4.0.0–v4.1.0 seeded only the log prefix; the v4.1.0
   comment already claimed all four — a lying comment), fix timeline table
   (introduced 08-06 → full seeding 09-06 → v4.2.0 09-08 → retracts
   09-11), consumer audit (zero repo-internal data-path consumers; external
   adoption unknown but unlikely), and the lesson (a guarantee in a comment
   is not a guarantee until a close/reopen test proves it). ADR line 70
   corrected to scope the guarantee to v4.2.0+.

5. **tag-release.sh hardened: `--audit`, binary smoke probe, advisory
   pre-flight.** (a) `--audit` replays the path-vs-tag guard over every tag
   of every module — one-shot run: **1078 tags checked, 24 historical
   violations, 2 skipped**, all in known-dead paths (cqrs-lint v4.2.0–v4.7.0,
   cqrs-bench v4.2.0, example/* v3–v4, event/v4/eventtest v0.x). (b)
   `--smoke` now follows the proxy check with a clean-dir
   `go install module@version` + run for main-package modules — the probe
   class that catches a poisoned tag. (c) Non-mutating `pin-sweep --check`
   runs as an advisory pre-flight (a full sweep would mutate 80+ unrelated
   go.mods — wrong scope for a single-module cut). (d) Guard logic
   extracted into `path_matches_major`, shared by the release flow and the
   audit — one implementation, no drift.

6. **`scripts/test-tag-release.sh` — 10 fixture-repo smoke tests, all
   passing.** Audit FAIL/OK paths, guard rejection (with "no tag created"
   assertion), pre-push `--smoke` error, usage guard. Hermetic: local bare
   origin, `tag.gpgSign`/`forceSignAnnotated` stripped. No network.

7. **Version-reporting unified: hand-maintained const deleted (decision:
   buildinfo primary).** `resolvedVersion()` reports the toolchain-embedded
   version (`go install …@vX.Y.Z` → the real tag), falls back to
   `dev-<sha>[-dirty]` from stamped VCS revision, then `dev` (+ injected
   commit/date under Nix). tag-release.sh's const-bump block removed — a
   cut can no longer mutate source, killing the v4.8.0 sed-poison class
   structurally. TestVersionMatchesLatestTag retired;
   `bump-cqrs-lint.sh` reduced to tidy + vendorHash. New tests:
   TestVersionResolved, TestBuildInfoSetting, reworked TestVersionFormat.
   Evidence: full cqrs-lint module suite green (27s). **On master** —
   deliberately NOT in v4.10.1 (that tag was cut from pre-refactor HEAD so
   the retract could ship immediately).

8. **cqrs-upgrade CI dogfood job wired.** `check-upgrade-dogfood` flake app
   + nightly `sentinel.yml` job (`--workspace --dry-run --strict .`).
   Evidence: local full run — 83 modules, exit 0, **0 v5-removed API
   findings**, 4m21s network-bound. All the flag work (--json/--workspace/
   --to/--strict/x-mod) had shipped earlier; this was the last open piece.

9. **Docs made truthful in the same wave.** CHANGELOG (3 new [Unreleased]
   sections; honesty gate green — "18 pkg.Symbol citations verified"),
   TODO_LIST (all 6 items `[x]` with evidence), gotchas-module-management
   (new "retracts are inert until a tag carries them" bullet incl. audit/
   smoke mechanics), cmd/cqrs-lint CONTRIBUTING release checklist
   (no-const flow), ADR-0118 (above).

10. **Everything committed and pushed.** Working tree was absorbed into
    `e37642adf` at 01:38:34; all three tags verified on origin via
    `git ls-remote`.

---

## b) PARTIALLY DONE

1. **Buildinfo version reporting is on master but in NO published tag yet.**
   v4.10.1 (live) still reports via the old const mechanism (correctly —
   the tagger bumped the const inside the tag). The refactor ships with
   the next cmd/cqrs-lint tag (v4.10.2+). Effort: S — tag at the next
   natural release.

2. **The nightly dogfood job has never run in CI itself.** The flake app
   evals and the exact command passes locally (4m21s), but the first
   sentinel run on GitHub Actions (network topology, GOPROXY behavior,
   20-min timeout) is unobserved. Effort: S — watch the next nightly run.

3. **`--audit` leaves 24 historical violations red forever.** They are all
   in dead paths that cannot be "fixed" (invisible tags can't be
   un-invisible; the modules involved are dead or v0-era). The audit
   currently exits 1 on them, which makes it a one-shot tool, not a CI
   leg. A known-violations baseline (like art-dupl's) would let `--audit
   --check` gate CI for NEW violations only. Effort: S/M.

4. **The `--smoke` run-probe is weak for non-cobra CLIs.** Install is a
   hard gate (correct), but `--help` exit != 0 only warns. cqrs-bench's
   flag-based CLI semantics were never probed end-to-end via the tool
   (verified manually instead). A per-binary probe-command table would
   make the run check deterministic. Effort: S.

5. **`test-tag-release.sh` Test 5 is weak.** It claims "rejects a non-main
   module cleanly" but only exercises the `--smoke` usage guard (missing
   version arg), not the no-main-package skip path. Effort: S.

---

## c) NOT STARTED

1. **Dead-path example modules undecided.** example/taskmanager and
   example/getting-started still carry suffix-less module paths with
   permanently-invisible v3/v4 tags; event/v4/eventtest has invisible v0.x
   tags on a /v4 path. Options (delete module / leave + document /
   re-path) not weighed. Not started — needs a product decision.

2. **`docs-health` HARVEST of this report's section (f).** The 50-item
   brainstorm below is a timestamped file until HARVEST routes it into
   TODO_LIST.md / ROADMAP.md. Not started (waiting for your go, per
   "then wait for instructions").

3. **Post-wave pin sweep.** v4.10.1 and v4.2.1 are now the newest tags;
   sibling pins across the 84 modules may be stale against them
   (`pin-sweep --check` advisory passed pre-push but new tags change the
   picture). Not started — one command.

4. **CI green confirmation for `e37642adf`.** The commit touched CI
   (sentinel.yml), flake.nix, and cqrs-lint sources; the push-triggered
   run had just started at report time. Not observed yet.

5. **GitHub Releases for the three tags.** `scripts/create-github-releases.sh`
   exists; no release objects were created for v0.1.1 / v4.10.1 / v4.2.1.
   Not started.

---

## d) TOTALLY FUCKED UP

Radical honesty. Nothing here is hidden behind "mostly fine".

1. **My audit shipped two false-result bugs before it was correct — both
   caught only because I re-ran and sanity-checked the counts.**
   - Bug 1: `git show … || true | awk` — shell precedence short-circuited
     awk, so the audit "read" the RAW go.mod as the module path and
     reported 1302 FAILs (more FAILs than total tags — the tell). Root
     cause: `a || b | c` parses as `a || (b | c)`.
   - Bug 2: the `<dir>/*` glob matched nested-module tags (storage/*
     matches storage/memory/v4.5.0), inflating counts and cross-checking
     tags against the wrong module. Root cause: fnmatch `*` crosses `/`.
   - **Severity of the near-miss:** an audit tool whose whole job is
     producing TRUSTWORTHY one-shot verdicts first produced two wrong
     verdicts (vacuous-red, then false-red). The set -e/pipefail
     interaction (command-substitution subshell aborting with 128) was a
     third latent trap. Lesson recorded: verify tool output against
     independent counts (FAIL count ≤ tag count) before trusting a
     verdict table.

2. **I nearly shipped a lie in two docs.** First drafts of CHANGELOG and
   TODO_LIST said the buildinfo change "rides the v4.10.1 train" — false:
   v4.10.1 was cut from pre-refactor HEAD by design. Caught it in
   self-review before the daemon absorbed the tree; corrected both files
   to "on master, rides the next tag". If the daemon had committed 10
   minutes earlier, the lie would be in master's history.

3. **The repo carried an inert security-adjacent fix for ~10 days.** The
   `retract v4.8.0` directive was committed to master on ~2026-09-01 but
   never shipped in a tag — everyone (including the TODO item's XS
   estimate) treated "add the directive" as the task, when the task was
   "make fresh consumers stop resolving v4.8.0". The TODO text was
   actually correct ("a retract stops fresh consumers…") and the gap was
   visible; nobody closed the loop until now. This is exactly the
   status-report-is-not-truth class AGENTS.md warns about.

4. **badgerengine v4.1.0 shipped with a comment that lied.** The doc
   comment claimed four-prefix restart seeding; the code seeded only the
   log. The ADR (line 70) claimed the same. A reviewer reading either
   would have approved the release. This is the most dangerous class in
   the repo: comments asserting guarantees the tests don't enforce. (The
   fix shipped in v4.2.0; the ADR now carries the incident addendum.)

5. **`check-file-size` is red on master — pre-existing, unrelated to this
   session, and evidently ungated in CI.** 10 production .go files exceed
   the 350-line limit (scorecard_render.go 444, suppression/parser.go 540,
   boilerplate/b022_b025.go 494, …). Either the gate is decorative or CI
   is not running it. My changes added +1 line to an already-violating
   file; no new violations introduced.

6. **Minor, owned:** (a) my first badger-smoke invocation ran the script
   from /tmp and failed on `git rev-parse` — harness error, re-run from
   the repo; (b) the bench dead-path verification printed "run exit=0"
   because the exit code was measured through `head` (pipeline masking —
   the exact AGENTS.md anti-pattern); the stub's exit-1 was verified
   properly earlier against the built binary.

---

## e) WHAT WE SHOULD IMPROVE

1. **"Directive on master" must never be conflated with "shipped".** The
   retract gap (d3) is a process hole, not a one-off. Suggested fix: the
   release checklist gains a mechanical step — after any go.mod retract,
   `go list -m module@latest` from a clean dir is the acceptance test, and
   it fails while the directive is untagged. Consider a tiny gate:
   `check-retracts-shipped.sh` comparing master's retract directives
   against each module's newest tag.

2. **New tooling needs a "first verdict is suspect" ritual.** Both audit
   bugs produced confident-looking wrong tables. Suggested fix: every new
   check script must assert at least one internal invariant (counts vs an
   independent source — here, `git tag -l | wc -l`) and fail loudly when
   violated. The audit now effectively does this via the summary line, but
   make it explicit practice for check-* scripts.

3. **Comments that state guarantees should cite the test that enforces
   them.** badgerengine's lying comment is the canonical case. Suggested
   fix: a convention (lint rule or review checklist): any comment of the
   form "X is seeded/guaranteed/validated" must name the test
   (`restart_safety_test.go`) in the same breath.

4. **The smoke run-probe should be per-binary configurable.** `--help`
   exit-code semantics differ across CLIs (cobra 0, flag pkg 0, subcommand
   CLIs 2). Suggested fix: an optional `scripts/smoke-probes.txt` mapping
   module → probe command; default stays `--help`-with-warning.

5. **Status-report skill format vs repo convention.** The skill mandates
   an HTML dashboard; every report in `docs/status/` is `.md`. Suggested
   fix: either switch the skill default to `.md` (honor the repo) or
   generate both. Flagged per the skill's own override rule; this report
   is `.md` per your instruction.

6. **Timestamped reports rot — HARVEST immediately.** Section (f) below is
   the primary input for TODO_LIST/ROADMAP. The last reports fed harvest;
   this one should too, right after you read it.

7. **Concurrent-session hygiene worked, but by luck.** The detached-
   worktree tagging pattern (sanctioned by gotchas §2b) let me cut tags
   while another session held the tree dirty. Suggested fix: a tiny
   `scripts/cut-tag-worktree.sh <module> <version> <desc>` wrapper so the
   dance (worktree → cut → remove) isn't re-typed per release.

---

## Self-Review (brutal-self-review, all 11 questions)

1. **What did I forget?** Nothing from the 6 items — all shipped. Forgotten
   until late: (a) that the retract needed a TAG, not just the directive
   (research caught it before execution); (b) that v4.10.1 would NOT carry
   the buildinfo refactor when I chose to cut from HEAD — I wrote docs
   claiming otherwise and had to correct them; (c) the `go list -m
   @latest` acceptance check — the single most convincing verification —
   was an afterthought at the very end instead of the first gate.
2. **What is something stupid we do anyway?** We keep hand-maintained
   mirrors of mechanically-derivable facts (the version const was one;
   the 24 dead-path tags are another — invisible tags nobody can consume
   still cost audit noise). Also: CI has a check-file-size app that is red
   on master and apparently not wired as a gate — decorative red is worse
   than no gate.
3. **What could I have done better?** Verified `@latest` resolution
   immediately after each tag push instead of at the end; built the audit
   against a fixture repo BEFORE pointing it at 1217 real tags (the two
   false-verdict bugs would have died in fixtures); written the CHANGELOG/
   TODO text only after the release mechanics were final.
4. **What could I still improve?** The `--audit` known-violations baseline
   (turn a one-shot into a CI leg); per-binary smoke probes; HARVEST of
   section (f); a wrapper for the worktree tag dance.
5. **Did I lie to you?** Two doc statements were false in first draft
   ("rides the v4.10.1 train") — caught and corrected before commit. The
   final report's claims are all backed by commands shown in Appendix A.
   One imprecise moment mid-session: I reported "exit=1" for the dogfood
   run which was actually my grep's exit code, not the tool's — corrected
   to exit=0/4m21s after reading the job output.
6. **How can we be less stupid?** Make the acceptance test the FIRST thing
   written (here: `go list -m @latest` for retracts; "FAIL count ≤ tag
   count" for audits). Fixtures before real repos for any script that
   renders verdicts. Never let a doc claim ship before the mechanism it
   describes is merged.
7. **Ghost systems?** One candidate: `check-file-size` — an app that fails
   on master with no CI leg and no baseline = a ghost gate. Either wire it
   (with a baseline for the 10 violators) or delete it. Also
   `event/v4/eventtest`'s invisible v0.x tags are ghost artifacts: they
   can never be consumed; the only value left in them is historical.
8. **Scope creep trap?** Watched it twice: `pin-sweep --no-build` in the
   release pre-flight (rejected — mutating sweep is wrong scope; used
   non-mutating `--check` advisory instead) and the temptation to
   "fix" the 24 historical audit violations (resisted — they are
   unfixable-by-design; documented instead).
9. **Did we remove something useful?** The version const and its
   lockstep test: no — the const's only job (report the release version)
   is now done better by buildinfo, and the test's job (catch drift) is
   void when nothing can drift. The tag-release bump block: same. The
   `gci`-style risk of removing the bump block's protections is covered
   by the build gate + install probe.
10. **Split brains?** Two small ones to watch: (a) v4.10.1 (const-based
    version) vs master (buildinfo-based) — intentional, converges at the
    next tag; (b) `bump-cqrs-lint.sh` still accepts a version argument
    that does almost nothing now (informational only) — mildly
    misleading, kept for muscle memory; candidate for deletion.
    Third: SKILL.md's benchmarking section documents building
    `./cmd/cqrs-bench` from source — still correct (the /v4 module is the
    real tool); the stub lives only on the dead install path. No action.
11. **Tests?** cqrs-lint module suite green post-refactor; 10 new script
    smoke tests; dogfood run green. Gaps: no test asserts the audit's
    own counting invariants; the dogfood job is unobserved in CI; the
    smoke run-probe has no fixture with a deliberately-broken tag
    (the v4.8.0 class) because building a proxy-served broken tag in a
    test is not possible offline — the install probe path is only
    covered by its error branch in fixtures.

---

## f) Top 50 things we should get done next

Brainstorm, ranked by impact. Items marked ✅-harvest are already in
TODO_LIST.md; the rest are candidates for docs-health HARVEST routing
(TODO_LIST vs ROADMAP per its rigor rules).

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Confirm CI green on `e37642adf` (first run with new tag-release/audit/sentinel) | Critical | S | Quality |
| 2 | Watch the first nightly `upgrade-dogfood` sentinel run; fix env gaps if any | High | S | Quality |
| 3 | Run `scripts/pin-sweep.sh --check` post-push; sweep stale pins against v4.10.1/v4.2.1 | High | S | Cleanup |
| 4 | HARVEST this report's (f) into TODO_LIST/ROADMAP (docs-health) | High | S | Documentation |
| 5 | Tag cmd/cqrs-lint v4.10.2 (or v4.11.0) shipping the buildinfo version reporting; verify installed binary prints the real tag | High | S | Feature |
| 6 | Add `check-retracts-shipped.sh`: fail when a master go.mod retract is absent from the module's newest tag | High | S | Quality |
| 7 | Add `--baseline` mode to `tag-release.sh --audit` (known 24 violations; gate NEW ones in CI) | High | M | Quality |
| 8 | Create GitHub Releases for v0.1.1 / v4.10.1 / v4.2.1 via `create-github-releases.sh` | Medium | S | Cleanup |
| 9 | Decide dead-path example modules (taskmanager/getting-started suffix-less go.mod): re-path to /v4, or delete, or document as v0-only | High | M | Cleanup |
| 10 | Decide event/v4/eventtest invisible v0.x tags: document as dead in modules.md + pin-sweep note (already partially noted) | Medium | S | Documentation |
| 11 | `scripts/smoke-probes.txt`: per-binary probe command for `--smoke` run check | Medium | S | Feature |
| 12 | Strengthen test-tag-release.sh Test 5 (cover the no-main-package skip path with a fixture) | Medium | S | Quality |
| 13 | Worktree cut wrapper `scripts/cut-tag-worktree.sh` (formalize the dance used three times today) | Medium | S | Feature |
| 14 | Fix `bench-deadpath` oddity: `go install …cmd/cqrs-bench@latest` downloads root module `v1.7.1` — investigate why the stub's zip pulls the root | Low | S | Bug |
| 15 | Wire or delete `check-file-size` (red on master: 10 files; decorative gates erode trust) — add baseline for violators then gate | High | M | Quality |
| 16 | Shrink the 10 over-limit production files (suppression/parser.go 540, boilerplate/b022_b025.go 494, scorecard_render.go 444, …) | Medium | M/L | Cleanup |
| 17 | Convention: guarantee-comments must cite the enforcing test; add to review checklist + consider a cqrs-lint rule | Medium | M | Quality |
| 18 | cqrs-lint rule candidate: comment claims "X is seeded/validated" without a test reference (R&D) | Low | L | Feature |
| 19 | Run `nix run .#verify` exclusively when tree is quiet (last full verify predates today's wave) | High | M | Quality |
| 20 | Delete or repurpose `bump-cqrs-lint.sh`'s vestigial version argument (post-buildinfo) | Low | S | Cleanup |
| 21 | Record today's audit run (1078/24/2) as a dated baseline doc next to the audit code | Medium | S | Documentation |
| 22 | Add `--json` output to `tag-release.sh --audit` for machine consumption | Low | S | Feature |
| 23 | Prune stale worktrees (`git worktree prune` for the 4 prunable entries from old sessions) | Low | S | Cleanup |
| 24 | Consider retracting example/* v3/v4 invisible tags' *modules*… not possible — instead document in faq.md why old example tags 404 | Low | S | Documentation |
| 25 | Add the `@latest` acceptance check as the documented final step in CONTRIBUTING release flow (already in tag-release output; mirror in cmd/cqrs-lint/CONTRIBUTING) | Medium | S | Documentation |
| 26 | Verify pkg.go.dev displays the badger retract reason comment; screenshot/link in ADR-0118 addendum | Low | S | Documentation |
| 27 | Fold "retract needs a tag" into the release checklist in CONTRIBUTING.md root (gotchas has it; checklist doesn't) | Medium | S | Documentation |
| 28 | Run `nix run .#verify-ci` (GOWORK=off per-module matrix) after today's wave — catches pin breaks CI-per-module sees | High | L | Quality |
| 29 | Grep repo for other master-only "pending release" directives: unfinished retracts, unbumped pins awaiting tags (`pin-sweep --check` covers pins; retracts now covered by #6) | Medium | S | Quality |
| 30 | Add the three new tags to docs/agents/module-map.md internal notes (badger retraction note) | Low | S | Documentation |
| 31 | benchmark-regression gate: run `./scripts/benchmark-regression.sh` after the wave (timing paths untouched, but cheap insurance) | Low | M | Quality |
| 32 | Improve `emitJSON` in cqrs-upgrade: add `--strict` violation summary field at document level for CI dashboards | Low | S | Feature |
| 33 | cqrs-upgrade: `--probe-command` flag generalizing the smoke-probe idea for consumers | Low | M | Feature |
| 34 | Sentinel: alert when upgrade-dogfood reports >0 bumps available for 7+ days (signals repo pins rotting vs latest tags) | Medium | M | Feature |
| 35 | FAQ entry: "why does `go install …/cmd/cqrs-bench@latest` fail loudly?" (the stub) so users self-serve | Low | S | Documentation |
| 36 | Audit other modules for the badger class: KV engines whose restart seeding claims outrun their tests (bbolt/pebble spot-check) | High | M | Bug |
| 37 | restart-safety harness: extend to a shared enginetest close/reopen matrix test ALL engines run (badger lesson generalized) | High | L | Quality |
| 38 | ~~Inert-retract check~~ VERIFIED 2026-09-11 01:50: command/v4.10.0, query/v4.8.0, and storage/v4.9.0 all carry their retract directives — none inert. No action. | — | — | — |
| 39 | Audit `scripts/batch-release.sh` for consistency with the hardened tag-release.sh (may encode the pre-hardening flow: no guard/probe/audit) | High | M | Quality |
| 40 | Verify the LIVE `/v4` bench path was unaffected by the stub — VERIFIED 2026-09-11 01:50: `go list -m …/cmd/cqrs-bench/v4@latest` → v4.3.0. Done. | — | — | — |
| 41 | doc-check pass over the edited docs (ADR-0118, gotchas) — they're outside doc-check's file list, but link rot applies | Low | S | Documentation |
| 42 | Add `docs/status/README.md` index entry for this report (the dir has a README manifest) | Low | S | Documentation |
| 43 | Consider `tag-release.sh --audit --module <dir>` scoping flag for post-wave spot audits | Low | S | Feature |
| 44 | Telemetry nicety: audit summary line should include total tags seen vs checked (skipped breakdown by reason) | Low | S | Feature |
| 45 | Move the "one-off stub commit" recipe into a reference doc (currently only in commit messages + gotchas) | Low | S | Documentation |
| 46 | Re-run `nix run .#check-duplication` after the wave (tag-release.sh grew; shell dup gate may have opinions) | Medium | S | Quality |
| 47 | shfmt/treefmt gate on CI for scripts/: confirm test-tag-release.sh passes the pre-commit shfmt (it passed nix fmt locally) | Low | S | Quality |
| 48 | Add `TestResolvedVersionFromInstalled` style probe: a test that `go install`s the module from the proxy — nightly, not per-push (proxy-dependent) | Low | M | Quality |
| 49 | ROADMAP candidate: generalize the retract-publish flow into `tag-release.sh --retract-only <module>` (patch tag with only go.mod delta) | Low | M | Feature |
| 50 | Celebrate + archive: move superseded 2026-09-01/15 reports referencing these items to annotated-done per docs-health ANNOTATE | Low | S | Documentation |

---

## g) Three questions I cannot answer myself

1. **Are there any known external deployments of `metaengine/badgerengine`
   v4.0.0–v4.1.0** (Discord, client projects, colleagues)? The retrospective
   had to conclude "unknown but unlikely" — only you can bound the real
   consumer set, and the answer decides whether the retraction warrants a
   heads-up post/email beyond pkg.go.dev.

2. **For the dead-path example modules (taskmanager / getting-started with
   permanently-invisible v3/v4 tags): is the /v4 re-path worth doing, or are
   these examples v0-era frozen by policy?** I can implement either in one
   session; only you know whether example version lines are a support
   surface or throwaway teaching code.

3. **Should I be authorized to push release tags autonomously when a
   session's acceptance criteria are green** (as v4.10.1/v4.2.1/v0.1.1
   were), or should every push stay a manual human step? Today the push
   wait added ~an hour of latency to shipping retracts that consumers
   needed; I defaulted to not pushing per the safety rules and won't
   change that without your explicit policy.

---

## Appendix A — Verification log (evidence for section a)

```
# Tags on origin (git ls-remote)
cmd/cqrs-bench/v0.1.1: PUSHED          cmd/cqrs-lint/v4.10.1: PUSHED
metaengine/badgerengine/v4.2.1: PUSHED

# Proxy + install probe (scripts/tag-release.sh --smoke)
✓ proxy serves …/cmd/cqrs-lint/v4@v4.10.1 (attempt 1)
✓ installed cqrs-lint runs (--help exited 0)
✓ proxy serves …/metaengine/badgerengine/v4@v4.2.1 (attempt 1)

# Acceptance: @latest skips retracted versions / serves the stub
go list -m …/cmd/cqrs-lint/v4@latest        → v4.10.1   (v4.8.0 retracted)
go list -m …/metaengine/badgerengine/v4@latest → v4.2.1 (v4.0.0–v4.1.0 retracted)
go list -m …/cmd/cqrs-bench@latest          → v0.1.1    (stub; v0.1.0 superseded)
go list -m …/cmd/cqrs-bench/v4@latest       → v4.3.0    (live path unaffected by stub)

# Old retracts verified SHIPPED (not inert): command/v4.10.0, query/v4.8.0,
storage/v4.9.0 all carry their retract directives in the tagged go.mod.

# Local suites (pre-commit, env chain + -tags goexperiment.jsonv2)
cmd/cqrs-lint:  ok … 27.0s   (full module suite, incl. new version tests)
cmd/cqrs-upgrade: ok … 0.003s
scripts/test-tag-release.sh: 10/10 PASS
check-changelog-symbols.sh: ✓ honest (18 citations)
dogfood (local, pre-CI): 83 modules, exit 0, 0 findings, 4m21s
--audit (one-shot): 1078 checked / 24 violations / 2 skipped
nix fmt: 0 files changed (tree format-clean)
```

*Point-in-time snapshot — will go stale. Section (f) awaits docs-health
HARVEST. Report not committed by hand (no-commit rule); the auto-commit
daemon is expected to absorb it, as it did the working tree at 01:38:34.*
