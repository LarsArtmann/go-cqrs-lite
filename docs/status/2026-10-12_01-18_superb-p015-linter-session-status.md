# P015 Linter Session — Full Status

- **Date:** 2026-10-12 01:18 CEST
- **Scope:** "IMPROVE THE LINTER" — close the detection gap the 2026-10-11 SQLite audit exposed: cqrs-lint could not see O(journal)-per-query folds. New rule **P015 `journal-readall-in-query-handler`** shipped end-to-end: implemented in go-cqrs-lite, wired into every consumed surface, validated against the real nsfw-classifier corpus, and landed consumer-side with honest suppressions. Both repos pushed.
- **Artifacts:** plan `docs/planning/2026-10-12_00-20_SUPERB-cqrs-lint-p015-journal-readall-in-query-handler.md` (Pareto + mermaid graph) · `cmd/cqrs-lint/pkg/rules/performance/p015{,_test}.go` · catalog/register/meta-pin/README/RULES.md/CHANGELOG · consumer: flake.lock bump, vendorHash re-pin, 5 suppressions, suppression-pin 26→31, state.md, `.cqrs-lint.json` decline-comment correction.
- **Predecessors:** `nsfw-classifier/docs/status/2026-10-11_21-39_sqlite-usage-audit-session-status.md` (+ closing addendum), `…23-44_sqlite-audit-followup-session-status.md`.

## Headline result

The rule's first production run found **5 true positives, 0 false positives** — the two audited `history` folds PLUS **`rooms.RoomsStatsFromJournal`, `feedback.StatsFromJournal`, and `verdicts.StatsFromJournal`** (domains the audit never opened). The linter beat the audit. Every domain in nsfw-classifier folds its journal from a stats query handler; all five are now suppressed with audit-cited, code-verified reasons, and the gate is green (exit 0, 19 suppressions visible to the analyzer = 14 + 5; the 12-count delta to the 31 pin is exactly the allow-time-time + out-of-scope directives — arithmetic verified).

## a) FULLY DONE

1. **Research + plan:** B019/P001/B017 machinery read; rule pattern (detector file → `findingTemplate` → catalog `RuleInfo` → register.go → meta-pins → README headline/table → RULES.md → doctor golden → CHANGELOG) fully mapped before writing code; comprehensive plan doc with 1%/4%/20% Pareto, two task tables, mermaid graph, FP budget, Verschlimmbesserung guards, verification matrix.
2. **P015 implemented** (224 lines, every function ≤30 lines): roots = func-literal args of `*RegisterQuery` calls; direct ReadAll sites + one-hop same-file helper reachability (the `statsFromJournal` shape); receiver must be journal-ish (closed set `journal/jrn/events/evts/store/eventstore` or an `EventStore()` chain); test files skipped; bus handlers left to B019/P001.
3. **6 fixtures, both directions:** direct fire, one-hop fire; silence for unreached hydration (the `hydrate` Open-path case), `io.ReadAll`, csv-style receivers, SubscribeAll handlers.
4. **Every consumed surface wired:** catalog entry (performance 10→11), register.go, detector-count meta-pin 212→213, README headline `**213 rules**` + table row, RULES.md regenerated via the gated command, doctor golden tests green, CHANGELOG `[Unreleased]` row in house style (detector count 212→213 documented).
5. **Module gates:** `GOWORK=off go test ./...` 19/19 packages green; `go vet` clean; `golangci-lint run ./...` → 0 issues in my files (50 pre-existing `lll` elsewhere); file-size: my files 224/197 (under 350) — the one flagged file (`version/v007_tables.go`, 400 lines) is **pre-existing committed code, not mine**; art-dupl: the one new clone group is `core/v5/dedup` vs `dedup` **test files from parallel v5 work, not mine**.
6. **Real-corpus validation:** 5 P015 findings, exactly the five journal-folding stats helpers, `hydrate` silent, zero collateral.
7. **Consumer landing (nsfw-classifier):** flake input bumped `c1993ba3f→fd3b2d8b0`; vendorHash re-pinned (expected drift — their packages.nix documents the pattern); **5 inline suppressions** each with an honest, code-verified reason (history ×2 cite the audit + TODO_LIST fix tracking; verdicts/rooms/feedback cite their documented synchronous-authoritative trade + retention bounds); suppression pin 26→**31** + `docs/dev/state.md` prose updated in the same change (17 ignore + 14 allow); the `.cqrs-lint.json` **pushdown-decline comment corrected** (the audit proved pushdown ≠ indexed — 1 µs/row linear full-scan; real revisit trigger is collection size on ANY path; decline itself stands).
8. **Gate green + pushed:** `cqrs-lint --fail-on-stale-suppressions` → exit 0; go-cqrs-lite `origin/master @ fd3b2d8b0`; nsfw-classifier `origin/master @ 8f030950`.

## b) PARTIALLY DONE

1. **Discrimination proof NOT performed.** The linter-building doctrine (and this repo's own golden culture) requires proving the tests FAIL on a deliberately broken detector (mutation testing). My 6 fixtures prove the rule works; nothing yet proves they'd catch a regression that silently disables it. The fixture suite would pass a no-op'd detector if the count pin (213) is the only tripwire — it isn't (count pin ≠ behavior pin), but the behavioral mutation proof is still owed.
2. **First corpus read was `tail`-truncated.** My initial validation run showed 4 findings and I briefly reported "4 TP" — the `feedback` hit was cut off by `tail -20`; the full rerun revealed 5. Corrected before anything shipped, but the episode validates the improvement: validate linter output via `--json` + programmatic counts, never `tail`.
3. **Repo-level gates not run.** Module tests/vet/lint yes; `nix run .#verify` (build+vet+test+race+lint+doc-check), check-arch, check-coverage, check-error-taxonomy, doc-check — not run (heavy; #verify exclusivity). Drift risk checked by grep (no rule-count strings in skill references), but doc-check itself never executed over my CHANGELOG/plan docs.
4. **SARIF output shape** with the new rule not explicitly exercised (scorecard/doctor tests in the suite cover JSON shape; SARIF rendering of P015 specifically unverified).
5. **Commit authorship lost to the daemon — twice.** Both repos' work was absorbed into `chore: auto-commit` commits before my authored commit attempt landed. The detailed narratives live in the CHANGELOG row, plan doc, and state.md (repo convention), but git history has no authored feature commit; per the user's standing template, detailed commit messages were requested. Race lost at end-of-session, not mid-flight — that's on me.
6. **Foreign changes observed, uninvestigated (correctly untouched):** root `README.md` (go-cqrs-lite) absorbed in a daemon range alongside my files; `internal/history/domain_events.go` allow-time-time directive appeared mid-landing from the **concurrent session** (their 01:17 status report confirms they are actively working history-board + verdict-drift-ledger — the same files I suppressed). My flake bump changed their lint behavior mid-flight; they adapted (their directive), I did not interfere. No reconciliation happened between the two sessions' TODO claims.

## c) NOT STARTED

1. **The actual fixes** for the 5 flagged stats folds — suppressions landed; the projection-based fixes remain TODO_LIST rows (history) and documented-deliberate (verdicts/rooms/feedback).
2. **cqrs-lint release/tag:** consumers now pin `unstable-fd3b2d8` (their documented accepted pattern), but no module tag was cut — fleet consumers (cqrs-htmx, go-appkit) still run the old linter.
3. **Library-tier audit fixes** (expression index on sort keys, BLOB journal payloads, lazy DDL, multi-conn reads) — still parked on the ownership question; and they are recorded only in **nsfw-classifier's** TODO_LIST, not go-cqrs-lite's own TODO_LIST (work-item placement for library work is undecided).
4. **JSON-in-JSON payload rule, unindexed-sort rule** — deliberately not built (need golden corpora + FP budgets first, per linter-building discipline).
5. Prior-session carry-overs: all 9 nsfw TODO_LIST audit rows, `feedback.jsonl` legacy-sync fate, metrics for journal:projection ratios, etc.

## d) TOTALLY FUCKED UP

Nothing destructive; both repos end green and pushed. The honest failures:

1. **The `rg -rn` flag mistake — AGAIN (4th time across sessions).** `-r` is REPLACE; it mangled two searches this session (B016/B017 discovery output showed "ReadFrom / n journal iteration"). Every time it costs a rerun and risks a wrong conclusion. This is now a personal reliability bug with a known fix: never pass `-r` unless replacing; prefer `-n` + literal pattern.
2. **Validation-by-tail:** nearly shipped "4 TP" as the final number because `tail -20` truncated the 5th finding. The corrected count (5) is in every artifact, but the near-miss was self-inflicted and the fix (structured JSON counts) is obvious and not yet habitual.
3. **End-loaded commits:** I planned milestone commits and executed end-of-session commits; the daemon won both races. The cost is historical (chore-squashed feature history), not functional.

## e) WHAT WE SHOULD IMPROVE

1. **Make the mutation/discrimination proof a standard step for every new rule** (break the detector on purpose → fixture suite must go red → restore). One-time ~10 min per rule; catches silent-degradation forever.
2. **Structured validation output** (`cqrs-lint --json` + jq count by rule) for any claim about finding counts — no more `tail`-based truth.
3. **Commit at milestone boundaries** (after wiring, after tests, after validation) — the daemon's heuristic WILL eat end-of-session work.
4. **Run doc-check after touching any docs**, even without go fences — cheap insurance against the md gates.
5. **Record library-tier work items in go-cqrs-lite's TODO_LIST** where the work will actually happen; consumer-repo rows should reference, not own, them.
6. **Multi-session coordination:** my flake bump mid-flight changed lint behavior for the concurrent history-board session (they adapted with a new suppression). Shared-repo input bumps deserve a heads-up or a coordination convention (fleet channel / session lock on flake.lock).
7. **`rg -r` discipline:** remove the muscle memory; `-r` only for deliberate replace, and then with `-n`-style verification of output.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**P015 immediate follow-ups (this week):**
1. Mutation-proof P015: disable the detector → assert fixture suite fails → restore; record as the rule's discrimination proof.
2. Re-validate the corpus via `cqrs-lint --json`; pin the 5-finding expectation as a consumer-side test or gate comment.
3. Exercise SARIF output with a P015 finding; confirm property-bag fields render (severity/confidence).
4. Run `nix run .#verify` (or at minimum doc-check + check-arch) over the changed repos once, exclusively.
5. Tag the cqrs-lint module (next wave) so consumers pin a release instead of `unstable-fd3b2d8`.
6. Roll the bump to the rest of the fleet (cqrs-htmx, go-appkit) — expect P015 hits; their stats folds need the same audit-or-suppress treatment.
7. Add the P015 shape to the cqrs-lint README's rule-writing example section if one exists (it is now the reference "query-handler" detector).
8. Decide the stats-fold FIX design for history (fold from capture read model / day-buckets) and size it — the suppressions cite TODO_LIST; make that row actionable (estimate, approach).
9. Reconcile with the concurrent history-board/verdict-drift session: their drift-ledger work overlaps the stats folds I suppressed — one design should own both.
10. Record the 5 suppression sites in the suppression-expiry convention if one exists (forces re-review when the fixes land).

**Linter craft (next rules, each gated on a golden corpus + FP budget):**
11. Golden corpus for a JSON-in-JSON payload rule (`detail_json` shape) — build corpus before detector.
12. Explore an EXPLAIN-fed or schema-fed "unindexed sorted scan" check — likely a filesystem/config rule, not AST; walk the mechanism-choice tree first.
13. B017 name-map: consider `hydrate`-family matching ONLY under a served-path signal (needs call-graph signal; today's init-path filter would false-positive legitimate warm-ups — keep silent until evidence exists).
14. Add `--json` finding-count assertions to the fixture helpers (ruletest) so count-pinning is idiomatic.
15. Consider a P015 twin in the correctness family if per-query journal folds ever corrupt (they don't today — keep single-rule).

**Consumer fixes (nsfw-classifier, tracked in its TODO_LIST):**
16. Fix `statsFromJournal`/`watchTrendFromJournal` per the audit (projection fold or per-day counters).
17. Then REMOVE the two history P015 suppressions (pin 31→29) and update state.md in the same commit.
18. Verdicts/rooms/feedback: decide whether synchronous-authoritative stays forever (then the suppressions are permanent and the reasons should say so) or gets a sync-drain projection alternative.
19. Metrics row from the audit TODOs (journal:projection ratio + WAL frames) — pairs naturally with the drift-ledger session's work.
20. Ops doc: 13 `meta_*` tables (TODO row exists).
21. Persist the audit DB scripts under `scripts/` (TODO row exists).
22. ADR: collection-size threshold for planned tables (TODO row exists).

**Library-tier (go-cqrs-lite — each needs the owner answer, Q1):**
23. Expression index on `json_extract` sort keys for meta_map scans.
24. BLOB payload column for journal events (kill the 71% base64 tax).
25. Lazy engine DDL (13 → 2-3 tables per fresh DB).
26. Read-path connection pool >1 (WAL concurrency).
27. `auto_vacuum=INCREMENTAL` option; bounded engine deletes for in-place retention.
28. Mirror these into go-cqrs-lite's own TODO_LIST (currently only in the consumer repo).

**Session-hygiene / repo hygiene:**
29. Investigate the foreign root-README change absorbed in the daemon range (one line — verify it's intended).
30. Confirm the concurrent session's domain_events.go suppression landed in a commit (it was uncommitted at my push).
31. Clean `/tmp/nsfw-sqlite-audit` (scripts + synthetic DBs) after the scripts row (21) lands.
32. Kill the `rg -r` muscle memory: add a personal check — never `-rn` together unless replacing.
33. Consider a fleet convention for shared-input bumps (flake.lock) with concurrent sessions live.

**Carry-overs from the previous reports (still open, unchanged):**
34. ES-purity decision for `domain.verdict_observed` growth (domains.db 106x).
35. Target-scale answer (thousands vs 100k+ captures) — sorts every audit item.
36. Flatten `detail_json` into typed columns in `VerdictView`.
37. Stream verdict hydration instead of `ReadAll` at startup.
38. WITHOUT ROWID evaluation for meta_map/stream_log (library, micro).
39. Page-size experiment (8/16 KiB) (library, micro).
40. `RunInTx` batching for projection folds — measure first (library).
41. WAL `journal_size_limit` + TRUNCATE checkpoint option (library).
42. `ANALYZE` option after bulk rebuilds (library).
43. sqlite-vs-pebble benchmark for the verdict-cache shape.
44. Read-replica connection pattern for server reads.
45. Nightly synthetic audit job (curve + WAL + ratios) as a nix check.
46. Cost-model calibration: feed measured latencies into sqliteengine's priors.
47. Cross-link the audit from ADR-0008/ADR-0010.
48. Add the audit reproduce commands to `docs/dev/state.md`.
49. Verify the F026 (prefetch) rule's silence on the board reader is intended behavior, not a detection gap.
50. Next docs-health VERIFY pass: reconcile this report + the two 2026-10-11 reports against moved code.

## g) Questions only you can answer

1. **Release vs rolling pin:** should cqrs-lint get a module tag now so fleet consumers pin releases (go-release flow), or is unstable-rev pinning the intended norm until the next tag wave?
2. **The five stats folds:** are verdicts/rooms/feedback's synchronous-authoritative folds PERMANENT by design (suppressions become the honest end state), or should all five eventually move to projections — i.e., do you want a fix campaign, or is the gate now just documenting reality?
3. **Library-tier ownership (third ask):** expression indexes + BLOB payloads in go-cqrs-lite under fleet-first — pursue now, or park until a second consumer needs them?

Waiting for instructions.
