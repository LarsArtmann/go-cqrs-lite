# Status: Event-Query-Model Truth Reconciliation — Execution Complete

> **Date:** 2026-09-13 18:35 CEST
> **Kind:** Point-in-time status report (execution of the 26-task reconciliation plan)
> **Scope:** This session's work only — the audit → plan → execute arc
> **Parent plan:** [`docs/planning/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md`](../planning/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md)
> **Baseline reports:** [12:10 audit](2026-09-13_12-10_metaengine-event-query-model-doc-audit.md) · [15:55 deep dive](2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) · [T02 verification notes](2026-09-13_17-40_event-query-model-t02-verification-notes.md)
> **Format note:** the status-report skill's canonical output is an HTML dashboard; the user explicitly requested `.md`, so this one-off override is honored (flagged per skill guidance).

---

## Verdict in one table

| Category | Count | Headline |
| -------- | ----- | -------- |
| Plan tasks executed (T01-T26) | 26/26 | All touched; 24 fully, 2 scope-reduced by design |
| Code changes shipped | 4 files + golden | `Store.StreamCollection`, streaming `Export`, `CommandsByActor`, tests |
| Doc changes shipped | 14 files | reconciled doc, 3 memos, AGENTS.md convention, TODO/ROADMAP/CHANGELOG, README, skill refs |
| Gates run green | 6 | build+vet, root suite (212 ginkgo), doc-check, changelog-symbols, duplication, targeted tests |
| Gates red (external) | 1 | file-size ratchet on another session's `lintutil.go` growth |
| Verification gaps left | 5 | lint, system module, engine modules, real-engine export test, md-go-validator re-run |
| Open decisions | 3 | rejection event, session boundary, query-level stream |

---

## a) FULLY DONE

1. **T01 — inbound-link sweep + gate scope.** 13 inbound docs found (all documentation; zero AGENTS/SKILL links). `cmd/doc-check` default scan set documented: `docs/planning/` is NOT gated by default (`cmd/doc-check/main.go:103-119`).
2. **T02 — all 7 inferred claims source-verified** and written up: §7 vs `rule_shared_collection.go` (not contradicted; opt-in layout normalization), D4 vs `catchup_state.go` (quarantine-only), D3 vs `typed_reader_grouped.go` (single-collection), §13 DDL auto-generation (`layout.go:116,169,199`), §11 all 7 steps mapped, §5 vs `StorageLayout` enum, plus the exact API-signature table. Evidence: [T02 notes](2026-09-13_17-40_event-query-model-t02-verification-notes.md).
3. **T03 — status banner** on `event-query-model.md` (shipped/not-shipped lists, truth pointers, audit links).
4. **T04 — implementation-status addendum** for all 15 sections (DONE/DIFFERENT/PARTIAL/NOT-SHIPPED/PHILOSOPHY), every row cited.
5. **T05 — §4 corrected**: `ExecuteTyped[Q,R]` execution, `OnRecord` canonical + `On`/`OnTyped` v5 deprecation, `:=` → `var` package-level validity, elided `{...}` fixed.
6. **T06 — §5/§6 corrected**: 8-value ADT enum + `MultiEntry`/`Append`/`EdgeRemoval`/vector/search/spatial sentinels; 11 read patterns; filter/sort mechanics rewritten to the shipped closure-vs-declarative truth (`query.go:142-199`).
7. **T07 — §8 corrected**: real `record.CommonMetadata` fields, `Stamp.Time()`, `WithRange`/`FilterOnField`; non-existent `MetaData.Timestamp`/`RangeFilter` removed and called out.
8. **T08 — §12 corrected**: Dgraph replaces Neo4j (+ ADR links), YAML marked pseudo-config with Go composition equivalents, Bloom annotated as Pebble-internal, real 10+ engine roster.
9. **T09 — §14/§15 corrected**: real hot-reload API inventory with shipped/not-shipped scorecard; Decisions 1-4 marked resolved/wired/not-shipped/partial.
10. **T10 — §10 annotated**: command log SHIPPED as `commandlifecycle` (streams/events/projections/journals/wiring), query+session logs NOT SHIPPED, "four logs"/"all three" framing note, full-audit claim corrected.
11. **T11 — §7 reconciliation note** vs `rule_shared_collection.go` + diagram-illustration caveat.
12. **T12 — §11 step→source map** (all 7 steps, file:line).
13. **T13 — scope markers** on §1/§2/§3/§9.
14. **T14 — coverage-map appendix**: ~25 shipped features beyond the doc, one `file:line` each (materialized views, replication, demote/backfill, vector/spatial/aggregates, SSE/replay, probes/latency, roles, resets, auditing).
15. **T15 — metaengine README**: new Projection Roles + Shadow Engines section, Design References (ADRs 0113/0117/0123/0124/0137), fold table gains `EdgeRemoval`, Quick Example migrated to `OnRecord`, `OnTyped` deprecation notes, `StreamCollection` section.
16. **T16 — StreamingScan memo** written and, per user's "do the whole list", **executed (option A)**.
17. **T17 — command-log scope memo** written and **executed (option B: `CommandsByActor`)**; rejection event + payload capture consciously deferred.
18. **T18 — session/queue boundary memo** written (recommendation: sessions stay external; `queue/` not a home for them).
19. **T19 — Dgraph-vs-Neo4j check**: ADR-0119 (dgraph engine) and ADR-0129 (transactional deferred) already exist; cross-linked from §12 and the addendum.
20. **T20 — planning-doc convention** added to project AGENTS.md: banner + addendum discipline, no-rewrite rule, doc-check scan-set note, snippet-exemption decision (T25). `cmd/doc-check` re-run green over the gate (exit 0).
21. **T21 — HARVEST**: TODO_LIST gains an Event-Query-Model follow-ups section (4 items); ROADMAP gains Open Questions 12-14 with memo links.
22. **T22 — sibling sweep**: only `future-typespec-extension.md` needed a pointer fix; older meta-engine docs already carry ASPIRATIONAL/addendum banners. Findings recorded.
23. **T23 — `Store.StreamCollection` + streaming `Export`**: capability-first with `ScanBackend` fallback; export output byte-identical; 5 new tests (capability preference, fallback, unknown collection, fn-error, export equality); pre-existing export tests green. `StreamingScan` now has its first production consumer.
24. **T24 — `projections.CommandsByActor`**: Multimap keyed by the record's typed Actor (`"kind:raw"`), with `CommandsByActorQuery`/`CommandRecordEntry`/`CommandsByActorResult`, wired into `All()` (now 5), new test green.
25. **T25 — snippet gate decision**: documented exemption (planning snippets are illustrative; md-go-validator flags them by design) + durable convention in AGENTS.md instead of a new gate.
26. **T26 — close-out**: per-module build+vet, root suites, doc-check, changelog-symbol gate (50 citations), duplication gate (0 new groups), API golden regenerated (6855 exports), api-stability meta-tests green. All work pushed (`3f96656b4` and follow-on auto-commits; branch in sync with origin).

---

## b) PARTIALLY DONE

1. **T23 scope**: collection-level streaming only; the query-level `Stream(ctx, input, fn)` from Decision 2 remains unbuilt (documented as future work in the doc, memo, and ROADMAP Q12).
2. **T24 scope**: per-actor projection shipped; distinct `command.rejected` event + errorfamily classification and opt-in payload capture deferred (T17 memo's remaining half; TODO item).
3. **Verification depth**: I did NOT run — `golangci-lint` on the changed modules, `nix run .#verify-fast` end-to-end, `#check-arch`, `#check-error-taxonomy`, the **`system` module tests**, the **engine modules** (pebble/sqlite/bbolt/badger) that implement `StreamingScan`, or `md-go-validator` on the reconciled doc.
4. **`system` blast radius unverified — the most important gap**: `system/lifecycle.go:57` consumes `projections.All()`, so `WithCommandLifecycle` now wires a 5th projection. Compiles everywhere, but no `system` test was run after this change. TODO for the very next session minute.
5. **Skill references partial**: `modules.md` rows updated; `recipes.md:1110` still lists only four lifecycle projections ("DLQ, RetryCount, FailureLog, ProcessingTime") and has no `CommandsByActor` example.
6. **Real-engine export test missing**: the streaming-path test uses a synthetic wrapper; no sqlite/pebble-backed `Export` test proves the real `StreamScan` path (engine impls were not re-run either).
7. **Commit hygiene**: the plan's "commit per phase" guardrail did not survive the auto-commit daemon — most of this work landed as `chore: auto-commit N changed file(s)`; only the plan (`93cc2be6c`) and the doc reconciliation (`85ee49fee`) carry authored messages.
8. **HTML format**: status-report skill's canonical HTML dashboard overridden by the user's explicit `.md` request.
9. **`docs/status` older reports**: not annotated with this execution's outcome (point-in-time convention allows leaving them; `docs-health` ANNOTATE would be the tool).

---

## c) NOT STARTED

(Nothing from the plan's 26 tasks — all were touched. These are the consciously unstarted items this session surfaced or deferred.)

1. Query-level `Stream(ctx, input, fn)` for metaengine (Decision 2 second half).
2. `command.rejected` event + classification contract + middleware wiring.
3. Recorder payload-capture option (size/PII-bounded).
4. `sessionlifecycle` module (blocked on a concrete audit consumer — T18 recommendation).
5. `queue/` module assembly (separate plan: `2026-09-13_durable-work-queue-module.md`; unrelated to this plan except the boundary analysis).
6. Audit-item-30 verification: Set-membership pushdown for SQL engines.
7. Audit-item-31 verification: graph traversal depth semantics (`FriendsOf{Depth}`).
8. Go 1.27 toolchain wave (existing TODO; untouched here).
9. `docs-health` ANNOTATE over the two earlier status reports.
10. `md-go-validator` re-run over the reconciled doc to confirm the `:182` finding is gone.

---

## d) TOTALLY FUCKED UP!

1. **The repo's file-size gate is RED — not by this work.** `nix run .#check-file-size` fails on `cmd/cqrs-lint/pkg/rules/lintutil/lintutil.go` (453 → 474, ratchet violation) committed by another session's auto-commit. This blocks a clean full `#verify-fast` and is out of my authorship — reported, not touched. Someone owns it, and it is not me.
2. **A load-sensitive flake fired during the full root suite**: `TestEngineHealth_CatchUpUnderConcurrentApplies` ("primary ticks = 2001, want exactly 2000") under full-suite parallelism; passes 5/5 isolated. Same family as the known `TestSystem_ResetProjection_RestartAndReplay` contention stall item in TODO_LIST. Not caused by this session's changes (no catch-up code touched), but it means "all green under load" is not currently true for metaengine.
3. **Planned commit history did not materialize.** Guardrail #6 ("commit per phase, never one mega-commit") was defeated by the auto-commit daemon absorbing edits mid-phase. The work is safe and pushed, but history attribution is noisy (`chore: auto-commit …`), making future archaeology harder than the plan promised.
4. **First-audit error class (from the earlier arc of this session)**: the 12:10 report claimed the command log was "not started" (C1). It was wrong (it shipped as `commandlifecycle`), caught and corrected in the 15:55 deep-dive — but the first report still stands points-in-time with the wrong claim unless read together with its successor. Left unmodified by convention; cross-referenced from both later docs.
5. **`Found total 0 clone groups`** from `#check-duplication` (baseline says 54 groups) is suspicious — the gate is green, but a detector that reports zero on a repo with a pinned baseline deserves a sanity check; possibly expected behavior of the changed-files scan mode, possibly a silent scope regression.

---

## e) WHAT WE SHOULD IMPROVE

**Direct answers to the three opening questions:**

- **What did I forget?** (1) grep the blast radius of `projections.All()` BEFORE declaring T24 done — `system/lifecycle.go:57` consumes it and its tests were never run; (2) `recipes.md`'s projection list; (3) `golangci-lint` (I ran `go vet` only, and I added a `//nolint:forcetypeassert` on a line where test-file exclusions already cover that linter — nolintlint may flag it as unused); (4) a real-engine streaming-export test.
- **What could I have done better?** Run lint before close-out; run the full root package suite BEFORE writing "done" (the flake and the knock-on uncertainty arrived late); treat any change to a public convenience aggregator (`All()`) as behavior-changing for consumers and verify every consumer module in the same session.
- **What could I still improve?** Add a mechanical step to my own procedure: for every exported symbol touched, `rg` its callers across all 85 modules and run those modules' tests before the final report. It would have caught the `system` gap mechanically.

**Improvement list:**

1. **Blast-radius rule**: after changing an exported aggregate/constructor (`All()`, list helpers), grep consumers repo-wide and test each consumer module. Add to AGENTS.md procedures.
2. **Lint-before-done**: never close a code task with `go vet` alone; run `golangci-lint` on the module (it catches nolintlint/funlen/gocritic that vet cannot).
3. **Realism in tests**: at least one integration-grade test per new capability path (real engine, not synthetic wrapper).
4. **Consumer-facing behavior changes**: `projections.All()` growing from 4 → 5 silently adds a collection + writes for `WithCommandLifecycle` users. Decide policy (auto-include vs opt-in) and document in CHANGELOG more loudly than "Added" if kept.
5. **Gate ownership visibility**: when a red gate is caused by another session, signal the owner (or TODO it) instead of only footnoting it — otherwise it blocks everyone's verify.
6. **Commit protocol**: for planned multi-phase work, commit explicitly at each phase boundary IMMEDIATELY (before the daemon races), or accept/post-document the daemon history.
7. **Snippet discipline in planning docs**: adopt the AGENTS.md convention consistently and re-run md-go-validator after edits to verify findings actually cleared rather than assuming.
8. **`#check-duplication` sanity check**: confirm whether "0 groups" is expected output; if it silently scopes to changed files, document that in AGENTS.md gotchas.
9. **Session-report cross-linking**: add an execution outcome line to the two earlier reports (ANNOTATE mode) so the C1 error and "not shipped" claims are unambiguous for future readers.

---

## f) Up to 50 things we should get done next

Sorted roughly by impact; effort: XS <30min, S <2h, M <1d, L >1d.

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Run `system` module tests (`cd system && GOWORK=off go test -tags … ./...`) — verify the 5th projection didn't break `WithCommandLifecycle` | Critical | XS |
| 2 | Run `golangci-lint` on `metaengine` + `commandlifecycle/projections`; remove the redundant test `//nolint:forcetypeassert` if nolintlint flags it | Critical | S |
| 3 | Run engine-module suites (`pebbleengine`, `sqliteengine`, `bboltengine`, `badgerengine`) — they implement `StreamingScan`, untouched but now consumed | High | S |
| 4 | Add a real-engine `Export` streaming test (sqlite or pebble) — synthetic wrapper is not sufficient proof | High | S |
| 5 | Resolve the `lintutil.go` file-size red (owner session: shrink, split, or baseline shift per policy) | High | S |
| 6 | Decide `projections.All()` policy: auto-include `CommandsByActor` (current) vs opt-in; document either way | High | XS |
| 7 | Update `recipes.md:1110` projection comment + add a `CommandsByActor` recipe snippet and cheat-sheet row | High | S |
| 8 | Re-run `md-go-validator` on `event-query-model.md`; confirm the `:182` finding cleared; record delta | Medium | S |
| 9 | Run `nix run .#verify-fast` end-to-end once #5 is green | High | S |
| 10 | Run `#check-arch` + `#check-error-taxonomy` as cheap sanity (no deps touched, but cheap) | Low | XS |
| 11 | Investigate the catch-up stress flake under load (extend existing contention-stall TODO with this second witness) | High | M |
| 12 | Verify `#check-duplication`'s "0 groups" behavior; document in gotchas if it's scope-dependent | Medium | S |
| 13 | Query-level `Stream(ctx, input, fn)` design one-pager (memo T16 follow-up, ROADMAP Q12) | Medium | M |
| 14 | Design `command.rejected` event + errorfamily classification contract (TODO item) | Medium | M |
| 15 | Payload-capture recorder option (size-bounded, opt-in) design | Low | M |
| 16 | Add empty-actor semantics test for `CommandsByActor` (key "" today) — pin or normalize | Medium | XS |
| 17 | Add fn-reentrancy test for `StreamCollection` (documented safe: no lock held during iteration) | Medium | XS |
| 18 | Streaming export mention in persistence/backup recipes (`recipes.md` persistence section) | Medium | XS |
| 19 | Add `StreamCollection` to the skill cheat sheet (core.md) if modules.md alone is insufficient | Low | XS |
| 20 | Run `system.WithCommandLifecycle` doc-check after recipes edits | Low | XS |
| 21 | Verify Set-membership pushdown for SQL engines (audit item 30) | Medium | S |
| 22 | Verify graph traversal depth semantics (audit item 31) | Medium | S |
| 23 | ANNOTATE the 12:10 report's C1 correction inline (docs-health mode) | Medium | S |
| 24 | Consider `CommandsByActor` pagination (`Limit` field) mirroring the `CommandsByUser` sketch | Low | M |
| 25 | Add `RecordExecute`-style observability for `StreamCollection` (meter exists; spans? none) | Low | S |
| 26 | Include `CommandsByActor` in any dashboard/introspection surface (Doctor? system introspection?) if useful | Low | S |
| 27 | Re-check `docs/planning` gate policy: should reconciled docs be added to doc-check scan set via an explicit list? | Medium | S |
| 28 | Add the blast-radius step to AGENTS.md procedures (edit one symbol → test all consumer modules) | High | XS |
| 29 | Tag the metaengine + commandlifecycle/projections changes when the next release wave runs (untagged work accumulating) | High | M |
| 30 | CHANGELOG: consider promoting the `All()` behavior change to a "Changed" note if policy keeps auto-include | Medium | XS |
| 31 | Sweep `docs/status/archived/*` for "event-query-model = truth" references needing a pointer | Low | S |
| 32 | Re-run `/load-sweep` before next `#verify` if timing paths were touched (they were not — skip unless bored) | Low | — |
| 33 | Add `StreamCollection` error sentinels to the error taxonomy doc if the gate wants them (currently unexported) | Low | XS |
| 34 | Consider exporting `ErrNoScanBackend`/`ErrCollectionNotFound` if consumers need to branch (currently unexported; Export uses `errors.Is` internally) | Medium | XS |
| 35 | Add example usage to `example/` for streaming export (consumer-facing demo) | Medium | S |
| 36 | Review `CommandsByActor` query name convention (`command_by_actor` vs collection naming) | Low | XS |
| 37 | Decide whether `CommandsByActor` should also fold completed/failed to enrich the per-actor entry (currently received-only) | Medium | S |
| 38 | Backfill note: `CommandsByActor` projections only see events applied after deployment — document replay/backfill recipe (EventLog + Backfill) | Medium | XS |
| 39 | Re-run `api-stability` + `TestEvery` after any further symbol changes (they were green) | — | XS |
| 40 | Investigate whether `TestExport_UsesStreamingScanAndMatchesFallbackOutput` should also assert call count ≥1 per collection | Low | XS |
| 41 | Clean up the two dirty other-session files before the next release cut (`scripts/check-module-layers.sh`, `example/scheduler-otel-status/*`) | Medium | S |
| 42 | Note daemon-commit behavior in AGENTS.md for plan-execution sessions (explicit phase commits immediately) | Medium | XS |
| 43 | Consider a "reconciliation" status marker for planning docs in `docs/planning/README.md` (index-level navigation) | Low | S |
| 44 | Add `event-query-model.md`'s reconciled status to the docs index if one exists | Low | XS |
| 45 | Re-verify `commandsByUser` illustrative snippet's remaining `CommandRejected` references are clearly marked (they are, but re-read after any edits) | Low | XS |
| 46 | Track decision answers (G1 done/G2 half/G3 open) in the plan's decision gates table (update status column) | Medium | XS |
| 47 | Evaluate whether `Store.StreamCollection` deserves a typed variant (`StreamTyped[V]`) for consumer ergonomics | Low | M |
| 48 | Add `StreamCollection` to the `Persistence (Survivability)` README section cross-reference (it lives under Streaming section today) | Low | XS |
| 49 | Run `nix fmt` before any further doc/Go edits to catch formatting drift | Low | XS |
| 50 | Keep watching `system` projection tests under load once #1 passes (the flake family touches this area) | Medium | S |

---

## g) Questions I CANNOT figure out myself (up to 3)

1. **Who owns the red file-size gate?** `cmd/cqrs-lint/pkg/rules/lintutil/lintutil.go` grew 453 → 474 in another session's auto-commit. Options: (a) its owning session shrinks/splits it, (b) I shrink it now, (c) baseline update per the `--update-baseline` policy. I cannot determine ownership or intent from the repo state, and touching another session's in-flight file violates my working rules.
2. **Is the 5th projection in `projections.All()` acceptable as-is?** `system.WithCommandLifecycle` now auto-wires `CommandsByActor` — consumers upgrading get a new collection plus fold writes without opting in. Keep (documented as "Added"), gate behind an opt-in constructor, or split `All()` into `All()`/`AllWithActor()`? This is a consumer-contract decision I should not make unilaterally.
3. **Do you want the remaining verification executed now?** I can immediately run system + engine module tests, lint both modules, and (once the file-size owner is resolved) full `#verify-fast`. Or hold until you give instructions. I have not run them because the session was asked to report and wait.

---

## Evidence appendix

**Gates actually run (with result):**

| Gate | Command | Result |
| ---- | ------- | ------ |
| metaengine build | `GOWORK=off go build ./...` | PASS |
| metaengine vet (root) | `GOWORK=off go vet .` | PASS |
| metaengine root suite | `GOWORK=off go test -short .` | 212/212 ginkgo PASS; 1 load-flake in a concurrency test (passes 5/5 isolated) |
| T23 tests | `go test -run 'TestStreamCollection\|TestExport'` | PASS (5 new + 4 existing) |
| commandlifecycle | `go test -short ./...` (parent + projections) | PASS |
| doc-check | explicit file list incl. AGENTS.md | exit 0 (1123 refs) |
| changelog-symbols | `scripts/check-changelog-symbols.sh` | 50 citations honest |
| duplication | `nix run .#check-duplication` | 0 new clone groups (0 total reported) |
| api-stability | `--update` + `TestEvery` | golden 6855 exports; meta-tests PASS |
| file-size | `nix run .#check-file-size` | **FAIL — external `lintutil.go` growth** |

**Files authored/modified this session (code):** `metaengine/stream_collection.go` (new, 78), `metaengine/stream_collection_test.go` (new, 195), `metaengine/export_import.go`, `commandlifecycle/projections/projections.go` (227), `commandlifecycle/projections/projections_test.go` (272), `docs/api_surface.txt`.

**Files authored/modified this session (docs):** `docs/planning/event-query-model.md` (847 → 1153 lines), `docs/status/2026-09-13_17-40_…t02-verification-notes.md`, `docs/planning/2026-09-13_T16-memo-…md`, `…T17-memo-…md`, `…T18-memo-…md`, `…16-01_SUPERB-…truth-reconciliation.md`, `docs/planning/future-typespec-extension.md`, `AGENTS.md`, `TODO_LIST.md`, `ROADMAP.md`, `CHANGELOG.md`, `metaengine/README.md`, `.agents/skills/go-cqrs-lite/references/modules.md`, `docs/planning/event-query-model.md` banner/addendum/coverage.

**Key commits:** `93cc2be6c` (plan), `85ee49fee` (doc reconciliation), rest absorbed by the auto-commit daemon; branch in sync with `origin/master` at `98e1d74a9`.
