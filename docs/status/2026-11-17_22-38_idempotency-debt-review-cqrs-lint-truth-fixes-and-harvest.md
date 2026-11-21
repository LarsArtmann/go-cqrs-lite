# Status Update — idempotency debt review, cqrs-lint truth fixes, backlog harvest

**Date:** 2026-11-17 22:38 CET
**Session window:** ~15:40–22:40 CET (plan review at 13:33 was the prior turn)
**HEAD:** `8520f6c82` (3 commits ahead of `origin/master`, not pushed)
**Scope of this report:** only what happened in this session — the three cqrs-lint defects found while reviewing
`idempotency` usage, their fixes, the verification receipts, the fleet evidence gathered, and the backlog harvest.
Everything else in the repo is out of scope here.

---

## Session in one paragraph

The previous turn produced a plan (`docs/planning/2026-11-17_13-33_SUPERB-cqrs-lint-truth-and-idempotency-debt.md`)
and three owner decisions. The owner authorized fleet edits (D1), asked to expand D2, and asked for a D3
matrix. This session executed plan phases P1 (linter truth) and the P2 harvest, deferred all fleet work
behind D2/D3 (as asked), and produced the D2/D3 analysis with per-app evidence gathered from 10 checkouts.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                     | Evidence                                                                                                                                                                                                                                  |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| a1  | **Scorecard false-MISSING fixed** — `go-cqrs-lite/idempotency` added to the `idempotency` import hints                                                                                                                   | `cmd/cqrs-lint/v4/pkg/analyzer/module_catalog_data.go:138-147`. Live check: `scorecard --path` now reports `Idempotency: used` for `games/SEC`, `InboxClean`, `github-local-sync` (all previously `missing`); `Zlota44` control unchanged |
| a2  | **Hint regression tests** — parent-only import ⇒ `UsageImported` with exact evidence; lookalike path ⇒ `UsageAbsent`                                                                                                     | `module_detect_test.go` (`TestDetectUsedModules_ParentIdempotencyShimImport`, `…LookalikePathAbsent`)                                                                                                                                     |
| a3  | **F030 became a deleted-module registry** — 7 entries: `transport/http`, `transport/grpc` (ADR-0127), `idempotency/kvstore` (2026-10-17), parent `idempotency` + `codec` + `retry` + `flightrecorder` (ADR-0128)         | `cmd/cqrs-lint/v4/pkg/rules/adoption/f030.go` — each entry carries removal record + ADR path + replacement                                                                                                                                |
| a4  | **Longest-fragment matching replaces plain substring matching** — a specific entry is never shadowed by a deleted ancestor; segment boundaries on both sides                                                             | `longestDeletedModuleMatch`, `fragmentMatchesImport`, unit tables in `f030_internal_test.go`                                                                                                                                              |
| a5  | **Present-module suppression list** — `idempotency/sqlstore` (deprecated, present) is never reported as deleted                                                                                                          | `presentModuleFragments` + `TestF030_DeprecatedSqlstoreIsNotReportedAsDeleted`                                                                                                                                                            |
| a6  | **F030 fixtures** — parent fires, kvstore fires exactly once, sqlstore silent, 3 ADR-0128 shims fire, sanctioned paths silent                                                                                            | `f030_test.go` (7 tests total)                                                                                                                                                                                                            |
| a7  | **V007 covers `idempotency/sqlstore`** — module entry with a real successor (`(*system.System).Dedup()` / `metaengine.DedupStore`), `adrForFragment` cases for `scheduling`→ADR-0142 and `idempotency`→ADR-0158/0167     | `v007_tables.go:36-41`, `v007.go`                                                                                                                                                                                                         |
| a8  | **V007 detector fixtures** — `sqlstore.NewSQLiteStore` fires (message cites ADR-0158/0167); `go-idempotency` import stays silent                                                                                         | `v007_test.go` (`TestV007_DetectsIdempotencySqlstoreRetirement`, `…SilentOnExternalIdempotency`)                                                                                                                                          |
| a9  | **Drift-scanner signal widened** — `no v5 successor`, `adr-0158`, `adr-0167` added; 16 previously invisible sqlstore markers now enter the inventory                                                                     | `v007_drift_scan_test.go` + `TestV5DriftScan_SeesIdempotencySatelliteMarkers` (asserts ≥10) + phrase table test                                                                                                                           |
| a10 | **Registry drift gates (reverse direction)** — a listed-deleted module must have no `go.mod` at HEAD (root or `/vN`); every cited ADR must exist; every present-module guard must still exist                            | `f030_internal_test.go`, `moduleGoMods` helper                                                                                                                                                                                            |
| a11 | **Artifacts regenerated truthfully** — `RULES.md` diff is 2 lines (name + description only, no ID churn); cqrs-lint `README.md` F030 row; `presets.go` disable comment                                                   | `34973524b`; `rules_doc_test.go` freshness test green                                                                                                                                                                                     |
| a12 | **Doc-count lie fixed** — `docs/DOMAIN_LANGUAGE.md:345` 202 → 217; `references/modules.md` idempotency/kvstore/sqlstore rows now name `system.Dedup()`; `references/core.md:204` dedup routing points at the engine path | `2bb70ab84`                                                                                                                                                                                                                               |
| a13 | **Verification receipts** — full `cmd/cqrs-lint/v4` suite green (`go test ./... -count=1`); `cmd/doc-check` "All 1391 references valid across 57 package(s)"; `check-canonical-facts.sh` green                           | run this session                                                                                                                                                                                                                          |
| a14 | **My own canonical-facts drift fixed** — the 12-26 status report was missing from `docs/status/README.md`; row added, gate re-run green (`missing=0 dangling=0`)                                                         | `d185f80c6`                                                                                                                                                                                                                               |
| a15 | **HARVEST done** — 10 fleet-repair rows (owning repo per row, gated where needed), D28/D29 on the owner decision sheet, D1 recorded as authorized, ROADMAP questions 23/24                                               | `8520f6c82` (61 insertions)                                                                                                                                                                                                               |
| a16 | **D2/D3 evidence base built** — 10-checkout survey: store type, TTL, sweeper presence, restart survival, middleware order, lying comments, per-repo swap file lists, indirect vs direct `go-idempotency` version         | **PERSISTED** in [`docs/evidence/2026-11-17_idempotency-fleet-wiring-survey.md`](../evidence/2026-11-17_idempotency-fleet-wiring-survey.md) (its §7 records the six claims that pass corrected)                                           |

**Rule-count integrity:** `RULES.md` anchors = 217; `meta_test.go` pin = 217; cqrs-lint README = 217; DOMAIN_LANGUAGE = 217.
No rule IDs were added, removed, or renumbered.

---

## b) PARTIALLY DONE

| #  | Item                                | What is done                                                                     | What is missing                                                                                                         |
| -- | ----------------------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| b1 | **T7 deleted-module registry gate** | Reverse gate (no ghost entries), ADR-existence gate, guard-liveness gate         | The **forward** direction: every module an ADR records as deleted must have a row. No enumeration source is wired yet   |
| b2 | **T5 verification**                 | Unit suites, doc-check, canonical-facts                                          | `nix run .#lint-scope` and `nix run .#verify` were never run — "all tests green" is true, "all gates green" is unproven |
| b3 | **T9 process rules**                | Rowed in TODO_LIST                                                               | No gotchas entry written: "tag exists ≠ module exists", `who-uses`-before-greps, daemon-blends-foreign-files            |
| b4 | **F030 rename**                     | Catalog, `RULES.md`, README, presets comment, V007 allowlist reasons all updated | No CHANGELOG `[Unreleased]` entry; never grepped consumer `.cqrs-lint.json` files for name-keyed suppressions; no tag   |
| b5 | **D2/D3 answer**                    | Evidence + matrix + recommendation delivered in chat                             | Owner has not chosen; no code depends on it yet                                                                         |
| b6 | **Fleet work (D1 granted)**         | Every repo/item catalogued with file:line evidence                               | Zero repos touched (correctly — D2/D3 gate the ordering)                                                                |

---

## c) NOT STARTED

1. All 10 fleet repairs (3 pure shim swaps, 4 bundled swaps + store migrations, 4 sweeps, Zlota44 store retirement, Swetty dangling `replace`, duplicate checkout) — `TODO_LIST.md` §Harvest 2026-11-17.
2. T8 doc-check module-existence rule.
3. T18 `cmd/cqrs-lint` release + fleet pin bumps.
4. T19 `nix run .#verify`.
5. T20 CHANGELOG entries + postmortem.
6. `cmd/cqrs-upgrade` codemod for the shim swap.
7. Per-repo receipts (build/test/scorecard) for any fleet change.
8. Census re-run + ADR-0167 addendum refresh after the fleet moves.
9. The plan file `2026-11-17_13-33_…` is still marked **PROPOSED** with T1–T20 open — no status banner/addendum.

---

## d) TOTALLY FUCKED UP

1. **Commit hygiene — I violated the repo's own rule.** AGENTS.md rule 4 says to hand-commit at phase
   boundaries or the daemon absorbs edits into `chore:` commits. I never did, so T1–T6 now live across
   `006262476`, `a00ff3a06`, `34973524b`, `2bb70ab84` + two later chore commits — and my one authored
   attempt printed "nothing to commit" because the daemon had already taken it. **Worse:** `2bb70ab84`
   contains my three doc files _and_ another session's `v003*.go`/`v006.go`/`v002_v006_test.go`. The
   history for this work is now unauditable, and I cannot honestly claim any commit as "the fix".
2. **The surveyed fleet evidence exists only in this conversation.** The 10-checkout wiring table (store
   per app, TTLs, sweeper presence, the lying comments, retry/idempotency inversion) has no file on disk.
   If this session ends, that evidence is gone — and it is the basis for every D3 claim I made.
3. **Two red iterations on the reverse gate, both self-inflicted.** First version asserted the _directory_
   was gone (it exists — `sqlstore` lives under it), second asserted `go.mod` at the module path (the module
   is at `/v4`). One `ls` would have avoided both.
4. **A test that lied about the contract.** I asserted `example.com/…/go-cqrs-lite/idempotency` must NOT
   match, but `analyzer`'s own boundary test documents the opposite (line 223). I encoded my intuition as a
   requirement instead of checking the precedent; the fix was to correct the test, not the code.
5. **"All tests green" is an overclaim as written.** I ran the linter suites, doc-check and canonical-facts;
   I did not run `golangci-lint`/`#lint-scope`. `f030.go` grew three new functions and a nested loop — if
   complexity or line-length rules fire, my claim was premature.
6. **Stale plan, created by me, left stale.** I wrote a 452-line plan two turns ago and this session made
   ~30% of it obsolete without adding the dated banner the repo's reconciliation procedure demands. This is
   exactly the "timestamped doc entombment" failure mode.
7. **The F030 name change is an unratified consumer-facing decision.** Rule IDs stayed stable, but the human
   name changed (`deprecated-transport-import` → `deleted-module-import`). I did not check whether any
   consumer config or doc keys on the name, and I did not surface it as a decision until now.
8. **Late discovery of the actual bug (carried over, still mine).** The previous turn framed the problem as
   "nobody uses it"; the _real_ defect (scorecard lying to three apps) only surfaced in the second pass.
   "Tag exists" was read as "module exists" for a full turn.

---

## e) WHAT WE SHOULD IMPROVE

1. **Commit at phase boundaries, not at the end.** Concrete habit: the moment a package's tests go green,
   `git add <that package>` + authored commit. Multi-file refactors must never ride the daemon.
2. **Persist research the moment it is produced.** A subagent report answering two owner questions belongs
   in `docs/evidence/` before the next tool call, not in the chat transcript.
3. **Read the layout before writing existence checks.** Module existence = a `go.mod` (root or `/vN`), not a
   directory; encode that in one helper instead of retrying ad-hoc `os.Stat` calls.
4. **Check the precedent test before asserting a contract.** `analyzer`'s boundary table already documented
   the fork/vendoring behavior; my test contradicted it.
5. **One table, one state field.** `deletedModuleImports` + `presentModuleFragments` are two parallel lists
   for one concept; a `state: deleted|present` field removes the possibility of them drifting apart.
6. **Never say "green" without naming which gates ran.** Use "unit suites + doc-check + canonical-facts green;
   lint and #verify NOT run".
7. **Banner planning docs in the same session that obsoletes them**, not in a follow-up.
8. **Treat rule names as API.** Any rename needs a CHANGELOG entry + a consumer-config grep in the same commit.

---

## f) UP TO 50 THINGS TO DO NEXT

**Closing this session's own loose ends (do first)**

1. Add the dated status banner + per-task addendum to `docs/planning/2026-11-17_13-33_SUPERB-…` (P1 shipped, P2 partial, P3 gated).
2. Persist the fleet wiring survey as `docs/evidence/2026-11-17_idempotency-fleet-wiring-survey.md`.
3. CHANGELOG `[Unreleased]`: F030 rename (name + ID note), V007 `idempotency/sqlstore`, drift-signal widening.
4. Grep every consumer `.cqrs-lint.json` for the F030 name before ratifying the rename; update any hit.
5. Run `nix run .#lint-scope` over `cmd/cqrs-lint` (close the "green" overclaim).
6. Run `nix run .#verify` on the changed set.
7. Merge `deletedModuleImports` + `presentModuleFragments` into one table with a `state` field.
8. Add the forward registry gate (ADR-recorded deletions ⇒ registry rows).
9. Decide F030's `_test.go` behavior (currently test-only deleted imports are invisible).
10. Add a scorecard golden/fixture pinning `Idempotency: used` for a shim-importer profile.
11. Verify the V007 replacement string names an exported symbol that exists (`(*system.System).Dedup`).
12. Check `system.Dedup()` godoc states the sweep requirement; link it from the V007 suggestion.
13. Add sweep guidance to the skill's dedup recipe (`references/recipes.md` §2.13 area).
14. ADR-0158 addendum: the linter now enforces the retirement (F030/V007 coverage).
15. Postmortem: first-turn "nobody uses it" framing + this session's commit-hygiene failure.

**Fleet — pure shim swaps (no store change; unblocked by D1 alone)**

16. InboxClean `internal/emailcqrs/setup.go` → `go-idempotency`; build + test.
17. games/SEC `internal/cqrs/app/app.go` + test → `go-idempotency`; build + test.
18. github-local-sync `internal/branch/service.go` → `go-idempotency`; build + test.
19. SwettySwipperWeb: 3 files → `go-idempotency`, delete the dangling `replace`, collapse the adapter onto `metaengine.NewFromEngine`.
20. Each of 16–19: confirm `go mod tidy` produces a DIRECT require and no shim line remains.

**Fleet — bundled swap + store migration (needs D28/D29)**

21. KeyCountdown: swap + sqlstore → engine dedup + `DedupSweep`; remove the `NewIdempotencyStore` DI function.
22. browser-history: swap + engine dedup, collapse the memory/sqlite dual branch.
23. browser-history-precheck: same, plus stop discarding a `MemoryStore` (its 10m ticker leaks).
24. Zlota44: swap 5 files + `go mod tidy && go mod vendor` + retire the 172-line hand-rolled store.
25. Zlota44: delete the false "growth is bounded … no sweeper needed" comment with the store.
26. browser-history + precheck: fix `/debug/idempotency` (`SweepEnabled` and hardcoded `DedupLayers`).
27. browser-history: fix the "native TTL" comment; mirror the correction into precheck.
28. github-local-sync: fix the retry-outside-idempotency inversion (retry attempt 2 currently gets `ErrDuplicate`).
29. InboxClean: close the `MemoryStore` the setup creates (goroutine leak).
30. DiscordSync: replace peek-then-record with `CheckAndRecord` (atomic claim).
31. Sweeps: browser-history, precheck, KeyCountdown, Swetty (`DedupSweep` never called anywhere).
32. precheck: bump indirect `go-idempotency` v0.3.1 → v0.3.2 (only project off the fleet version).
33. Decide the DiscordSync/discordsync canonical checkout; trash-or-archive the twin.
34. Per-repo receipts: build + test + `cqrs-lint scorecard` before/after each change.
35. Re-run `scripts/census-fleet.sh` and diff against the ADR-0167 addendum counts.
36. Re-run `bash scripts/check-versions-manifest.sh --check` after any `go.mod` edit.

**Linter / repo hygiene**

37. doc-check rule: a cited module path must exist at HEAD or carry a deleted marker (T8).
38. Add "tag exists ≠ module exists" + `who-uses`-before-greps to `gotchas-module-management.md`.
39. Add the daemon-blending hazard with this session's evidence (`2bb70ab84` mixed two sessions).
40. Add a deleted/tag-only row convention to the module tables in `references/modules.md`.
41. Cut `cmd/cqrs-lint` (F030/V007) + bump fleet/CI pins; note the rename prominently.
42. FEATURES.md row: cqrs-lint now covers deleted-module imports and the idempotency retirement.
43. Refresh the 2026-11-13 deep-dive's stale "ten fleet go.mod requires pinning `idempotency/v4`" claim (now 7 modules / 17 files, verified).
44. Add `check-canonical-facts` coverage for the deleted-module registry (count + ADR links).
45. Triage the 2 moderate dependabot alerts on the default branch (noticed at push).
46. Decide whether `docs/planning/` should join the `cmd/doc-check` default corpus.
47. Write the `cqrs-upgrade` codemod rule for the shim swap.
48. Add a `system.Dedup()` + sweep recipe to the skill (durability + retention in one place).
49. Re-run the scorecard on all 8 consumers as the fleet baseline before any repair.
50. File the one genuine upstream ask surfaced by this session: nothing yet — record it as "none" rather than invent one.

---

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **D29 semantics — replay or dedup?** For InboxClean and github-local-sync, is re-running the same
   command IDs meant to be a no-op (dedup wins) or a deliberate replay (dedup must forget)? I can see the
   code paths but not the intent, and it decides whether durable dedup is a bug fix or a regression for
   those two apps. (browser-history/precheck are unambiguous bugs; SEC is unambiguously keep-as-is.)
2. **Release timing for the linter changes.** Cut `cmd/cqrs-lint` now so the fleet gets the honest
   scorecard and the F030/V007 findings, or batch with the next wave? This is release policy, and it
   determines whether fleet suppressions must be updated today or later.
3. **Which DiscordSync checkout is authoritative?** Both are real clones with distinct remotes
   (`DiscordSync.git` ahead 2 with runtime config; `discordsync` behind 5 / ahead 3 without config). I can
   diff them but cannot decide which one you actually run from.

---

## Appendix — receipts

| What                      | Command                                                                  | Result                                                                               |
| ------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| analyzer suites           | `GOWORK=off go test ./pkg/analyzer/ -count=1`                            | ok                                                                                   |
| adoption + version suites | `GOWORK=off go test ./pkg/rules/adoption/ ./pkg/rules/version/ -count=1` | ok (after fixing my own boundary-test expectation)                                   |
| full linter suite         | `GOWORK=off go test ./... -count=1`                                      | ok (green twice, incl. after the registry gates)                                     |
| doc-check                 | `go run . SKILL.md references/*.md AGENTS.md`                            | `✓ All 1391 references valid across 57 package(s)`                                   |
| canonical facts           | `bash scripts/check-canonical-facts.sh`                                  | red on my own missing index row → fixed → `all citations match repo-derived truth`   |
| scorecard (before/after)  | `cqrs-lint scorecard --path <repo> -o markdown`                          | SEC / InboxClean / github-local-sync `missing` → `used`; Zlota44 `used` both times   |
| lint (real repos)         | `cqrs-lint /home/lars/projects/games/SEC`                                | now emits F030 "deleted module … idempotency/v4" at `app.go:14`                      |
| lint (real repos)         | `cqrs-lint /home/lars/projects/games/KeyCountdown`                       | now emits F030 (`cqrs.go:20`) + V007 (`cqrs.go:55`, "removed at v5 (ADR-0158/0167)") |
| rule count                | `grep -c '^<a id="' RULES.md`                                            | 217                                                                                  |
| not run                   | `nix run .#verify`, `nix run .#lint-scope`                               | deliberately deferred (listed in f5–f6)                                              |
