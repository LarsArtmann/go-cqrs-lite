# P015 Validation Session — Review & Full Status

- **Date:** 2026-10-12 13:30 CEST
- **Scope:** closes the review questions ("what did you forget / do better / improve") over the 02:30–02:57 validation session, and re-baselines the full status. Covers ONLY that session's run plus what it noticed — no new research after 02:57 (machine idled ~10.5 h since; parallel sessions may have moved the repo since; everything below is as-of the 02:57 state).
- **Predecessors:** `2026-10-12_02-56_p015-validation-session-status.md` (this session's own report), `2026-10-12_01-18_superb-p015-linter-session-status.md` (P015 build session; its three §g questions remain unanswered — 4th ask below).

## Headline

All four owed P015 proofs landed and are on record: **3/3 mutants killed**, corpus **5 TP / 0 FP** via structured output, **SARIF property bag verified**, and one **foreign-at-HEAD red suite repaired** (taskmanager golden, root-caused to the issue-#64 detector change). cmd/cqrs-lint: 19/19 packages green at 02:57. Two new product gaps found and filed instead of fixed. The three owner questions are now asked for the 4th consecutive session.

## a) FULLY DONE

1. **Session setup done right:** status-report §f read first, matching skills loaded (linter-building, go-cqrs-lite, jq — loaded before first jq use), 7-step todo list created and worked strictly one-step-at-a-time with verification after each step.
2. **Discrimination proof (01-18 §b1 closed).** Three deliberate mutants in `p015.go`, each applied → suite run → restored via `git restore` → green re-verified: M1 `p015JournalishName` always-false → both TPs red; M2 root suffix `RegisterQuery`→`RegisterQueryXX` → both TPs red; M3 `p015JournalHelpers` emptied → ONLY the one-hop TP red (paths independently covered). 3/3 killed, 0 collateral NP failures. Recorded as a dated addendum in the plan doc (`docs/planning/2026-10-12_00-20_SUPERB-cqrs-lint-p015-journal-readall-in-query-handler.md`).
3. **Corpus re-validation, structured (01-18 §b2 closed).** cqrs-lint built from working tree (`4.13.6-0.20261012004318-ad4f824f5610`); nsfw-classifier: `--fail-on-stale-suppressions` exit 0; "19 finding(s) suppressed" (5 P015 + 14); `--show-suppressed` text stream yields exactly 5 `[P015]` blocks at the five predicted sites — feedback/domain.go:214, history/domain_queries.go:205 + :264, rooms/domain_queries.go:93, verdicts/domain.go:165 — each with a Suppressed marker; `--format json` shows zero unsuppressed P015 (only 1 info-level F021). No `tail` anywhere. Expectation pinned in the plan-doc addendum; mechanically enforced consumer-side (suppression pin 31 + stale-suppression gate).
4. **SARIF check (01-18 §b4 closed).** Synthetic compilable fixture (interface-method `X.RegisterQuery` selector shape, `id/v4` import to pass the go-cqrs-lite engagement gate): SARIF 2.1.0 parses; P015 result at the exact ReadAll call site (fixture.go:28:15); `level: warning`; property bag carries the full verified key set — `go-finding/severity`, `-confidence` (0.5 Medium), `-category` (performance), `-fixStrategy` (suggest), `-suggestion`, `-snippet`, `-id`, `-toolName`.
5. **Foreign-at-HEAD red suite repaired.** `TestIntegration_TaskmanagerExpectedFindings` failed at HEAD (C009 golden 4, live 3). Root-caused: foreign commit `46187d98a` (01:11, issue #64 — bare `Must`/`New` helper exemption) changed detector semantics and updated its unit tests but missed the paired integration golden. Root cause confirmed by reading the actual diff (old `len(name) > 4` guard excluded taskmanager's generic `Must[T]`); golden re-pinned to 3 with a dated, cause-citing comment (`cmd/cqrs-lint/pkg/rules/integration_test.go`).
6. **Gates:** doc-check standard set — 1284 references valid (3 pre-existing union-verified ambiguity notes from the parallel v5 session, not mine); full cmd/cqrs-lint suite 19/19 green after the repair.
7. **Hygiene §f29/§f30:** foreign root-README daemon-range change = legitimate release-wave table row (`projectionhost/v4.6.0`) — intended, no action; nsfw-classifier `domain_events.go` allow-time-time directives confirmed committed (`301f3eb0`), its live tree edits (AGENTS/TODO_LIST/images) belong to the concurrent session — untouched.
8. **New findings filed, not buried:** `--show-suppressed` structured-format parseability gap → TODO_LIST §cqrs-lint dated harvest block; golden-update trap → documented in the 02-56 report §d/§e.

## b) PARTIALLY DONE

1. **Full `nix run .#verify` still not run** — this session ran the module suite, doc-check, and targeted gates only; the exclusive full verify (race, per-module matrix, check-arch/coverage) remains owed before any tag, and will still hit the foreign file-size (`version/v007_tables.go` 400 L) and art-dupl reds documented at HEAD.
2. **Lint/vet not re-run after the golden edit** — the edit is an 8-line comment + count change in a test file; suite green proves compile health, but `golangci-lint`/`go vet` on the touched file were not re-executed this session (pre-existing 50 `lll` in cmd/cqrs-lint are foreign).
3. **Authored history: 2 of 4 artifacts won the race** — the proof addendum (`ad4f824f5`) and the 02-56 status report (`358e18c48`) got authored commits; the TODO_LIST row, plan-doc validation addendum, and the golden fix were daemon-absorbed into `chore:` commits within minutes (the golden-fix commit attempt literally returned "nothing to commit, working tree clean"). Narratives survive in the files themselves (repo convention), but the §d3 lesson from 01-18 was NOT solved, only re-experienced.
4. **Validation binary vs pinned binary skew noted but not reconciled** — the corpus was validated with the working-tree binary, which now INCLUDES the C009 semantics change; nsfw-classifier's gate still runs the flake-pinned older build. P015 behavior is identical across both, but the next flake bump will move C009 behavior consumer-side — nobody has checked what that does to nsfw-classifier's C009 profile (it has C009-relevant code; a golden-style drift there would surface as gate findings).

## c) NOT STARTED

1. `--show-suppressed` parseability fix (filed in TODO_LIST; payload-or-stderr + regression test).
2. `CQRS_LINT_UPDATE_GOLDEN=1` write-mode (currently only logs; rename or implement).
3. `ruletest` JSON-round-trip count helper (01-18 §f14).
4. Mirror the 5 library-tier work items into go-cqrs-lite's own TODO_LIST (unblocked bookkeeping; consumer rows then reference, not own — 01-18 §f28).
5. Tag cqrs-lint + fleet rollout (cqrs-htmx, go-appkit) — blocked on §g1.
6. Stats-fold fix campaign — blocked on §g2.
7. Library-tier audit fixes (expression index, BLOB payloads, lazy DDL, read pool, auto_vacuum) — blocked on §g3.
8. Golden-corpus-gated next rules (JSON-in-JSON payload, unindexed sorted scan) — deliberately gated per linter-building discipline.
9. docs-health VERIFY pass reconciling the three 2026-10-12 reports against moved code.
10. All carry-over consumer rows (nsfw TODO_LIST: history fold fix, metrics, ops doc, scripts persist, ADR threshold).

## d) TOTALLY FUCKED UP

1. **False comfort from `CQRS_LINT_UPDATE_GOLDEN=1`** — I read the green exit as "golden updated"; the mode only LOGS the profile for manual pasting. Caught only because I reflexively checked `git diff` (the never-trust-unverified lesson working as designed). A less paranoid run would have walked away believing the suite was fixed while the tree was untouched. The flag's name lies; I nearly bought it.
2. **Detector-semantics archaeology done backwards** — when the count-drift failure appeared, my first two hypotheses were fixture edits and panic-line diffs (3 tool calls); the correct first move was `git log` on `c009.go`, since detector semantics had changed 100 minutes earlier. Cost: ~3 wasted round trips. The general rule: when a detector-count golden drifts, suspect the DETECTOR's own history first.
3. **Sloppy comment draft almost shipped** — my first version of the golden comment contained thinking-out-loud text ("…wait, the two features.go panics…"). Caught on immediate self-review and rewritten. Quality bar is per-keystroke; that one was a miss the linter itself would never catch.
4. **Commit race lost 3 of 4 times despite the lesson being written down** — the daemon absorbs within minutes; my "commit at milestone boundaries" execution was still end-loaded relative to the daemon's loop. Unresolved personal/process bug, third session in a row.
5. **Two wasted fixture attempts before the SARIF fixture compiled** — I initially forgot the CLI loads REAL packages (go-cqrs-lite import gate + unresolvable import = "could not analyze any packages") and burned two attempts on AST-level assumptions. Also produced a jq parse-error detour before I checked where `--show-suppressed` actually writes.

## e) WHAT WE SHOULD IMPROVE

1. **Mutation proof as the default closing step for every new rule** — ~4 minutes end-to-end for 3 mutants; it demonstrably proves the suite tests. Make it ritual, not exceptional.
2. **Fix the two product gaps found:** `--show-suppressed` must ride inside structured output (payload array or stderr) with a parseability regression test for every flag × format combo; and the golden "update" env var should actually write (or be honestly renamed `PRINT_GOLDEN`).
3. **Detector-semantics changes owe their goldens in the same edit** — same rule as api-surface ⇒ api-golden. A one-line "which goldens watch this rule" note in the rule-writing docs would have prevented `46187d98a`'s miss.
4. **Count-drift triage order:** detector git log → fixture git log → source enumeration. Write it down where the integration golden lives.
5. **Commit immediately after each single file write** (write→commit pairs), or accept daemon authorship deliberately and stop spending attempts. Status quo wastes one tool call per artifact.
6. **Version-stamp every validation claim** (worked well this session: the binary version string is in every record) AND check the flake-pinned-vs-working-tree delta when validating a consumer corpus — see b4.
7. **Status-report format convention:** this repo's established convention is `.md`; the status-report skill's canonical default is HTML. The explicit user instruction wins here, but the repo-convention exception deserves recording in the skill (standing-exception note covers task-queue dispatches; this repo's demand recurred again).

## f) Up to 30 things we should get done next (brainstorm, impact-sorted; NOT a commitment)

**Product gaps from this session (unblocked, small, high-trust-value):**
1. Fix `--show-suppressed` structured-format pollution (TODO_LIST row filed) — render suppressed findings inside json/sarif/csv payloads or move the text block to stderr; add parseability regression tests.
2. Make `CQRS_LINT_UPDATE_GOLDEN=1` write the golden (or rename to `PRINT`); keep a `--print` fallback for review workflows.
3. ruletest JSON-round-trip count helper (pin output shape at fixture level — §f14 from 01-18).

**Unblocked bookkeeping / verification:**
4. Mirror the 5 library-tier items into go-cqrs-lite's TODO_LIST (consumer rows become references).
5. Full exclusive `nix run .#verify` + check-arch + check-coverage — after resolving or explicitly baselining the foreign file-size/art-dupl reds.
6. Check what the new C009 semantics (issue #64) will do to nsfw-classifier's profile BEFORE its next flake bump (b4) — a dry-run lint with a locally built new binary exists from this session (`/tmp/cqrs-lint-p015`).
7. docs-health VERIFY pass over the three 2026-10-12 reports.
8. docs-health HARVEST routing for this report's §f (items 1–4 already landed in TODO_LIST this session; the rest route to ROADMAP per rigor).

**Release & fleet (gated on §g1):**
9. Tag the cqrs-lint module (next wave) so consumers pin releases instead of `unstable-fd3b2d8`.
10. Roll the bump to cqrs-htmx + go-appkit — expect P015 hits; same audit-or-suppress treatment.
11. Suppression-expiry convention for the 5 consumer P015 sites (forces re-review when fixes land).
12. Reconcile with the concurrent history-board/verdict-drift session: one design should own drift-ledger + stats folds.

**Consumer fixes nsfw-classifier (gated on §g2):**
13. Fix `statsFromJournal`/`watchTrendFromJournal` (day-buckets or projection fold); then remove the 2 history suppressions (pin 31→29) + state.md in the same commit.
14. Decide permanent-vs-fix for verdicts/rooms/feedback synchronous-authoritative folds.
15. Metrics row: journal:projection ratio + WAL frames (pairs with drift-ledger work).
16. Ops doc: the 13 `meta_*` tables.
17. Persist the audit DB scripts under `scripts/`; then clean `/tmp/nsfw-sqlite-audit`.
18. ADR: collection-size threshold for planned tables.

**Library-tier (gated on §g3):**
19. Expression index on `json_extract` sort keys (kills the 1 µs/row sort cliff at ~100k rows).
20. BLOB journal payload column (kills the 71% base64 tax).
21. Lazy engine DDL (13 → 2–3 tables per fresh DB).
22. Read-path connection pool > 1 (WAL read concurrency).
23. `auto_vacuum=INCREMENTAL` + bounded engine deletes for in-place retention.
24. sqlite-vs-pebble benchmark for the verdict-cache shape.
25. Nightly synthetic audit job (curve + WAL + ratios) as a nix check.

**Linter craft (each gated on a golden corpus + FP budget):**
26. Golden corpus for the JSON-in-JSON payload rule (`detail_json` shape) — corpus BEFORE detector.
27. EXPLAIN-fed or schema-fed "unindexed sorted scan" check — walk the mechanism-choice tree first.
28. B017 hydrate-family matching only under a served-path signal (keep silent until evidence exists).
29. Verify F026 (prefetch) silence on the board reader is intended, not a detection gap.
30. Carry-overs: ES-purity decision for `domain.verdict_observed` (106× amplification); target-scale answer (thousands vs 100k+ captures) which sorts the whole audit roadmap.

## g) Questions only you can answer (4th ask — each gates a whole tier above)

1. **Tag policy:** tag cqrs-lint now (go-release flow) so fleet consumers pin releases, or is unstable-rev pinning the intended norm until the next tag wave? (Gates f9–f11.)
2. **Stats folds:** are verdicts/rooms/feedback's synchronous-authoritative folds PERMANENT by design (suppressions = honest end state), or do you want a fix campaign — including history's 2 performance-relevant folds? (Gates f12–f14.)
3. **Library-tier ownership:** expression indexes + BLOB payloads + lazy DDL in go-cqrs-lite's sqliteengine — pursue now under fleet-first, or park until a second consumer needs them? (Gates f19–f25; the TODO_LIST mirroring in f4 is doable regardless on your word.)

If these stay unanswered, my recommended defaults on your behalf would be: keep rolling pins until the next wave (no tag), document suppressions as permanent-with-revisit-triggers, and park library-tier until the 100k-scale answer exists — but I have not acted on any of them.

Waiting for instructions.
