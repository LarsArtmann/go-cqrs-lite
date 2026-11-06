# Status update — issue sweep #116 / #115 / #114 (metaengine Verify sentinels, PrefetchCache wiring, per-group caps)

**Date:** 2026-11-06 10:54 CET
**Branch:** `master` (all work absorbed by the auto-commit daemon; tree clean)
**Scope of this session:** the three open issues from `gh issue list`, end to end — read, diagnose, fix, verify, close.
**Session diff:** 34 files, +1,154 / −203 (`git diff --stat 254e4839c..HEAD`)

---

## a) FULLY DONE

### Issue #116 — `Verify`/`Backfill` failures are exported, classified, branchable

| What | Where |
| --- | --- |
| `ErrNoEventLog` (Rejection, `metaengine.no_event_log`) | `metaengine/errors.go:200` |
| `ErrNoQueryDecls` (Rejection, `metaengine.no_query_decls`) | `metaengine/errors.go:209` |
| `ErrCollectionCountMismatch` (Corruption, `metaengine.collection_count_mismatch`) | `metaengine/errors.go:217` |
| `ErrVerifyDrift` (Corruption, `metaengine.verify_drift`, still wrapped with live/replay counts) | `metaengine/errors.go:226` |
| `Store.EventLogAttached()` — ask the question without provoking the error | `metaengine/consistency.go:118` |
| `Store.Backfill` returns `ErrNoEventLog` instead of a silent `nil` | `metaengine/runtime_backend.go:156` |
| `replayableEvents()` — the one shared prologue (also killed a new clone group) | `metaengine/consistency.go:122` |
| `system.VerifyProjections` documents the composition-root no-op | `system/introspection.go:259` |

Design decision worth recording: I exported **all four** outcomes, not just the missing-log sentinel the issue named. With only `ErrNoEventLog` exported, the caller still could not tell `ErrCollectionCountMismatch` from `ErrVerifyDrift` — i.e. the actual "no-op vs real drift" branch the issue is about would have stayed broken.

Families are deliberate: Rejection for the two "the deployment has to change" cases (retrying cannot help), Corruption for the two drift cases (projection state no longer follows from its journal). This also removes the trap where a plain `errors.New` sentinel defaults to Transient and makes a structurally impossible `Verify` look retryable.

Tests: `metaengine/verify_sentinels_test.go` (5 tests: sentinel code+family table, no-op branching incl. non-retryability, drift branching, Backfill loud, Backfill empty-log no-op), the ginkgo Backfill spec flipped to the new contract, and `system/system_wiring_test.go` now asserts `errors.Is(err, metaengine.ErrNoEventLog)` instead of `_ = sys.VerifyProjections(ctx)`.

### Issue #115 — `PrefetchCache` is attachable, and its key is now correct

| What | Where |
| --- | --- |
| `metaengine.WithPrefetch(cache)` scan option (explicit `nil` = uncached for that call) | `metaengine/scan_options.go:170` |
| `scanConfig.prefetchCacheFor()` precedence rule (option > reader default) | `metaengine/scan_options.go:24` |
| `(*TypedReader).WithPrefetch` doc now states both forms + precedence | `metaengine/typed_reader.go:31` |
| `PrefetchCache` extracted to its own file, both attach forms documented | `metaengine/prefetch_cache.go` |
| `PrefetchCache.Len()` — the only "is it used" signal (no eviction) | `metaengine/prefetch_cache.go:55` |
| Shape-qualified cache key (collection + filters + sort + cursor) | `metaengine/typed_reader_cursor.go:113` |
| Cache read moved AFTER range/IN expansion so read shape == write shape | `metaengine/typed_reader_scan.go:108` |

**Correctness bug found beyond the filed issue:** the key was `collection + cursor` only. A page cached for `Status=open` could be served to a scan of `Status=closed` that shared a cursor value — silent wrong rows, no error, no metric. Now shape-qualified. Pinned by two tests that prove the mechanism rather than merely the row count: a cache hit is proven by **emptying the engine** before page 2 (rows can only come from the cache), and a shape mismatch is proven by the empty engine returning 0 rows.

**cqrs-lint F026 had two bugs, not one:**
1. Its detection looked for `metaengine.WithPrefetch`, so a project using the *real* `reader.WithPrefetch(...)` form was flagged anyway (correct code reported as wrong — the worst class).
2. An aliased import (`metaeng "…/metaengine/v4"`) evaded the rule entirely.
3. Additionally it fired on ANY `.Scan(`, including one-shot filtered scans, where a prefetch cache is a pure loss (it doubles the fetch to stage a page that is never requested). That is exactly why `example/taskmanager` was flagged.

Fixed: alias resolution (`import_names.go`), receiver-agnostic attach detection, pagination-only gate (`hasCursorPaginationCall`), and a suggestion that names the reader form (every v4 release) plus the option with its version (metaengine v4.21.0) instead of an API that no tag ever had. Goldens updated: `cmd/cqrs-lint/RULES.md`, `testdata/taskmanager_golden.txt` (F026 line gone), `pkg/rules/integration_test.go` profile (with the reasoning recorded).

### Issue #114 — verified, documented, closed (it had already shipped)

`WithCapPerGroup` / `EnforceCapGrouped` / `EnforceCaps` / `CapGroupTrimmer` pushdown (sqlite window DELETE, turso by delegation) / portable fallback with `ErrCapGroupPushdownUnsupported` / `adttest.AssertGroupedCapContract`, ADR-0164, CHANGELOG + capability matrix. Re-ran: 9 grouped-cap tests + sqliteengine + tursoengine contract suites, all green.

One genuine addition from this pass: `metaengine/adttest/cap_contract_test.go` runs **both** cap contract helpers against the reference memory engine. Reason: those helpers are executed from every engine suite, so a helper defect surfaces there and is blamed on the engine. This also repaired adttest's coverage floor (40.2 % vs a 43.8 % baseline → **44.7 %**), which had drifted when the #114 assertion code landed.

### Verification actually run (and green)

| Gate | Result |
| --- | --- |
| metaengine suite | ok (36.7 s) |
| metaengine/adttest, sqliteengine, tursoengine | ok |
| system suite (`-short`) | ok |
| cmd/cqrs-lint full suite (18 packages) | ok |
| scoped lint (`metaengine system cmd/cqrs-lint`) | 0 issues, 3/3 clean |
| `nix fmt` | 0 changed |
| file-size ratchet | no new offenders, no growth |
| package-surface ratchet | re-pinned once: 836 → 841 exports, 168 → 169 files |
| duplication (`art-dupl check -t 3`) | 0 new groups (2 accepted, 1 deduped) |
| cycles / arch / lint-config | green |
| untested-exports | 1,081 / 6,455 (was 1,086 / 6,423) |
| error-taxonomy + taxonomy-tests | 548 codes / 20 modules; 10 tested, 540 frozen |
| api-stability (with `-race`) | ok; golden +6 exports |
| changelog-symbols | 53 citations honest |
| md-go / ADR numbering / canonical-facts | green |
| recipes compile harness | ok (23 s) |
| doc-check (SKILL + references + AGENTS) | 1,385 references valid |
| test sleeps / skips / matrix / systemscenario adoption / turso citations | green |
| Issues #114 #115 #116 | closed with evidence comments |

---

## b) PARTIALLY DONE

1. **`check-coverage` is not fully green.** After my fixes the only remaining drift is `cmd/cqrs-upgrade` at **67.7 % vs a 74.7 % baseline** (module + package rows). I did not touch that module; the `--to-v5` feature landed 2026-11-04 without a re-baseline. Every other module/package is within tolerance. I deliberately left it rather than mask it with `--update` (see e7).
2. **`nix run .#verify` (the full composed gate) was never run.** The load guard refused (`load1 = 28.3 ≥ ceiling 10`, 37 users on this host) and I chose not to force it, because forcing a benchmark/timing-sensitive gate on a loaded host manufactures false reds. Instead I ran every verify leg that my change can affect, individually. This is a documented deviation, not a completion.
3. **The `#verify`-equivalent coverage of the *other* 80+ modules** (untouched by me) is unproven this session. My blast-radius search says nothing outside `metaengine`, `metaengine/adttest`, `system` and `cmd/cqrs-lint` consumes the changed surface, and I verified that by grep — but I did not run their tests.
4. **`example/taskmanager` could demo the prefetch pattern.** I removed its false F026 finding instead of giving it a paginated endpoint that uses `WithPrefetch`. The example is the consumer copy-paste surface, so a positive demo is a real gap (it has no cursor pagination at all today).

---

## c) NOT STARTED

1. `cmd/cqrs-upgrade` coverage repair (67.7 → ≥ 74.7) or an honest re-baseline with a reason.
2. A CHANGELOG line for the adttest contract self-test (test-only addition; I judged it below the release-note bar — arguable).
3. `nix run .#verify` in a quiet window (`nix run .#quiet-window-run -- nix run .#verify`).
4. A release tag for metaengine (v4.21.0) — the CHANGELOG now promises the option at that version; nothing is tagged yet.
5. Fleet adoption: webphone (the #114 consumer) can delete its two hand-rolled trim loops; nothing was done on their side.
6. `docs/agents/module-map.md` / `FEATURES.md` rows for the new sentinels — I judged CHANGELOG + taxonomy + faq + api golden sufficient.

---

## d) TOTALLY FUCKED UP

Nothing destructive, nothing reverted, nothing left half-written in the tree. But three honest self-criticisms:

1. **I overwrote the coverage artifacts mid-run.** My second `check-coverage` invocation (killed after it hung) rewrote `build/coverage/*`, so when I first read "REGRESS adttest 40.2" I was reading a *partial* run and briefly mistook it for a real regression signal. I recovered by measuring the single package directly — but I should have pinned the first run's log to a file instead of killing a measurement run.
2. **I created two clone groups and one gate failure of my own making** while fixing things (`Verify`/`Backfill` identical 8-line prologues; a digit-scan loop twin of `IsRuleID`), and only found them because I ran the duplication gate. Lesson reaffirmed: run the ratchets after each structural edit, not at the end.
3. **First `dx.go` edit pushed a baselined file over the line ratchet** (443 → 472). The fix (extract `prefetch_cache.go`) is better structure, but it forced a package-surface re-pin in the same session. I should have checked `scripts/file-size-baseline.txt` *before* adding 29 lines to a baselined file.

---

## e) WHAT WE SHOULD IMPROVE

1. **The ratchets exist; check them FIRST, not last.** `file-size-baseline.txt` and `package-surface-baseline.txt` should be part of the pre-edit checklist for any file in `metaengine` (the repo's biggest package, baselined in both).
2. **Coverage floors do not follow feature work.** `cmd/cqrs-upgrade` proves the pattern: a feature lands, its tests cover 67 % of new surface, nobody re-measures, and the gate goes red weeks later for a stranger. A post-merge coverage check per changed module would catch it at the source.
3. **`check-coverage --profile` is misleading.** Feeding it an existing `merged.out` reported `none`/DRIFT for every package (it evidently needs `module-coverage.txt`/`pkg-coverage.txt` too). A `--profile` mode that silently mis-reports is worse than not having one.
4. **`Verify`'s API shape invited the bug.** A method that can only end in "cannot run" is a design smell; `EventLogAttached()` is the patch, but the v5 shape is probably `Verify` on a store that always has a log (the composition root owning one).
5. **F026's original implementation shows a rule can ship with a suggestion that never existed.** Any rule whose suggestion names an API should be compile-checked against the tag it claims (the repo already does this for doc fences — same machinery could cover rule messages).
6. **Prefetch cache eviction.** `PrefetchCache` still only grows until `Clear()`. `Len()` makes the leak observable; it does not bound it. A page budget or LRU is the honest fix.
7. **`addressed`-style comments in goldens.** Three goldens encode *why* a finding left the list; that discipline saved me from "why did F026 disappear?" archaeology. Worth making a rule.
8. **The `verify` load guard vs a busy multi-session host.** On this host `#verify` is effectively unusable in the daytime. Either the ceiling needs a per-host tier or the composed gate needs a documented "structural legs only" profile (which is essentially what I ran).
9. **Batch-verify of touched modules.** A `nix run .#verify-modules -- a b c` (build+vet+test+lint for a module set) would have given me the composed guarantee in ~5 minutes instead of four ad-hoc invocations.
10. **Issue-comment voice.** My comments are evidence-dense and long; a shorter body plus a linked commit range would serve a future reader better.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Correctness / follow-through on this session**
1. Repair or re-baseline `cmd/cqrs-upgrade` coverage (67.7 vs 74.7) — the only red gate on master right now.
2. Run `nix run .#quiet-window-run -- nix run .#verify` and record the result; delete this caveat.
3. Tag `metaengine/v4.21.0` (and `cmd/cqrs-lint` next minor) so the CHANGELOG's version promise is real.
4. Add a paginated endpoint to `example/taskmanager` that demonstrates `WithPrefetch` end-to-end (removes the F026 finding by *showing* the pattern, not by narrowing the rule).
5. Tell webphone their two hand-rolled per-owner trim loops can go (`WithCapPerGroup`).

**Prefetch / typed-reader surface**
6. Bound `PrefetchCache` (page budget or LRU) instead of relying on `Clear()`.
7. Consider `WithPrefetchSize(n)` or `NewPrefetchCache(WithCapacity(n))` rather than an unbounded map.
8. Document/decide whether `PrefetchCache` should coalesce concurrent identical page requests (singleflight, like `decider.Repository`).
9. Cache-hit/miss counters (`PrefetchCache.Stats()`), so operators can see whether the cache earns its doubled fetch.
10. Decide whether a cache hit should still validate against the engine's `hasMore` (today the cached page trusts itself).

**Verify / consistency surface**
11. Post-v5: fold `EventLogAttached` away by giving the store a mandatory log in the composition root.
12. Make `Verify` return a structured report (per-collection live/replay counts) instead of failing on the first mismatch.
13. Consider exposing "verify could not run" as a health status rather than an error (`System.Health()` already exists).
14. Add a `Verify` conformance case to `systemtest` so the composition-root no-op is pinned there too, not only in `system`.

**Lint rules**
15. Add a meta-test that every rule *suggestion* naming `metaengine.X` resolves in `docs/api_surface.txt` (kills the F026 class).
16. Audit the other adoption rules for the same alias-blindness my new `importLocalNames` fixed (F018/F019/F021 use `hasCallIn` with hardcoded `metaengine`).
17. Audit rules that fire on one-shot APIs vs paginated ones (the F026 precision class).
18. Extend `hasCallToNameAnywhere` adoption where a method form exists (e.g. `.Scan(` vs `.ScanPage(`).

**Coverage / gates**
19. Fix `check-coverage --profile` to consume a merged profile correctly, or fail loudly instead of reporting `none`.
20. Add a per-changed-module coverage check to the PR/CI path (catch the next cqrs-upgrade early).
21. Re-examine the verify load-guard ceiling for multi-session hosts.
22. Add a `verify-modules` flake app (build+vet+test+lint for an explicit module set).
23. Have `check-file-size` print the headroom for baselined files (`443 → 404, 39 lines free`).
24. Have `check-package-surface` print remaining headroom per baselined package, so an edit knows its budget before it lands.

**Docs**
25. Add the Verify-sentinel table to `references/modules.md` (currently only `faq.md` + code docs).
26. Add a `docs/agents/gotchas-*.md` line: "`Verify`/`Backfill` are structural no-ops under `system.New` — branch on `ErrNoEventLog`".
27. Document the shape-qualified prefetch key in `readmodels.md` (cursor pagination is described there too).
28. FEATURES.md: record the new sentinel surface under metaengine's row.
29. Sweep `docs/` for other "Verify is meaningless" folklore that the exported sentinels now obsolete.

**Release / process**
30. Decide whether the CHANGELOG should list test-only additions that repair a ratchet (the adttest case).
31. Add a pre-commit hook reminder when a baselined file grows (file-size + package-surface both).
32. Codify the "golden removal needs a reason comment" convention in CONTRIBUTING.md.
33. Add this session's `check-coverage` incident (killed run poisoning artifacts) to the testing gotchas.

**Broader metaengine**
34. Revisit `pkg/cqrs-lint` F026's confidence now that it is precise (info/low may undersell it).
35. Consider a `PrefetchCache` interop test against sqliteengine (only memory is covered for the option path).
36. Check `tursoengine` inherits the scan-option prefetch behaviour (it delegates, but no explicit test).
37. Add a bench for the prefetch hit path (is 2× fetch + decode actually cheaper than another round-trip on SQL?).
38. Consider whether `EnforceCapGrouped` should accept a `groupCol` override (per-query multi-tenancy).

**Repo hygiene**
39. `docs/status/` is accumulating; the ARCHIVE convention in `docs/status/README.md` may want a monthly roll-up.
40. Two `ErrNoDatabase` sentinel-name attributions vanished from the regenerated taxonomy — check whether erraudit's name attribution regressed or the committed doc was hand-attributed.
41. The taxonomy regen this session also fixed *pre-existing* drift (`storage.open_turso_sync` was missing) — a nightly `--check` on that generator would have caught it.
42. `scripts/import_names.go`-style helpers deserve a `ruletest` package-level test (alias matrix) rather than only rule-level tests.
43. Re-check `cmd/cqrs-lint/README.md` rule counts and the `meta_test.go` detector count after this change (unchanged rule count, but worth a drift assertion run).
44. Consider an ADR for the shape-qualified cache key (it is a behavioural contract, currently only in code comments + CHANGELOG).
45. Re-verify `docs/engine-capabilities.md` regeneration after the `cap_grouped.go` comment edit (capability column untouched, but the generator probes source).
46. Check `metaengine/MIGRATION.md` for pre-v5 text about `Verify`/`Backfill` semantics.
47. Sweep for other unexported sentinels returned by exported methods (`errSwapEngineNotFound`, `errScanTruncated` is already exported) — the #116 class is general.
48. Consider exporting `ErrCapGroupPushdownUnsupported`'s family doc in `docs/error-taxonomy.md` prose (it is a documented code already).
49. Add a `Store.Backfill` variant that returns the number of replayed events (today it returns only an error).
50. Have the auto-commit daemon annotate commits with the issue numbers from CHANGELOG entries, so `git log` links the sweep.

---

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **`cmd/cqrs-upgrade` coverage (67.7 vs 74.7 baseline): repair with tests, or re-baseline with a reason?** It is the only red gate on master and it is not mine — but I can spend the next block adding tests for the `--to-v5` rewrite internals if you want the floor to keep pointing up.
2. **Should `example/taskmanager` grow a cursor-paginated endpoint that demonstrates `WithPrefetch`?** That is the only way the F026 finding disappears *by showing the pattern* rather than by narrowing the rule, but it changes the canonical example's HTTP surface.
3. **Do you want the composed `nix run .#verify` run in a quiet window tonight (and the caveat removed), or is the per-leg evidence in section (a) sufficient for this change?** The load guard refuses right now (`load1 = 28`).
