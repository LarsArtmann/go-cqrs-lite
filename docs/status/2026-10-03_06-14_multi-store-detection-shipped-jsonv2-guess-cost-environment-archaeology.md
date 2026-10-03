# Status Report: Multi-Store Detection + Memory Inference IMPLEMENTED and Verified; json/v2 API Guess Cost; Environment Archaeology Tax

- **Date:** 2026-10-03 06:14
- **Scope:** The implementation session ("GO FIX IT! PROPERLY!") — the 4-point multi-store proposal from the 03-44 report, executed step-by-step with per-step verification. Includes everything noticed along the way (concurrent-session collisions, repo go.sum rot, gate failures). Prior reports in series: `2026-10-03_02-12_…stale-untracked-binary.md`, `2026-10-03_03-44_…concurrent-edits.md`.
- **Tree at report time:** 1 dirty file (`query/go.mod` — concurrent session's; all my work absorbed by the auto-commit daemon).
- **Evidence-gap closure this run:** package-main targeted tests (JSONCLoader, FilterLibrarySelfLint, Explain, Preset) re-run green before writing this report.

---

## a) FULLY DONE

All 11 tracked steps completed, each verified at its boundary:

1. **`analyzer.StoreSpec`** (`store_spec.go`, 119 lines) — config `"store"` accepts scalar OR array; unknown names rejected at load with the valid list; doctor emits scalar-when-one. Implemented via encoding/json/v2 `UnmarshalJSON`/`MarshalJSON`. Pinned by `store_spec_test.go` (scalar/array/empty/number/object/bad-kind + marshal roundtrip + Primary/String).
2. **`ConfigFeatures.Store *StoreSpec`** (`presets.go`) with all downstream compile fixes.
3. **Multi-store model** (`feature_profile_stores.go`, 99 lines; `feature_profile.go`) — `Stores []StoreKind` field; `addStore`/`refineStore` mutators; `EffectiveStores` (filters unknown/none/empty-zero-value); exists-quantified `AnyStoreSQL`/`AnyStorePersistent`/`AnyStoreDistributed`; `String()` renders the union + the inference caveat; `ResolveFeatureProfile`/`ToConfigFeatures` wired for the spec.
4. **Detection restructure** (`feature_detect.go`, 316 lines) — pure `storeKindFromImportPath` + `storeKindForEngine` (moved to `feature_kinds.go` after hitting the 350-line edge at 349); every signal recorded into `Stores` while the primary keeps the historical first-seen selection; sqlite-driver fallback records too.
5. **Memory inference** — engine-less metaengine ⇒ `store: memory` + engines gain `"memory"`, grounded in verified facts (core init-registers the driver; engines register only via imports); in-app-custom-engine caveat in code comment AND doctor render.
6. **AST refinement hook** (`feature_detect_helpers.go`) — custom→concrete constructor refinement now updates `Stores` via `refineStore`.
7. **Rule migration** — F022, F023/F024/F025 (3 gates), C017, C036 all on exists-quantified predicates; orphaned `isPersistentStore` deleted. Correctness/adoption suites green.
8. **Doctor merge** (`doctor_profile.go`) — `Stores` union + `MetaengineEngines` dedup (fixes a latent duplicate-engine merge bug).
9. **Explain restructure** — `explain.go` shrank **542 → 413** (baseline allows shrink); new `explain_features.go` (187) with the features table + full STORE DETECTION contract docs.
10. **Tests** — 5 detection fixtures (engine-less inference, mixed pool, engine-only, no-backend, stack/memory overlap), predicate truth tables (6 cases), StoreSpec config override, JSONC loader e2e ×4 (scalar, array, rejection naming bad value + valid list, commented JSONC).
11. **Verification** — api-stability golden in sync (`StoreSpec`, `AnyStoreSQL`, `EffectiveStores` present at docs/api_surface.txt:698,707,751); `nix fmt`; changelog-symbols gate green (53 citations honest); CHANGELOG Added entry in repo style; file-size gate passes for ALL my files; T20-1 engine-mapping pin test green.

**Headline empirical result** (frozen-snapshot doctor, throwaway module cache):
`system/` and `metaengine/` flip `store: none → memory` with `engines: memory` + rendered caveat; `sqliteengine` stays `sqlite`; `duckdbengine`/`pebbleengine` doctor as memory-with-caveat (documented edge); mixed pools union correctly.

**Unblocked the repo twice**: repaired committed go.sum rot in `deriver/` and `system/` (mechanical `go mod tidy`) — the daemon had swept mid-edit states; the rot silently degraded `packages.Load` profiles repo-wide (root cause of the TestMultiModule "regression" that turned out NOT to be mine — proven via worktree archaeology at 92226ebff).

## b) PARTIALLY DONE

1. **Full-suite verification is targeted, not exhaustive.** All pkg/... suites + key package-main tests green (isolated worktree + final targeted run), but `nix run .#verify` (exclusive gate) NOT run — the tree was hot with concurrent edits all session. Effort: M when tree settles.
2. **`#check-duplication` not run** — new code (StoreSpec, predicates, fixtures) could carry novel clone shapes. Effort: S.
3. **Repo-level pin of the flip absent** — detection is pinned by fixtures, but no test pins THIS repo's system/metaengine doctor profiles (would have caught the go.sum-rot degradation class instantly). Effort: S.
4. **Engine self-detection deferred** (again, deliberately) — engine modules doctor as memory-with-caveat; the module-own-path seeding would fix it. Effort: M.

## c) NOT STARTED

1. Scorecard rendering of `Stores` (scorecard.go consumes `MetaengineEngines` only).
2. doctor `--evidence` mode (which import set each store signal).
3. v5 list-only config migration decision.
4. Foreign-session debts (see d-3): scanfixture gate registration (×3), stale.go ratchet + stale_test vet, taskmanager golden drift, repo-wide go.sum sweep — deliberately untouched to avoid collisions.
5. HARVEST of this report's (f) — third consecutive deferral per "report and wait".

## d) TOTALLY FUCKED UP

1. **Guessed the json/v2 API instead of reading it.** First StoreSpec used `UnmarshalFrom`/`MarshalTo` — the old GOEXPERIMENT names; the graduated stdlib renamed them (`UnmarshalJSONFrom`/`MarshalJSONTo`; the []byte-style interfaces are `UnmarshalJSON`/`MarshalJSON`). Two failed builds + one wrong-signature iteration before `go doc` settled it. The user's interjection ("READ, UNDERSTAND, RESEARCH, REFLECT.") was aimed at exactly this. The probe-test-first pattern was right; the doc-lookup-first pattern was missing.
2. **Shipped a precedence lie, caught by my own test.** I wrote "stack preset > engine import" into the explain docs (repeating a claim from the earlier design answer); TestDetectFeatures_MixedPoolRecordsAllStores disproved it — primary is FIRST-SEEN in visit order, no categorical ranking between signal kinds. Docs and test now state the truth. Lesson: assertions from a prior session's reasoning are hypotheses until a test pins them.
3. **(Attribution discipline held, but at a tax)** — I briefly treated TestMultiModule/F031/Taskmanager failures as possibly mine before isolating. The worktree-at-baseline pattern was correct but arrived only after two noisy live-tree rounds; the FIRST unexplained repo-wide failure should have triggered it (it eventually proved: CommandFlow failure = foreign go.sum rot, F031 = foreign suppression churn, Taskmanager = pre-existing fixture drift failing at pre-my-work baseline too).
4. **Left session artifacts behind**: `/tmp/cqrs-lint-v2/-v3/-final`, `/tmp/iso-modcache`, `/tmp/iso-gopath`, and my removed-worktree's binary (`/tmp/cqrs-lint-final` still exists). Harmless (tmpfs) but unowned artifacts are how sessions rot — same note as report 1, d-3.
5. **Missed verification window in the final output**: the last targeted test run's `head -8` truncated the package-main `ok` line, leaving the self-lint/explain/loader evidence unclosed at report time. Fixed by the pre-report re-run — but it should never have been truncated in the first place (verify, then eyeball the verification).

## e) WHAT WE SHOULD IMPROVE

1. **`go doc` before implementing against any stdlib API surface that moved recently** (json/v2 graduated this cycle). Cost of guessing: 3 iterations; cost of reading: 30 seconds.
2. **Every design claim carried across sessions is a hypothesis.** The first implementation step touching it must pin it with a test (the precedence chain is the case study).
3. **Isolation worktree at the FIRST unexplained cross-module failure** in a hot tree — the pattern (HEAD + rsync my module + throwaway GOMODCACHE) took ~10 min to assemble ad hoc and settled every attribution question; make it the reflex, not the third resort.
4. **Throwaway module cache from the start** when the tree is concurrent-edit territory — the shared redirected cache corrupted mid-session ("invalid package name" on otter/koanf) and cost a full debugging round.
5. **Never let truncated verification output count as verification** — the final-suite truncation gap repeated the report-1 class of "confident output, incomplete evidence".

## f) Next tasks (up to 50 — 24 new + carryover)

| #  | Task                                                                                                                                                                                    | Impact   | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Run `nix run .#verify` (exclusive) once the tree settles — full gate incl. lint/race/doc-check                                                                                          | Critical | M      | Quality       |
| 2  | Run `nix run .#check-duplication` (dirty-tree guard: commit-adjacent run)                                                                                                               | High     | S      | Quality       |
| 3  | Repo-level pin test: doctor profiles of system/metaengine stay `store: memory, metaengine: true`                                                                                        | High     | S      | Quality       |
| 4  | Engine self-detection: seed detection with the module's own import path (pkg.Module.Path) so engine modules report their real backend, not memory-with-caveat                           | Medium   | M      | Feature       |
| 5  | Scorecard: render `Stores` alongside engines                                                                                                                                            | Medium   | S      | Feature       |
| 6  | doctor `--evidence`: print the import/call site behind each store signal                                                                                                                | Medium   | M      | Feature       |
| 7  | doctor partial-load loudness: per-profile `partial` tag or hard-fail flag (this session re-demonstrated silent profile garbage under load errors — third occurrence)                    | High     | S      | Quality       |
| 8  | Register foreign `cmd/cqrs-lint/testdata/scanfixture` in the three gates (api-stability modules slice + exclusion maps, check-module-layers LAYER/DEP_BUDGET, cqrs-lint module catalog) | High     | S      | Cleanup       |
| 9  | Foreign stale.go shrank-to-fit (489→533 violates ratchet) + stale_test.go `finding.New` vet fix                                                                                         | High     | S      | Cleanup       |
| 10 | Taskmanager golden drift (C026 ×2 + F009 + count mismatch) — triage rule vs fixture                                                                                                     | High     | S      | Bug           |
| 11 | Repo-wide go.sum tidy sweep (only deriver + system repaired; TestEveryModuleGoSumIsTidy still red)                                                                                      | High     | M      | Cleanup       |
| 12 | `query/go.mod` dirty file — confirm the concurrent session owns/absorbs it                                                                                                              | Low      | S      | Cleanup       |
| 13 | Decide + document v5 list-only config (scalar deprecated?) at the v5 migration doc                                                                                                      | Low      | M      | Decision      |
| 14 | Engine-module doctor UX: suppress memory inference when the analyzed module path IS an engine dir (cheap complement to #4)                                                              | Medium   | S      | Feature       |
| 15 | CHANGELOG follow-up if #4 lands (engine self-detection semantics)                                                                                                                       | Low      | S      | Documentation |
| 16 | Explain: mention `stores` doctor line + scorecard rendering once #5 lands                                                                                                               | Low      | S      | Documentation |
| 17 | Config discovery upward-search decision (carried from report 1 f-6/f-7; now also affects multi-store pins)                                                                              | Medium   | M      | Decision      |
| 18 | Empty `monetary:` doctor line (carried, report 1 f-8)                                                                                                                                   | Low      | S      | Bug           |
| 19 | Stale untracked go1.26 binary `cmd/cqrs-lint/cqrs-lint` + committed-binary policy (carried, report 1 f-1)                                                                               | Medium   | S      | Cleanup       |
| 20 | `cqrs-lint version` prints build Go version (carried, report 1 f-2)                                                                                                                     | Medium   | S      | Feature       |
| 21 | HARVEST all three reports' (f) into TODO_LIST.md/ROADMAP.md via docs-health                                                                                                             | High     | S      | Documentation |
| 22 | Skill/reference sweep: does any skill reference document `store` config behavior that needs the multi-store update? (`rg "features.*store" .agents/`)                                   | Medium   | S      | Documentation |
| 23 | Consider a fixture-based golden for doctor's multi-store rendering (stores line ordering)                                                                                               | Low      | S      | Quality       |
| 24 | Session retrospective: add "isolation worktree + throwaway cache" recipe to docs/agents/gotchas-testing.md (concurrent-session verification pattern)                                    | Medium   | S      | Documentation |

**Carryover status from earlier reports:** report-1 f-13/f-15 pins superseded by #3 above; f-4/f-5 (engine self-detection) = #4; f-11 (evidence mode) = #6; f-3 (partial-load loudness) = #7. Report-2 items 1–21: #1 (implement) DONE this session; #2 (coordinate) resolved — their tree settled; #3/#5/#6/#7/#8/#9 DONE; #4 (verify system default engine) still open (below, Q2-adjacent); #10 = #3 above; #12/#13 (caveat hints) DONE in doctor render; #14 = #4/#14; #15/#16 DONE (explain); #17 engine-fold = #4; #18 (api-stability check) DONE (golden in sync); #19 (v5 timing) = #13; #20 scorecard survey = #5; #21 DONE (rules migrated); #22 (known-facts checklist) — recurred once more this session (d-2).

## g) Questions (cannot figure out myself)

1. **Verify gate timing:** run `nix run .#verify` now (tree is nearly clean — only `query/go.mod` dirty, presumably the other session's), or wait until that session signals completion? Verify runs exclusively and I cannot tell whether their session is finished or mid-task.
2. **Foreign-debt ownership:** the concurrent session left breakage (scanfixture unregistered, stale.go ratchet violation, stale_test vet failure, taskmanager golden drift, repo-wide go.sum drift, two prunable worktrees at /tmp/gcl-head-check and /tmp/pre-check2). Repair those as repo hygiene from THIS session, or leave strictly to theirs to avoid another mid-edit collision? (I can do either; ownership is the unclear part.)
3. **Engine self-detection priority:** with the memory inference shipped, engine modules (duckdbengine, pebbleengine, …) doctor as `memory` with a caveat. Should module-own-path seeding (#4) land as a fast follow, or is caveat-rendering acceptable until v5? Your call on urgency; technically it is a ~half-day change with per-engine pins.

---

_Point-in-time snapshot. Section (f) across three reports still NOT harvested into TODO_LIST.md/ROADMAP.md — third deferral, awaiting user instruction per docs-health HARVEST._
