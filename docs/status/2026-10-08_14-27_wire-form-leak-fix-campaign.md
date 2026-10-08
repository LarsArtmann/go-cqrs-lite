# Status Report — 2026-10-08 14:27 CEST

**Session:** "Fix it fully" — root-cause and repair the branded-ID wire-form leak class behind the red module suites
**Branch:** `cqrs-lint/a014-d013-scoped-fixes` — tree clean, work absorbed by auto-commit daemon (commits `8019d5957` → `54e5335b4` (229-file fmt wave) → `0d95d3ca5`)
**Scope guard:** This report covers only this fix session's work and observations. Prior-session review findings live in `docs/status/2026-10-08_13-11_branch-pr-issue-review-and-ci-red-campaign.md`.

---

## Executive Summary

The prefix-leak class is fixed at every failing site plus its production core: **23 verification modules green**, including all 9 originally failing suites. Four real production bugs shipped fixes (SQL timer lease lookups that could neither renew nor remove claimed timers; broker metadata leaking brand display forms; `StreamIDFrom` storing display-form raw values; signing canonical bytes coupled to marker names — now format-stable under canonical v2). The session also healed ~20 untidy module go.sums, the release-train gate, and 229 files of standing formatting drift. Remaining red (gci war, example budgets, integration/nightly jobs) is flagged and needs your policy calls, not more code from this session.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Root cause proven before fixing: `cbid.ID.String()` is prefixed display; wire contract (`id/stream_id.go:60-65`) says bare; branded-id `Value()` writes bare while `String()`-fed SQL args query prefixed — asymmetric by construction, shipped in the Aug-22 branded-type adoption, uncaught because CI was already dead | Source reads of `id/stream_id.go`, `claiming/stmt.go`, branded-id `id_sql.go`/`id_binary.go`; version diffs v0.5.1→v0.7.0 ruled deps out |
| 2 | `scheduling/sqlstore` production fix: `MarkFired`, `Cancel`, `RenewLease`, MySQL lease-stamp now bind `id.Get()` (bare) matching INSERT's Valuer-bare writes | store.go:249,254; claiming.go:310; claiming_mysql.go:57 — module suite green incl. claim-metrics tests that were red |
| 3 | `watermill` production fix: `stream_id`/`aggregate_id` metadata keys emit bare; readers unchanged (already accept both) | protocol.go:74-79, command_protocol.go:42-46 — `TestGolden_MessageMetadata` green |
| 4 | `id.StreamIDFrom` fix: strips one leading brand prefix (ParseStreamID semantics) instead of storing the source's display form raw | id/stream_id.go:160-169 |
| 5 | `signing` canonical fix: bytes now built from RAW id values; `canonicalFormatVersion` v1→v2 so old signatures fail LOUD; golden refreshed | signing/payload.go:12,21-23; signing suite green; `hmac-signed-metadata.snap` updated |
| 6 | Test drift re-pinned to the bare contract across 8 modules: schema, snapshot (golden + legacy CBOR bytes), storage (timer + migration), storage/memory, commandlifecycle, watermill (5 prefixed-expectation assertions) | All suites green in final sweep |
| 7 | ~20 modules' go.mod/go.sum tidied (dep-wave debris back to July): `GOWORK=off go mod tidy` loop over all workspace modules, zero tidy failures | `TestEveryModuleGoSumIsTidy` green |
| 8 | `cmd/cqrs-lint/testdata/busfixture` registered: 3 api-stability exclusion maps + LAYER[tier 7] + TEST_INFRA_MODULES + DEP_BUDGET=1 | `TestEveryGoModDirIsInModulesList` + `TestEveryModuleHasLayerEntry` green |
| 9 | CHANGELOG: v4.14.1 "(1 module: cmd/cqrs-lint)" wave declaration (release-train gate), one Changed (signing v2) + four Fixed bullets; symbol-citation gate reworded and passed | `check-changelog-symbols.sh`: "55 citations verified, honest" |
| 10 | `nix fmt` healed 229 files of standing treefmt drift (the red format-gate class) | commit `54e5335b4`; post-format compile + spot suites green |
| 11 | Final certification sweep: **23 modules GREEN** post-format (`id schema signing snapshot storage storage/memory commandlifecycle cmd/api-stability cmd/cqrs-lint scheduling scheduling/sqlstore watermill decider deriver kv graph listing benchkit cmd/cqrs-bench integration event/v4/eventtest metaengine/bboltengine` with `SOAK_SKIP_BOLT=1`), taskmanager + mesh-demo build | background sweep `ALL 23 MODULES GREEN` |
| 12 | Memory recorded: display-vs-wire contract, double-prefix round-trip trap, gci-vs-treefmt war (gotchas-language-footguns); SOAK_SKIP_BOLT sweep trap (gotchas-testing) | 4 appended entries |

---

## b) PARTIALLY DONE

| # | Item | What works | What remains | Effort |
|---|------|-----------|--------------|--------|
| 1 | Same-class wire-leak sweep | Failing + known sites fixed; grep identified further suspect sites | **NOT fixed or verified**: `watermill/catchup_replay.go:86,118` (replay watermark as `evt.ID().String()` + `after.String()` cursor), `storage/sql_aggregate_reader.go:105` (`opts.After.String()` cursor), `storage/command_store_journal.go:57`, `storage/query_store_load.go:94` (pagination-cursor args). Same latent pattern; no failing test covers them. I flagged these in-session and then **dropped them from the final response's flag list — a real miss** | M |
| 2 | CI redness repair | Module-test class fixed; tidy + format + busfixture legs of the gate failures addressed locally | Integration (PG/Redis), nightly (Gates/Sentinel), and remaining gate jobs NOT re-verified; my claim that the fmt wave "heals the shfmt/fmt gate class" is plausible but **unproven until a CI run or local gate script run** | M |
| 3 | Lint state | 5 touched modules carry zero findings from MY changes | Standing gci-vs-treefmt war: `.golangci.yml:832` re-enabled `gci` contrary to AGENTS.md #18's documented removal; bare `golangci-lint run` red on untouched files (gci grouping vs treefmt goimports). Config policy decision, not mine to relitigate silently | S (decision) + M (aftermath) |
| 4 | Example dep budgets | Identified precisely: mesh-demo 6>5, goal-shaped-app 11>10; verified they **pre-date my session** (master commits Oct 6-7) | Raise budgets or shed deps — explicit budget review per AGENTS.md; red in the layer gate | S–M |
| 5 | Signing v2 rollout | Implemented, documented in CHANGELOG with one-time-break warning | No regression test pinning that a v1 signature FAILS v2 verification (only v2-passes is pinned); no consumer migration note beyond CHANGELOG; references/faq.md not updated | S–M |
| 6 | Wire-form decision record | Contract enforced by tests + memory entries | **No ADR written** — the bare-wire decision (my call under "fix it fully") lives in code comments, CHANGELOG, and gotchas; the repo's convention for cross-module contract decisions is an ADR | S |
| 7 | Carried from prior report | — | Branch PR, 8 stale-branch deletions, worktree prune, issues #54/#56/#26/#36 — untouched this session | S–L |

---

## c) NOT STARTED

| # | Item | Why |
|---|------|-----|
| 1 | Cursor/watermark same-class fixes (b1) | Scope discipline: fixing failing tests first; class sweep is a distinct pass |
| 2 | ADR for branded-ID wire form | Decision documented informally; ADR authoring not started |
| 3 | v1-signature-verifies-v2-fails regression pin | Not written |
| 4 | PR for this branch | Awaiting go-ahead; branch now carries the fix campaign + fmt wave |
| 5 | Branch/worktree cleanup (8 branches, 3 /tmp worktrees) | Awaiting authorization (unchanged from prior report) |
| 6 | Issues #54 (nil guards), #56 (doc convention), #36 (stack split design), #26 (tag v4.4.2) | Untouched |
| 7 | CI honesty (slim matrix / alerting on consecutive failures) | Policy decision pending |
| 8 | HARVEST of the two reports' (f) sections into TODO_LIST/ROADMAP | Awaiting instruction per report-then-wait |
| 9 | Upstream go-branded-id work (strip-on-unmarshal option; ULID-vs-string String() asymmetry) | Cross-repo, not started |
| 10 | Post-merge release train incl. signing/watermill/scheduling/id/storage modules | Blocked on PR + merge |

---

## d) TOTALLY FUCKED UP

1. **I dropped two known latent same-class bugs from my final report.** The catchup-replay watermark and storage pagination-cursor `String()` sites were identified mid-session, consciously deferred, and then **omitted from the response's "still red" list** — they survived only in my working notes. A flagged-not-fixed list that loses items is worse than no list. (Recovered now: report b1.)
2. **The signing v2 break is shipped on the branch with no v1-fails test.** The loud-break design is sound, but the only pinned property is "v2 signs/verifies"; nothing proves old signatures actually reject (the property that makes the break loud instead of silent). Until that test exists, a future refactor could silently re-accept v1 bytes and nobody would notice.
3. **Attribution muddle:** my surgical fixes were absorbed by the daemon into heuristic commits mixed with the 229-file fmt wave. The fix campaign's history is now un-bisectable as authored work — the exact anti-pattern AGENTS.md warns about (commit at phase boundaries). My fault for not committing or fencing the fmt wave separately.
4. **Session own-goals (all caught, all real):**
   - Applied `.Get()` to `evt.ID()` (ULID-backed) in storage/memory golden test without type-checking — compile error, caught by LSP, reverted. Pattern-applied before understanding that ULID-backed ids are the bare-`String()` family.
   - Declared bboltengine RED and spent three background-job round trips on it — the "failure" was the documented 509–1145s soak hitting the 5m timeout; `SOAK_SKIP_BOLT=1` is written in the test's own header comment. Forgot my own prior-report lesson about reading the failure mode before diagnosing.
   - Created a duplicate map key via blind `sed -i` after a partial multiedit, then fixed it with blind `sed -i '175d'` line deletion — fragile on two counts; should have re-viewed and edited precisely.
   - Tripped `check-changelog-symbols` with backticked `evt.ID`/`evt.StreamID` prose (the gate regex counts dotted identifiers as symbol citations) — reworded after the gate caught it.
   - Ran `nix fmt` only at session end, mixing 229 reformat files into the same absorbed commit as the fixes instead of fencing them early.

---

## e) WHAT WE SHOULD IMPROVE

1. **Type-check before pattern-applying.** The `.String()` vs `.Get()` rewrite has TWO families (ULID-backed bare-String, string-backed prefixed-String); a one-line type check would have prevented the compile error and sharpened the whole fix.
2. **Read a failing test's own header before diagnosing.** The soak env-gate was documented three lines into the file. "Check `running tests:` in the timeout dump" is now in memory — it should be reflex.
3. **Never blind-delete by line number.** Re-view, then edit. The `sed '175d'` reflex worked once; it will eventually delete the wrong line.
4. **Format + lint at phase boundaries**, not session end: `nix fmt` after edits, gates after each doc change. Fencing mechanical waves from authored fixes preserves bisectability.
5. **Fix the class, not the failures.** I fixed failing sites; a 10-minute grep pass found four more latent sites of the identical bug. Every wire-leak fix session should end with the class sweep — and its findings must land in the REPORT, not just the working notes.
6. **Gate-aware prose.** Any backticked dotted identifier in CHANGELOG [Unreleased] Added/Changed is treated as a symbol citation by `check-changelog-symbols`; write around it or verify against the golden first.
7. **Contract decisions get ADRs, not just tests.** Tests enforce bare-wire today; the ADR is what tells next year's refactor why.
8. **Carry a persistent "flagged, not fixed" list in the artifact**, not in session memory — items must survive to the report unconditionally.

---

## f) TOP 50 THINGS WE SHOULD GET DONE NEXT

> Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Fix `watermill/catchup_replay.go` watermark/cursor to raw form; add round-trip test | High | S | Bug |
| 2 | Audit + fix storage cursor args: `sql_aggregate_reader.go:105`, `command_store_journal.go:57`, `query_store_load.go:94` | High | M | Bug |
| 3 | Repo-wide class sweep: `\.String\(\)` feeding SQL args/wire/cursors/canonical bytes; fix all hits | Critical | M | Bug-hunt |
| 4 | Add regression test: v1-signed event FAILS v2 verification (pins the loud break) | High | S | Quality |
| 5 | Write ADR: branded-ID wire form is bare; display form is `String()` only | High | S | Documentation |
| 6 | Update `references/faq.md` (consumer-facing) with display-vs-wire rule + signing v2 migration | High | S | Documentation |
| 7 | Open PR for `cqrs-lint/a014-d013-scoped-fixes` (fix campaign + fmt wave) | High | S | Cleanup |
| 8 | Delete 8 verified-stale branches + prune 3 /tmp worktrees | Medium | S | Cleanup |
| 9 | #54: nil-receiver guards in otel + prometheus `Provider.Shutdown` + typed-nil tests | High | S | Bug |
| 10 | #56: document optional `io.Closer` shutdown convention on `event.Subscriber` | High | S | Documentation |
| 11 | #26: cut stack/postgres v4.4.2 tag; close issue | High | S | Release |
| 12 | #36: design `stack/metaengine` split | Medium | L | Feature |
| 13 | Decide gci removal from `.golangci.yml` (per AGENTS.md #18) and align or re-suppress | Medium | S–M | Decision |
| 14 | Budget review: mesh-demo (6>5), goal-shaped-app (11>10) — raise with rationale or shed deps | Medium | S | Decision |
| 15 | Run remaining gate scripts locally to verify the heal claims: file-size, actionlint+shellcheck, shfmt, check-arch via nix | High | M | Quality |
| 16 | Run `nix run .#verify-fast` as the official local gate on this branch | High | M | Quality |
| 17 | Triage PG/Redis integration job failures | High | M | Bug |
| 18 | Triage Nightly Gates + Sentinel nightly failures | High | M | Quality |
| 19 | Post-merge: coordinated release train (signing, watermill, scheduling/sqlstore, id, storage, schema, snapshot, commandlifecycle, cmd/cqrs-lint, +tidied) with REAL per-module test sweep — never verify=ok-only | Critical | M–L | Release |
| 20 | Notify known signing consumers (DiscordSync, cqrs-htmx, go-taskqueue) of canonical v2 before the tag ships | High | S | Process |
| 21 | Decide: legacy-verify opt-in flag for v1 signatures, or clean break only | Medium | S | Decision |
| 22 | Property test across ALL branded id types: `Parse(String()) == id` and wire forms bare | High | M | Quality |
| 23 | Decide defensive normalization for prefixed legacy bytes in snapshot decode (accept+strip vs reject) | Medium | S | Decision |
| 24 | Unify ULID-backed vs string-backed `String()` semantics in go-branded-id (marker `Name()` presence) or document the asymmetry | Medium | M | Feature |
| 25 | go-branded-id upstream: optional strip-on-unmarshal to kill the class at the root; then pin-sweep | Medium | L | Feature |
| 26 | Verify busfixture needs no cmd/cqrs-lint analyzer-catalog registration (meta-test check) | Medium | S | Quality |
| 27 | Run `check-duplication` after the 229-file fmt wave (clone shapes may have shifted) | Medium | S | Quality |
| 28 | Run `check-file-size` (no baseline growth from the campaign) | Medium | S | Quality |
| 29 | CI alerting: fail a job when master consecutive-failure count > N | Medium | S | Process |
| 30 | CI honesty decision: slim matrix to required-critical vs repair-to-green | Critical | S (decision) | Decision |
| 31 | HARVEST both status reports' (f) sections into TODO_LIST.md / ROADMAP.md | Medium | S | Process |
| 32 | Split-brain check: ActorID `PrefixedString()` vs branded `String()` — one documented rule | Medium | S | Documentation |
| 33 | Commit-phase-boundary discipline for next campaign (authored commits; daemon gets only mechanical waves) | Medium | S | Process |
| 34 | Record "ask when was CI last green first" + "read the test header before diagnosing" as standing lessons | Low | S | Process |
| 35 | Post-merge: `nix run .#verify` full gate once (official picture) | High | L | Quality |
| 36 | Add snapshot wire test asserting prefixed-bytes decode behavior is intentional (pin or reject) | Medium | S | Quality |
| 37 | Review `watermill/stream_id.go` parse robustness while the context is fresh | Low | S | Quality |
| 38 | Confirm remaining watermill metadata keys (correlation/causation — ULID-backed bare) are all genuinely bare | Low | S | Quality |
| 39 | Consider CHANGELOG bullet ordering convention (newest-first within sections?) and normalize | Low | S | Documentation |
| 40 | Sweep for other golden tests that emit `String()` into goldens (grep `String()` in `golden_test.go` files repo-wide) | Medium | S | Bug-hunt |
| 41 | Add `-short`-mode documentation to the soak test headers (discoverability of SOAK_SKIP_BOLT) | Low | S | Documentation |
| 42 | Make the multi-module sweep script (with SOAK env) a repo script so nobody re-discovers the bbolt trap | Medium | S | Process |
| 43 | Decide whether `docs/api_surface.txt` should gain the new test-visible symbols (busfixture exclusion interplay) | Low | S | Quality |
| 44 | Time-box retro: measure fix-campaign duration vs estimate; feed planning | Low | S | Process |
| 45 | Re-run `check-module-layers.sh` fully after budget decisions (13/14) to confirm the gate clears | Medium | S | Quality |
| 46 | Reconcile AGENTS.md #18 (gci removed) vs `.golangci.yml:832` (gci enabled) — whichever wins, update the loser | Medium | S | Documentation |
| 47 | Add the EventID/StreamID `String()` asymmetry note to the data-model-review skill notes | Low | S | Documentation |
| 48 | Sweep `example/` modules for the same `.String()` wire pattern (they build, but do they round-trip?) | Medium | M | Bug-hunt |
| 49 | Verify `check-readme-deprecated.sh` + `check-md-go` stay green after CHANGELOG edits | Low | S | Quality |
| 50 | Post-train: re-run the versions-manifest gate so the new tags land in `versions.json` + README matrix | Medium | S | Release |

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — Signing v2 consumer policy:** Is the loud break (v1 signatures no longer verify) acceptable for all known consumers, or do you want an explicit legacy-verify opt-in for one release? I cannot judge your consumers' re-signing appetite — DiscordSync was named in the issues as an active signing user, and its full-mode tests were the ones that surfaced the nil-panic issue (#54), so it is live on this code.

**Q2 — Scope of the class sweep:** Should the four known latent cursor/watermark sites (and the repo-wide `String()` sweep they imply) land in THIS branch before the PR, or in a dedicated follow-up branch? Fixing them here widens the diff but ships the class complete; parking them keeps the PR reviewable. I can argue both; only you can pick.

**Q3 — Release shape:** After merge, do you want the coordinated tag train (signing + watermill + scheduling/sqlstore + id + storage-family + schema + snapshot + commandlifecycle + the tidied modules), and does signing's canonical v2 warrant a MINOR bump convention note in the train (this repo's convention seems minor-bump-per-behavior-change)? The train composition and version numbering are yours to call; I can execute once decided.

---

## Session one-glance

| Metric | Value |
|---|---|
| Production bugs fixed | 4 (sqlstore lease args, watermill metadata, StreamIDFrom, signing canonical) |
| Modules failing before → after | 9+ → 0 (23-module sweep green; bbolt explained as documented soak) |
| Test sites re-pinned | 15 across 8 modules |
| go.sums tidied | ~20 modules |
| Formatting healed | 229 files |
| Gates newly passing locally | go.sum-tidy, module-dir registry, layer-entry, changelog-symbols, release-train declaration |
| Regressions introduced | 0 (all caught in-session: ULID Get() compile error, duplicate map key, changelog citations) |
| Open policy decisions for you | 3 (see g) |
