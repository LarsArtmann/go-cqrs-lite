# Status: Fleet-Libs Pareto Plan — W1–W3 shipped, W4 partial (T19 in flight)

> 2026-11-22 18:47 · Session: execution of
> [`docs/planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md`](../planning/2026-11-22_16-34_SUPERB-fleet-libs-audit-pareto-execution-plan.md)
> (T01–T25). Companion log with per-task verdicts + tag-prep lists:
> [`2026-11-22_fleet-libs-plan-execution-log.md`](2026-11-22_fleet-libs-plan-execution-log.md).

## a. What is done

**W1 version-truth sweep (T01–T04) — SHIPPED, commit `1e6ad6fd9` (101 files).**
go-sse uniformly v0.7.0 (41 modules swept; metaengine literal →
`sse.DefaultSubscriberBuffer`), go-idempotency uniformly v0.6.0 (5 modules),
httputil v1.9.0, cmdguard v5.1.0 ×5 tools, templ-components v1.27.2 (templ
regenerated with the pinned v0.3.1100 CLI from the canonical cwd;
docs-ui.css rebuilt), recipe fence pins aligned. All module tests green at
the new pins; CHANGELOG row landed; changelog-symbols gate green.

**W2 file safety + golden path (T05–T07) — SHIPPED, commit `e5276cfc4` (+ daemon splits).**
cqrs-upgrade + cqrs-lint write through go-atomic-write (kill-mid-write
crash-consistency test; codemod goldens unchanged); taskmanager gained the
production posture (Recovery outermost, 1 MiB MaxBodySize → 413,
httputil liveness/readiness probe, Server-Timing behind `-server-timing`,
drain order HTTP-before-projections, `MeasureWithDesc`); recipe §2.47
mirrors it (job table +3 rows, fence sync, prod-posture paragraph).

**W3 bridges + seams (T08–T14) — SHIPPED, commit `665153fed` (amended).**
SSE correctness: malformed Last-Event-ID → **400** before header commit
(was: silent full-journal replay), `WithSSEReconnectAfter` (retry:
advertisement), drop-old-vs-Broadcaster rationale in docs. ssetest/sseparse
adopted in our own harnesses (metaengine raw-TCP legs rewritten sleep-free
on typed asserts; systemscenario parser replaced; `ReceivedIDs()`). cqrs-bench
`system-memory` backend via `FactoryFromSystem` (presets marked legacy).
go-health seam: `/healthz` in the example + recipe fence #5. Docserver:
ETag 304s on the six deterministic routes (HTML deliberately unwrapped —
per-request CSP nonce changes bytes; decision recorded in code), buffered
templ render (clean 500s), `display.CodeBlock` swap. **go-appkit refreshed
AND released**: cqrs/v0.10.0 + docs/v0.3.4 tagged and pushed (tests green
first). New `example/scheduler-engine-status` (engine-path timers, fully
registered in all 6 gates). Rode along: repo-wide go.sum tidy repair (60
modules) + three DEP_BUDGET raises with rationale.

**W4 tail (T15–T25) — partial, mostly uncommitted:**

- T15 catalog D2 → go-output: **REFUSED by the plan's own gate** (see b).
- T16 retry consolidation: middleware loop → `retry.Do` — DONE, tests green
  (sentinels preserved by wrapping; jitter + deadline-vs-cancel now arrive);
  PublishRetry (event/v4) migration REFUSED on the v4 byte-freeze — decision
  recorded in the log.
- T17+T18 errorfamilytest sweep: 19 sites converted across 12 files/8
  modules (schema, command, id, storage, event, turso, integration,
  benchkit), all module suites green. Plan estimated ~98; only 25 real sites
  existed (audit overcounted the pattern) — 6 non-mechanical remain rowed.
- T19 cqrs-upgrade → cmdguard: **IN FLIGHT, zero code written** — skill
  loaded, design settled (see b).
- T20 signing DeterministicCodec guard: DONE, test green (pins both the
  stored-bytes contract and the codec marker for any future re-encoding).
- ~~T21 cqrs-gen dogfood: DONE — `//cqrs:command task.create` marker +~~
  ~~committed `handlers_gen.go` + README loop.~~ **Superseded — the dogfood
  never ran: `RegisterCreateTaskCmdHandler` had zero callers and no test compiled
  or executed generated code. Retired in `0b4d58409`;
  `example/goal-shaped-app` is the live dogfood (ADR-0168).**
- T22 go-sse CHANGELOG: DONE in the go-sse repo (commit `1b7d168`) —
  [0.7.0] section cut from Unreleased, dated to the tag.
- T23 audit infra: DONE — `scripts/check-libreports.sh` + `nix run
  .#check-libreports` (citation sweep + index↔report links; `--self-test`
  plants and repairs faults). First live run caught a stale citation MY OWN
  T05 edit caused → re-derived + report annotated. **Local gate by
  construction** (siblings live in ~/projects; CI cannot resolve them).
- T24 session infra: DONE — status row for the audit report landed via
  `add-status-row.sh` (m69); border-before-trim screenshot lesson committed
  to crush-config `bc637bf` (m70); sequential-dispatch+resume pattern
  committed to the library-deep-dive runbook `f4cd4a8` (m71).

## b. What is partially done or broken

- **T19 (cqrs-upgrade → cmdguard) is mid-integration.** cmdguard skill
  loaded; design settled: flat root command on `NewCLI[T]` with all 7
  existing flags as typed T-fields, `RootCommand().RunE` adapter into the
  unchanged pipeline, tests ported to `ExecuteWithArgs`. NOT started in
  code; the plan's m60–m62 remain open. Deviation flagged: the plan's
  "subcommands (v5-mode)" would break the flat flag surface callers/tests
  use; compat-first chosen, owner ratification requested (question 2).
- **W4 work is uncommitted** (~25 files staged/dirty): retry.go, the
  errorfamilytest sweep, signing test, cqrs-gen artifacts, check-libreports
  - flake app, .gitignore (stray-binary pattern), status README row. All
    listed suites are green; the wave lacks its boundary commit + log section.
- **T15 refusal not yet written into the execution log** (decision made in
  session, recorded here): `escape.SlugifyID` lacks ToLower and adds
  bracket/asterisk handling vs `sanitizeID`, and go-output/d2's renderer
  cannot reproduce the pinned D2 bytes — "golden unchanged" is unreachable
  without re-baselining, which the verschlimmbesser guard forbids.
- **T25 closeouts untouched**: `#verify-fast` after the tree settles, the
  `level=warn` stray (needs owner confirm → trash), error-taxonomy doc
  header v0.10.0→v0.12.0, and the local-vs-remote tag policy (Q3).

## c. What is left (priority order)

1. T19 implementation (m60–m62) per the settled design + suite port; then
   the W4 boundary commit + execution-log W4 section.
2. T25 closeouts (m72–m75) — mechanical once the tree settles.
3. Decide the W1–W4 pin-wave: the changed modules (list in the execution
   log's m06 section) are candidates for `batch-release.sh` — tests-first
   per AGENTS (release verify=ok does NOT run tests; this session ran the
   per-module suites green but a fresh pre-tag sweep is still owed).

## d. Testing and verification

Green at execution time: metaengine (full + new SSE tests), systemscenario,
system, middleware (full, after retry.Do swap), commandlifecycle,
idempotency/sqlstore, integration/-short, example/taskmanager (full + race),
example/scheduler-engine-status, catalog (incl. docserver + 304 test),
cqrs-lint, cqrs-upgrade (kill test + goldens), cqrs-bench (77s), benchkit,
schema/command/id/storage/event/turso (errorfamilytest sweep), signing (new
determinism test), go-appkit cqrs/docs/integration, doc-check recipes
harness (28s). Gates: check-templ (both legs), check-csp (browser),
check-arch (after 3 budget raises), check-test-matrix, md-go, example-
standalone (2 examples), changelog-symbols, TestEvery (api-stability,
after the 60-module tidy repair), check-libreports (900/0/23) + self-test.
NOT yet run: `#verify`/`#verify-fast` (T25), the full `#test` matrix over
the W4 dirty set, `nix fmt --fail-on-change` over W4 files (gofmt applied
per module).

## e. Next steps

1. Owner: answer the three questions below; T19 design + T15 verdict hang
   on two of them.
2. Implement T19, commit W4 boundary, write the log's W4 section (including
   the T15/T16-m55 refusal records).
3. Run T25: verify-fast, stray disposition, taxonomy header, tag policy.
4. If the pin-wave is green-lit: tests-first sweep over the changed set,
   then `batch-release.sh` with the m06 module list.

## f. Failure ledger (this session, honestly)

- **Stray 100755 binary rode the W3 commit** (`example/scheduler-engine-status/scheduler-engine-status`
  from a bare `go build ./...`): caught post-commit, amended out, gitignore
  pattern added. Lesson: build with `go vet`/`go build -o /dev/null` inside
  modules, or check `git status` for executables before `git add -A`.
- **`rg -rn` hook strikes ×3** (the documented trap — I re-hit it in this
  session despite the lesson existing).
- Self-test fixture bugs (line-count out of citation range; missing asset
  reference) — two debug rounds before the gate's self-test went green.
- python-subprocess `rg` read the heredoc stdin (rc 1, zero files) —
  explicit path argument fixes it; cost a debugging detour.
- The appkit changelog-cut script crashed midway (marker mismatch); manual
  repair. `git tag` needed `-a -m` (editor refused under `editor false`).
- The 304 test initially asserted HTML routes that templ's runtime
  flush-streams (and later, nonce-varies) — root-caused via probes before
  redesigning to data-routes-only. Right outcome, slower than planned.
- The errorfamilytest sweep converted 19 of 25 sites mechanically; 6
  non-standard shapes remain hand-work (rowed, not silently skipped).

## g. Owner questions

1. **T15 verdict:** accept the refusal (keep the hand-rolled D2 DSL + its
   goldens; reconsider when go-output/d2 grows a byte-compatible renderer
   or the D2 format is deliberately re-baselined at v5) — or commission the
   re-baseline now?
2. **T19 shape:** flat-flags compat (settled design; `--to-v5` stays a
   flag, no invocation break) vs the plan's subcommand restructure
   (cleaner chassis, breaks the flag surface scripts/tests use)?
3. **Pin-wave + tag policy (Q3 from the audit, now concrete):** tag the
   W1–W4 changed modules as one batch-release wave now (tests-first), and
   should future version stamps derive from remote tags (`git ls-remote`)
   instead of local tags? The citation gate being local-only (m67) folds
   into this: CI cannot see sibling checkouts by construction.
