# Session-7 Continuation — Execution Status (2026-09-01 00:11 CEST)

> Continuation run of the session-7 execution (plan `ad77f5223`, prior report
> `2026-08-31_16-28_session-7-execution-status.md`). Scope of THIS report:
> the continuation session only — leftover-commit completion, ledger closes,
> T33 pin sweep, T35 exclusion audit. User directive: execute remaining plan
> items one at a time; interrupted for this status report; WAIT after.
>
> HEAD at report time: `7be10e144` (+ this report commit). Unpushed: `761b6d2c0`,
> `7be10e144` (+ this report commit). Working tree otherwise clean.

## a) FULLY DONE (verified, committed, gated)

1. **Leftover-commit completion — resolved by reality, not by me.** The
   interrupted `git add`+commit had already landed as `f9aad87bc` (all 4
   files: status report, SKILL.md row, dgraph soak hatch, turso CTE test),
   and **origin/master == HEAD**: everything through `f9aad87bc` is already
   PUSHED (user or daemon). Plan T38 (push) is closed; session-7 §g Q1
   (push policy) is moot for that set.
2. **Post-commit lint of the two just-committed modules** (`#lint-module`):
   tursoengine GREEN on first run. dgraphengine had **2 findings** in the
   just-committed soak file (golines >120 skip message + gofumpt import
   blank line) — fixed (message shortened, rationale moved to doc comment,
   gofumpt applied), re-lint GREEN, test compile green. Commit `761b6d2c0`.
3. **CHANGELOG [Unreleased] entries for the session-7 API surface** — the
   gap flagged by the prior report §b.3: `metaengine.LayoutPlanEvolver`
   (pg/mysql `EvolveLayoutPlan`), `metaengine.KeyScanBackend` +
   `BackfillPlannedCollection` + `ErrBackfillUnsupported`,
   `PlannedTablesReporter` + `PlannedTableInfo` +
   `PlannedTablesDoctorSection`, `EffectiveDurability` adoption on
   badger/bbolt/pebble/sqlite/pg, `adttest.RunPlannedOpsMatrix` +
   `Factory.PreClean`, and the duckdb MapScan planned-routing Fixed entry.
   Symbols gate GREEN: **143 citations verified**.
4. **api-golden scare investigated and closed**: `RunPlannedOpsMatrix`/
   `PreClean`/`RunMatrix` are absent from `docs/api_surface.txt` because
   `cmd/api-stability/collect.go` reads only module-ROOT packages — adttest/
   enginetest are packages-inside-modules and structurally invisible. NOT a
   golden-regen miss; no action.
5. **TODO_LIST ledger closes — 10 rows**: D3-slice-4 parity matrix,
   information_schema evolution, backfill helper, Doctor planned tables,
   durability tiers, `-shuffle=on` verdict, SOAK_SKIP_* dgraph explanation,
   turso CTE probe, the session-4 `LayoutPlanApplier` PARTIAL row (all four
   REMAINING items now landed), plus the turso row. WS-M row (G1
   calibration) deliberately untouched (concurrent session's).
6. **AGENTS.md**: turso libSQL recursive-CTE gotcha line added.
7. **doc-check gate GREEN** (956 refs, 42 packages) after the AGENTS edit.
8. Commit `7be10e144` (CHANGELOG + TODO_LIST + AGENTS).
9. **T35 exclusion audit — evidence phase COMPLETE for the linter blocks**
   (see b for the not-yet-applied edit):
   - **Decisive discovery**: golangci resolves exclusion-rule `path:` regexes
     relative to the **CONFIG file's directory (repo root)**, NOT the module
     CWD — proven by paired probes (throwaway global+named-return file:
     suppressed in `metaengine/`, fired in `kv/`, reported as `kv/…`). All
     repo-prefixed rules are ALIVE. Nothing was committed from the probes;
     both throwaway files trashed immediately.
   - **Broad-block probe** (4 blocks removed from a probe config — the three
     known ones + a second `path: system/` rule at line 780 — run over 18
     modules): per-linder finding counts collected. Zero-finding (stale)
     exclusions identified: `metaengine/` block → drop `cyclop`, `gocyclo`,
     `funlen`; `system/` rule 1 (line 466) → drop `gosec`,
     `gochecknoglobals`, `gochecknoinits`, `godoclint`, `musttag`, `revive`,
     `gocyclo`, `cyclop`, `maintidx` (9 of 19!); `system/` rule 2 (line 780)
     → drop `goconst`, `funlen`, `golines`; `cmd/cqrs-lint/` → drop
     `gocyclo`, `maintidx`. **17 stale linter exclusions total**, each
     proven by zero findings with the block fully removed.
   - Bonus evidence: `metaengine/sqliteengine` produced **zero findings with
     the whole blanket removed** — fully clean module.

## b) PARTIALLY DONE

1. **T33 record/v4 consumer-pin sweep** — running at report time (~55/66
   modules processed, 50+ OK): full enumeration, `GOWORK=off go test -run
   ZZNONE` per module (tests compile too — the MarshalBinary lesson). **1
   REAL FAILURE found**: `example/getting-started` pins `storage/v4 v4.6.0`
   (indirect) which does not compile against `scheduling/v4 v4.3.0`'s
   branded `TimerID` (`timer_store.go:69/77`) — the exact known
   version-sequence class; fix is bumping the pin to `storage/v4.8.0` +
   `go mod tidy`. Log: `/home/lars/projects/.gotmp/record-pin-sweep.log`.
2. **T35 config edit — NOT applied.** The audit data (a.9) is complete but
   `.golangci.yml` is untouched; the edit + `config verify` + affected-module
   re-lint + commit remain.
3. **Formatter-exclusion probe — aborted.** Probe #2 (are the
   `formatters.exclusions.paths` entries `event/`, `event/v4/eventtest/`,
   `metaengine/sqliteengine/`, `storage/view/` stale, given treefmt formats
   them under the `nix fmt` CI gate?) died mid-run ("context canceled") with
   a script bug: the config-edit step matched only 2 of 5 entries (indent
   mismatch; 3 WARNed) and I did not diff the probe config before running.
   One solid finding survives: the `event/eventtest/` entry is DEAD (no such
   directory). Probe needs rewrite (correct indent + diff-assert) + re-run.

## c) NOT STARTED

1. T35 final: apply the 17-exclusion drop + engine-rule redundancy prune
   (e.g. pgengine rule's wrapcheck/varnamelen/wsl_v5 are blanket-covered)
   - `nix run .#check-lint-config` + re-lint of affected modules.
2. Formatter-exclusion cleanup (b.3 re-run).
3. Plan-HTML execution-status banner (deliberately deferred until after
   verify so it records final truth).
4. `nix fmt` full-repo convergence (prior report §b.5).
5. T36 load-sweep (exclusive window).
6. T37 full `nix run .#verify` (SOAK_SKIP_BOLT=1) — the unproven GREEN.
7. Push of the 3 new commits (rule: never push without explicit ask).
8. T39 GitHub Releases / tag wave — user-gated (session-7 §g Q2, open).
9. AGENTS note for the golangci path-base discovery (probe method).

## d) TOTALLY FUCKED UP

1. **`f9aad87bc` shipped with lint findings** — the close-out commit's
   dgraph soak file carried a >120-char skip line and a gofumpt import
   violation; the commit message asserted pre-verification that had not
   covered that file. Third occurrence of the red-gate-commit class. Root
   cause: committed from an interrupted flow on "already vetted" claims
   without a fresh `#lint-module` in the same action. Fixed `761b6d2c0`.
2. **Formatter probe was sloppy four ways**: launched while the pin sweep
   was still running (two heavy jobs); probe-config edit silently matched
   only 2/5 entries (indent bug) — and 3 WARNs did not abort the script; no
   `diff` of probe-vs-original config before running; the run died
   "context canceled" and left `.golangci-probe2.yml` untracked in the repo
   root until cleaned at report time.
3. **Foreground long probe**: ran a 3-5 min probe as a foreground command
   (auto-background threshold) — it got canceled. Multi-minute jobs must be
   `run_in_background` from the start.

## e) WHAT WE SHOULD IMPROVE

1. **Verify-then-commit, same action**: any commit touching module code gets
   a fresh module lint (and test compile) immediately before `git commit`,
   regardless of prior claims from interrupted flows.
2. **Probe configs are code**: `diff` them against the original before use;
   make the edit step FAIL LOUDLY if the removal count ≠ expected.
3. **Serialize heavy jobs**: one sweep/probe at a time; exclusive windows
   for timing-sensitive gates (unchanged lesson, honored for load-sweep/
   verify planning).
4. **Record the golangci path-base discovery in AGENTS** (config-dir
   relative paths + the throwaway-probe method) so no future session
   re-derives it.
5. Background-job hygiene held up well this run (disk logs under
   `.gotmp/`, `timeout -k` wrappers, no `| tail` traps) — keep it.

## f) NEXT — prioritized (18, do-first at top)

1. Collect T33 sweep final tally; fix `example/getting-started` storage pin
   (`v4.6.0` → `v4.8.0`, tidy, GOWORK=off re-probe); close the ledger row.
2. T35 edit: drop the 17 proven-stale linter exclusions (+ prune redundant
   engine-rule entries), `check-lint-config`, re-lint affected modules
   (metaengine×16, system, cmd/cqrs-lint), commit.
3. Rewrite + re-run formatter probe #2; drop stale formatter paths (incl.
   the dead `event/eventtest/` entry) or record version-skew evidence.
4. AGENTS: golangci exclusion path-base + probe method note (S).
5. Plan-HTML execution-status banner (S).
6. `nix fmt` convergence + commit any drift (S).
7. T36 load-sweep, exclusive (M).
8. T37 full `#verify` with SOAK_SKIP_BOLT=1, exclusive window, background
   log; fix any fallout; record honestly (M-L).
9. Push the accumulated commits — user-gated (see g Q1).
10. T39 GitHub Releases via `scripts/create-github-releases.sh` — gated on
    the tag-wave decision (g Q2).
11. WS-M remainder: pg/mysql `CounterGet` live recalibration — concurrent
    session's workstream (g Q3).
12. dgraph suite `-shuffle=on` evaluation (the one suite not yet covered by
    the adopt verdict).
13. cqrs-lint ApplyLayout→ApplyLayoutPlan rule design pass (design-gated).
14. `exhaustruct` → `exhaustruct_v5` migration (deprecation warning fires on
    every lint run today).
15. Blanket-exclusion burn-down, module by module — sqliteengine is proven
    clean; the metaengine blanket keeps 20 linters for the others (hundreds
    of suppressed findings; NOT a quick win, do deliberately).
16. LSP go-mod-tidy warnings: `catalog` (templ-components should be direct),
    `watermill` (dedup unused) — investigate/tidy.
17. Session-7 §f carry-forwards still open from the prior report (D3/D8
    trains, observability, v5 train) — see that report's §f, still valid.
18. Fix the ~1 FAIL-class per sweep going forward: add the record-pin sweep
    script to `scripts/` + CI leg so strand-class breakage is caught
    pre-tag, not by hand (proposal, small).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Push policy (new commits)**: `761b6d2c0` + `7be10e144` (+ this report
   commit) are unpushed. I do not push without an explicit instruction —
   and someone/something pushed `f9aad87bc` between sessions. Should I push
   on GREEN once T36/T37 pass, or do you want to push yourself / stay
   unpushed?
2. **Tag wave go/no-go (T39 GitHub Releases)**: no session-7 tags exist
   (latest tags predate the new API surface: LayoutPlanEvolver, backfill,
   observability, durability, ClaimMetrics…). Releases require a tag wave
   (39+ tags last time, with pin sweeps). Go, no-go, or defer to a combined
   wave with WS-M?
3. **WS-M ownership**: is the concurrent session (pg/mysql CounterGet
   recalibration, ADR-0133 row) still active? If it has ended, should I
   take over the live-PG/MariaDB recalibration window, or leave it parked?
