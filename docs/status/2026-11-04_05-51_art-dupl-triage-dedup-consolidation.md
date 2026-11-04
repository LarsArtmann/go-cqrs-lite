# Status Report: art-dupl Triage & Dedup Consolidation Session

**Date:** 2026-11-04 05:51 CET
**Session scope:** Respond to three pasted `art-dupl` outputs (text `-t 5` with directive warnings; SARIF `-t 3`), triage all flagged clone groups, extract harmful duplication, fix gates that were red on master, and record what was learned.

---

## Scope Line — what this report VERIFIED vs NOT

**VERIFIED in-session (command → result):**
- `art-dupl check . --threshold 3` → **green** ("No new clones detected", baseline 189; was **red** with 2 novel groups at session start)
- `art-dupl --sort total-tokens -t 3 --sarif` (parsed) → **12 distinct groups before, 9 after** the extractions; zero directive-placement warnings anywhere
- `nix run .#check-file-size` → **green** (was **red**: `cmd/cqrs-upgrade/main.go` at 357 lines, NEW offender, pre-existing on master — not authored by this session)
- `cd cmd/doc-check && GOWORK=off go run . …AGENTS.md` → exit 0, all 1372 references valid
- Full test suites: `metaengine/duckdbengine` (158.6s, ok), `metaengine/adttest` (ok), `cmd/cqrs-upgrade` (13.1s, ok), `stack/memory` contract tests (ok), `stack/bbolt` contract tests (ok)
- `go vet` on 10 modules, `golangci-lint` on 10 packages (**0 issues** each), `gofumpt -l` clean on every touched file

**NOT verified (and why):**
- `nix run .#verify` / `#verify-fast` (composed gate incl. race) — too heavy for the session window; individual gates run instead
- Test suites of the 6 engine modules whose *_test.go files I edited (badger/bbolt/pebble/mysql/pg/dgraph) — they need DB backends; changes are Skip-guarded dump functions + comment removals, so compile (vet+lint) was the practical bar. **Runtime unproven.**
- `stack/pebble`, `stack/duckdb`, `stack/postgres`, `stack/mysql` contract-suite consumers — only `stack/memory` + `stack/bbolt` were actually run
- Race detector on touched packages — not run
- CI/nightly status for master — not investigated (network/gh out of session scope)
- Final re-run of the user's exact original command (`-t 5 --timing` text view) — only `-t 3` check + SARIF re-verified at the end
- `CHANGELOG.md` / `FEATURES.md` — intentionally not updated: every change is unexported/internal, nothing user-visible

---

## a) FULLY DONE

1. **7 inert `//art-dupl:accept` directives fixed.** Six calibration dump tests (badger/bbolt/pebble/mysql/pg + dgraph's `calibration_pin_test.go`) carried doc-comment directive copies 8 lines above the flagged range — outside the 5-line window, therefore never suppressing anything. Removed the inert copies (the working in-body trailing ones remain). `mysqlengine/pushdown.go`'s directive relocated into the body directly above the flagged statement.
2. **adttest clone eliminated.** `value_determinism.go`/`value_encode.go` duplicated the MapBackend skip-guard → extracted `requireMapBackend` into new `metaengine/adttest/requirements.go`.
3. **contracttest clone eliminated.** Factory-error guard shared by `newBundle`/`testFullRouting` → generic `requireFactory[T]` in `stack/contracttest/contract.go`, plus `newMultiDBTest` in `multidb.go` (no Cleanup by design — the multi-DB suite closes the bundle explicitly as part of the routing proof; documented).
4. **duckdbengine init-block clones eliminated (5 sites).** `aggregatePlanned`, `groupedAggregatePlanned`, `multiAggregatePlanned`, `multiGroupedAggregatePlanned`, `distinctPlanned` each repeated `var b strings.Builder` + `initialArgs(...)` + `initialArgIndex(plan)` → new `plannedQueryInit` helper returning `(*strings.Builder, []any, int)`, mirroring the pre-existing `stdQueryInit` pattern. File shrank 654 → 649 lines (baseline 722, shrink allowed).
5. **The `-t 5` expression artifact resolved.** Group `(title, description, docsPrefix, nonce string)` / `(brand, …)` / `sortIndexKey(col, field, encodedValue, primaryKey string)` is an alpha-normalized false positive (four-string parameter lists, unrelated functions). Accepted via in-tree directive with rationale on `metaengine/pebbleengine/sort_index.go:26` — `.templ` files cannot carry directives, so the Go participant carries it.
6. **Master's red file-size gate fixed.** `cmd/cqrs-upgrade/main.go` 357 → 311 lines by moving `printReport`/`printBumps`/`emitJSON` into `report.go` (264 lines) where the rest of the reporting concern already lives. Full module suite green after the move.
7. **Memory recorded.** AGENTS.md contract 14 gained "FOURTH semantics (2026-11-04)": directives >5 lines from the flagged range are inert + warned; plain report mode and `--sarif` ignore the committed baseline (they list groups the `check` gate accepts — 9 at `-t 3` today), so triage SARIF findings against `art-dupl check` before extracting. doc-check re-run green.
8. **All touched code verified:** vet ×10 modules, golangci-lint ×10 packages (0 issues), gofumpt clean, duckdb/adttest/cqrs-upgrade/memory/bbolt suites green.

## b) PARTIALLY DONE

1. **Triage of the 12 SARIF groups:** 3 extracted (above), 1 accepted in-tree, **9 accepted by judgment only** — cross-module dep-isolated pairs (pg↔sqlite aggregations, bbolt↔pebble `GracefulClose`, middleware↔projectionhost DLQ, iroh transport trio) and idiomatic lock-read accessors (`memory_stream_log.go` ×3, `system/introspection_extended.go` trio). The acceptance reasons exist only in this report and the baseline JSON — **no in-tree `//art-dupl:accept` rationale** for those 9 (consistent with repo practice, but the "why" is not where the code is).
2. **Consumer verification for contracttest changes:** `RunSuite`/`RunMultiDBSuite` consumers compiled everywhere and ran for memory + bbolt only. pebble/duckdb/postgres/mysql/sqlite/turso preset suites unrun (DB/container prerequisites).
3. **Gate parity:** the gate uses `-t 3`, the user's scans use `-t 5`; my final verification ran only `-t 3` + SARIF. The exact `-t 5` view was not re-run after the last edit.
4. **Test coverage for the new helpers:** `requireMapBackend`, `requireFactory`, `plannedQueryInit`, `newMultiDBTest` have no dedicated micro-tests — they are exercised only indirectly through the suites that passed. For test-infra packages that is arguably fine, but a fake-engine skip test and a factory-error Fatalf test would pin the contracts cheaply.

## c) NOT STARTED

1. **HARVEST** of section (f) into `TODO_LIST.md`/`ROADMAP.md` (docs-health) — deliberately deferred per your "then wait for instructions" (the 07-37 entombment lesson says a timestamped report is not a backlog home; this is the one open obligation from the Close-a-Wave procedure).
2. Accept-directive pass over the 9 remaining groups (in-tree rationales).
3. Assessment of the iroh closed-guard trio (`closed := t.closed` under RLock, three packages, three distinct sentinel errors) for a shared helper — it is in the same module, so extraction is actually feasible; I accepted it without that analysis.
4. Upstream report to art-dupl: expression-level parameter-list false-positive class + SARIF-ignores-baseline semantics (`verify-before-filing` gate required first — not started).
5. Race-run and `-t 5` parity re-run (see b.3/b.4 scope gaps).
6. CHANGELOG/FEATURES — skipped by decision (nothing exported changed); revisit if you disagree.

## d) TOTALLY FUCKED UP

Nothing this session broke — but the session **found** two things that were already fucked on master:

1. **Master was red on two gates when the session started.** `art-dupl check` failed with 2 novel clone groups, and `check-file-size` failed on `cmd/cqrs-upgrade/main.go` (357 lines). Since `check-file-size` is wired into `#verify`/CI since 2026-10-26, a red master implies either CI didn't run/loud-fail on the offending commits, or the failure was ignored. Both are now fixed here — but **the monitoring gap itself is unfixed**: nothing alerted anyone.
2. **Seven shipped directives never worked.** The calibration dump tests' doc-comment directives were inert from the day they were added (window rule), silently riding on the in-body copies or the baseline. No mechanism flags an `art-dupl:accept` that suppresses nothing — the tool only warns at scan time, and nobody scans with warnings in view. Silent no-op safety annotation is a false-security pattern.
3. **Auto-commit waves bury multi-part work.** This session's changes landed as `chore: auto-commit` waves (expected per ground rule 4, and the T03.2 ruling authorizes hand-commits at phase boundaries on commissioned plans — which I did not invoke). Reading this session's history later means reconstructing intent from a status report instead of commit messages.

## e) WHAT WE SHOULD IMPROVE

1. **Gate health monitoring.** A nightly fail-loud audit ("list every `check-*` gate and its last-green commit") would have caught both red-on-master gates within a day. The README/versions nightly gates exist; gate-the-gates does not.
2. **Directive hygiene.** After ANY `art-dupl:accept` edit, immediately re-run `art-dupl check` to prove suppression (I did this — but only after first writing a directive that then needed empirical testing; make it reflexive). Better: a tiny `--lint-directives`-style check upstream that fails on inert directives.
3. **Accept = rationale in-tree.** The 9 accepted groups should carry `//art-dupl:accept <reason>` at the region (Go participants), so the judgment survives next to the code instead of in a JSON blob.
4. **Blast-radius discipline for shared test-infra.** When touching `stack/contracttest`/`adttest`, run at least `-short` of every preset consumer, not two.
5. **Hand-commit at phase boundaries** (T03.2): directive-fix phase, extraction phase, gate-fix phase each merited a real commit message.
6. **Detector feedback loop.** The param-list expression group is a recurring false-positive class; the SARIF-ignores-baseline behavior burned triage time. Both belong upstream (art-dupl) — with source-level verification first.
7. **duckdb aggregations.go continues to shrink** (649 lines vs 350 target): the natural next seam is a small `plannedQuery` builder struct (`b`, `args`, `argIdx`, `whereStarted` travel together; `appendDuckDBFilter` signature simplifies), and eventually the standard/planned split into two files.
8. **`stdQueryInit` vs `plannedQueryInit` asymmetry** (standard path still declares its builder locally) is now the file's largest remaining self-inflicted shape-similarity — unify or document why.

## f) NEXT — up to 50 items (session-derived; marked [R] = ROADMAP fuel, most are brainstorm not commitment)

**Immediate (this week, high impact / low effort)**
1. HARVEST this report into `TODO_LIST.md` (docs-health HARVEST mode).
2. Investigate why red-on-master gates went unnoticed: check last CI/nightly runs for `check-duplication` + `check-file-size`; wire a nightly gate-health audit if the answer is "not covered".
3. Re-run the user's exact original command (`art-dupl --sort total-tokens -t 5 --timing`) end-to-end and confirm zero warnings + only the accepted groups.
4. Run `-short` suites of the 6 engine modules whose test files changed (badger/bbolt/pebble/mysql/pg/dgraph).
5. Run remaining contracttest consumers (`stack/pebble`, `stack/sqlite`, `stack/turso`, `stack/postgres`, `stack/mysql`, `stack/duckdb`) at `-short` where backend permits.
6. Race-run `metaengine/adttest`, `stack/contracttest` consumers, and `metaengine/duckdbengine` once.
7. Add micro-tests: fake engine without MapBackend → `requireMapBackend` skips; erroring factory → `requireFactory` Fatalf (pins both contracts against silent drift).
8. Add `//art-dupl:accept <reason>` directives for the 9 accepted groups (Go participants only).
9. `nix run .#check-test-runtime` after the 158.6s duckdb suite — confirm no STALE row.
10. `nix run .#verify-fast` once to certify the composed gate end-to-end.
11. Confirm CI's duplication job flags match the flake app (`--threshold 3 --semantic`).
12. Sweep the repo for OTHER inert `art-dupl:accept` directives (script: parse directives, run art-dupl, flag any that suppress nothing) — the calibration pattern may recur.

**Short-term (quality follow-through)**
13. Unify `stdQueryInit`/`plannedQueryInit` into one query-init seam in duckdbengine. [R]
14. Introduce a `plannedQuery` builder struct in duckdbengine (b/args/argIdx/whereStarted together; simplifies `appendDuckDBFilter`). [R]
15. Split `duckdbengine/aggregations.go` standard vs planned halves as it shrinks toward 350. [R]
16. Assess the metaengine-core pull-up for the pg/sqlite aggregation init block (contract 27 precedent: shared scan plumbing already lives in core). [R]
17. iroh closed-guard trio: same-module extraction assessment (accept or extract with per-package sentinels passed in). [R]
18. `system/introspection_extended.go` trio: decide closure-helper vs documented accept. [R]
19. `memory_stream_log.go` triple: same decision pass. [R]
20. `record.GracefulClose` assessment — deliberately deferred while storage/* is v5-sentenced; record the revisit trigger (third backend needing it). [R]
21. Prune resolved entries from `.art-dupl-baseline.json` (189 → post-consolidation count) via surgical deletion, keeping the shrink-only ratchet honest. [R]
22. docserver `layout.templ` `docsPageProps`/`docsNavProps` near-clone: explicit accept/extract verdict (likely accept — different return types — but it was never explicitly decided).
23. Upstream: file art-dupl issue for the expression-level param-list false positive (verify-before-filing gate first).
24. Upstream/decide: SARIF honoring the baseline or a `--respect-baseline` flag.
25. Decide repo stance on SARIF consumers (is anything wired to consume `--sarif`? If not, say so in AGENTS.md to prevent future confusion).
26. Standardize the documented dedup threshold (gate=3, ad-hoc scans=5) in AGENTS.md contract 14.
27. Naming pass over the four new helpers (`requireMapBackend`, `requireFactory`, `plannedQueryInit`, `newMultiDBTest`) with the naming-review lens — names were chosen fast.
28. CHANGELOG decision: internal-quality waves — add a one-line `[Unreleased]` note or explicitly adopt "unexported-only changes skip CHANGELOG" as written policy.
29. `check-canonical-facts.sh` was edited by a parallel session today (backtick-stripping) — re-run `nix run .#check-canonical-facts`-equivalent gates to confirm no interaction with my AGENTS.md count claims.
30. Decide whether `newMultiDBTest` should register Cleanup given `CloseIdempotent` is a contract — current "no Cleanup" is intentional and documented; confirm owner agrees.

**Mid-term / adjacent hygiene**
31. Run `nix run .#check-untested-exports` + `check-package-surface` to certify no exported surface moved (expected: none).
32. Adopt `art-dupl --incremental` for session loops (10–16s per full scan today).
33. Consider a committed `docs/status/` index or prune policy — reports are accumulating (two today from parallel sessions).
34. Cross-reference the two gosec status reports (parallel session) — if they cite gates this session touched, annotate.
35. Gate-the-gates: extend the nightly audit (item 2) to enumerate ALL `check-*` apps with last-green SHA.
36. Add "run `art-dupl check` after every directive edit" as an explicit line in AGENTS.md contract 14 (currently implied).
37. Evaluate art-dupl `--explain` output in future triage (used once; the type/category/explainability fields made the 12-group triage fast).
38. Time-boxed: try `--type-aware` on the 9 accepted groups to see whether any dissolve (same-name methods on different types are a false-positive class).
39. If (38) dissolves groups, prefer `--type-aware` in the gate and re-pin. [R]
40. Bench-check `plannedQueryInit` escape-to-heap (returns `*strings.Builder`) against the old value-builder — aggregation paths are not hot, but the repo has alloc pins; confirm none fire.
41. Write the templ-side suppression decision down (AGENTS.md contract 14 already says templ can't carry directives; the pebble-carried directive for a templ-involving group is a NEW pattern — document it).
42. Hand-commit policy: pre-agree which session types count as "commissioned plans" under T03.2 so agents stop guessing.
43. Add a `docs/status/README.md` note that HARVEST must run within the same session as any report with an (f) section (the entombment rule, made mechanical).
44. Re-verify one inherited claim per session against source (brutal-review Q12) — candidate: "check-file-size is wired into #verify/CI since 2026-10-26" (AGENTS.md claim; today's red master suggests verifying the wiring claim itself).
45. Confirm the adttest module's dep budget unchanged after `requirements.go` (same imports, but the module-map row may cite file lists).
46. Consider extracting `requireFactory` to a shared internal testutil package IF a third suite needs it (YAGNI now — note the trigger).
47. Re-run `nix run .#check-arch` for adttest/contracttest (no new imports; certify formally).
48. Add the duckdb 158.6s datapoint to the test-runtime baseline conversation if CI machines diverge from local.
49. Purge `-t 5` vs `-t 3` confusion in future session prompts: record "always verify with the gate's exact flags" in AGENTS.md gotchas.
50. Close-out ritual: `nix run .#check-adr-numbering` — no ADR was added this session; run anyway if any parallel session added one.

## g) Questions I cannot answer myself

1. **Why did red gates on master go unnoticed?** `check-file-size` is documented as wired into `#verify`/CI since 2026-10-26, yet master sat red on it (and on `check-duplication`). Was CI simply not run on the offending commits, is the wiring claim itself stale, or was the failure seen and deprioritized? This decides whether item 2 is "fix CI" or "fix docs".
2. **Acceptance style for the 9 remaining groups:** do you want in-tree `//art-dupl:accept` rationales (self-documenting, more comment lines) or the current baseline-only acceptance? I chose baseline-only by precedent but it is genuinely reversible.
3. **Was this session a "commissioned plan" under your T03.2 ruling?** If yes, I should have hand-committed the three phases (directive fixes, extractions, cqrs-upgrade split) with real messages instead of letting the daemon produce `chore:` waves. Retroactive hand-commits are not possible without history rewrites (off-limits), so the answer only changes future behavior — but I'd rather know your intent.

---

*Report convention: point-in-time snapshot. Section (f) is HARVEST input for `TODO_LIST.md` — do not let it entomb here. Per the harness contract this file is not hand-committed; the auto-commit daemon picks it up.*
