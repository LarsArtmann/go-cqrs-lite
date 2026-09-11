# Status Report — SUPERB Pareto Waves 1–2 (continuation session), 2026-09-09 ~02:00 CEST

> **RESOLVED (docs-health pass 2026-09-11):** **Superseded — archived by the docs-health pass 2026-09-11.** Its 'next' list was executed by `2026-09-09_04-10_pareto-w2-w3-execution-green-verify.md` (P19 tail, P20, P21, P23, P24, P25, P26, P27-feasible, exclusive `#verify` GREEN).
> Open work lives in [`TODO_LIST.md`](../../TODO_LIST.md); shipped surface in [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]`.


**Scope:** continuation of `docs/planning/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`
(W0 done prior; this session executed W1-remainder + W2 chunks). Working tree CLEAN
(auto-commit daemon absorbed; local master `c23e1847e` is 7 commits AHEAD of origin `d2bb71fbe` — not yet pushed).

## What got DONE this session

- **CI re-triage (final commits):** no run processed after 20:33 (runner-minutes/billing — user-gated).
  Failure classes on the 20:33 run: FlakeHub-auth (dominant, billing-gated), File Size 350 (real),
  **WASM leg: REAL BUG found+fixed** (job built deleted `codec/` dir since ADR-0128 extraction),
  go.work-sync "version drift" PASSES locally at final state (judge on next CI run),
  gosec failure was cache-throttle class, not findings.
- **P07 benchkit load-scaling — DONE, root-caused (not just scaled):** the closed-store
  "expected error, got nil after ~26s" flake was a SEMANTICS GAP: a caller deadline expiring
  before any phase ran made every phase "gracefully skip" and `Run` return `(partial, nil)`.
  Product fix in `benchkit/runner.go` (`runPhases` not-started guard, Duration-bounded runs keep
  partial semantics) + `TestRun_ExpiredContext_ReturnsError` pin + load-scaled test budgets
  (`loadScaledCeiling` over `soakTestScale` in `mustRun`; ClosedStore ctxs) + system hardening
  tests' outer ctxs aligned with the already-scaled inner wait budgets (4 tests).
  Verification: fast timing classes 3×`-race` GREEN (65s); Pebble+Recovery 3×`-count=1 -race` GREEN
  (93s/247s/55s); system hardening suite GREEN.
- **P14 cqrs-upgrade growth — DONE:** `--strict` (v5-readiness gate exit code), `--json`
  (deterministic struct-order wire format), `--to <version>` (ceiling clamp, never downgrade —
  `held` status), `--workspace` (multi-module tree walk, skips vendor/testdata/.git/node_modules);
  `--dry-run` now includes the deprecation report. 6 new tests. Dogfooded LIVE against
  `example/getting-started`: all 11 pins up-to-date, zero v5 findings, exit 0 (also re-validates
  yesterday's release train). New CI job `cqrs-upgrade-dogfood` (plain go, no nix cache needed).
- **P17 CI wiring — DONE (minus user-gated pieces):** `check-csp` CI job; `lint-scripts` CI job
  (actionlint+shellcheck) with ALL 16 pre-existing shellcheck findings FIXED (real fixes + scoped
  `disable` directives with reasons for intentional word-splitting/glob passthroughs); NEW
  `.github/workflows/sentinel.yml` (nightly days-since-green alarm ≥3d + weekly EventCatalog
  render validation); pre-commit gained 3 staged-aware cheap gates (shellcheck on scripts/,
  workspace-sync on go.mod/go.work/flake.nix, changelog-symbols on CHANGELOG.md).
  actionlint GREEN on both workflows. M17.2 was ALREADY covered by the existing
  Module Isolation Build job (GOWORK=off builds fail on missing go.sum under -mod=readonly).
- **P18 coverage — verified GREEN:** all 11 documented modules within ±2% tolerance
  (kv actually improved 71.9→73.7; worst documented gap remains kv ~74%). Gap-closing
  beyond tolerance-repair remains open (polish class).
- **P19 first split wave — DONE (3/3 files):** `storage/sql/dialect.go` 590→94 (+4 per-dialect
  files), `cmd/cqrs-lint/pkg/rules/architecture/helpers.go` 628→182+241+224 (composites/project
  concerns), `metaengine/typed_reader.go` 1127→161+286+129+332+120+119 (scan/aggregates/grouped/
  options/cursor concerns). All files now <350; builds+vet GREEN; targeted tests GREEN;
  doc comments preserved correctly at every split boundary (caught + fixed 6 dangling/lying docs).
  Full metaengine suite re-run was still in flight at report time (targeted reader/aggregation
  tests + build + vet already GREEN).

## What is next (ranked)

1. Finish P19 verification: full metaengine suite result + per-module lint on touched modules +
   `check-duplication` + CHANGELOG entry for the split wave.
2. P20 ApplyLayout rule behind `--typed-info` (gate exists since P11) + fixtures.
3. P21 encryption docs + wire-format golden + v1↔v2 symmetry test.
4. P23 repo hygiene (gocognit pg_integration_test, sqlstore lint findings, aggregate-code
   tripwire, 5 clone groups, awaitAck log line).
5. P24 docs truth batch (error-taxonomy, DOMAIN_LANGUAGE matview terms, exhaustruct canary,
   templ tripwire, doc-check --json, example READMEs).
6. P25 v5 sweep §4 — safe-first order: benchkit key, bbolt CBOR tags, v6 markers, wire-key table
   doc, watermill dual-read; SQL column renames only after assessing the T18 migration interplay.
7. P26 T18 tail + V5-MIGRATION-GUIDE expansion (live MariaDB on :33061 available).
8. P27 feasible chunks (tag-release.sh proxy smoke-check, cqrs-bench stub + v4.8.0 retract,
   version-reporting decision…).
9. Push local master (7 commits ahead) — bundled with the next work chunk or user ping.

## How I know (verification evidence)

- benchkit/system test logs: `/tmp/p07a.log` (timeout kill — superseded), `/tmp/p07c.log`
  (3×race fast classes `ok`), `/tmp/p07d.log` + job 23D (Pebble 3× `ok`), `/tmp/p07b.log` (system `ok`).
- cqrs-upgrade: module tests `ok`, live dogfood transcript (11 pins up-to-date, exit 0).
- shellcheck: `SC_EXIT=0`, 0 findings repo-wide on scripts/*.sh; actionlint exit 0 both workflows.
- coverage: `/tmp/p18.log` all `ok` within tolerance.
- splits: per-module `go build` + `go vet` GREEN; `storage/sql` tests `ok`; architecture pkg
  tests `ok`; metaengine targeted tests `ok`.

## Surprises / lessons

- The ClosedStore "flake" was not timing — it was a lying success path (all-skip ⇒ nil error).
  Load-scaling alone would have papered over a real semantics defect.
- shellcheck directives: trailing same-line directives are INVALID (SC1126) and a comment inside
  a `\`-continuation breaks parsing; directives must sit ALONE on the line before the command.
- Sed-based file splitting is fast but doc comments crossing the cut need a boundary pass —
  one comment ended up lying on the wrong function (worse than dangling).
- Exit-code-after-pipes bit again in the first actionlint check (`| head` reported 0);
  re-ran with proper capture.

## User-gated (unchanged)

CI billing/FlakeHub creds (blocks red-leg triage + new nix jobs actually running), PR #8257
permalink + turso upstream A+B filing (P22), strict-vs-lenient typo'd-DSN ruling, 350-line
policy ruling (ratchet vs exemptions; first split wave landed without it), doctor-JSON
pre-merge semantics, next tag-wave authorization, EventCatalog package-lock pinning decision.
