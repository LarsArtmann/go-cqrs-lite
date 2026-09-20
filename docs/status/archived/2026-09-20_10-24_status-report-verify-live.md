> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** Superseded hours later: attempt 4 died in the 10:31 concurrent corruption (10-56 report), the tree was repaired, and the S03 record landed GREEN at 15:04 (16-39 report). §f items are done or tracked in [TODO_LIST.md](../../../TODO_LIST.md).

# Status Report: Publish-and-Prove Execution — W0 Verified, W2 Code Wave Landed, Composed Verify LIVE

**Date:** 2026-09-20 10:24 CEST · **Session:** 2026-09-19 22:44 → ongoing (reboot gap 04:15–08:39)
**Directive:** execute the SUPERB publish-and-prove plan end to end (`docs/planning/2026-09-19_22-34_SUPERB-publish-and-prove-pareto-plan.md`).

---

## a) FULLY DONE (all verified)

### W0 PUBLISH — the 1%→51% — complete to its acceptance bar

- **The 92-tag train was cut+pushed by a parallel session (21:08–21:45)** — this session found it already shipped (the plan's "0 published" §0 was stale), then **verified every tail item**:
  - 1314 tags local == 1314 remote; wave tags match the batch manifests; sampled tags carry zero local replaces, coherent pins.
  - Proxy: `@latest` resolves wave versions from a clean dir for 10 key modules; pre-warmed all 92.
  - **92 GitHub Releases created** (152 total) after extending `create-github-releases.sh` to match train-section headers (bounded-token match, newest-section-first, 09-08-precedent trimmed bodies) + `--dry-run`; 92/92 extract, bogus tags skip.
  - `pin-sweep --check --remote` green · `check-retracts-shipped` green · `--audit --baseline` 0 NEW (CI leg already existed) · clean-dir `go list -m @latest` acceptance green.
  - **Smoke 91/92 green**; taskmanager proxy-served + dependency-green (its full install was starved 4× by host contention, never by a tag defect).
- **Post-wave integrity repairs:** 13 go.sums completed (missing claiming/sqliteengine `/go.mod` hashes — the api-stability tidy gate went red→green), typedfixture re-pinned (the REAL cause of verify attempts 1–3's cqrs-lint failures), `scheduling/engine` registered in flake testModules (creation gap; sync gate was failing every code commit), 4 committed binaries untracked + ignored (5.9–27MB).
- **T09/T12 recorded:** consumer-propagation handoff (CV, go-taskqueue, PapDashboard, go-graph-rag, go-localsync) + 16 release rows struck with evidence in TODO_LIST.

### W1 (partial) + W2 code wave

- **T16:** api-stability fully green (incl. foreign `readme_claims_test.go`, TestEvery, tidy gate).
- **T26 (3-session ask):** `wait-for-quiet.sh` + `can-run-composed-gate.sh` (both mutation-tested via planted fixtures — the first drafts' vacuous passes were caught and rebuilt with honored env hooks), `#verify` `-p 4` cap, cqrs-lint typed-fixture helpers fail-loud.
- **T20:** `check-canonical-facts.sh` — derives go.mod count / module-map census / recipes-catalog size from the repo; **caught the 80→81 recipes drift on first real run**; wired into nightly.
- **T19:** `check-doc-annotations.sh` — marker-OR-banner for archived reports, 1128-entry conscious baseline, live-drift warnings; wired into nightly.
- **T27 (P0):** camelCase planned-column pushdown FIXED in `metaengine.ExtractFields` (tag-OR-name matching + acronym-aware snake_case; the naive splitter's `parent_i_d` caught before shipping; 3 parity tests; the CRM workaround is deletable on its next bump).
- **T28:** ctx-scoped transactions ported to **pg/mysql/duckdb engines** (engine-global `activeTx` → ctx marker, sqliteengine 22ab7b218 pattern): PG isolation test validated live (testcontainer), MySQL build+vet green (live-gated on `MYSQL_TEST_DSN`), DuckDB full suite green 74s (stress companion deliberately not ported — go-duckdb C-level prepare convoy, documented).
- **T22:** `# Experimental` doc.go stamps across all 17 metaengine-family modules (pkg.go.dev renders the heading); stray package comments consolidated.
- **T35a:** doc-check repo-root relative-path regression pin + CHANGELOG Fixed + gotcha.
- **T29-docs:** queue README MySQL quickstart + conformance doc names all 3 engines.
- **T44-selftest:** restore-depguard `--self-test` (3 mutation legs via honored `RD_ROOT`).
- **T47/T48:** module-map census to all 95 go.mods (scripted-diff verified) + TODO `[x]` sweep (34 blocks deleted per header policy; 104 open rows remain).
- **Gates all green at session close:** check-doc-links 0 broken/786 targets · doc-check 1223 refs/50 packages (2 known ambiguous-alias advisories) · changelog-symbols honest · canonical-facts clean · doc-annotations clean · api-stability green.
- **Session reports:** `docs/status/2026-09-20_09-40_publish-and-prove-execution.md` (full a–g) + README ledger entry; all work pushed (`3afefc076` at last push, later commits pushed as landed).

## b) IN FLIGHT RIGHT NOW

- **Composed `#verify` attempt 4 is RUNNING** (supervisor-armed at the 10:0x load dip): verify-docs ✓, Module Coverage ✓, Build ✓, Vet ✓, **Test phase FULLY GREEN (245 pkgs ok — the phase that killed attempts 1–3)**, now in the **Race phase** (8+ min in, `-p 4` cap active, zero FAIL lines). Remaining phases after: lint, arch, modsums, lint-config, css, duplication, turso-version, templ, bench-gate, coverage, api-stability, error-taxonomy, doc-check. Supervisor records `VERIFY4-EXIT`; **S03 record lands the moment it goes green** (date/commit/durations into the TODO composed-verify row).

## c) QUEUED (sequenced behind verify / owner)

1. **T13/T14/T15** (same window): `#load-sweep`, `benchmark-regression.sh --save` (1.27+claimkit+engines provenance), `#verify-ci` + verify-docs e2e + S03 acceptance record.
2. Opportunistic: taskmanager full-install smoke; T29's PG `-race -count=2` leg.
3. W2 slice remainders: T23–T25, T30–T43 (benchkit/README/goal-app tails, Doctor refused-ADTs, cqrs-lint FP harness).
4. W3 owner bundle: G-T01 direction ruling, Zenoh go/no-go, 350-line ratification, ADR-0139 questions, CI billing fix, Turso filing approval, iroh P99 ratification.

## d) HONESTY LEDGER (the session's own fuckups)

1. Three shell-mangling rounds building the smoke list (awk `NF--` OFS bugs: spaces-then-slashes) — two smoke runs + a prewarm burned.
2. The tool-shell's `kill` builtin silently no-ops (rc=2 hidden) — two no-op rounds while 3 duplicate smoke trees hammered the host; `/run/current-system/sw/bin/pkill` worked first try. Gotcha recorded.
3. `GOTOOLCHAIN=local` leaked into pre-commit hooks twice (hook's `go build` dies on go.work 1.27.1) — the documented fix was ignored by habit; each retry cost minutes.
4. Two first-draft self-tests passed vacuously (unhonored env overrides, lying fixture numbers) — caught by their own mutation legs, rebuilt.
5. Ported the sqlite stress test to DuckDB before checking the driver could run it — 13 minutes of C-level prepare convoy before accepting the driver limit.
6. Six authored commits absorbed by the daemon despite pre-add status checks (its window is seconds); content complete, history says `chore:`. Two more stale `.git/index.lock` clears (proven ownerless first).

## e) WHAT WE SHOULD IMPROVE

1. Ship a `#verify-when-quiet` flake app (wait → gate → verify in-repo) — tonight's supervisor is a /tmp nohup that died with the reboot once already.
2. `tag-release.sh --smoke` should refuse duplicate live smokes of the same module (3 trees on taskmanager) and set `GOTOOLCHAIN=auto` in its probe env (both gotcha-recorded).
3. The api-stability tidy gate should also cover the cqrs-lint testdata fixture (replace-based consumer excluded by design — but a stale fixture burned 3 verify attempts; the fail-loud helpers + gotcha mitigate).

## f) NEXT (ordered)

1. Verify attempt 4 finishes → record S03 (or triage any red to zero; the pipeline + fail-loud tooling make the next diagnosis minutes, not hours).
2. Same window: T13 → T14 → T15 in that order.
3. W2 slice remainders by impact: T44 ownership/load-threshold guards → T43 Doctor refused-ADTs → T32–T34 benchkit → T36–T38 README/goal-app.
4. W3 stays owner-gated (the 12-minute decisions unblock >20 tasks each).

## g) OWNER QUESTIONS (unchanged from 09:40 report)

1. Annotation-gate ratification — largely MOOT (T19 shipped the gate with a conscious baseline).
2. 92-tag wave authorization — MOOT (cut, pushed, verified).
3. `readme_claims_test.go` ownership — verified green; still unclaimed.
4. iroh P99 50→150ms — keep or revisit.

— Verify attempt 4 live at report time (Race phase, 245 pkgs green through Test); supervisor armed; tree clean; master pushed.
