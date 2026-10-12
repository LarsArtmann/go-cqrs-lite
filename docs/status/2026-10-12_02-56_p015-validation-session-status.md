# P015 Validation Session — Full Status

- **Date:** 2026-10-12 02:56 CEST
- **Scope:** close the owed verification from the 01-18 P015 session (§b items): discrimination proof, `--json` corpus validation, SARIF check, repo gates. Plus §f hygiene rows 29/30 and the §f2 "pin the 5-finding expectation" record.
- **Predecessor:** `docs/status/2026-10-12_01-18_superb-p015-linter-session-status.md` (P015 shipped there; all headline numbers unchanged).

## Headline result

Every owed P015 proof is now on record: **3/3 mutants killed** (suite demonstrably tests), **corpus re-validated programmatically** (exactly 5 P015 sites, all suppressed, gate exit 0 — no `tail` in sight), **SARIF property bag verified** (severity/confidence/suggestion/snippet all render), and a **foreign-at-HEAD red suite repaired** (taskmanager golden C009 4→3, completing the issue-#64 detector change's own coupling obligation). cmd/cqrs-lint module: 19/19 packages green.

## a) FULLY DONE

1. **Discrimination proof (§b1 closed).** Three deliberate mutants in `p015.go`, each restored via `git restore` and re-verified green: M1 receiver-signal dead → both TPs red; M2 root `RegisterQuery` match blinded → both TPs red; M3 one-hop helper map emptied → ONLY the one-hop TP red (paths independently covered). Recorded as a dated addendum in the plan doc.
2. **Corpus validation via structured output (§b2 closed).** Built cqrs-lint from `ad4f824f5` (`4.13.6-0.20261012004318-ad4f824f5610`); against nsfw-classifier: `--fail-on-stale-suppressions` exit 0, "19 finding(s) suppressed" (5 P015 + 14), text `--show-suppressed` yields exactly 5 `[P015]` blocks at the five predicted sites (feedback:214, history:205/:264, rooms:93, verdicts:165), each with a Suppressed marker; `--format json` shows 0 unsuppressed P015. Expectation pinned in the plan doc addendum + mechanically enforced consumer-side (suppression pin 31 + stale gate).
3. **SARIF check (§b4 closed).** Synthetic compilable fixture (interface-method `X.RegisterQuery` selector shape, `id/v4` import for the go-cqrs-lite gate): SARIF 2.1.0 result at the ReadAll call site, `level: warning`, property bag carries `go-finding/severity`, `-confidence` (0.5 Medium), `-category` (performance), `-fixStrategy` (suggest), `-suggestion`, `-snippet`, `-id` — the full verified round-trip key set.
4. **Repo gates (§b3 closed for the linter surface).** doc-check standard set: 1284 refs valid (3 pre-existing union-verified ambiguity notes from the parallel v5 session). Full cmd/cqrs-lint suite: 19/19 green — after one repair (see d1). check-arch unaffected (no dependency changes). File-size/art-dupl pre-existing foreign reds at HEAD unchanged (not mine, documented in the 01-18 report).
5. **Foreign-at-HEAD red suite repaired (d1).** `TestIntegration_TaskmanagerExpectedFindings` failed at HEAD: C009 golden 4, live 3. Root cause: foreign commit `46187d98a` (01:11 today, issue #64) exempted bare `Must`/`New` helpers — taskmanager's generic `Must[T]` (must.go:10) legitimately left the raw count. That commit updated its unit tests but missed the integration golden (the env-var "update" only LOGS the profile; it never writes — trap noted). Golden re-pinned to 3 with a dated comment citing the cause.
6. **Hygiene §f29/§f30.** Foreign root-README daemon-range change = legitimate release-wave table row (`projectionhost/v4.6.0`), intended, no action. nsfw-classifier `domain_events.go` allow-time-time directives confirmed committed (`301f3eb0`); its live tree edits (AGENTS/TODO_LIST/images) belong to the concurrent session — untouched.

## b) PARTIALLY DONE

1. **`nix run .#verify` (full) still not run** — module suite + doc-check + the targeted gates cover the touched surface, but the exclusive full verify (race tests, per-module matrix) remains owed before any tag; it will still hit the foreign file-size/art-dupl reds at HEAD.
2. **Authored commits again lost to the daemon** — the docs addendum (plan doc + TODO_LIST row) and the golden fix were each absorbed into `chore:` commits within minutes; narratives live in the files themselves (repo convention), but the race documented in the 01-18 report §d3 repeats. Only the discrimination-proof doc commit won.

## c) NOT STARTED

Everything parked on the three unanswered owner questions (tag/fleet rollout = §g1; stats-fold fix campaign = §g2; library-tier audit fixes = §g3), plus §f craft rows 11–15 (golden-corpus-gated next rules). The `--show-suppressed` parseability gap is filed in TODO_LIST §cqrs-lint, not fixed here.

## d) TOTALLY FUCKED UP

1. **First `CQRS_LINT_UPDATE_GOLDEN=1` run produced false comfort** — I read the green exit as "golden updated"; the mode only pastes to the test log. Caught it because `git diff` came back empty, then pasted the map by hand. Lesson: an env var named UPDATE that doesn't write is a trap — verify the tree changed after ANY "update" invocation.
2. **One sloppy comment draft** briefly landed in the golden edit (thinking-out-loud text); caught and rewritten in the next tool call. Quality bar is per-keystroke, not per-commit.

## e) WHAT WE SHOULD IMPROVE

1. **Mutation proofs are cheap and decisive** — 3 mutants ≈ 4 minutes end-to-end; make it the default closing step for every new rule (now demonstrated twice over).
2. **`--show-suppressed` should ride INSIDE structured output** (filed as TODO_LIST row): json/sarif/csv must stay parseable under every flag combo; the text block belongs on stderr or in the payload.
3. **`CQRS_LINT_UPDATE_GOLDEN=1` should actually write** (or rename to `CQRS_LINT_PRINT_GOLDEN`); a silently-non-writing update mode invites the false-comfort class.
4. **Cross-coupling checklist for detector-semantics changes:** c009.go's change owed the integration golden in the same edit (same rule as api-surface ⇒ api-golden); a one-line "goldens touched by this rule" note in the rule-writing docs would prevent the miss.

## f) UP NEXT (unchanged from 01-18 §f, minus what this session closed)

1–4 closed. Remaining near-term: §f5 tag (blocked §g1), §f7–10, §f14 (ruletest JSON-round-trip helper — still a good idea), consumer fixes §f16–18 (blocked §g2), library-tier §f23–28 (blocked §g3). New: fix the `--show-suppressed` parseability gap (TODO_LIST), and the `CQRS_LINT_UPDATE_GOLDEN` write-mode improvement.

## g) Questions only you can answer

Unchanged — the three from the 01-18 report (tag vs rolling pin; stats-fold campaign vs documented-permanence; library-tier ownership). No new questions; this session needed none.
