> **RESOLVED — docs-health 9th pass (2026-09-20):** Fully closed: W0 verified end-to-end; T01 composed verify GREEN (S03, 2026-09-20 15:04 — 16-39 report); T13 load-sweep + T15 verify-ci GREEN (16-39 §a); T14 deliberately deferred to the next quiet window (recipe in 16-39); W2 slices landed (T27 pushdown, T28 ctx-tx ports, T22 stamps, T29-docs, T44-selftest, T47/T48). Remaining: the W3 owner bundle (`2026-09-20_11-36_owner-bundle-w3.md`, OPEN) and TODO_LIST-tracked tails.

# Publish-and-Prove Execution: W0 verified end-to-end, W2 code fixes landed, verify gated on host load

> Session start 2026-09-19 22:44, resumed post-reboot 08:39 (host rebooted ~04:15; all
> background pipelines died and were relaunched). Directive: execute the SUPERB
> publish-and-prove plan (`docs/planning/2026-09-19_22-34_SUPERB-publish-and-prove-pareto-plan.md`)
> end to end. Machine reality: a 92-tag release train shipped from a PARALLEL session at
> 21:08–21:45 (before this session's plan was even written — the plan's "0 published" §0
> claim was stale), an autonomous task-queue agent fleet drives continuous nix builds
> (load 15–171 all night, never a <10 window for more than minutes), and the auto-commit
> daemon absorbed six authored commits mid-flight.

## a) FULLY DONE (verified this session)

1. **W0 PUBLISH — verified end-to-end, completed to its acceptance bar.** The train was
   cut+pushed by the parallel session; this session verified and finished every tail item:
   - Tags: 1314 local == 1314 remote; all 92 wave tags match the batch manifests; sampled
     tags carry ZERO local replaces with coherent pins.
   - Proxy: `@latest` resolves all 10 probed key modules to wave versions from a clean
     dir; proxy pre-warmed for all 92 (three awk/sed mangling rounds — see §d).
   - **GitHub Releases: 92 created** (152 repo total) — `create-github-releases.sh`
     extended to match release-train sections (bounded-token match on full tag or last
     two path components, newest-section-first, trimmed body + CHANGELOG pointer like
     the 09-08 precedent) + `--dry-run`; all 92 extract, bogus tags skip.
   - `pin-sweep.sh --check --remote` green; `check-retracts-shipped` green (5 modules);
     `tag-release.sh --audit --baseline` green (24 known, 0 NEW; CI leg `#check-tag-audit`
     already existed — T11 was already done, verified rather than rebuilt).
   - Clean-dir `go list -m @latest` acceptance (T10) green.
   - **T06 smoke: 91/92 modules fully smoked green** (`--smoke-all` over the corrected
     wave file); taskmanager's tag is proxy-served and its dependency tree installed
     green through the proxy repeatedly — its final full-binary install was killed four
     times by host contention (see §b), not by a tag defect.
2. **Post-wave integrity repairs (the wave's own residue):**
   - Repo-wide `GOWORK=off go mod tidy` sweep: 13 go.sums were missing `/go.mod` hashes
     (claiming/sqliteengine transitive pins) — `TestEveryModuleGoSumIsTidy` red → green.
   - `cmd/cqrs-lint/testdata/typedfixture` re-pinned (schema v4.4.0→v4.4.1) — the stale
     fixture was the REAL cause of verify attempt 2's P014/V007 failures, not the
     env-divergence hypothesis the 00-19 session was pursuing.
   - `scheduling/engine` registered in flake `testModules` (missed at module creation;
     check-workspace-sync was failing every code commit) + tidied.
   - Four committed binaries untracked + ignored (typedfixture 5.9MB daemon accident,
     ec-fixture, goal-shaped-app 27MB, scheduler-otel-status 22MB).
3. **T16/M19/M20: api-stability fully green** including the foreign
   `readme_claims_test.go`, TestEvery, and the tidy gate.
4. **T26 (3-session ask): quiet-window tooling** — `wait-for-quiet.sh` (1-min AND 5-min
   ceilings, planted-fixture `--self-test` that caught two real bugs on first run),
   `can-run-composed-gate.sh` (no-release-procs + tree-stability-window + load assert;
   canary mutation-tested), `#verify` `-p 4` parallelism cap (VERIFY_TEST_P override),
   and the cqrs-lint typed-fixture helpers made fail-loud (packages/TypesInfo/LoadErrors
   counts + repro command).
5. **T20: canonical-facts gate** — `check-canonical-facts.sh` derives go.mod count,
   module-map census, and recipes-catalog size from the repo and fails on any disagreeing
   doc citation. First real run caught the recipes 80→81 drift live. Mutation-tested;
   wired into nightly-gates.
6. **T19: doc-annotations gate** — archived reports must carry a resolution marker
   (banner or `~~`) or sit in the conscious 1128-entry pre-convention baseline; live
   reports >14d without KEEP-LIVE surface as drift. The first self-test draft passed
   vacuously (env override ignored) — rebuilt with honored CHECK_DOCS_ROOT hooks.
   Wired into nightly-gates.
7. **T27 (P0): camelCase planned-column pushdown fixed** — `metaengine.ExtractFields`
   matched columns only against json tags; Go-name columns over snake tags extracted
   NULL and every pushdown filter silently matched nothing (2026-09-18 CRM). Now matches
   tag OR Go field name; map path falls back to acronym-aware snake_case
   (`ParentID`→`parent_id`; the naive splitter's `parent_i_d` caught by tests before
   shipping). 3 parity tests; metaengine/adttest/sqliteengine green; CHANGELOG entry.
8. **T28: ctx-scoped transactions ported to pg/mysql/duckdb engines** — the
   engine-global `activeTx` (dirty reads + `sql: Rows are closed`) replaced by the
   sqliteengine 22ab7b218 ctx-marker pattern in all three; `conn(ctx)`, nested-tx
   rejection, ambient fast paths keyed off the marker. Isolation tests per engine: PG
   validated live (testcontainer suite green, 33s), MySQL build+vet green (live-gated),
   DuckDB full suite green 74s — its sqlite stress companion deliberately NOT ported
   (go-duckdb C-level prepare convoy on file DSNs; documented in the test). Workspace
   build + system consumer tests green.
9. **T47/T48: docs census + hygiene** — module-map rows all 95 go.mods (8 added,
   scripted-diff verified, banner + AGENTS updated); TODO_LIST `[x]` sweep deleted all
   34 done blocks per header policy (104 open rows remain; doc-links 0 broken).
10. **T22: experimental stamps** — all 17 metaengine-family modules carry doc.go
    package comments with the `# Experimental` heading pkg.go.dev renders; stray
    package comments consolidated.
11. **T12/T9 handoffs recorded** — TODO_LIST Release section now carries the train
    summary, post-wave verification evidence, and the consumer-propagation handoff
    (CV, go-taskqueue, PapDashboard, go-graph-rag, go-localsync). 16 release-wave rows
    struck with evidence.

## b) PARTIALLY DONE

1. **T01 composed `#verify` — attempt 4 is QUEUED, not run.** The patient pipeline
   (`wait-for-quiet` → `can-run-composed-gate` → `#verify`) is honest: load never held
   under 10 for the tree-stability window (fleet builds drove 15–171; one window at
   09:2x was refused because my own smoke-tail processes were alive — correct refusal).
   The typedfixture root cause that killed attempts 1–3 is fixed and the -p cap
   de-risks the timeout class; the run is mechanically ready the moment the host quiets.
2. **T06 taskmanager full install smoke** — proxy-verified + dependency-green, but the
   binary install itself was starved four times (three duplicate smoke trees from the
   release session + mine were discovered and killed via /run/current-system pkill —
   the interpreter's `kill` builtin is unsupported and silently no-ops, see §d).

## c) NOT STARTED (sequenced behind the quiet window / owner rulings)

1. **T13/T14/T15** — load-sweep, benchmark baseline re-pin, `#verify-ci` + verify-docs
   e2e + S03 record: all want the same quiet window as T01, in that order.
2. **T17** calibration re-anchor (same window class).
3. W2 slices not yet touched: T21 (external repo), T23–T25, T29–T46 individual slices.
4. W3 rulings (G-T01, Zenoh, 350-line, ADR-0139, billing, Turso filing approval,
   annotation-gate ratification — the last now partially moot: T19 shipped the gate).

## d) TOTALLY FUCKED UP (honesty ledger)

1. **Three shell-mangling rounds building the smoke list.** `awk NF--` with default OFS
   joined nested module paths with SPACES (`storage memory`), then OFS=/ removed the
   separator entirely (`storage/memory/v4.5.2` as one token). Two smoke runs and a
   prewarm burned before `printf` + explicit verification fixed it. The wave files'
   own format was correct all along; I rebuilt what existed.
2. **`kill` silently no-ops in this shell** — two "kill" rounds did nothing while three
   duplicate taskmanager smoke trees hammered the box; `kill -9` returned rc=2
   ("unsupported builtin") hidden by 2>/dev/null. `/run/current-system/sw/bin/pkill`
   worked first try. Same class as pipeline-masking: verify the rc, not the intent.
3. **GOTOOLCHAIN=local leaked into two pre-commit hooks** (one forgotten env prefix)
   — the hook's `go build` died on go.work's 1.27.1 requirement. The fix is documented
   in gotchas line 42; I paid it twice anyway.
4. **Two first-draft self-tests passed vacuously** — wait-for-quiet's (garbage probe
   read as load 0 via awk string coercion) and check-doc-annotations' (env overrides
   not honored; the fixture's own numbers lied). Both caught by their own mutation
   legs — the self-test discipline works, but only because I kept the legs honest.
5. **Ported the stress test before asking whether DuckDB can run it** — 13 minutes of
   C-level prepare convoy in the panic dump before accepting the driver limit. Should
   have profiled the donor's assumptions (WAL multi-connection) against DuckDB first.
6. **Six authored commits absorbed by the daemon** (typedfixture fix, tooling wave,
   facts gate, extract fix .go files, engine port .go files, stamps) despite
   re-checking `git status` before add — the daemon's window is seconds, not minutes.
   Content complete; history says `chore:`. The stale `.git/index.lock` class hit
   twice more (cleared via trash after proving no live owner).

## e) WHAT WE SHOULD IMPROVE (structural)

1. The verify pipeline should LIVE in the repo: a `#verify-when-quiet` flake app that
   wraps wait-for-quiet + can-run-composed-gate + verify — tonight's nohup pipeline is
   a /tmp artifact that died with the reboot.
2. `tag-release.sh --smoke` should detect a duplicate live smoke of the same module
   (three trees on taskmanager!) — the release-session-alive check exists in
   can-run-composed-gate but not in tag-release itself.
3. The interpreter's broken `kill` builtin deserves a gotcha line: it resolves via
   `command -v` but no-ops at runtime with rc=2.
4. Smoke probe GOTOOLCHAIN: `--smoke` inherits the caller's env; a `GOTOOLCHAIN=local`
   shell makes every proxy poll fail invisibly (the probe's `go list` dies on go.work).
   tag-release.sh should set GOTOOLCHAIN=auto explicitly in its env-clearing probe.

## f) NEXT (ordered)

1. When load holds: `can-run-composed-gate.sh --wait` → `#verify` (attempt 4) → S03
   record (date/commit/durations) into the TODO composed-verify row.
2. Same window: T13 `#load-sweep`, T14 `benchmark-regression.sh --save` (provenance
   header: Go 1.27.1 + claimkit + tonight's engines), T15 `#verify-ci` + verify-docs.
3. Opportunistic: taskmanager full-install smoke (one quiet compile).
4. W2 slices in impact order: T44 (golangci ownership guard), T43 (Doctor refused-ADTs),
   T29/T30 (queue polish), T32–T34 (benchkit slices), T35–T38 (README/goal-app tails).
5. W3 owner bundle (unchanged): G-T01 ruling, Zenoh, 350-line ratification, ADR-0139,
   billing fix, Turso filing approval.
6. Consumer propagation handoff executes consumer-side (CV, go-taskqueue,
   PapDashboard, go-localsync — the workaround dies with `watermill/v4.6.1`).

## g) QUESTIONS FOR THE OWNER

1. Same three from the 8th-pass §g stand: annotation-gate ratification (now largely
   MOOT — T19 shipped the marker-OR-banner gate with a conscious baseline), the 92-tag
   wave authorization (MOOT — cut, pushed, verified), and `readme_claims_test.go`
   ownership (verified green this session; still unclaimed).
2. The iroh P99 50→150ms ratification row is still open (shipped + gated green; keep
   or revisit).

— verify attempt 4 remains queued behind the load gate at session close; no gates
running; tree clean; logs at /tmp/verify-run3.log (gate), /tmp/smoke-all3.log (91/92),
/tmp/duck-run3.log (duckdb green).
