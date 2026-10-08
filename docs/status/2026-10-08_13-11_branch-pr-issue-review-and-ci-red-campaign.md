# Status Report — 2026-10-08 13:11 CEST

**Session:** Review of all open branches, GitHub Issues, and PRs in go-cqrs-lite
**Branch:** `cqrs-lint/a014-d013-scoped-fixes` (clean tree, 3 commits ahead of master, no PR yet)
**Scope guard:** This report covers ONLY what this session observed. No new research was done for the report itself.

---

## Executive Summary

The review task completed, but it surfaced something much bigger than the review targets: **master's CI has produced zero green runs in 300 attempts, reaching back to 2026-07-16** (227 failures, 73 cancellations, ~3 months). On top of that sits a **real data-compatibility regression** (pre-v5 CBOR snapshot decode double-prefixes stream IDs), not just golden drift. The four open issues are all legitimate and scoped; only one open PR exists and its red checks are inherited from master, not caused by the bump.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                             | Evidence                                                                                                          |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| 1  | Full branch inventory with merge-state audit (10 branches, local + remote)                                                                                                                                                                       | `git rev-list --left-right --count` per branch; `git branch --merged master`                                      |
| 2  | Proved `a014-d013-review` AND `e004-e006-decider-reemit` contain zero unique work (both point at `b4f29e517`, an ancestor of the current branch)                                                                                                 | `git merge-base --is-ancestor` → both "IS contained in scoped-fixes"                                              |
| 3  | Proved `systemnix-cqrs-lint-hashfix`/`origin/cqrs-lint-vendorhash-fix` (1-line vendorHash fix, Sept 13) is superseded by two later vendorHash refreshes on master                                                                                | `git log --all --grep vendorHash` → `dba6f007b`, `31b2e1bd8` postdate it                                          |
| 4  | Proved `discord-sync-pin` content is fully absorbed on master (storage/eventstore + multi-batch tests exist in evolved form; 5957-file divergence from master)                                                                                   | `git ls-tree master storage/memory/` shows `store_multi_batch_test.go`; `git diff master discord-sync-pin --stat` |
| 5  | Read and assessed all 4 open issues (#56, #54, #36, #26) with per-issue action verdicts                                                                                                                                                          | `gh issue view` on each; #26 comment thread confirms retract directive on master, blocked only on next tag        |
| 6  | Reviewed the only open PR (#55): 3 patch bumps (modernc.org/sqlite 1.60.1, fxamacker/cbor patch, go-flightrecorder), MERGEABLE, red checks inherited from master                                                                                 | `gh pr view 55` (files, checks, body, mergeState UNSTABLE)                                                        |
| 7  | Reproduced the failing module suites locally on the current branch: 9 of 9 tested modules fail (schema, signing, snapshot, storage, storage/memory, commandlifecycle, cmd/cqrs-lint→passes, cmd/api-stability, scheduling/sqlstore, watermill)   | `GOWORK=off go test -count=1 ./...` per module after `source scripts/go-env.sh`                                   |
| 8  | Source-level analysis of the failure class: `cbid.ID.String()` is prefixed display form; `ParseStreamID` strips one prefix; `cbid.ID.UnmarshalBinary/UnmarshalJSON` do NOT strip → asymmetric round-trip; snapshot legacy decode double-prefixes | `id/stream_id.go:60-100`, `snapshot/wire.go`, go-branded-id `id_binary.go`/`id_json_v2.go` at v0.7.0              |
| 9  | Ruled out two dependency-bump suspects with exact diffs: go-branded-id v0.6.0→v0.7.0 (zero diff in serialization files), go-codec v0.3.0→v0.3.1 (zero Go code changes)                                                                           | `git diff` in sibling repos                                                                                       |
| 10 | Established the CI red-window empirically: last success is absent across 300 runs (oldest fetched 2026-07-16); identified the current failing-job set (24 jobs incl. all gate jobs, 9 module suites, PG/Redis integration)                       | `gh run list --branch master --workflow ci.yml --limit 300`; `gh run view 37644184698`                            |

---

## b) PARTIALLY DONE

| # | Item                                                | What works                                                                                                                                                                      | What remains                                                                                                                                                                                                                                                              | Effort       |
| - | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| 1 | Root-cause the module-test failures                 | Failure class localized: display form (`StreamMarker:` prefix) leaks into serialized output; double-prefix bug reproduced and understood at the decode side                     | The **introducing commit/change is NOT identified** — red CI predates every pinned dependency version, and the Sept-21-era failures may differ from today's set. A targeted bisect was never run                                                                          | M–L          |
| 2 | CI failure triage                                   | Module test class + full failing-job list mapped                                                                                                                                | Gate jobs (shfmt drift, actionlint+shellcheck, file-size, arch-check, nix flake check, go.sum tidy, integration-tag lint) named but NOT investigated. Integration jobs (PG, ephemeral PG, Redis) named only. Nightly Gates + Sentinel nightly failures noticed, untouched | M each       |
| 3 | Current branch (`a014-d013-scoped-fixes`) readiness | Substantive content verified (A014 stale-deprecation fix + schema-version rule default-awareness + 261-file treefmt import reorder); `cmd/cqrs-lint` passes locally; tree clean | PR not opened; full `nix run .#verify` not run on it                                                                                                                                                                                                                      | S to open PR |
| 4 | Issue #54 fix scope                                 | 2-line nil guards ×2 (otel, prometheus `Provider.Shutdown`), repro claimed in issue                                                                                             | **Not verified in-repo by me** (issue text trusted); fix not implemented; no tests written                                                                                                                                                                                | S            |
| 5 | Issue #56 fix scope                                 | Convention identified (optional `io.Closer` note on `Subscriber`, mirroring `Bus`)                                                                                              | Doc change not made; all `Subscriber` implementations not audited for goroutine-leak exposure                                                                                                                                                                             | S–M          |
| 6 | Issue #26 closure path                              | Confirmed retract directive is on master; blocker is exactly "next tagged release ships it"                                                                                     | stack/postgres v4.4.2 tag not cut                                                                                                                                                                                                                                         | S            |
| 7 | PR #55 disposition                                  | Risk-assessed (patch-level, low risk); red checks attributed to pre-existing master breakage                                                                                    | Local per-module test sweep over the bump NOT done — unknown whether the bumps fix, ignore, or worsen the 9 red suites                                                                                                                                                    | S–M          |

---

## c) NOT STARTED

| #  | Item                                                                                       | Why not started                                                              |
| -- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| 1  | Opening the PR for the current branch                                                      | Awaiting user go-ahead                                                       |
| 2  | Branch cleanup: delete 8 stale branches + prune 3 `/tmp` worktrees                         | Deletion is destructive-adjacent; recommended, not executed without approval |
| 3  | Implementing #54 nil guards                                                                | Awaiting prioritization; fix is trivial but deserves its own verified change |
| 4  | Implementing #56 doc convention                                                            | Awaiting prioritization                                                      |
| 5  | Design for #36 (`stack/metaengine` split)                                                  | API restructure, needs design first                                          |
| 6  | Wire-format decision: bare vs prefixed stream-ID serialization (ADR)                       | The pivotal open decision for the whole failure class                        |
| 7  | Golden refresh campaign for the 8+ golden-drift modules                                    | Blocked on #6                                                                |
| 8  | Fix for the legacy CBOR double-prefix decode                                               | Understood, not coded                                                        |
| 9  | Targeted bisect to find the introducing commit of the prefix leak                          | Not started; requires known-good baseline which may be months back           |
| 10 | Investigating the 3 nightly workflow failures (Nightly Gates, Sentinel, Autoretry context) | Out of review scope this session                                             |
| 11 | HARVEST of this report's section (f) into TODO_LIST.md / ROADMAP.md                        | Per user instruction: report then WAIT                                       |

---

## d) TOTALLY FUCKED UP

1. **Master CI is effectively dead as a merge gate.** 300 consecutive `ci.yml` runs without a single success, oldest examined 2026-07-16 (~3 months). 227 failures, 73 cancellations. Severity: **blocks development trust** — every "mergeable"/check signal on every PR is noise. Root cause: layered/evolving, unknown in full; the current layer includes real test failures (below), so it is NOT purely infra. Mitigation: none in place — no alerting fired on a 3-month red streak.

2. **Real regression: pre-v5 CBOR snapshot decode mangles stream identity.** `TestWire_CBORRoundTripAndLegacyKeys` (snapshot/wire_test.go:192) fails on master AND the current branch, locally and in CI: legacy bytes carry `"StreamMarker:01HK…"` (the documented `String()` display form) but `cbid.ID.UnmarshalBinary` stores bytes raw with no prefix strip, so re-displaying yields `StreamMarker:StreamMarker:01HK…`. Severity: **data-compat bug** — any consumer decoding pre-v5 snapshot bytes gets corrupted stream IDs. Root cause: asymmetry between `ParseStreamID` (strips prefix) and the marshaler-implemented decode paths (don't). Mitigation: none for consumers; decode via `ParseStreamID` would fix it.

3. **Wire-contract inconsistency across the ID layer.** `id/stream_id.go` documents "`String()` is a display form; the wire form (`MarshalText`) is bare," yet snapshot JSON now serializes the prefixed display form and at least one golden expects bare. Two components disagree about a core contract, and 8+ modules' goldens drift because of it (schema `TestGolden_UpcasterOutput`, signing, snapshot, storage, storage/memory, commandlifecycle, cmd/api-stability, scheduling/sqlstore). Severity: **blocks the module suites**. Root cause: the exact introducing change is unidentified (red CI predates all current dependency pins).

4. **`watermill` suite red** (locally reproduced, cause not yet diagnosed beyond the prefix class). Severity: module blocked. Root cause: unknown — needs its own session.

5. **Repo archaeology is hostile:** the auto-commit daemon produces 466–852-file heuristic commits daily, which destroys bisectability and blame value. This session's root-causing was directly slowed by it. (Known repo trade-off, but it compounds every incident like this one.)

6. **My own session misses (honesty items):**
   - A false-negative early conclusion ("storage/eventstore not on master") from a truncated `git ls-tree | head` — corrected one step later, but wrong narration shipped mid-session.
   - I asked "when was the last green CI run?" ~6 tool calls too late; it should have been question #1 and would have reframed the whole investigation immediately.
   - I did not check `docs/status/` history for prior reports that may already contain CI root-cause work before re-deriving it.
   - Dependency-theory testing (branded-id, go-codec diffs) was sequential guesswork rather than a disciplined bisect plan; two dead ends cost real time.

---

## e) WHAT WE SHOULD IMPROVE

1. **Make "when was CI last green?" the FIRST query** in any CI investigation. It separates "fresh break" from "chronic rot" and prevents wasted archaeology. (Candidate: lesson entry / skill note.)
2. **Existence checks must query exact paths** (`git ls-tree master <dir>/`), never top-level listings truncated by `head`.
3. **The CI matrix needs an honest mode:** either repair to green, or cut it down to the critical set and mark the rest non-required, plus a "consecutive-failures > N" alerting job. Three months of red means the gate no longer gates.
4. **Pin the stream-ID wire contract in an ADR** (bare vs prefixed on the wire), then enforce it with round-trip property tests (`ParseStreamID(String()) == id`, legacy-bytes identity) instead of frozen display-form snapshots that drift whenever rendering changes.
5. **Move prefix handling into one layer** (cbid wrappers or id/v4) so parse/marshal/text/binary paths cannot disagree again — today only `ParseStreamID`/`ParseStreamIDStrict` strip.
6. **Commit at phase boundaries with authored messages** during fix campaigns (AGENTS.md already advises this) so bisect works when the campaign itself causes a regression.
7. **Check `docs/status/` for prior incident reports before re-investigating** known-looking failures.
8. **Verify issue claims in-source before fixing** (#54's panic, #56's interface shape were trusted from issue text this session; fine for review, not for landing code).

---

## f) TOP 50 THINGS WE SHOULD GET DONE NEXT

> Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h
> This section is the primary input for `docs-health` HARVEST (TODO_LIST / ROADMAP routing).

| #  | Task                                                                                                                                                                                                                                                        | Impact   | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Fix snapshot legacy CBOR decode: strip brand prefix on unmarshal so pre-v5 bytes round-trip                                                                                                                                                                 | Critical | S      | Bug           |
| 2  | Decide + ADR: bare vs prefixed stream-ID wire form (decides the entire golden campaign)                                                                                                                                                                     | Critical | M      | Decision      |
| 3  | Find the introducing commit for the display-form leak (targeted bisect on schema golden test)                                                                                                                                                               | High     | M      | Bug           |
| 4  | Refresh goldens in schema, signing, snapshot, storage, storage/memory, commandlifecycle, cmd/api-stability, scheduling/sqlstore after #2                                                                                                                    | Critical | M      | Bug           |
| 5  | Add round-trip property tests: `ParseStreamID(id.String()) == id` incl. legacy byte-string path                                                                                                                                                             | High     | M      | Quality       |
| 6  | Grep-repo hunt for other `String()`-into-wire serialization sites beyond snapshot                                                                                                                                                                           | High     | M      | Bug-hunt      |
| 7  | Verify PR #55 locally (per-module test sweep over the bump), then merge                                                                                                                                                                                     | High     | S      | Quality       |
| 8  | Open PR for current `a014-d013-scoped-fixes` branch                                                                                                                                                                                                         | High     | S      | Cleanup       |
| 9  | Delete 8 stale branches (`a014-d013-review`, `e004-e006-decider-reemit`, `systemnix-cqrs-lint-hashfix`, `cqrs-lint-vendorhash-fix`, `discord-sync-pin`, `lint-sweep-recovery`, `consolidate-catalog`, `nix/go-1.26.7-toolchain`) + prune 3 `/tmp` worktrees | Medium   | S      | Cleanup       |
| 10 | #54: nil-receiver guards in otel + prometheus `Provider.Shutdown`, with typed-nil tests (verify repro in-repo first)                                                                                                                                        | High     | S      | Bug           |
| 11 | #56: document optional `io.Closer` shutdown convention on `event.Subscriber`                                                                                                                                                                                | High     | S      | Documentation |
| 12 | #26: cut stack/postgres v4.4.2 tag so the v4.2.0 retract reaches the proxy; close issue                                                                                                                                                                     | High     | S      | Release       |
| 13 | Diagnose watermill suite failure (prefix class vs broker/test-env)                                                                                                                                                                                          | High     | M      | Bug           |
| 14 | Triage PG Integration + Ephemeral PG + Redis job failures (env vs real)                                                                                                                                                                                     | High     | M      | Bug           |
| 15 | Establish CI honesty: repair-to-green or slim the matrix to required-critical; make the rest advisory                                                                                                                                                       | Critical | M      | Process       |
| 16 | Add alerting: CI job fails when master consecutive-failure count > N                                                                                                                                                                                        | Medium   | S      | Process       |
| 17 | Triage Nightly Gates failure (2026-10-08)                                                                                                                                                                                                                   | High     | M      | Quality       |
| 18 | Triage Sentinel nightly failure (2026-10-08)                                                                                                                                                                                                                | High     | M      | Quality       |
| 19 | Diagnose systemtest module job failure                                                                                                                                                                                                                      | High     | M      | Bug           |
| 20 | Diagnose storage/turso module failure (check known IVM quirks first)                                                                                                                                                                                        | High     | M      | Bug           |
| 21 | Explain cmd/cqrs-lint CI-vs-local discrepancy (passes locally, fails CI)                                                                                                                                                                                    | Medium   | S      | Bug           |
| 22 | Diagnose metaengine/bench module job failure                                                                                                                                                                                                                | Medium   | S      | Bug           |
| 23 | Investigate shfmt drift job (gate)                                                                                                                                                                                                                          | Medium   | S      | Quality       |
| 24 | Investigate actionlint + shellcheck job                                                                                                                                                                                                                     | Medium   | S      | Quality       |
| 25 | Investigate File Size Check job                                                                                                                                                                                                                             | Medium   | S      | Quality       |
| 26 | Investigate Module Layer Architecture Check job                                                                                                                                                                                                             | Medium   | S      | Quality       |
| 27 | Investigate Nix Flake Check job                                                                                                                                                                                                                             | Medium   | M      | Quality       |
| 28 | Investigate go.mod/go.sum Tidy (cold-cache) job                                                                                                                                                                                                             | Medium   | S      | Quality       |
| 29 | Investigate Lint integration-tagged-files job                                                                                                                                                                                                               | Medium   | S      | Quality       |
| 30 | Run full `nix run .#verify` on the current branch for the official gate picture                                                                                                                                                                             | High     | L      | Quality       |
| 31 | Check `docs/status/` prior reports for earlier CI root-cause work; annotate if found                                                                                                                                                                        | Medium   | S      | Documentation |
| 32 | Reconcile AGENTS.md "Verify CI" quick-reference with the 3-month red reality                                                                                                                                                                                | Medium   | S      | Documentation |
| 33 | Verify #54 panic repro in-repo before landing the fix (verify-external-claims discipline)                                                                                                                                                                   | Medium   | S      | Quality       |
| 34 | Audit all `event.Subscriber` implementations for goroutine-leak exposure (follow-up to #56)                                                                                                                                                                 | Medium   | M      | Quality       |
| 35 | Design `stack/metaengine` subpackage split for #36 (additive, root stays lean)                                                                                                                                                                              | Medium   | L      | Feature       |
| 36 | After merge: release cqrs-lint fixes per release procedure (incl. per-module test sweep, NOT just verify=ok builds)                                                                                                                                         | Medium   | M      | Release       |
| 37 | E004–E006 decider-reemit: check whether the findings still fire in cmd/cqrs-lint; do the work or delete the intent                                                                                                                                          | Low      | M      | Cleanup       |
| 38 | Confirm `storage/eventstore` package on master is intentional API (discord-sync origin) per the library-not-app rule                                                                                                                                        | Medium   | S      | Quality       |
| 39 | Check whether discord-sync-pin's `otel/attributes.go` +3 lines were absorbed on master                                                                                                                                                                      | Low      | S      | Cleanup       |
| 40 | Document the display-form vs wire-form lesson in `references/faq.md`                                                                                                                                                                                        | Medium   | S      | Documentation |
| 41 | Sweep `/tmp` worktrees for other stale checkouts beyond this repo                                                                                                                                                                                           | Low      | S      | Cleanup       |
| 42 | Decide the fate of the 130-job CI matrix (keep vs slim) once green-critical subset exists                                                                                                                                                                   | Medium   | S      | Decision      |
| 43 | Future dependabot waves: include go-branded-id/go-codec behavior notes in review checklist                                                                                                                                                                  | Low      | S      | Process       |
| 44 | During fix campaign: authored commits at phase boundaries (bisectability)                                                                                                                                                                                   | Medium   | S      | Process       |
| 45 | Edge-check: `ParseStreamIDStrict` TrimPrefix vs derived SHA-256 IDs that legitimately start with `"StreamMarker:"`                                                                                                                                          | Low      | S      | Bug-hunt      |
| 46 | After fixes: full `nix run .#test-integration` to confirm PG/Redis jobs genuinely pass                                                                                                                                                                      | High     | L      | Quality       |
| 47 | Add a weekly-updated tracked doc of remaining red CI jobs until green                                                                                                                                                                                       | Low      | S      | Process       |
| 48 | Consider `git worktree list --porcelain` hygiene script for prunable worktrees across repos                                                                                                                                                                 | Low      | S      | Cleanup       |
| 49 | Once #2 lands: regenerate `md-go` baseline/goldens where frozen history cites old output shapes                                                                                                                                                             | Medium   | M      | Quality       |
| 50 | Harvest this list into TODO_LIST.md (top ~15) + ROADMAP.md (rest)                                                                                                                                                                                           | Medium   | S      | Process       |

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — Wire-form intent (decides tasks #1–#6):** Is the `StreamMarker:`-prefixed string leaking into serialized output **intentional** (the wire contract is moving to the prefixed display form; goldens should be refreshed), or is it a **regression** to fix at the serialization call sites (wire stays bare per the `id/stream_id.go` doc comment)?
_I tried:_ reading the documented contract, diffing go-branded-id v0.5.1/v0.6.0/v0.7.0 and go-codec v0.3.x, tracing snapshot/event serialization paths. The code says bare-wire; the behavior says prefixed. Only you know the intent.

**Q2 — CI policy (decides task #15):** Has master CI been _consciously_ abandoned in favor of local `nix run .#verify` (i.e., the red matrix is known and accepted since July), or is this an unnoticed rot that should be repaired to green? This decides whether we invest in repairing the matrix, slimming it, or formally re-scoping it.

**Q3 — Cleanup authorization (task #9):** May I delete the 8 verified-stale branches and prune the 3 `/tmp` worktrees now? All are proven to contain zero unique work, but deletion is destructive-adjacent and I won't do it without your go-ahead.

---

## Review targets, one-glance recap

| Target         | State                                                                                                                                                       |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Open PRs       | 1 — #55 dependabot patch bump, MERGEABLE, red checks inherited from master                                                                                  |
| Open issues    | 4 — #56 (doc fix), #54 (2-line nil guards), #36 (design work), #26 (waiting on tag)                                                                         |
| Branches       | 1 active with value (current, no PR yet); 8 stale/deletable; 3 prunable worktrees                                                                           |
| Current branch | 3 commits ahead (cqrs-lint A014 fix + rule default-awareness + 261-file import reorder); cmd/cqrs-lint green locally; inherits all 9 red suites from master |
