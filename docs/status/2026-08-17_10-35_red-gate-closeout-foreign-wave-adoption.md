# Red-Gate Closeout — Adopting the Concurrent Session's Lint & Duplication Debt

**Date:** 2026-08-17 (session ran 2026-08-16 23:37 → 2026-08-17 01:20; report written 10:35)
**Scope:** Takeover of the two RED gates (lint + duplication) left by the finished
concurrent engine-correctness session, closing the last open question (Section G)
of the honesty-&-flake-gates wave.
**End state:** `nix run .#verify-fast` fully GREEN (lint 76/76 modules, 0 new
clone groups). Three commits landed. Only macOS hardware verify remains from the
19-item quality section.

---

## a) FULLY DONE (verified this session)

1. **Concurrent-session completion confirmed** — final commit `d7e583c82`
   ("engine-correctness batch completion", 23:32) left a clean tree; takeover was safe.
2. **Lint gate: 8 findings → 0** (7 previously reported + 1 NEW `unparam` that
   appeared in the concurrent wave's final commits — re-verified rather than trusted
   from the stale report):
   - `dgraphengine/graph.go` — `doWithAbortRetry` now returns `error` only; all 3
     callers updated (the `*api.Response` result was never used).
   - `metaengine/store.go:737` — `Edge{From: removal.From, To: removal.To}` →
     direct `Edge(removal)` conversion (S1016; the types are structurally identical).
   - `metaengine/graph_vector_features_test.go` — duplicate interface `graphCapable`
     deleted, assertions retargeted to `graphBackend` (the duplication was NOT
     deliberate scaffolding, as the prior report feared).
   - `metaengine/pebbleengine/vector.go` + `badgerengine/vector.go` — `//nolint:nilnil`
     with the documented not-found contract, matching the 6 existing repo sites.
   - `id/entropy.go` — `epochMu` nolint (gochecknoglobals) + G115 annotation
     refactored to the documented short-conversion-helper form
     (`seq := uint32(s)` on its own line) after golines wrapping moved the first
     attempt's nolint off the finding line.
3. **Duplication gate: 12 clone groups → 0, baseline untouched, 4 iterative rounds**
   (art-dupl reports one group per region pair — suppressing the visible 5 exposed
   7 masked ones). Annotated classes, all documented-intentional:
   - cross-engine dialect twins: mysql/sqlite `graph_undirected` dispatch + row
     scanners, pebble/pg vector-search prologues ×2 (plain + filtered), badger/sqlite
     marshal-with-fallback helpers;
   - same-file twins: memory engine RLock + BFS prologues (×2 groups), badger
     directed/undirected guards + view epilogues, metaengine typed-field reflect
     extractors;
   - quickstart demo setup boilerplate (graph_demo/vector_demo).
4. **Per-module verification** — build + test + lint green for all 8 code-touched
   modules (id, metaengine, badger/pebble/dgraph/pg/mysql/sqlite engines) plus the
   comment-only example module (its bare-`go` failure was the host-Go-vs-replace
   chain artifact; the nix `goPkg` build is authoritative and green).
5. **Authoritative gate** — `nix run .#verify-fast` EXIT=0: build, vet, test(short),
   race(short), **lint 76/76**, check-arch, depguard, docserver CSS, **duplication
   (0 new clones, baseline 111)**, coverage, api-stability. CHANGELOG honesty gate
   green on the new entry (20 citations verified).
6. **Docs closed out**:
   - CHANGELOG: new "Fixed — lint/duplication gate closeout" section.
   - TODO_LIST: art-dupl item → done with the iteration + directive-placement
     lessons; macOS item annotated as the last open item (18/19).
   - AGENTS.md #14 extended: annotation is ITERATIVE (12 groups took 4 rounds);
     directive must sit on/above the region's FIRST line.
   - Previous session's HTML report: Section G answered in place with an
     "ANSWERED 23:37" card (retargeted to the existing `card-solution` CSS class).
7. **Prettier table truncation repaired** — the BuildFlow hook's prettier step
   destroyed the AGENTS.md "Load sweep" row (raw `|` inside
   `` `-run 'Latency|Timer|Deadline'` `` was parsed as cell delimiters; the row was
   truncated mid-content and the closing delimiter dropped). Rebuilt byte-exact at
   the table's column width with `\|` escapes; dprint clean; prettier no longer
   flags it. Root cause: the row was authored in a prior session without escaping —
   a latent landmine that finally detonated.
8. **Commits (3)**: `45eacb25e` (all fixes + docs), `dc39822af` (hook's own
   whitespace auto-fix, committed so the tree stays clean), `53296052f`
   (.gitignore the hook's tailwind intermediate `docs-ui.src.out.css`, which
   re-dirtied the tree after every commit — nothing serves it; docserver ships
   `docs-ui.css`).

## b) PARTIALLY DONE

- **Verification depth**: verify-fast (soaks skipped) is the AGENTS.md minimum
  bar, met. Full `#verify` (soaks) and a **live Dgraph integration run** against
  the `doWithAbortRetry` signature change were NOT run — unit suite is green but
  the dgraph engine's graph paths only prove out against a real server
  (`nix run .#integration-dgraph`).
- **Lint-config convergence**: BuildFlow's hook run printed err113/goconst/mnd
  findings in `example/metaengine-quickstart/*` that canonical `#lint` does not
  emit (`.golangci.yml` excludes `example/` paths; the hook's per-module golangci
  invocation apparently resolves that exclusion differently). The example module
  was NOT among the hook's failed steps, and the canonical gate is green — but
  "what does lint-clean mean" differs between the two runners. Unresolved.

## c) NOT STARTED (out of this session's scope)

- macOS hardware verification of `scripts/ephemeral-pg.sh` (needs a Darwin host).
- Full `#verify` with soaks; `#load-sweep` (not required — no timing paths touched).
- The BuildFlow preflight/persistent-warning backlog (see f).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Two failed commit attempts (~15 min burned)** — the documented commit-env
   gotcha bit me anyway: attempt 1 lacked the cache env (hook's go step died on
   host-Go 1.26.5 vs go.work 1.26.6); attempt 2 (GOTOOLCHAIN only) hit golangci
   failures in 7 modules via nix eval-cache contention ("SQLite database is busy",
   "9 tools unavailable") — env-flaky, not real (canonical #lint 76/76 green).
   Only attempt 3, with the FULL documented env block
   (GOTOOLCHAIN/GOCACHE/GOMODCACHE/GOPATH/GOLANGCI_LINT_CACHE), passed. The
   gotcha was known; the discipline wasn't applied. Fix listed in (e).
2. **First nolint placement was formatter-fragile** — annotated the same line the
   linter flags, on a line long enough that the hook's golines wrapped it into a
   multi-line call, stranding the nolint off the finding line (would have silently
   re-reddened the lint gate had I not re-linted after the hook run). The
   AGENTS.md rule ("nix fmt BEFORE placing nolint") exists; the robust pattern is
   the short single-line conversion (`seq := uint32(s)`), which I used on retry.
3. **wsl_v5 whitespace miss** — the retry insert needed a blank line above the
   assignment; caught by module lint, fixed immediately. Cost: one lint round-trip.
4. **First edit-batch partial failures** — 2 of 6 edits in the first multiedit
   failed on exact-match (store.go not yet View-read; byte-level re-verify fixed
   both). Minor, self-inflicted by editing too eagerly.

## e) WHAT WE SHOULD IMPROVE

1. **Make the commit-env block copy-pasteable** — add the exact export line to
   AGENTS.md Quick Reference (or a `scripts/with-commit-env.sh` wrapper) so no
   session re-derives it under pressure. The gotcha text exists; the ergonomics
   don't.
2. **Pre-flight the hook before the hook** — run gofmt+golines-equivalent and the
   module lint on ALL touched files (including docs/markdown via dprint) BEFORE
   `git commit`, since the hook chain (golines, wsl, prettier) enforces more than
   `gofmt -l`. My gofmt-only pre-check was insufficient.
3. **Escape `|` inside markdown table cells, always** — the truncation class is
   silent and destructive (content loss, not just formatting). Cheap guard: run
   `dprint check`/prettier on any edited .md before commit; better: a grep
   tripwire for raw pipes inside backticked spans in table rows.
4. **Re-verify stale reports before acting** — paid off this time (found the 8th
   finding); keep doing it. Conversely, gate claims should name the log line
   ("lint 76/76", "EXIT=0") — done here.
5. **Converge BuildFlow-golangci and #lint configs** — two lint truths on one
   tree is a split brain; pick one config resolution and make the hook use it.

## f) NEXT (up to 50, priority-ordered)

1. Run full `nix run .#verify` (soaks) — upgrade verify-fast green to full green.
2. Run `nix run .#integration-dgraph` live against the `doWithAbortRetry` change.
3. macOS hardware verify of `ephemeral-pg.sh` (needs a Mac).
4. Add the full commit-env export line to AGENTS.md Quick Reference.
5. Investigate + converge BuildFlow golangci vs `#lint` config resolution (example/ exclusions diverge).
6. Sweep ALL markdown tables repo-wide for unescaped pipes inside code spans (same truncation class as the Load-sweep row).
7. Repair or retire `docs/reviews/2026-08-14_14-25_brutal-self-review.html` — malformed (unclosed body) → 9 prettier parse findings EVERY hook run.
8. `go mod tidy` the 9 modules BuildFlow flags (cqrs-bench, metaengine-quickstart, integration, projectionadapter, middleware, projectionhost, schema, storage/bbolt, storage/pebble).
9. Decide the gomod-check "eventtest required but no replace" ×23 findings — audit against ADR-0045's documented exception.
10. Tame markdown-lint noise (735 findings, MD013/MD024 dominated) — exclude status/CHANGELOG or set line-length config; also fixes the MD024 duplicate "Added" headings pattern.
11. Add codespell ignore-list (`unparseable`, `deriver`, …) — 983 findings, mostly false positives on domain words.
12. Fix lychee empty-URL findings in AGENTS.md (2) + TODO_LIST.md (3) — empty `()` link syntax.
13. Fix vulnix wiring in BuildFlow (it prints usage help — invocation is wrong).
14. Add `homepage` + `mainProgram` to the two flake apps flake-meta-checker flags.
15. Extract flake.nix fixed-output hashes to dedicated files (nix-checker suggestion).
16. Decide go-structure-linter policy (13 findings: assets/, internal/, api/, examples/ dir-name heuristics vs this repo's layout).
17. buf-lint findings on transport/grpc proto — moot at v5 removal (ADR-0127); suppress until then.
18. Upgrade host Go to ≥1.26.6 — kills the GOTOOLCHAIN gymnastics class entirely.
19. `buildflow doctor` — clear the 9 unavailable-tool warnings.
20. Remove/justify the `.buildflow.yml` GOEXPERIMENT=jsonv2 preflight warning (verify claim "project does not import encoding/json/v2" — it DOES; the preflight check itself may be wrong).
21. go-licenses availability outside devShell — wire into hook env or silence.
22. The 74 go.work use-path-mismatch warnings — BuildFlow heuristic noise vs gitignored go.work; configure the exception.
23. Make tailwind-build hook step `--check` (or fix its output path) so it stops dropping `.out.css` artifacts; gitignore is a band-aid.
24. Extend `scripts/install-hooks.sh` to also export the commit-env (single source for hook ergonomics).
25. nilnil+iface enablement review for engine modules (carried item #25 from the prior report — this session proved both catch real smells).
26. Wire a CI step asserting `dprint check`/prettier clean on AGENTS.md/CHANGELOG/TODO_LIST (would have caught the pipe landmine before it landed).
27. Consider an art-dupl version pin (baseline semantics could drift across versions).
28. docs/status has both `archive/` and `archived/` dirs — consolidate.
29. Prior-session report footer said "verify-fast state: mine green, gate red" — now superseded by the ANSWERED card; consider a one-line pointer edit if reports get indexed.
30. gopls phantom-error noise — the documented GOTOOLCHAIN=local LSP flood; a devShell GOTOOLCHAIN=auto default would quiet every future session's diagnostics.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **There is a LIVE concurrent session RIGHT NOW** — `system/cache.go` modified
   and `system/cache_test.go` created at 10:32–10:34 today (minutes before this
   report). I have not touched those files. Is that session yours/intentional, and
   should future work in this window stay out of `system/`?
2. **Verification bar for this wave**: is verify-fast green sufficient to call the
   closeout done, or do you want the full `#verify` (soaks) + live Dgraph run
   before we consider the gate takeover "fully verified"?
3. **Lint truth ownership**: which runner is canonical when BuildFlow's hook and
   `nix run .#lint` disagree (example/ exclusions)? Should I make the hook use
   `.golangci.yml` strictly, or is the divergence acceptable hook noise?

---

_Every green claim above names its evidence: verify-fast EXIT=0 with "Lint: 76/76
modules clean" and "No new clones detected (baseline: 111 groups)" log lines;
per-module GOWORK=off build+test+lint runs; art-dupl check EXIT=0 after round 4._
