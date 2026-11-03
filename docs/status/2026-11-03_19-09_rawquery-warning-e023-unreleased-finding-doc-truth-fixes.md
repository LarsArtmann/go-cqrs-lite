# Session Status: RawQuery Warning, E023-Unreleased Finding, Doc-Truth Fixes

**Date:** 2026-11-03 19:09 CET
**Scope:** This session only — three turns: (1) explain `system.RawQuery`, (2) when to use/not use it, (3) owner rant "the lib should OWN I/O / why is there no warning / bank-sync uses sqlc and thinks it's fine".
**Verdict up front:** All planned edits landed green (doc-check ×2, api golden regen + verify, system build + coeffect tests, graph-native example build). The session's biggest *finding* — E023 enforcement has never been released — is rowed but NOT acted on (no tag cut, no bank-sync edit; both await owner instruction). The session's biggest *mistake* — a botched first cqrs-lint invocation that returned 0 E023 findings and nearly produced a false "the lint is blind to bank-sync's layout" conclusion — was self-caught and corrected, but only on the second attempt.

---

## d) TOTALLY FUCKED UP (lead with the failure, per form)

Nothing destructive, no red suites, no reverts needed. But two real self-inflicted wounds:

1. **Near-miss false conclusion from a sloppy invocation (the big one).** First bank-sync run: `cqrs-lint --strict /home/lars/projects/bank-sync` from the *go-cqrs-lite* cwd → **0 E023 findings**, while simultaneously *displaying* E003/C040 findings that bank-sync's `.cqrs-lint.json` explicitly disables. I almost read this as "E023 is structurally blind to bank-sync's package layout." The disabled-rules-displayed anomaly was itself the tell (config not discovered), and the corrected run — binary built, executed from *inside* bank-sync exactly as its CI does (`cqrs-lint --strict .`) — fired **20 E023 findings**. Root cause of the confusion: cqrs-lint's config discovery diverges between "absolute path from foreign cwd" and "`.` from repo root" (silent divergence — itself a latent tool bug, rowed in §f). Method lesson: **replicate the consumer's exact CI invocation before drawing conclusions about what their lint sees.**
2. **Turn-2 answer shipped on stale docs without noticing until mid-answer.** I answered "when should RawQuery be used" partially on the README/skill guidance that *still described the pre-M08 workaround* ("the gate cannot see through RawQuery — declare Events only when…"), which had been false since system v4.13.0 (2026-10-12). I caught it during source verification and fixed 4 sites, but a reader of my first answer's framing ("originally the gate was blind") underweighted that the *docs themselves* were the bug.

## a) FULLY DONE (verified green unless noted)

1. **Explained `system.RawQuery`** — definition (`system/projection_builder.go:53`), sealed-interface rationale, opacity-by-design, coeffect-gate history. Question-answer only.
2. **Fixed 4 stale pre-M08 doc sites** (the "RawQuery is opaque to the coeffect gate" workaround class):
   - `system/README.md:481-484` → now documents the fixed behavior (gate reads `metaengine.Fold.EventType` since v4.13.0).
   - `.agents/skills/go-cqrs-lite/references/recipes.md` §2.44-area NOTE → same correction.
   - `example/graph-native/composition.go:32-36` comment → now states declaring `Events` would compose cleanly and add typo protection.
   - `TODO_LIST.md` coeffect-RawQuery row → flipped `[x]` with SHIPPED receipt (M08, v4.13.0, pinned by `system/coeffect_gate_test.go:150`).
   - Verified: doc-check ALL-GREEN (1372 refs / 57 packages), graph-native `go build` green.
3. **Investigated the bank-sync/E023 rant with receipts** (read-only on bank-sync):
   - bank-sync's satellite-SQL surface mapped: sqlc-generated `internal/db`, hand-SQL `INSERT … ON CONFLICT` projections (`internal/cqrs/projections.go:121,234,288,353,419`), same-file cqrs-import + `_ "modernc.org/sqlite"` (`internal/storage/sqlite/storage.go:18,20`), journal on the metaengine sqlite engine sharing one DB file.
   - **E023 is UNRELEASED**: added to master 2026-10-24 18:06 (`dcb786f0c`); `cmd/cqrs-lint/v4.18.0` was tagged 06:44 that same morning — **zero cqrs-lint tags contain the rule**.
   - bank-sync pins `@d15e0a48` (2026-10-12) in CI + hermetic flake package — 12 days older than the rule; it structurally cannot see E023.
   - Ground truth run (current working tree, exact CI invocation): **20 E023 findings**, exit 0.
   - `--strict` semantics verified: fails only on load errors; info advisories never gate the exit code — pinned by design (`TestShouldExitWithError_InfoAdvisoryNeverGates`).
4. **Added the "huge warning" the owner demanded** — "raw = hand-declared, NOT raw SQL", on-substrate statement, ADR-0142 + E023 pointer:
   - `system/projection_builder.go` RawQuery godoc → **api golden regenerated (6422 exports) and verified**; system module `go build` + `TestSystem_Coeffect*` green.
   - `system/README.md` (declaring-projections section, WARNING block).
   - `.agents/skills/go-cqrs-lite/references/core.md` (quickstart NOTE) and `recipes.md` (graph recipe prose).
   - Final doc-check re-run green after these edits.
5. **TODO_LIST rows added (2):**
   - RELEASE cqrs-lint ≥ v4.19.0 carrying E023 (+ bank-sync pin bump with its audit-marker discipline) — in the substrate section next to the E023 DONE row.
   - v5: rename `system.RawQuery` (e.g. `DeclaredQuery`) + tighten `decl any` → typed `metaengine.QueryDecl` — in the system section.

## b) PARTIALLY DONE

- **"The lib should own I/O" enforcement story.** Diagnosis complete (rule exists on master, unreleased; consumer pins predate it; advisories never gate CI). The *fix* is 0% executed: no tag, no consumer pin bumps, no teeth. Deliberately awaiting owner (release + cross-repo actions).
- **RawQuery misleading-name mitigation.** Warnings shipped everywhere the symbol is documented; the *name itself* and the type-unsafe `decl any` remain (v5 row only — no additive typed constructor was attempted this session; see §e/§f).
- **bank-sync comment fix.** The comment now says declaring `Events` "would compose cleanly" — but the example still doesn't declare them. The *demonstration* of the fixed gate is a next task, not done.

## c) NOT STARTED (noticed this session, deliberately not touched)

- Cutting `cmd/cqrs-lint/v4.19.0`.
- Any edit inside `~/projects/bank-sync` (pin bump, disable-list re-audit, sqlc→metaengine migration).
- CHANGELOG `[Unreleased]` row for this session's doc-truth/warning changes (see §e — this is a *miss*, not a choice).
- Full `#verify` / `#verify-fast` — only targeted gates ran.
- golangci-lint on the touched Go files (godoc-only change; format believed fine, unverified).
- Fleet-wide pin audit beyond bank-sync (DiscordSync known from the E023 row's field receipt; others unexamined).

## e) WHAT WE SHOULD IMPROVE (self-critique, session-derived)

1. **Session-closing checklist not fully run.** "Close a Wave" procedure says: FEATURES/CHANGELOG inventory + harvest. I wrote TODO rows but **no CHANGELOG `[Unreleased]` row** for the RawQuery godoc WARNING + doc-truth fixes. The api golden moved; the changelog-symbols gate doesn't force doc-only rows, but drift discipline says row it.
2. **Verification breadth.** Targeted tests only; no `#verify-fast`, no lint pass on `system/`. For a godoc-only change the risk is low, but "low" was asserted, not measured.
3. **Invocation fidelity discipline.** First lint conclusion was nearly wrong (see §d). Rule to internalize + row in `docs/agents/gotchas-testing.md`: *when testing a consumer-facing tool's behavior, replicate the consumer's exact invocation (cwd, path form, config) before concluding anything.*
4. **The invocation artifact exposed a real tool trap that I only rowed implicitly.** `cqrs-lint <abs-path>` from a foreign cwd silently analyzed bank-sync with NO config (different findings, different suppressions) — a user/CI-divergence footgun independent of my mistake. Needs its own TODO row + a fix (discover config from the *target* root, or fail loudly).
5. **Near-miss catch was late.** The "disabled rules displayed" anomaly sat in my first run's output; I treated 0-findings as the headline instead of the anomaly. Read the whole output, not the count.
6. **Authored history.** Three logical changes landed uncommitted; the auto-commit daemon will blur them into `chore:` commits. Project rule 4 says commit at phase boundaries when authored history is wanted; I defaulted to global never-commit. Owner should say which norm wins here.
7. **bank-sync evidence could be sharper still.** 20 findings counted, but I did not save the categorized breakdown (which packages, how many per class) into the TODO row or an evidence file — the run log lives in `/tmp` only.

## f) NEXT — up to 50 things, session-derived, ranked

**Release & enforcement (the rant's actual fix):**
1. Cut `cmd/cqrs-lint/v4.19.0` (tag-release.sh; check-release-scripts + check-example-standalone first; per AGENTS: run the changed-set module suites — `verify=ok` builds don't test).
2. Bump bank-sync's cqrs-lint pin (CI `@d15e0a48` + hermetic flake package) and re-verify its disable list per its `cqrs-lint-disable-audit` marker discipline.
3. Fleet pin audit: `scripts/census-fleet.sh` + grep every consumer's cqrs-lint pin; row the stale ones (DiscordSync first candidate).
4. Decide E023's teeth: scorecard wiring (row already noted on E023 DONE row) so satellite SQL degrades the ModernityGrade — vs keeping print-only advisories.
5. Consider `--fail-on info` / per-rule severity override so strict consumers CAN gate on E023.
6. SARIF output for E023 so CI annotations render findings without exit-code semantics.

**cqrs-lint tool health (found this session):**
7. Fix/decide the config-discovery divergence: `cqrs-lint /abs/path` from foreign cwd vs `cd path && cqrs-lint .` must not silently differ (discover config from target root, or loud error).
8. Test pinning #7 (fixture repo with config, run both invocation forms, assert identical findings).
9. Surface "config file not found at <path>" as a notice so silent-default runs are visible.
10. Audit other cwd-dependent behaviors in cqrs-lint (suppression discovery, inline directives are path-anchored — verify both forms).

**RawQuery / system API:**
11. v5 rename (`DeclaredQuery` or similar) + typed `metaengine.QueryDecl` param (rowed).
12. ADDITIVE v4 option now: a typed constructor (e.g. `RawQueryDecl(d metaengine.QueryDecl) ProjectionDeclaration`) alongside `any`-based RawQuery; deprecate the loose one toward v5 — kills the type-safety gap without breaking.
13. `example/graph-native`: actually declare the `Events` universe (post-M08 it composes) — turns the fixed comment into a demonstrated gate pass + typo protection.
14. Sweep remaining docs for pre-M08 gate claims (`grep -rn "cannot see through"`-class) — archived status docs stay frozen, live docs only.
15. CHANGELOG `[Unreleased]` row for this session: RawQuery godoc WARNING + 4 stale-doc corrections + golden regen.
16. Consider a short "I/O ownership" section in root README pointing at ADR-0142 + E023 (consumer-facing statement of the contract-30 rule).

**bank-sync migration (contract-30 debt, cross-repo):**
17. Plan the sqlc retirement: which read models map to metaengine planned tables / collections; which queries need `QuerySet`-class filters.
18. Migrate the hand-SQL `INSERT … ON CONFLICT` projections to `metaengine` folds via `projectionadapter` (the dual-run metaengine models already exist there — README says projectionhost runs them).
19. Decide the fate of bank-sync's migrations/preflight journal SQL (journal_* files touch `database/sql` against the shared DB file) — engine-owned checkpointing vs deliberate ops tooling.
20. Re-run bank-sync's own gates after any pin bump (its `check-cqrs-lint-audit.sh` tripwire).

**Verification & hygiene (this repo, from §e):**
21. Run `nix run .#verify-fast` over the session's tree (or full `#verify` if quiet).
22. Lint the touched modules (`system`, `cmd/cqrs-lint` untouched but rebuilt — lint `system` at minimum).
23. Save the bank-sync E023 run breakdown as `docs/evidence/` (categorized 20 findings) and link it from the TODO row.
24. Add the gotchas-testing.md row: replicate consumer CI invocation exactly when testing fleet-facing tools.
25. Clean `/tmp/cqrs-lint-bin`, `/tmp/cqrslint-banksync*.txt` after evidence is preserved.
26. Decide the commit-vs-daemon norm for authored sessions (owner call; then commit the 3 logical changes as authored commits if wanted).

**Secondary, noticed in passing:**
27. TODO_LIST §2.26 row (recipes.md §2.26 sqlstore deprecation note) — pre-existing, still open, unrelated-but-adjacent to this session's doc work.
28. bank-sync's stale inline suppressions (6 "safe to remove" warnings in the run) — bank-sync hygiene task.
29. E023 exemption-table audit cadence: `storage/`, `stack/` prefixes go away at v5 deletion — the exemption list will need shrinking with the deletion waves (row a reminder on T19–T21).
30. Consider a doctor-mode hint in cqrs-lint output when the analyzed tree has NO go-cqrs-lite imports at all ("wrong repo?" guard).

## g) QUESTIONS FOR THE OWNER (cannot be figured out from the repo)

1. **Cut the tag?** Shall I run the release ritual for `cmd/cqrs-lint/v4.19.0` now (tag + push are gated on you; per AGENTS the tag wave needs the changed-set test suites first — ~10 min warm)?
2. **Does E023 get teeth?** Print-only advisory (current, test-pinned design), scorecard-grade (degrades ModerntyGrade), or strict-gate opt-in (`--fail-on info` / per-rule override)? This decides whether "bank-sync thinks it's fine" can ever recur silently.
3. **Who touches bank-sync?** Do I do the pin bump + disable-list re-audit (and later the sqlc→metaengine migration plan) inside `~/projects/bank-sync` in a follow-up session, or does that stay with bank-sync's own sessions (I only ever read it this session)?

---

*Session receipts: doc-check ×2 green (1372 refs), api-stability regen+verify green (6422 exports), system build + `TestSystem_Coeffect*` green, graph-native build green, bank-sync lint ground-truth run captured (20 E023, exit 0). No commits authored; working tree left for the auto-commit daemon.*
