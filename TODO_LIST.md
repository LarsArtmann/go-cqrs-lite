# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, 2026-09-08, 2026-09-11, 2026-09-16, 2026-09-19, and
2026-09-20 — the 8th pass struck 815+ verified-done items inline across 57 reports;
the 9th pass closed the 09-19/20 wave reports + executed plans and harvested the
release-train tails below).
The Declined section at the bottom is a do-not-re-litigate guard, not a backlog.

> **Prioritized execution plan (2026-09-08):**
> [`docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`](docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md)
> ranked the then-list into Pareto waves (W0 release train → W1 trust →
> W2 efficiency → W3 v5 train) and was EXECUTED through 2026-09-11 (W3's v5
> items live in the v5 section below; the user-gated P22 halves remain in the
> Turso section). **Current plan (2026-09-20 17:40):**
> [`docs/planning/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md`](docs/planning/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md)
> (M01–M27, all open rows mapped). This file remains the living source of truth.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## Metaengine Universal Storage Substrate (proposed 2026-09-18)

> Owner directive 2026-09-18: metaengine becomes the ONE way data is stored/retrieved from disk.
> Full Pareto plan (23 tasks / 82 micro-tasks): [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

- [x] ~~**T18b: load-sweep + benchmark baseline regen under Go 1.27** — `#load-sweep` on timing paths and a `benchmark-regression.sh --save` refresh (the committed baseline predates the 1.27 toolchain AND now needs the new claimkit entries)~~ **DONE 2026-09-20** — quiet window opened 16:58 UTC (load1/load5 3.73/4.88 < 5, `calibration-gate.sh` PASS); `#load-sweep` PASS in 91s (all 8 timing modules survived a 31-core soaker under go1.27.1); re-pin clean on attempt 2 (attempt 1 was noise-gate-rejected and auto-restored — the protection works), compare vs the 2026-09-11 baseline: **0 regressions, 15 improvements, 1 stable + 5 new entries** (claimkit ×4 + `BenchmarkBenchkitSuite_SQLite`); header carries re-pin timestamp + uptime sample + `# toolchain: go1.27.1` provenance line; daemon-committed (a91e7cd90). Caveat: the post-save verification re-run could not land a green gate tonight — 4 runs all noise-gate-rejected on chronic `write_p99_ns` CoV volatility, and the >25% flag landed on a different matview variant each run (sampling noise, never reproduced; see new row below). **UPDATE 2026-09-21:** headline demotion implemented (row below) and production-validated — first post-fix decision-grade run 08:46 UTC (noise gate PASSED at load 1.87/4.84, write_p99 advisory CoV 22.5%); it flagged `MIN/matview +33%` (11062 vs 8319) per the bounded-attempts ruling → baseline stands, bimodal lottery suspected (same variant 3+ times across runs, samples 7.9-15µs; the baseline caught the low mode). Cost pass (benchtime/count widening, chained quiet-window) decides the widening; until then MIN/matview flags are EXPECTED noise, not perf evidence.
- [ ] **T19–T21 (v5-gated): fold capabilities into universal `Engine`, delete the duplicate SQL stacks, release train** — blocked on the v5 train per ADR-0142 §decision; DO NOT execute in v4.x (growing core interfaces is breaking, contract 21g discipline). The tag waves for claiming + the queue family are the separately-tracked item below. _(Effort: L; v5-gated)_
- [ ] **Go 1.27 follow-ups** — the 94-module `go 1.27.1` sweep completed the jsonv2 graduation (2026-09-19, un-broke every workspace-mode compile); the coordinated tag sweep (flake.nix/scripts/CI/workflows/Go-strings/docs → plain `go build`) landed 2026-09-19. ~~Remaining: re-baseline load-sweep benchmarks under the 1.27 toolchain.~~ DONE 2026-09-20 via the T18b row above (`#load-sweep` PASS + re-pin under go1.27.1) — nothing remains in this row. _(Effort: S)_
- [x] ~~**benchkit noise gate: `write_p99_ns` chronically exceeds the 10% CoV threshold on this host (sqlite/dev profile)** — observed 2026-09-20 across 4 gate runs (CoV 11.5% → 20.6% → 22.5%) including a deep-quiet window (load 2.89/4.23) and at `--noise-repeat 7`; the other 3 headline metrics stay stable. An extreme-quantile estimator (~100-iter runs) has intrinsic cross-run variance, so a clean baseline save is a coin flip around the threshold.~~ **RESOLVED 2026-09-20 late (owner-delegated calls, implemented):** (1) `write_p99_ns` DEMOTED from the headline set (default now `write_throughput write_p50_ns load_p50_ns` — p99/max measure estimator variance, not loudness; the median compare routes contention through throughput/p50 first); (2) `benchmark-regression.sh --save` now REFUSES on a noise-gate failure unless `--force-save` (the D1 footgun: attempt 1's suspect save was only caught by an external guard); (3) re-pin acceptance = bounded attempts (verify up to 3×; all-noise-rejected → save stands with rejections recorded; decision-grade regression → flag for triage). Fixture-tested (p99-noisy→saves; p50-noisy→refuses; `--force-save`→saves; compare-only+save→saves, CI path intact). Open remainder: the matview bimodality (MIN 7.9-15µs samples) still makes the 25% median-of-5 compare flag random variants on clean runs — investigate/widen benchtime (see T18b row caveat). **UPDATE 2026-09-21:** first post-fix decision-grade run confirmed the fix (noise gate PASSED with p99 advisory at CoV 22.5%); matview widening = owner ruled MEASURE FIRST — cost pass chained (100x/5, 1000x/5, 100x/9, 1000x/9 wall-clock + MIN/matview medians; results `/tmp/matview-cost-results.txt`). **UPDATE 2026-09-21 midday (owner rulings):** (a) matview gate entry PRE-WIDENED to `100x::9` as the cheapest candidate; closure pipeline (detached, chained after the cost pass) autonomously validates the accepted rule — median-of-9 within ±10% across 3 consecutive quiet runs, suite ≤60s wall — then re-pins (noise-clean save) and verifies; on failure reverts the entry to defaults; (b) KNOWN-UNSTABLE annotation shipped in `benchmark-regression.sh` (MIN/matview suppressed as advisory `UNSTABLE-KNOWN` until **2026-10-21** expiry; a quiet-night green fixture-verified; real regressions still fail) so nightly runs stay green while the sampling fix lands; (c) NIGHTLY WIRED (owner ruled yes): `scripts/nightly-bench.sh` + systemd user units `scripts/nightly/` (install: `cp scripts/nightly/go-cqrs-nightly-bench.{service,timer} ~/.config/systemd/user/ && systemctl --user daemon-reload && systemctl --user enable --now go-cqrs-nightly-bench.timer`) + flake app `nightly-bench` — logs `/var/tmp/cqrs-nightly/<date>.log`; (d) shellcheck CLEAN on all three touched scripts; (e) host benchmark-ceiling policy remains OPEN (owner undecided). **UPDATE 2026-09-21 afternoon (incident + three more rulings):** the first cost-pass run FAILED silently — `GOTOOLCHAIN=local` (inherited shell env) vs go.work `>= 1.27.1` killed every bare `go test` at `wall=0s`, and the closure began confirming against `median=0` garbage before being killed pre-revert; both session scripts now `export GOTOOLCHAIN=auto` + carry an implausible-median guard (lesson: the env chain applies to session tooling too, and plausibility guards precede mutations). Re-armed chain (cost → closure) executes three further owner rulings: (f) **CI parity** — closure updates `.github/workflows/benchmarks.yml` matview leg to the chosen config; (g) **baseline history** — `--save` now auto-archives the superseded baseline to `docs/benchmarks/baselines/benchmark-baseline-<UTC>.txt` (fixture-verified); (h) **fallback if no config stabilizes MIN/matview**: matview suite leaves the gate set entirely and moves to a nightly NON-GATING stability suite (`scripts/nightly/matview-stability.sh`, wired into nightly-bench by the closure) — closure installs it automatically.
- [x] ~~Quiet-window tooling promotion (W2 row)~~ **DONE 2026-09-21 (owner ruled promote-now):** `scripts/quiet-window-run.sh` — waits load1/load5 < `--ceiling` (default 5), runs the wrapped command with `--attempts`/`--no-retry` bounded retries, `--deadline` (default 6h), `--log`; `--self-test` 6/6 green (quiet+pass, never-quiet deadline, retry-then-pass, always-fail exhausted, usage); wired as flake app `nix run .#quiet-window-run -- CMD` (mirror of load-sweep). Open halves: nightly-gates wiring **DONE 2026-09-21 midday (see row above: systemd units + `nightly-bench` app; user installs the timer)**; host benchmark-ceiling policy (owner UNDECIDED 2026-09-21 — ceilings stay strict <5 until ruled).

## Durable Work Queue module (proposed 2026-09-13)

> ~~T20 PapDashboard adoption evaluation~~ and ~~M4 polish tail~~ done
> 2026-09-20 — verdict + receipts in
> [`docs/reviews/2026-09-20_papdashboard-queue-adoption-evaluation.md`](docs/reviews/2026-09-20_papdashboard-queue-adoption-evaluation.md)
> (ADOPT for the notify pipeline, gated on the tag wave; no PapDashboard-side
> blocker) and the CHANGELOG 2026-09-20 entries (pool options, deadlock
> backoff+jitter, `testutil/mysqltestcontainer`, PG engine-test DB isolation,
> PG `-race -count=2` + MySQL `-race -count=2` legs green). Items the 09-19
> harvest listed but later sessions already landed: conformance/doc.go
> 3-engine list, README MySQL quickstart, `MYSQL_TEST_DSN` in the nix legs.

- [BLOCKED] **Owner ratification: queue dep-validation semantics (M4 §f1)** —
  decision memo with options + recommendation:
  [`docs/reviews/2026-09-20_queue-dep-validation-ratification-memo.md`](docs/reviews/2026-09-20_queue-dep-validation-ratification-memo.md).
  Reply A (ratify `ErrDanglingDep` at-enqueue validation; recommended) or B
  (restore donor-faithful blindness). Freezes with the queue-family tag wave. _(Effort: XS — owner reply)_

## Command-side domain depth (2026-09-13 plan)

> Prioritized execution plan:
> [`docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md`](docs/planning/archived/2026-09-13_11-45_SUPERB-command-side-depth.md)
> — Pareto waves (W1 decider causation → W2 first-class command records → W3 lifecycle
> upcasting + docs parity → W4 gates/filing), derived from the three 2026-09-13 reviews
> (`docs/reviews/2026-09-13_*`). Additive-only v4.x; `decider` gains ZERO new deps
> (local `CausedCommand` capability interface). Guardrails: decider.go is 377/350
> baselined (new file required), api golden regen in same edit, CHANGELOG symbols gated.
>
> **EXECUTED IN FULL — W1–W3 shipped 2026-09-13; W4 closed 2026-09-20 (composed
> `#verify` S03 GREEN, 16-39 report); the 92-tag train published
> `decider/v4.7.0` + `command/v4.11.0` + commandlifecycle (2026-09-19). Plan
> archived.** The section keeps only the demand-gated remainder:
- [BLOCKED] **ADR-0138: command sourcing draft (consumer demand)** — design doc only, builds on W2's bridge, reconciles ADR-0112's planned `CommandAwareFold`. _(Effort: M)_

## Go 1.27 upgrade wave (closed 2026-09-19)

> ~~Toolchain + go-directive wave to Go 1.27~~ DONE 2026-09-19 — shipped as the 92-tag
> "Toolchain cutover wave": go 1.27.1 directives repo-wide, flake `goToolchain` =
> `go_1_27`, jsonv2 graduation (tags stripped repo-wide), consumers served
> (`go list -m @latest` = 1.27-based tags on the proxy). The remaining bench re-pin
> under 1.27 is the T18b/T14 row in the Metaengine Universal Storage Substrate
> section above; owner decisions on the flake go-pin ride the W3 owner bundle below.

## Investigate: `TestSystem_ResetProjection_RestartAndReplay` contention stall (found 2026-09-13)

- [ ] **TestEngineHealth_CatchUpUnderConcurrentApplies load-sensitivity** — failed once under the full metaengine package suite ("primary ticks = 2001, want exactly 2000"); passes 5/5 isolated — genuine timing sensitivity to full-suite contention, no correctness signal observed since. Revisit only if it recurs. _(Effort: S — observe)_

## Turso materialized views (ADR-0135) — upstream handoffs

> Created 2026-09-07 (matview operator option shipped; three upstream
> turso-go defects verified and documented — see
> `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`
> and the archived report
> `docs/status/archived/2026-09-07_19-25_turso-materialized-views-operator-option.md`).

- [BLOCKED] 🔥 **File the standalone upstream issue for the silent wrong-results bugs (defects A+B)** — grouped views diverge from the second transaction on and collapse at ~27k rows; draft is ready and fully verified in `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (everything below its first `---`). Blocked on user approval (external action). The COMMIT-abort half (defect C) is already reported: PR #8257 comment https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646. _(Effort: XS once approved)_
- [ ] **Code guard follow-up: make grouped-spec safety mechanical** — today the danger is advisory-only (Doctor WARN + docs). Options: `MaterializedViewSpec` validation refusing `GroupBy` on turso-go ≤ v0.8.0-pre.10 (breaking for legitimate small deployments) vs a config flag (`AllowGroupedViews`) vs silent status. Decide + implement once the upstream timeline is known (still unknown: PR #8257 unanswered, defect re-verified live on pre.10 2026-09-11). The mechanical flip point now exists: `TestTursoMatView_GroupedSumDefectAEnvelopeGuard` + `TURSO_IVM_ENFORCE_FIX=1` asserts exactness at the 2k-row repro shape the day upstream fixes it. _(Effort: S)_
- [ ] **Matview v2 feature surface** — planned-table matviews (ordered with `ApplyLayout` + backfill), filtered-view spec variants, multi-aggregate/DISTINCT serving, `DropMaterializedView` off-boarding, per-view IVM write-amp otel counter, `system.Introspection()` surface, cqrs-lint rules (matview-on-unsupported-driver; matview-plus-planned-table staleness trap), `example/materialized-views/`. Route individually when a consumer asks. — source: archived 19-25 §f23-35, 05-33 §f29-35
      _(Effort: M/L each)_
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). DESIGN STEP DONE 2026-09-21 (SUPERB S28/F110):
      one-pager at [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md)
      — `AggregateOn(fn, column, group)` as a QueryOption stamped on `QueryDecl`,
      `MatViewSpecReporter` capability, scalar-covered-first scope. REMAINING:
      ratification + implementation — routing v1: scalar-covered shapes price O(1) (matview-served), grouped shapes stay O(N) with a Doctor note (upstream defect A makes grouped routing unsafe). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A — scope the first cut to scalar-covered shapes only. — source: archived 19-25 §f29, 05-33 §f32, SUPERB S28/05-51 §f16-17
      _(Effort: M)_
- ~~[ ] **Tag wave for the matview feature**~~ done 2026-09-19 — matview family published: `metaengine/v4.14.0` + `sqliteengine/v4.4.0` + `tursoengine/v4.2.0` + `system/v4.8.0` (tags carry zero local replaces; pins coherent per `pin-sweep --check --remote`).<br>**Original:** metaengine/sqliteengine/tursoengine/system carry sibling replaces for unpublished symbols (`MaterializedViewSpec` family); pins must be bumped and replaces stripped at the next release wave so consumers can use the feature from published tags. _(Effort: M — see AGENTS.md tag-wave procedure)_
- [ ] **Sharpen the defect-A characterization before filing upstream** — bisect the actual onset boundary (rows × groups × tx) for a principled property envelope and investigate the anomaly cluster (collapse at 26k vs draft's ~27k; wall onset through tursoengine observed at 24k-25k — the "deterministic at 27000" claim is scan-activity-sensitive, confirmed by the `-tags ivmrepro` suite logs 2026-09-11; post-abort views absorb the aborted tx's deltas). The scalar-at-scale exactness pin and the three-defect repro suite now exist (`metaengine/tursoengine/ivm_repro_test.go`); what remains is the principled onset-boundary characterization for the upstream issue. — source: 02-48 §d4/§f2/§f9/§f10
      _(Effort: M)_

---

## Cordis spatiotemporal-composability follow-ups (2026-09-10)

> Source: [`docs/planning/archived/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](docs/planning/archived/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md) (executed in full 2026-09-10; archived by the 2026-09-11 docs-health pass)
> (mapping report: `docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`).
> Operationalizes the Cordis learnings — revertible effects, reactive coeffects,
> observational equivalence — as correctness + trust wins **without breaking a
> v4 consumer**: every behavior change warns-first in v4.x, hard-errors at v5
> (rides the ADR-0123 wave).
>
> **ALL 27 tasks (M-01..M-27) DONE 2026-09-10** — shipped surface lives in
> CHANGELOG `[Unreleased]`: Reset warn-guard + projectionadapter `Resettable`
>
> - ADR-0136, the coeffect gate (`DomainConfig.Events` /
>   `ErrDanglingEventSubscription`), cqrs-lint E018, `ValidateCoeffects` +
>   `coeffects.md`, the arXiv grounding + figure recounts, ADR-0137 engine
>   deactivation (quarantine/reroute/reprobe + Doctor/Stats health), and the
>   equivalence tooling (`scenario.Interleaved` /
>   `AssertObservationalEquivalence` + rapid property). Vocabulary stays
>   internal-only until v5 (M-26 default held, verified leak-free). The
>   2026-09-11 follow-up wave closed the rest in-tree (CHANGELOG
>   `[Unreleased]`): `EngineResetter` on every engine (sqlite 🔥 + pg, mysql,
>   duckdb, pebble, bbolt, badger, dgraph, iroh; turso by delegation), reset
>   capability surfaced in Doctor/`GetEngineStats` (`CanReset`), C040 fold-case
>   coverage with E018 provider parity, goleak for `metaengine` +
>   `projectionhost`, the `[Unreleased]`-position tripwire in `verify-docs.sh`,
>   and fold-write failover with `CatchUpEngine` (ADR-0137 completion —
>   writes reroute like reads; reprobe rebuilds before reactivating).

- ~~[ ] **Release-train note**~~ done 2026-09-19 — the 92-tag wave stripped the
      `metaengine/projectionadapter`/`irohengine` sibling replaces and repinned
      every consumer (`pin-sweep --check --remote` green; tags verified
      replace-free — 10-25 §a2/§a3, now archived).

---

## cqrs-lint

> Point-in-time execution plan (T01–T24 / F001–F096) with per-row resolution
> markers: `docs/planning/archived/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`.
> T01–T12, T20–T24, F089, F090(a+b), F091 Tiers 1–3 (incl. P014 ApplyLayout)
> and the 2026-09-08 hardening batch are DONE (CHANGELOG `[Unreleased]`); this
> section carries the living remainder.

- [ ] **cqrs-lint audit follow-ups: loose heuristic gates (deliberately
      deferred 2026-09-11).** Documented, low-severity FP/FN vectors that
      each need their own false-positive analysis + golden churn; confidence
      levels already mitigate. Candidates: import-scope substring gates
      (V001 `/v3` `/v4`, V004/V005 `eventtest`, T001 `/decider`, T002/T005
      `/projection`, T003/T004 `catalog`/`snaps`, T007 `/event`, E016
      `Bundle`, A008 `/event/` exclusion); B018 `containsBus` lowercase-only
      and its "identical error-handling structure" claim; A015 name-collision
      write-matching at error severity; A016/A013 project-wide suppressions;
      A017 unqualified `NewRepository` matching + `NewTypedRepository`
      asymmetry; A019 vendor-path heuristic; F006 payload-class wiring under
      the strong/weak split; F009/F010 pattern tokens; V002/V003/V006
      root-go.mod-only scope; b022_b025.go (495) and
      a020_a021_a022_a023.go (~357) over the 350-line convention — bundle
      with the file-size-gate policy decision.
- [ ] **cqrs-upgrade strict-gate residual holes** — (a) run the
      deprecation scan even for NoPins modules (indirect-only cqrs consumers
      currently escape); (b) consider a `schemaVersion` field for the
      `--json` wire; (c) E2E test of `run()` against a fixture module
      (flags→report→strict exit codes). — source: 05-26 §e4/§f6-10
      (DONE 2026-09-13: --strict now fails on module errors — unscanned =
      unproven — and `bumps` is always-present in --json, symmetric with
      `deprecations`; both pinned by tests.)
      _(Effort: S)_
- [ ] **cqrs-lint FP-sweep harness refresh (2026-09-19 harvest)** — surface
      stderr from the sweep harness (5 empty repo rows were silent failures),
      re-run the corrected 12-repo baseline
      (`docs/status/2026-09-17_fp-sweep-baseline.md` is the known-bad snapshot),
      investigate the crush-daily 39-finding outlier. — source: archived 08-45
      §f11-13 _(Effort: M)_
- [ ] [BLOCKED] **Doctor-JSON pre-merge semantics ruling** — should
      `doctor --format json` report RAW config (today, golden-pinned) or
      EFFECTIVE post-`applyConfigOverrides` values (what the text path shows)?
      Consumer-scripting contract decision; implementable in minutes either
      way once ruled. — source: 05-31 §g2
- [ ] [BLOCKED] **Release-policy Q3: severity tightening in a minor.**
      S008/S009 now emit `error` (were `warning`); consumers using
      `--min-severity error` see new failures after ≥v4.9.0. Acceptable in a
      minor (documented in CHANGELOG), or gate behind a "Changed" section +
      dedicated minor? User decision. The S011 financial escalation is
      classified under CHANGELOG "Changed" (2026-09-08) and is governed by
      this ruling. Concretized by the envelope v2
      wire-format-in-minor question (08-26 §g3). — source: 02-40 §g3
- [ ] [BLOCKED] **Daemon Q2: `.golangci.yml` exclusion from the auto-commit
      formatter.** ROOT-CAUSED 2026-09-06: BuildFlow's built-in golangci
      defaults regenerate config at pre-commit; no user-facing knob found in
      `~/.config/buildflow`. `scripts/check-formatters.sh` self-heal repaired
      every occurrence (4+ incidents) and is the durable defense. REMAINING
      DECISION: accept self-heal permanently or fix upstream. User decision. —
      source: 02-40 §d1/§g2
- [ ] [BLOCKED] **F040 — required status checks / branch protection.** Master
      has no branch protection at all; enabling it would block direct pushes
      and the daemon workflow. Owner decision on protection + which checks +
      exceptions. — source: 06-58 §g1
- [ ] 🔥 **350-line policy: ratify the shipped ratchet, then split waves.**
      STATE 2026-09-11: the baseline+ratchet gate SHIPPED and is GREEN
      (`scripts/check-file-size.sh` + `scripts/file-size-baseline.txt`, 58
      historical offenders baselined; fails on NEW offenders and on
      baselined-file GROWTH, allows shrinking; mutation-proven ×2; wired
      into `nix run .#check-file-size` + the CI `file-size-gate` job).
      REMAINING: (a) owner ratifies the ratchet as POLICY vs full split
      waves vs harness exemptions (adttest/enginetest are exported test
      harnesses — 953/935 lines); (b) then the code-file split waves
      (typed_reader 1127, adttest/harness 953, metaengine/store 935,
      enginetest 935, execute 778, engines 725/722/663,
      architecture/helpers 627, suppression/parser 540, explain 516,
      b022_b025 495, a020 ~357 …). The gate stops being decorative either
      way. — source: 06-56 §a9/§d1, 05-51 §a (ratchet shipped)
      _(Effort: decision + L, multi-session)_

---

## Release / Tagging

> The full 39-tag v4 wave (B1–B7) was cut, pushed, and verify-ci-green on
> 2026-08-29. The SUPERB wave tagged `otel/v4.4.0` + `cmd/cqrs-upgrade/v4.0.0`
> (2026-09-07) and a coordinated release re-tagged 15 modules (2026-09-08).
> Zero local `=> ../` replaces remain EXCEPT `storage/go.mod` (`=> ../encryption`,
> `=> ../snapshot` — the documented unpublished-sibling pattern).

- ~~[ ] **Reconciliation-wave untagged surfaces (2026-09-13)**~~ done 2026-09-19 — shipped in the 92-tag train: `metaengine/v4.14.0` (`Store.StreamCollection`), `commandlifecycle/projections/v4.2.0` (`CommandsByActor` + query/result types), golden regenerated.<br>**Original:** `metaengine` (`Store.StreamCollection`), `commandlifecycle/projections` (`CommandsByActor` + query/result types), plus the regenerated API golden. — source: [`docs/status/archived/2026-09-13_18-35_…execution.md`](docs/status/archived/2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md) _(Effort: XS note; M at tag time)_

- ~~[ ] [BLOCKED] 🔥 **Next v4 tag wave**~~ done 2026-09-19 21:08–21:45 — the 92-tag train (B0–B6): every listed surface published, incl. `watermill/v4.6.1` (issue-#21 typed causation; the go-localsync workaround can die), `otel/v4.5.0` (DBSystem) cut BEFORE `storage/v4.10.0` (B2 before B3, ordering honored), `cmd/cqrs-lint/v4.12.0`, `cmd/api-stability/v4.4.0`, engine re-calibrations, `scheduling/sqlstore/v4.1.0`, `system/v4.8.0` (matview + coeffect gate). Tags verified replace-free with coherent pins.<br>**Original:** substantial unpublished surfaces on master: `encryption` (key helpers + envelope v2), `snapshot` (`NewRewritingTransformedStore` + wire tags), `storage` (`MigrateSnapshotColumnsToStream`, EventSchema re-exports, bytea fix), `cmd/cqrs-lint` (working `--fix`, C005, RULES.md, doctor JSON, scorecard panel, preset policy), `cmd/api-stability` (sub-package golden), `catalog`, `benchkit` (system harness), `metaengine` (planner capability partition, record context, `SortPaginate[T]`, planned-table parity, matview family) + engines (irohengine v4.2.0 for the pin repair, mysqlengine, pgengine, sqliteengine, duckdbengine, dgraphengine recalibration, badgerengine — now consumes new `metaengine.SortPaginate`, pin bump + replace-strip REQUIRED), `scheduling/sqlstore` (MySQL claiming), `watermill` (**v4.7.0 — the issue-#21 typed-causation wire protocol: `writeCausation`/`parseCausation` + legacy custom-mirror promotion, landed 2026-09-09 and STILL UNTAGGED**; go-localsync runs its documented workaround until this tag exists), `system` (v4.7.0: materialized views + the MV recipe's UNRELEASED marker flips when tagged). **Strip `storage/go.mod`'s two local replaces in the same wave.** **NEW 2026-09-16 preconditions:** (1) re-tag `otel/v4` carrying `DBSystem` BEFORE any storage/v4 tag — the published storage would otherwise reference an unpublished symbol (broken for consumers until otel re-tags); (2) the wave also strips the newer sibling replaces: `cmd/cqrs-bench => ../../benchkit` (statistical-rigor APIs: `RunRepeated`, `MetricVariation`, `WriteBenchstatRepeated`), `queue/sqlite` + `queue/postgres => ../queue`, `commandlifecycle/projections`, and bumps `metaengine` (`Store.StreamCollection`) + `commandlifecycle/projections` (`CommandsByActor`). Order constraints per CONTRIBUTING pre-tag checklist; cut→push→next interleave (GOPRIVATE resolves siblings via VCS). — source: 08-26 §c3, 15-09 §f47, SUPERB §f14-16; 08-04 §b5 + 02-09 §f3 + 09-35 §f18 (2026-09-16 additions) _(Effort: M)_
- ~~[ ] **Tag `cmd/cqrs-lint` v4.10.2 (ships the buildinfo version reporting)**~~
  done 2026-09-19 — superseded by `cmd/cqrs-lint/v4.12.0` in the 92-tag
  train (buildinfo version reporting rides it; smoke probe covers the
  installed binary).<br>**Original:** on master since 2026-09-11;
  v4.10.1 deliberately predates it. Verify the
  installed binary prints the real tag after `go install …@v4.10.2`. —
  source: 01-47 §b1/§f5 _(Effort: S)_
- ~~[ ] **`check-retracts-shipped.sh`**~~ done — script exists, wired, green
  2026-09-19 (5 modules with retracts checked); the clean-dir `go list -m
      module@latest` acceptance ran green for 10 key modules post-train.<br>**Original:**
  fail when a master go.mod retract
  directive is absent from the module's newest tag (the inert-retract
  class: `retract v4.8.0` sat on master ~10 days before v4.10.1 shipped
  it). Acceptance test for every retract = clean-dir `go list -m
      module@latest`. — source: 01-47 §d3/§e1/§f6
  _(Effort: S)_
- ~~[ ] **`tag-release.sh --audit --baseline` mode**~~ done — mode +
  `#check-tag-audit` CI leg exist; 2026-09-19 post-train audit: 24 known
  violations, 0 NEW, 0 fixed (1175 tags checked).<br>**Original:** the one-shot audit found
  24 historical violations (1078 tags), all in dead paths that cannot be
  fixed; a known-violations baseline (art-dupl pattern) turns `--audit
      --check` into a CI leg gating NEW violations only. — source: 01-47
  §b3/§f7
  _(Effort: S/M)_
- ~~[ ] **`scripts/smoke-probes.txt` + strengthen test-tag-release.sh Test 5**~~
  done — smoke-probes.txt exists (12 lines, explicit per-CLI probes) and
  Test 5's lib suite covers the no-main skip path ("library module takes
  the no-main skip path" ✓).<br>**Original:**
  per-binary probe command for the `--smoke` run check (`--help` exit
  semantics differ across CLIs); Test 5 covers the `--smoke` usage guard,
  not the no-main-package skip path. — source: 01-47 §b4/§b5/§f11/§f12
  _(Effort: S)_
- [ ] [BLOCKED] **Dead-path module/tag decisions (owner)** — (a)
      ~~example/taskmanager + example/getting-started carry suffix-less module
      paths with permanently-invisible v3/v4 tags: re-path to /v4, delete, or
      document as v0-only~~ done 2026-09-19 — decided by the wave: correct
      v0-line tags cut (`taskmanager/v0.2.0`, `getting-started/v0.2.0`,
      `scheduler-otel-status/v0.1.0`, `goal-shaped-app/v0.1.0`); the dead
      v3/v4 tags stay baselined in `audit-tag-baseline.txt`. (b)
      `event/v4/eventtest`'s invisible v0.x tags:
      document as dead in modules.md + pin-sweep note. — source: 01-47 §c1/§f9/§f10
      _(Effort: M decision + S doc)_
- ~~[ ] **Create GitHub Releases** for the outstanding tags~~ done 2026-09-19
  — 92 releases created for the train (152 repo total);
  `create-github-releases.sh` extended to match train-section headers
  (bounded-token match, newest section first, trimmed body + CHANGELOG
  pointer like the 09-08 precedent) with `--dry-run`; all 92 extracted,
  bogus tags skip. `gh` auth working. — source: 05-00 §f12
  _(Effort: S)_
- ~~[ ] **Consolidate indirect dep references**~~ done 2026-09-19 — moot:
  ADR-0128 extracted codec/retry/idempotency/flightrecorder to external
  repos and the 92-tag train repinned everything; zero
  `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}` references
  remain in any go.mod (verified by grep across all 95).<br>**Original:**
  the transitive
  `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}/v4` indirect deps
  in ~49 consumer go.mod files clean up after new tags publish. Track and
  verify. _(Effort: M)_
- ~~[ ] **Run `scripts/pin-sweep.sh --check` as a standing post-release step**~~
  done 2026-09-19 — the train's cut loop ran per-batch pin-sweeps (commits
  2a9ccb75a…38c3b4fd4) and the post-wave `--check --remote` is green
  (local + origin tag sources); `storage/eventstore` is a package inside
  `storage/v4`, so its pin health is the storage pin coherence the sweep
  already covers. — source: archived 07-48 §b2/§f2
  _(Effort: S)_
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms (worst-of-30 sample inflates under gate load). Shipped + gated
      green; keep or revisit. _(Effort: XS)_

---

## Metaengine — follow-ups

- ~~[ ] 🔥 **Planned-table filter pushdown silently returns empty for camelCase fields over snake_case json keys**~~ done 2026-09-20 — fixed in `metaengine.ExtractFields`: columns match a field by json tag OR Go field name (case-insensitive); map path falls back to acronym-aware snake_case of the column (`ParentID`→`parent_id`; the naive splitter's `parent_i_d` caught by the new tests). Pinned by 3 parity tests over the CRM shape; metaengine/adttest/sqliteengine suites green. CHANGELOG [Unreleased] Fixed carries the entry; the CRM Go-side workaround (`tasksForParentID`) dies on the next consumer bump.<br>**Original:** — found 2026-09-18 in the Ledger CRM: a projection registered with `metaengine.FilterOnField[R]("ParentID", FilterEq)` gets a planned layout whose columns are named by the GO field names (`ParentID`, `DueAt`), but the planned-table writer populates those columns by extracting the stored JSON document with the same camelCase key — while the documents use the view's JSON TAGS (`parent_id`, `due_at`). Result: `ParentID`/`DueAt` columns stay NULL and every pushdown filter (`system.Where("ParentID", …)` → `WithFilter` → planned `WHERE "ParentID" = ?`) matches nothing, silently, no error. Repro: register a projection with a Filterable camelCase field, write a record, then `Find[R](Where("Field", v))` → 0 rows while unfiltered Find returns the row. Fix direction: planned-column extraction must use the same field-to-json-key mapping the encoder uses (or jsonPath must fall back field-name AND snake_case); add a parity test that every Filterable field round-trips through pushdown. CRM works around it today by filtering in Go (`tasksForParentID` in `internal/httpapi/helpers.go`) — delete that workaround once fixed. — source: 2026-09-18 CRM pareto-execution session, M9 (deal detail tasks panel); verified against live sqlite db: `meta_planned_task_views` row had `ParentID=NULL` while `value.parent_id` was set _(Effort: M — canonical key mapping, extraction fix, pushdown round-trip test)_

- ~~[ ] 🔥 **Port ctx-scoped transactions from sqliteengine to pg/mysql/duckdb engines (same engine-global `activeTx` leak)**~~ done 2026-09-20 — all three ported: txMarker ctx + `conn(ctx)` + nested-tx rejection + ambient fast paths keyed off the marker; `activeTx` deleted everywhere. Isolation tests per engine (`TxIsolationFromForeignContext`): pg validated live (testcontainer), mysql live-gated on `MYSQL_TEST_DSN` + build/vet green, duckdb full suite green 74s (stress companion deliberately not ported — go-duckdb C-level prepare convoy on file DSNs, a driver limit). Workspace build + system consumer tests green. CRM pin bump note below still applies.<br>**Original:** — found 2026-09-18 while fixing a production flake in the Ledger CRM (`sql: Rows are closed` on timeline loads): sqliteengine's `xc()/xd()` resolved the "active tx" from an engine-global `atomic.Pointer`, so any engine call from an unrelated goroutine landing inside another caller's `RunInTx` window was handed the foreign `*sql.Tx` — dirty reads of uncommitted writes, and readers dying with `sql: Rows are closed` when the foreign tx committed mid-iteration. FIXED in sqliteengine (commit `22ab7b218`): the tx now travels in the ctx under `txMarker{}` (value = `*txExecutor`), `xc(ctx)/xd(ctx)/txExec(ctx)` resolve affinity from ctx only, `activeTx` field deleted; regression-pinned by `metaengine/sqliteengine/tx_isolation_test.go` (`TestSQLiteEngine_TxIsolationFromForeignContext` — deterministic dirty-read guard, red before/green after — plus `TestSQLiteEngine_ConcurrentStreamReadVsAppendExpected`). pgengine (`engine.go:70`), mysqlengine (`engine.go:52`), and duckdbengine (`engine.go:55`) still use the identical engine-global pattern and have the same hazard; port the marker-carrying ctx + accessor refactor and the isolation test to each. NOTE for the CRM: its flake input pins go-cqrs-lite at `a0b5429a…` — the pin must be bumped to include `22ab7b218` (post-push) before the next `nix build` release binary. — source: 2026-09-18 CRM pareto-execution session, baseline-verified via worktree at `22ab7b218~1` _(Effort: M per engine — mechanical port + test)_
- [ ] **Calibration provenance protocol + quiet-window re-runs** — protocol HALF DONE 2026-09-11 (later session), re-runs remain gated on a quiet window: (a) DONE — `scripts/calibration-gate.sh` asserts 1-min load < 5 (overridable `--max-load`/`CALIB_MAX_LOAD`; CI exempt) and aborts loudly — verified against a live compile storm (load 207 → hard abort); `calibration-drift.sh` runs it before benching; (b) DONE — protocol items 6-8 in `docs/benchmarks/calibration-2026-08-30.md` define the per-entry PROVENANCE line (store path + binary version output + uptime samples) and ban secondhand version citations; the 2026-09-11 SearchQuery entry now carries an explicit provenance-gap note; (c) MECHANISM DONE, RUN PARTIAL — `benchmark-regression.sh --save` writes a titled provenance header (fixture-tested, parser-safe); the titled re-pin of `benchmarks/benchmark-baseline.txt` **DID run 2026-09-20 17:12 UTC** (T18b row above: noise-clean save, go1.27.1 provenance, claimkit/SQLite entries, 0 regressions vs the 2026-09-11 baseline); the quiet-window count=5 SearchQuery re-run remains pending (a 493-load storm held the 2026-09-11 session; gate correctly refuses); (d) PENDING — re-anchor ALL dgraph constants in one gate-passing window. Run when `scripts/calibration-gate.sh` passes: SearchQuery count=5 (supersede today's table if medians move >5%), then the benchmark-baseline re-pin, then the dgraph constant campaign. — source: 03-50 §b2/§b3/§f7/§f8/§f15/§f16, 02-48 §d3/§f8
      _(Effort: M)_

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

- [ ] **metaengine live-latency doc sync vs code** (harvested 2026-09-20) —
      recipes §2.11 (`ProbeEngine`/`LatencyTracker`/`Calibration` surface) has
      never been drift-checked against the shipped code; verify every named
      symbol/flow, fix the doc where the API moved. — source: archived 16-43 §f16 _(Effort: S)_
- [ ] [BLOCKED] **Turso strict-vs-lenient DSN param policy** — the driver
      silently ignores mistyped encryption params (`encryption_hexkkey=` opens
      the DB UNENCRYPTED). Strict posture (reject unknown `*encrypt*`/`*key*`
      params at construction) vs lenient (document + fix upstream). Lean:
      strict ("make impossible states unrepresentable"), but it changes
      behavior for existing DSNs — owner call. — source: 20-57 §g3
- [ ] [BLOCKED] **Turso sync/embedded-replica first-class support decision** —
      the ONLY Go path to Cloud BYOK (the `database/sql` driver has no remote
      client). Real consumer need or out of scope? Gates an L-effort design.
      — source: 20-18 §g1, 20-57 §f11-12
- [ ] [BLOCKED] **Upstream turso-go issues (verify-before-filing first)** —
      (a) missing `DriverContext`/`OpenConnector` (struct-level config without
      DSN stringification); (b) pure-remote connections cannot present a BYOK
      key; (c) mistyped DSN params silently ignored → silently-unencrypted
      DBs. Verify each against latest main, then file. — source: 20-18 §c4,
      20-57 §f13-16
- [ ] [BLOCKED] **dgraph one-RPC scope (Q1)** — ADTMap flipped to O1
      (2026-09-07); Set/Multimap/Log/StreamLog still OLogN with a comment
      promising per-ADT reassessment. Authorize the one-wave flip or confirm
      incremental. Needs per-ADT reassessment benches either way. — source:
      archived 22-33 §g1, 04-35 §f9/§f15
- [ ] [BLOCKED] **CapabilityGaps reach into Doctor (Q2)** — documented gaps
      silence PLAN diagnostics today; should they also silence Doctor's
      `--- Capability ---` violation lines (`CapabilityAudit` receives nil
      gaps)? — source: archived 22-33 §g2, 04-35 §f17
- [ ] **Conformance-sweep + hot-path tail — live-server runs remain** — (a) dedup
      no-op case shipped (`TestApplyIdempotent_DuplicateIsNoOp`); (b) micro-bench
      DONE 2026-09-16 (struct hot path 1.9 ns / 0 allocs through the shared
      funnel; `metaengine/encoded_bench_test.go` +
      `docs/benchmarks/2026-09-15_applyfold-raw-payload-funnel.md`); (c) PG half
      DONE 2026-09-16: `PG_MODULES="scheduling/sqlstore storage" nix run
      .#integration-pg` full suite PASS incl. `TestClaimingPostgres_MetricsSnapshot`
      (+ TwoClaimersNoDoubleFire, RenewLease, RenewVsClaimRace) on the repo's own
      ephemeral PG; the `#integration-mysql-nspawn` half remains (quiet-window).
      — source: 05-38 §b2; execution status 2026-09-16 08-04 report §a6
      _(Effort: M)_
- [ ] **scheduling/sqlstore hardening tail — `RenewLease` ownership/claim
      tokens remain** (race-stress test, counter-scope pin, counter property
      test, `FuzzDecodeDueTimer`, and the `example/scheduler-otel-status`
      worked `Metrics()` + OTel example all shipped 2026-09-13..16);
      `RenewLease` ownership/claim tokens are design-gated (code comment
      defers today).
      — source: 03-50 §f18-27; execution status 2026-09-16 08-04 report §a3-a4
      _(Effort: M, one rule per slice)_

---

## CI / Infrastructure

- ~~[ ] **Quiet-window verify tooling (asked by ≥3 sessions)**~~ done 2026-09-20 —
      `scripts/wait-for-quiet.sh` (1-min AND 5-min ceilings, self-tested),
      `scripts/can-run-composed-gate.sh` (no-release-procs + tree-stability +
      load assert), and the `#verify` `-p` parallelism cap (`VERIFY_TEST_P`, set
      to 4) all shipped in T26 (09-40 §a4, now archived); composed-`#verify`
      went GREEN the same day (S03). Remaining launcher ergonomics live in the
      release-train tail row below. — source: archived 06-47 §f6-8, 12-02 §f9/11/17, 18-11 §f11
- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s; broken since ~2026-07-17. Local `nix run .#verify` remains the
      authoritative gate. _(Effort: S, user action)_
- [ ] **cqrs-lint Self-Lint credentials** — BLOCK LIKELY STALE (re-verified
      2026-09-17): go-finding resolves via the public module proxy under
      GOWORK=off (verified again: proxy serves v1.11.0, published and fetched
      by cqrs-lint's go.mod bump today — no replace directive, no SSH/git
      remote needed; the old `git ls-remote` exit 128 hit the SSH path). The
      remaining blocker is purely the Actions billing entry above. Re-run the
      self-lint CI leg once billing works; close if green. _(Effort: S,
      re-run required, gated on billing)_
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_

- [ ] **Pre-commit hook still owns three TREE-WIDE gates** (workspace
      `go build ./...`, api-surface freshness, fmt.Printf grep) that can
      block an honest commit on a CONCURRENT session's in-flight files —
      the benign version observed 2026-09-18 (go.mod/go.work 1.27 stamp
      war), the hostile version is a sibling's mid-edit file. All three
      verified green 2026-09-18 ~18:50 (build OK, 7209 exports, zero
      Printf hits). Design staged-scoped or per-module variants where
      cheap; the workspace-build gate is the one worth keeping tree-wide
      (it is the daemon-commit safety net). — source: 2026-09-18 18-12
      report §e/9 _(Effort: M)_

- [ ] **asyncapi-react bundle requires `unsafe-eval` — interactive AsyncAPI
      UI is DEAD under strict CSP** (found 2026-09-18 via the repaired
      `#check-csp` browser gate): the vendored bundle evaluates strings at
      runtime and throws `Uncaught EvalError` under the eval-free `script-src`
      policy, so `EnableCSP: true` leaves `/docs/asyncapi` non-interactive
      (raw JSON endpoint + noscript fallback still serve). The gate currently
      classifies the eval refusal as a known degradation
      (`csp_browser_test.go`, cross-referenced). Real fix is an owner
      decision: upgrade/replace the bundle for an eval-free build, or add a
      PAGE-SCOPED CSP relaxation for `/docs/asyncapi` only (never a global
      `'unsafe-eval'`). Same session also fixed the REAL root causes the gate
      had never seen: nav scripts now nonce-gated
      (`docsNavProps` threads the nonce into `ThemeToggle` + `SimpleNav`
      `BaseProps`; previously blocked silently). _(Effort: M for page-scoped,
      S+upstream for bundle swap)_

- [ ] 🔥 **CI triage: master red across ~15+ jobs, no green run in the last
      30.** Classified 2026-09-11 (run 34548534824), RE-CLASSIFIED
      2026-09-13 (run 34747274058, full log triage): (a) FIXED same-day —
      the Module-matrix go.sum class (missing `/go.mod` hashes after the
      v4.5/v4.6 pin wave). (b) **ROOT-CAUSED 2026-09-13 — one dominant
      infra cause:** the deprecated `magic-nix-cache-action` was THROTTLED
      by the GitHub Actions Cache API ("ResourceExhausted: rate limit
      exceeded" / "GitHub Actions Cache throttled Magic Nix Cache"), its
      local substituter (127.0.0.1:37515) then returned HTTP 418, nix
      disabled the substituter mid-job, and every nix-based job starved on
      closure downloads: verify-fast, Dgraph Integration, CGo Build,
      Security Scan (gosec via nix-shell), Minimum Coverage. NOT FlakeHub
      auth — the Twirp rate-limit is the GH cache backend. **Fix needs an
      owner/infra decision**: migrate to a maintained cache backend
      (`DeterminateSystems/flakehub-cache-action` — needs the parked
      FlakeHub account decision) or drop the action and accept cold builds
      (timeout-minutes must rise). (c) FIXED LOCALLY 2026-09-13, next run
      should clear — File Size Check (store.go 953→940: EventInput moved
      out), Shell Format Drift + Nix Flake Check formatting leg
      (nix fmt clean), api-stability (golden updated), cmd/cqrs-lint module
      (taskmanager golden re-pinned — rule-output drift) + verify-fast's
      TestTagContentMatchesChangelog (green against current CHANGELOG).
      (d) FIXED LOCALLY 2026-09-16: the go.work sync check job now uses plain
      `actions/setup-go@v5` with `go-version-file: go.mod` (it previously had
      NO Go toolchain at all, and rode the throttled nix cache); the
      benchmarks.yml matview-gate is restructured into per-backend subshells
      with root-relative `tee current.txt` (the old single-`cd` form hopped
      `../metaengine/tursoengine` from `stack/bench` — a nonexistent dir — and
      teed into `stack/current.txt` while the compare step reads the root
      copy; BOTH legs verified live with real bench runs). **CACHE MIGRATION
      EXECUTED LOCALLY 2026-09-18:** the owner/infra decision resolved to
      option (b) — `magic-nix-cache-action` REMOVED from all 24 ci.yml jobs
      (flakehub-cache-action was rejected: it needs the parked FlakeHub
      account + auth token) with the policy documented in a ci.yml header
      (start with nightly-gates.yml when the FlakeHub decision lands), and
      timeout-minutes raised on the starved jobs (verify-fast 45, per-module
      matrix 25, CGo 25, Dgraph 30). Remote confirmation still gated on the
      billing fix. The test-tag-release.sh
      SC2086 item was already stale — the script uses the array form
      `git "${notag[@]}"` and shellcheck is clean (verified 2026-09-13). — source: run
      34548534824, run 34747274058, `gh run list`
      _(Effort: M-L, multi-session; the cache-backend migration is the
      single highest-leverage repair)_
- [ ] [BLOCKED] **Set the `ERRAUDIT_PAT` secret** (user action) — erraudit
      findings verified zero across all modules 2026-09-15; the `error-audit`
      CI job arms the moment the secret exists. — source: 08-05 §f12/§f19
      _(Effort: S, user)_
- [ ] **Green MySQL-VM shuffled suite in the quiet window** — two attempts
      died ~15s in with transport-level resets on DIFFERENT modules/seeds
      (zero assertion failures; the documented semi-dead-VM/host-contention
      class). Replay both logged seeds (`build/shuffle-seeds.log`) when the
      box is quiet; closes the [x] rollout item's caveat fully.
      — source: 08-05 §b1/§f5 _(Effort: M)_
- [ ] **VM-leg hardening (vehicle ruled 2026-09-20, W3 bundle)** — owner ruled
      the QEMU/VM leg stays the canonical local MySQL vehicle (native-MariaDB
      productization CLOSED-WONTFIX: do NOT build `scripts/ephemeral-mysql.sh`
      or an `#integration-mysql-ephemeral` app). Remaining: `vm-mysql.sh`
      pre-flight stale-port check (orphaned QEMUs holding port 33070) +
      process-group cleanup trap (from archived 23-21 §f11), plus the
      `testutil/mysqltestcontainer` leg (landed 2026-09-20, commit
      4b8ae5eab + follow-ups) serving CI/local Docker runs alongside the VM.
      — source: archived 23-21 §a8/§b1/§f3-4/§f11; W3 vehicle ruling 2026-09-20 _(Effort: M)_

---

## Code Quality

- [ ] **>350-line production files (~54, 2026-09-06 count)** — see the
      cqrs-lint section for the verified picture, gate-policy options, and the
      already-split offenders; the code-file split waves are a standalone
      multi-session program pending the policy decision. Decide
      harness-dir exemptions (adttest/enginetest are exported test harnesses)
      first. _(Effort: XL, multi-session)_
- [ ] [BLOCKED] **macOS verification of ephemeral PG** —
      `scripts/ephemeral-pg.sh` claims cross-platform but was only
      static-review-tested; a GitHub Actions macOS runner leg is the
      verification route. _(Effort: M)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) —
      userspace MariaDB coverage exists but not the full nspawn env. Now also
      covers the MySQL claiming integration tests. _(Effort: M)_
- [ ] **Shuffle eval + adoption for `scripts/test-integration.sh` /
      `test-all-backends.sh`** — the two composite runners execute the same
      suites UNshuffled (documented parity gap in gotchas-testing.md).
      Gated on ROADMAP OQ #9 (are the composite scripts staying?). — source:
      02-16 §b3/§f4/§f5
      _(Effort: S)_
- [ ] **Backport contention-retry review to turso/badger engines** —
      dgraph got the execution-layer retry; check whether turso/badger have
      an analogous transient-abort class worth the same treatment. — source:
      02-16 §f19
      _(Effort: M)_
- [ ] **Unify ephemeral-script passthrough conventions** — ephemeral-pg.sh
      uses positional EXTRA_ARGS, ephemeral-dgraph.sh uses
      TEST_ARGS/TEST_ARGS2, redis/nats use raw passthrough; three
      conventions for the same job complicate evaluations. — source: 02-16
      §e5/§f25
      _(Effort: M)_
- [ ] **Watch dgraph + redis CI jobs (~10 shuffled runs)** — record any
      seed that fails; rare orderings WILL eventually appear in CI (that is
      the point of shuffling). — source: 02-16 §e7/§f10
      _(Effort: XS)_
- [x] ~~[BLOCKED] **Quiet-window exclusive `nix run .#verify` composed GREEN**
      (supersedes the contention-fix verify item) — last composed GREEN was
      2026-09-09; three days of waves (Cordis, publish/reset/v5-train, both
      parallel sessions) are unverified as one chain, the dispatch-core
      fold-reroute refactor has never seen `-race`, and `scripts/verify-docs.sh`
      has never run end-to-end with its new tripwire. When the box is quiet:
      run `#verify`, then `-race` over `metaengine`, then `verify-docs.sh`;
      record date + commit + durations in TODO_LIST/plan (S03 acceptance).~~
      **DONE 2026-09-20 (S03):** composed GREEN 14:53:28–15:04:25 (~11 min,
      head `61e1a2080`-era tree) — all 19 phases: verify-docs tripwires,
      module coverage, build, vet, test, race (incl. metaengine + system),
      lint, arch, modsums, lint-config, docserver-css, duplication (0 new,
      baseline 60), turso, templ, bench-gate, coverage, api-stability,
      error-taxonomy, doc-check. Log: `/tmp/verify-attempt10.log`. Getting
      there surfaced and fixed three latent W2-wave tails along the way
      (lint godoclint debt; tx-isolation/graph-SQL clone groups; templ
      codegen drift) plus a real `GracefulClose` pre-cancelled-context
      nondeterminism — see `docs/status/2026-09-20_13-21_*.md` + CHANGELOG
      [Unreleased].
      — source: 02-16 §c4/§f13, 05-40 §f2/§f3/§f8, SUPERB S03
      _(Effort: M)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1–7 done. Pre-cut deprecation markers shipped 2026-08-17; migration
> guide at `docs/V5-MIGRATION-GUIDE.md`. Snapshot wire tags (T18) DONE
> 2026-09-06 — dual-read fallbacks in snapshot/pebble, SQL columns migrated by
> `MigrateSnapshotColumnsToStream` (auto-run by every InitSchema). Error-family
> codes renamed to stream vocabulary 2026-09-08 (17 codes, 9 modules; E4 of the
> extended review thereby RESOLVED).

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it. _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)** —
      multi-collection batch atomicity + auto-projection replaces them. Also
      removes the remaining `aggregate_*` SQL surfaces wholesale. _(Effort: M)_
- [ ] **Delete `graph.GraphProjection`** — auto-projection + graphadapter
      replaces it. _(Effort: S)_
- [ ] **Delete `stack.Bundle` + all 8 stack presets** — `system.System` is the
      only composition root; `stack/` module deleted entirely (incl.
      `stack/bench` and `stack.RunProjections` → `projectionhost.Host`). _(Effort: M)_
- [ ] **Delete deprecated compat shells from ADR-0126** — `schema.VersionedStore`
      + `NewVersionedStore`, `signing.Rejecting*` forwarders,
      `encryption.ErrInnerStoreNot*` aliases, `metadata.CustomData`. _(Effort: S)_
- [ ] **Delete `storage/sql.BuildWhereClause`** — `BuildWhereClauseChecked` is
      the validated replacement. _(Effort: XS)_
- [ ] **Breaking `record.NewStreamRef` validation** —
      `NewStreamRef(streamType, entityID string) (StreamRef, error)` rejecting
      an empty entityID; migrate call sites. Owner-confirmed 2026-08-22
      (decision memo:
      `docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md`
      Appendix B). _(Effort: M)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) — final
      v4.x tags exist; drop from go.work/flake testModules/api-stability list,
      then delete at the cut. Confirm they die BEFORE anyone renames their
      proto fields (sweep §4). _(Effort: M)_
- [ ] **Delete deprecated tombstone metadata API (ADR-0114 completion)** —
      remove `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/
      `TombstoneStatus`/`Metadata.Tombstone`; pre-reqs: type-driven status in
      `listing` (replaces the DetectTombstone call at listing/in_memory.go:155),
      migrate `example/taskmanager` off `OnTombstone`, regen golden. _(Effort: M)_
- [ ] **Rest of sweep §4 (wire-vocabulary renames):** watermill metadata keys
      `aggregate_id`/`aggregate_type` → `stream_*` with dual-read; events +
      commands table column renames + migrations (decide 5.0.0 vs later 5.x —
      08-41 §g1); benchkit `aggregates` output-key rename + re-golden;
      bbolt `command_serialization` CBOR tags + golden. Extend the sweep
      census with the newly-found pebble `slog` attribute keys
      (`aggregate_type`/`aggregate_id` in helpers.go + snapshot.go) and grep
      sibling consumer projects for old code strings in alert/dashboard
      configs at the cut. Consider a central wire-key table doc
      (JSON/CBOR/SQL × backend × fallback status) rewriting sweep §4 as a
      table. — source: 08-41 §b1/§f1–11, archived 07-48 §f5-6/§f7-13
      _(Effort: M)_
- [ ] **v5 ADR: encryption-at-rest configuration** — SKELETON SHIPPED
      2026-09-13 as [ADR-0139](docs/adr/0139-v5-encryption-at-rest-configuration.md):
      `DriverConfig.Encryption` + `KeyProvider func(ctx) ([]byte, error)` (vs raw
      `key []byte` — the KeyProvider lean enables rotation/hot-reload and keeps keys out of
      config structs; sets THE precedent for pg/mysql passwords too) +
      `system/` DeploymentConfig key-reference slot (env/file/secret-manager
      ref, never the key). Engines fail construction loudly when unable to
      honor (precedent: `RejectDurabilityTier`, `MaterializedViews`).
      REMAINING for the ADR: owner ruling on the 4 open questions
      (provider call semantics, reference validation timing, read-model
      scope, plaintext→encrypted migration), then implementation.
      — source: 20-18 §f15-17/§f21, 20-57 §f9-10
      _(Effort: L)_
- [ ] **Migration-verification tail for T18:** live MySQL/MariaDB +
      DuckDB `MigrateSnapshotColumnsToStream` runs; mixed-state corruption
      test; mid-migration failure-path test; concurrent-init idempotency test;
      property test for arbitrary legacy JSON subsets. — source: 08-41 §f13–23
      _(Effort: M)_
- [ ] **v5 items from extended review** — E1 (event-envelope Encoding →
      `record.Encoding`), E7 (watermill/middleware RetryConfig collision),
      E8 (typed Message Kind enum), E11 (AdapterCore.Encode error return),
      E13 (SQLTimerStore phantom param), E15 (middleware signature
      unification). _(Effort: M)_
- [ ] **More extended-review follow-ups** — E3 (bbolt command/query bare
      `fmt.Errorf` → pebble error-family pattern), E6 (`middleware.Option` vs
      `BundleOption` merge/bridge), E9 (turso Policy nil-write panics), E10
      (ShutdownDependency name validation), E14 (eventstore ownership
      asymmetry). (E4 — sentinel name↔code mismatch — RESOLVED by the
      2026-09-08 stream-code rename.) — source: `docs/reviews/2026-08-22_extended-data-model-review.md`
      _(Effort: M)_
- [ ] **Post-landing sweep for the data-model series** — api-stability
      meta-tests, doc-check over skill refs, consumer-pin sweep for `record/v4`
      consumers under GOWORK=off (MarshalBinary lesson). _(Effort: M)_
- [ ] **Expand V5-MIGRATION-GUIDE** with before/after examples per v1 tier
      (incl. `relational → metaengine`) at the cut; add the envelope v2
      consumer note ("old readers compatible; no action needed") and
      operator verification snippets for the snapshots migration; sweep the
      asrecord/MIGRATION_TO_STACK/PRESETS guides once v5 nears. — source:
      08-26 §c6, 08-41 §f25–27
      _(Effort: M)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate. _(Effort: M)_

---

## Core Data Model v4.x/v5 (2026-08-22 review + plan)

> Source: [core data-model review](docs/reviews/2026-08-22_core-data-model-review.html)
>
> - [execution plan](docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md).
>   Owner decision 2026-08-22 (Appendix B): string `record.StreamRef` SURVIVES
>   v5 with a validating constructor; the struct `record.Stream` proposal is
>   rejected.

- [x] ~~**T23 — upstream skill-maintenance pass** (the plan's one open task):
      docs/reviews↔brainstorming divergence; read-prior-reports +
      copy-template steps in the review skills.~~ **DONE 2026-09-21** —
      executed in `~/projects/SKILLS` (canonical skill repo). The 2026-08-22
      fix (`95ec7fc`) had covered only the data-model-review skill; this pass
      added the two steps ("read prior reports in the series" + "copy the
      template, never transcribe") to the other five HTML-report skills
      (architecture-review, brutal-self-review, code-quality-scan,
      full-code-review, naming-review) and to the canonical html-report-kit
      guide (new "Series Discipline" section carrying both observed failure
      modes from the 2026-08-22 session), then re-vendored into all 10 kit
      consumers (`sync-html-kit.sh --check` green; `check-skills.sh` 30/30,
      links clean). Divergence re-verified dead: no skill body names
      `docs/brainstorming` as an output target.

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [ ] **README review deep-read tail (2026-09-13 cluster, 8th-pass harvest)** —
      the 12-16 report verified all 93 READMEs mechanically but ~30 polish items
      stayed unharvested; the substantive ones: ~~(a) doc-check repoRoot regression
      test + `[Unreleased]` Fixed entry + gotcha note for the relative-path fix~~ done 2026-09-20 (`reporoot_test.go` + CHANGELOG Fixed + gotcha);
      (b) add READMEs to the doc-check gate (flake app/CI); ~~(c) deep-read the six
      big unread READMEs (catalog 587L, graph, stack, storage/view, watermill,
      otel, prometheus)~~ done 2026-09-20 (T37: 8 READMEs symbol-verified
      against source, zero drift); (d) quick-start drift-guard tests for stack/sqlite,
      storage/memory, decider, scheduling, projectionhost; ~~(e) deprecated-symbol
      grep gate over READMEs~~ done 2026-09-20 (`scripts/check-readme-deprecated.sh`:
      package-scoped + framing-aware, 7-leg mutation-tested self-test, nightly
      gate, ADR-0123 banner sweep + citation migration, baseline at 0); ~~(f)
      `scripts/check-readme-links.sh` link checker~~ done 2026-09-20 (673 targets,
      0 broken, nightly gate).
      — source: archived 12-16 §f (150-154, 158-161, 174, 179) _(Effort: M total,
      sliceable)_
- [ ] **Docs censuses (7th-pass items 1-3)** — ~~module-map census~~ DONE
      2026-09-20 (all 95 go.mods rowed, scripted-diff verified — census banner
      in the map header). REMAINING: FEATURES maturity matrix vs the 95 modules
      (+ last-verified stamps on guarantee rows); per-file index for the
      archived waves in `docs/status/README.md`. — source: archived 13-03 §f1-3 _(Effort: S/M)_

---

## Event-Query-Model reconciliation follow-ups (2026-09-13)

> The 2026-07-23 design doc was reconciled against source (status banner + per-section addendum
>
> - coverage map); `StreamingScan` was wired (`Store.StreamCollection`) and the per-actor
>   lifecycle projection shipped the same day (see CHANGELOG). Carry-forward items below. Source:
>   [`plan`](docs/planning/archived/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md).

- [BLOCKED] **Session-log boundary decision** — memo recommends sessions stay external
  (`cqrs-htmx/identity-model`) and NOT fold into the planned `queue/` module; revisit only on
  a concrete audit consumer. — source: [`T18 memo`](docs/planning/archived/2026-09-13_T18-memo-session-log-boundary.md) _(Effort: XS decision)_

---

## benchkit statistical-rigor tail (2026-09-16)

> The 02-09 session shipped P100 exactness, `RunRepeated`/`RepeatedResult`,
> per-metric `MetricVariation`, real benchstat samples, and load provenance —
> all verified green. The 09-35 session closed the two blockers (queue clones,
> soak gocyclo) and refreshed the FEATURES coverage line (151+43). What follows
> is the consolidated remainder. — source:
> [`docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md`](docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md)
> §b/§f, 09-35 §f P3

- [ ] **Benchkit CLI polish tail (2026-09-19 harvest)** — unharvested §f items
      from the statistical-rigor completion: render `Min` in output tables;
      `--strict` failing on NOISY headline metrics; `list-phases` metric
      mapping; `Load1` in env row; `--warmup` docs (README gap); CSV variation
      columns; sweep CoV column; per-repeat progress; reservoir size per-phase;
      `tail_ratio` true-max semantics; `RunSuite` testing.B variant over
      RunRepeated; metric-name constants for downstream tooling; stale-baseline
      re-pin protocol + gate-set rename guard; CI golangci version skew; P100
      in benchstat gate metrics; recipes statistical-rigor block + FAQ P100
      entry + readmodels/core cross-links; supersede-note on the oversubscribed
      2026-09-19 capture once a quiet-window one exists. — source: archived 15-37
      §f8-30, archived 02-09 §f14-42 _(Effort: M total, sliceable)_

---

## CV consumer-verdict follow-ups (2026-09-16)

> CV (real consumer, evented-funnel-core Phase 0 GO) ran a parity-gated
> four-tier read-model benchmark against metaengine/storage/sqlstore and
> Phase-0 seam spike over a real 6.2k-event store. All their API-fit claims
> were re-verified against source: [`docs/reviews/2026-09-16_cv-verdicts-reflection.md`](docs/reviews/2026-09-16_cv-verdicts-reflection.md).
> The `Scan` 100-row-default doc lie is FIXED in the same change (doc comment +
> FAQ entry); `ApplyBatch` atomicity stays tracked under v5 Unification
> (ADR-0123 §10) — CV's 3.4 s → 109 ms pragma measurement is the perf
> argument for it.
>
> **Execution sequencing for this section + the metaengine/system reliability
> items lives in the Pareto plan
> [`docs/planning/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md`](docs/planning/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md)**
> (P0: tag wave + replay-starvation fix + ApplyBatch atomicity; P1: lease +
> FilterContains + Forever + E9/E10 + matview guard; P2: v5 deletions + E-items +
> AggregateOn seam; P3: proof + docs + v5.0.0 cut).

- ~~[ ] **Decide a first-class single-writer/lease story for engines**~~ — DESIGN
      DELIVERED 2026-09-21:
      [`docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md`](docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md)
      (verified current reality: lease semantics live only in `queue/` + `claiming/`
      task claims; recommendation = `EngineConfig.SingleWriter` advisory
      `<dsn>.cqrs-lease` flock, fail-loud default-off; becomes ADR-0146 on
      ratification). REMAINS OPEN: owner ratification + implementation before v5
      freezes engine construction surfaces.<br>**Original:** CV's Phase-0 ADR
      conditions every library-store cutover on a CV-owned `metaengine.RegisterDriver`
      decorator wrapping their `<dsn>.lease` single-writer marker, because the library
      has NO engine/store-level lock. — source: reflection doc §4.2
- [ ] **`FilterContains`/`FilterPrefix` FilterOp extension** — metaengine
      FilterOp today is exactly eq/ne/lt/le/gt/ge/in (`enum_validation.go:71`);
      substring search degrades to a client-side full scan (CV measured
      2.3–28 ms vs 326 µs hand-rolled LIKE). Native engines map to
      LIKE/prefix; closure fallback evaluates in Go. Fits the v5 FilterOp
      window. — source: reflection doc §4.5 _(Effort: M)_
- [ ] **go-idempotency `Forever` → adapter mapping (gated on upstream v0.4.0)** —
      when go-idempotency ships the `Forever` sentinel, `idempotency/sqlstore`
      + `idempotency/kvstore` must write `expires_at = math.MaxInt64` DIRECTLY,
      never via `expiryFromTTL` (verified by execution: `now.Add(ttl).UnixNano()`
      wraps negative past year 2262 → key dead on arrival — the unrecoverable
      direction). Dedup the verbatim-copied `expiryFromTTL` (kvstore:46,
      sqlstore:173) while touching both; add the overflow boundary test; pin
      bump middleware/sqlstore/kvstore (all v0.3.0 today) via the
      go-ecosystem-upgrade skill. — source: reflection doc §3.1/§4.7 _(Effort: M once upstream lands)_
- [ ] **Tag `system` so the coeffect gate reaches consumers** —
      `DomainConfig.Events` + `ErrDanglingEventSubscription` sit unreleased on
      master while the newest real consumer (CV) enforces its event universe
      CV-side at system v4.7.0 (= latest tag). A release lets consumers
      delete their bespoke gates. — source: reflection doc §4.3; rides the
      existing "Next v4 tag wave" row (P0 in the SUPERB plan) _(Effort: S — routine tag-wave mechanics)_
- ~~[ ] **Docs-truth tail (2026-09-16 plan-surfaced)**~~ done 2026-09-21 —
      readmodels.md Scan-limit note, modules.md Scan-default mention, and the
      CHANGELOG `[Unreleased]` entry shipped 2026-09-17; ALL doc-check
      ambiguous-alias advisories resolved (it was 5 by then, not 3 — the
      alias set grew with queue/*: every affected fence now imports the exact
      package, doc-check zero warnings, recipes harness green); overflow
      probe source embedded in the review doc §3.1b on 2026-09-18.
      <br>**Original:** status report 2026-09-16 21-02 §f-2/12/13; SUPERB plan T27/M104
- [ ] **benchkit cross-tier PARITY gate** — before any tier-vs-tier benchmark
      number is trusted, assert cross-tier result identity (full snapshot,
      stat counts, ranked IDs) — the template CV's four-tier benchmark
      proved out. — source: reflection doc §5; SUPERB plan T27/M102 _(Effort: M)_
- [ ] **Tuned-tier metaengine benchmark** — `BuildLayoutPlanFromType`
      composite layouts vs hand-indexed SQL, BOTH sides tuned, so future
      latency claims carry no defaults-vs-tuned asymmetry (CV's fairness
      finding). — source: reflection doc §5; SUPERB plan T27/M101; Goal-closure plan G-T16 extends this to the fold-tier goal-parity shape _(Effort: M)_

---

## metaengine Goal-closure follow-ups (2026-09-17)

> Source plan: [`docs/planning/2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md`](docs/planning/2026-09-17_05-49_SUPERB-metaengine-goal-closure-pareto-plan.md)
> (distance analysis: ~55–60% consumer-experienced). Executes AFTER/ALONGSIDE
> the 2026-09-16 excellence plan (shared items cross-referenced there, never
> duplicated). Success definition in plan §7.

- [ ] 🔥 **DIRECTION RULING: what does the Goal's "declare ONLY" mean after the
      `Infer` deprecation?** — evidence pack + 3-option decision memo
      (reframe: Evolutions+Queries on the `system` surface IS the goal ·
      revive: Layer-1 inference via compile-time codegen, visible+auditable ·
      hybrid), owner ruling, ADR-0141, AGENTS.md Goal sentence amended to the
      ruled meaning. The Goal is undefinable at 100% until this is ruled.
      — G-T01/G-T02/G-T03 _(Effort: S memo + XS ruling; M if revive)_
- [ ] **Auto-projection completion** — Evolution-fold-inheritance coverage
      audit (which declaration shapes inherit vs hand-wire; close the top
      gaps), tombstone auto-fold (ADR-0114 × ADR-0116: type-driven `Remove`
      in auto-projection), planned-table auto-backfill option (kills the
      registration-before-data developer worry; idempotent,
      KeyScanBackend-gated, Doctor-visible). — G-T09..G-T12 _(Effort: M)_
- [ ] **Capability smoothing: ADTSet on pg/mysql** (meta_set DDL +
      SetContains parity) or, if the effort probe says no, plan-time
      Doctor-loud degraded warnings on every affected query — "operators
      pick any engine" must never break a declared query silently. — G-T13
      _(Effort: S/M by probe)_
- [ ] **Scan default v5 decision** — SURVEY DELIVERED 2026-09-21:
      [`docs/planning/2026-09-21_scan-default-v5-survey.md`](docs/planning/2026-09-21_scan-default-v5-survey.md)
      (consumer census incl. `system.Find` inheriting the cap + recommendation:
      flip to unbounded at the v5 cut + cqrs-lint nudge + optional operator
      ceiling). Documented-100 is the status quo, loud in godoc+FAQ. REMAINING:
      owner decision, implement at the v5 branch. — G-T14 _(Effort: S decision + S impl)_
- [ ] **FEATURES maturity flip for the closed surface (🧪→✅)** — earned by
      the plan's gates (not declared): evidence links per row, CHANGELOG
      Goal-story entry, release notes. Final stamp of Goal closure. — G-T25
      _(Effort: S, gated on gates A–D)_
- [ ] **goal-shaped-app polish tail (2026-09-19 harvest)** — extend the AGENTS
      "Add a New Module" procedure with the examplePaths/testModules split;
      compile-gate the example README's Go fences (docs_compile_test); a real
      postgres e2e leg for the config-swap test; README ns-figures
      machine-specific caveat; cqrs-lint consumer probe + AsyncAPI export demo
      on the example; `DomainConfig.Events` coeffect demo once `system` tags.
      — source: archived 18-16 §f1-12 _(Effort: S/M)_

---

## Vector-search verification tail (2026-09-15)

> Vector ADT shipped on EVERY engine 2026-09-15 (brute-force + DuckDB/libSQL
> pushdown; see FEATURES Metaengine section). Report:
> [`docs/status/archived/2026-09-15_18-32_vector-search-every-engine.md`](docs/status/archived/2026-09-15_18-32_vector-search-every-engine.md)

---

## Watermill sibling skill follow-through (2026-09-15)

> `.agents/skills/watermill/` shipped (broker powers/tradeoffs/limits). Report:
> [`docs/status/archived/2026-09-15_19-45_watermill-skill-session.md`](docs/status/archived/2026-09-15_19-45_watermill-skill-session.md)

- [ ] **NATS JetStream roundtrip test leg** — `watermill-nats/v2` +
      `scripts/ephemeral-nats.sh`, mirroring `TestRedisStreamRoundtrip`; if it
      lands, an `.#integration-nats`-style flake app + CI leg analog to
      `#integration-redis`. — source: 19-45 §c1/§f2/§f10 _(Effort: M)_
- [ ] **Skill-quality tail** — run the 3 drafted trigger-eval prompts
      (with/without skill); add `references/advanced.md` (Delayed Messages,
      Requeuing After Error, FanIn/FanOut, Metrics, Troubleshooting);
      cross-link go-cqrs-lite `SKILL.md`/`advanced.md` watermill sections →
      the sibling skill (discovery is currently AGENTS.md-only); verify
      upstream latests for the redisstream/kafka/amqp/sql plugins. Codify the
      claims-checklist rule (verify inline factual assertions, not just
      file:line cites) in docs/agents. — source: 19-45 §f1/§f3-5/§f8; 19-57 §f3/§f8
      _(Effort: M total)_

---

## md-go-validator CI integration (from 2026-09-13 audit)

> 167 md-go-validator errors remain (9 consumer-facing, 55 active docs, 108
> archived); P1 (6 fence fixes) is done, P2–P4 are not. Report:
> [`docs/status/archived/2026-09-13_12-21_md-go-validator-audit-and-p1-fence-fixes.md`](docs/status/archived/2026-09-13_12-21_md-go-validator-audit-and-p1-fence-fixes.md)

- [ ] **Make the validator a real gate** — commit `--init` config + baseline
      file (`scripts/md-go-baseline.txt`, mirror `check-file-size`); flake app
      `check-md-go`; verify/package the tool for CI (today it only exists as a
      host-level NixOS package); then P2 (`// skip-validate` the 9
      consumer-facing blocks, 7 files) and P3 (~55 active-doc blocks); decide
      the P4 archived-errors policy (baseline-forever vs shrinking ratchet).
      — source: 12-21 §b2/§c1-6/§f1-4/§f6-8 _(Effort: M)_

---

## Temporal versioned cells — ADR-0141 follow-ups (harvested 2026-09-18)

> From the temporal deep-dive reports
> ([14:07](docs/status/archived/2026-09-18_14-07_temporal-versioned-cells-deep-dive.md) §f items 19–44,
> [16:03](docs/status/archived/2026-09-18_16-03_temporal-versioned-cells-completion-gates.md)).
> Items 1–18 of the 14:07 list + docs/CHANGELOG/FEATURES/golden/lint work are DONE
> (see those reports); the core API (`VersionedStorage`, `MapSetAt`/`MapGetAsOf`,
> `temporal-asof` rule, memory/sqlite/bigtable engines) is green and documented
> (recipes §2.37, advanced §6.20, readmodels versioned-engine note). Engine
> enumeration in tooling (api-stability, cqrs-lint `StoreBigTable` +
> `metaengineEngineFromImport`) landed 2026-09-18 evening session.

- [ ] 🔥 **Real-GCP validation + prior calibration for `bigtableengine`** — the
      module ships 🧪 (bttest-fake-validated only, no credentials on this
      machine). Run the suite once against a real BigTable instance, then
      calibrate `NsPerOp`/RTT priors (currently UNCALIBRATED-marked constants)
      via `CALIB_DUMP=1` + `scripts/calibration-drift.sh`, then optionally add
      `BenchmarkCalibration_Bigtable_*` stubs. Owner decision pending: is a
      real-GCP smoke test tag-blocking for the next release? — source: 14:07 §f20/§g2,
      `metaengine/bigtableengine/README.md` _(Effort: S each, gated on GCP access)_
- [ ] **Property-based temporal tests for memory version chains** — rapid-based:
      out-of-order stamps, same-ms LWW collapse, retention-never-prunes-newest,
      tombstone-as-of visibility (engine-level, memory first). — source: 14:07 §f22,
      `metaengine/version_chain.go` _(Effort: M)_
- [ ] **sqlite versioned-cells restart soak** — prove `meta_cell_versions`
      history survives process restart (re-open DSN, as-of reads still answer).
      — source: 14:07 §f23, `metaengine/sqliteengine/` _(Effort: S)_
- [ ] **bigtableengine restart-safety test** — two engines over one bttest
      server; verify no cross-instance state bleed and clean re-reads. — source:
      14:07 §f24 _(Effort: S)_
- [ ] **Decide + document `MapUpdateAt` on bigtableengine** — optimistic
      ReadRow+Apply is non-atomic; fold-lock serialization may make it
      skippable. Decide, then document either the implementation or the
      exclusion rationale in the README. — source: 14:07 §f25 _(Effort: S)_
- [ ] **bigtableengine client-side MaxAge retention trim** — via the
      `DeleteTimestampRange` option (GC policy covers MaxVersions natively), or
      document GC-policy-only as the deliberate choice. — source: 14:07 §f26 _(Effort: S)_
- [ ] **Pebble/bbolt versioned cells — scope decision for the next wave** —
      both have natural prefix-range machinery for version chains; decide
      whether they join the 3 versioned engines. — source: 14:07 §f31 _(Effort: M once scoped)_
- [ ] **memory_versioned wall-clock path naming clarity** — `MapSet` on a
      versioned engine still stamps wall-clock (documented); consider a clearer
      name for the `recordVersionAt(now)` path so replay-vs-live writes are
      obvious at the call site. — source: 14:07 §f40, `metaengine/memory_versioned.go:31` _(Effort: XS)_
- [ ] **Cross-engine fuzz: `MapSetAt`/`MapGetAsOf` memory vs sqlite** — reuse
      the existing `fuzz_test.go` pattern to pin contract equivalence across
      the two emulating engines. — source: 14:07 §f41 _(Effort: M)_
- [ ] **Temporal capability gap notes in Dgraph/PG/MySQL engine READMEs** —
      one paragraph each: versioned cells not supported yet, temporal reads
      fail loud + `temporal-asof` WARNs (link ADR-0141). — source: 14:07 §f42 _(Effort: XS)_
- [ ] **projectionadapter-level temporal stamp test** — Store-level integration
      exists; pin the CQRS path (event stamps survive `ApplyRecord` → folds on
      versioned engines) at the adapter level. — source: 14:07 §f44,
      `metaengine/projectionadapter/` _(Effort: S)_
- [ ] **Soak env-var run for bigtableengine** — per
      `docs/agents/gotchas-testing.md` soak conventions (`-race` covered by
      `#verify`). — source: 14:07 §f33-34 _(Effort: S)_

## Dogfooding self-review follow-ups (2026-09-19)

> The dogfooding sessions adopted `dedup.Ring` in loopback and swept 54
> `defer func(){ _ = x.Close() }()` sites onto `metaengine.DeferClose` across
> engines + queue. What follows is the unharvested tail. — source:
> [`docs/status/archived/2026-09-19_16-50_dogfooding-self-review-execution.md`](docs/status/archived/2026-09-19_16-50_dogfooding-self-review-execution.md) §f +
> [`…17-25_dogfooding-self-review-final-status.md`](docs/status/archived/2026-09-19_17-25_dogfooding-self-review-final-status.md) §f

- [x] **Tier-0 close-helper decision (owner/ADR)** — RULED 2026-09-20:
      [ADR-0144](docs/adr/0144-deferclose-lives-in-tier0-record.md) —
      `record.DeferClose` is the canonical Tier-0 address; `metaengine.DeferClose`
      stays as a self-contained twin. Sweep executed the same day (kv, storage,
      storage/pebble, storage/turso/indexing, scheduling/sqlstore, projectionhost,
      stack, benchkit, examples; queue/postgres audited clean via bare defers;
      cmd/cqrs-lint → bare defer, at dep budget).
- [x] **Extract scan/paginate helpers** — done 2026-09-20: bbolt's
      `sortAndPaginateKV` now delegates to `metaengine.SortPaginate` (pebble/badger
      precedent); the mysql/sqlite `scanScoredVector` byte-twins + duckdb JSON
      variant delegate to new `metaengine.ScanScoredVector`/`RowScanner`. The
      `scanJSONValues` twins (mysql/duckdb) stay intentional (`art-dupl:accept`;
      sqlite's passthrough error contract would change under unification).
- [x] **Retry-idiom reconciliation audit** — done 2026-09-20:
      [ADR-0145](docs/adr/0145-retry-idioms-are-per-concern.md) — four sites are
      four DIFFERENT concern classes (transport op retry / crash-loop damping /
      shadow freshness budget / DB contention retry); documented per-class rules
      instead of forcing one idiom. `go-retry` stays middleware-only.
- [x] **quic/loopback const split brain + dedup parity test** — done 2026-09-20:
      one shared `irohengine.DefaultDedupCapacity` (quic re-exports; loopback
      references), parity pinned by `quic/dedup_parity_test.go`
      (`TestDedupParity_SharedCapacityConst` + `TestRing_ProductionCapacity10K`)
      mirroring loopback's `dedup_internal_test.go` contract.
      _(Effort: S)_

## go-graph-rag feedback follow-ups (2026-09-15, triaged 2026-09-19)

> Consumer evaluation of `metaengine/v4.13` + `system/v4.7` (adopted neither —
> "system is a category error for a library, metaengine wrong-shaped for
> GraphRAG"). Routed requests live in their home sections (#2 Turso §, #4 v5 §,
> #6 metaengine plans, #1/#7 ROADMAP); the unrouted ones:
> [`docs/feedback/reviewed/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md`](docs/feedback/reviewed/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md)

- [ ] **Fail closed on `EventAdapter.Save` racy fallback** — third-party engine
      authors get silent partial writes today; make the fallback loud or refuse.
      — feedback #3 _(Effort: S)_
- ~~[ ] **Stamp experimental status in each engine/module `doc.go`**~~ done 2026-09-20 — all 17 metaengine-family modules carry a doc.go package comment with the `# Experimental` doc heading (the section pkg.go.dev renders; readers can now tell 🧪 from ✅ in the godocs — FEATURES knows the rest); stray package comments on cost.go/engine.go consolidated. T22 in the 09-40 report (now archived). — feedback #5 _(Effort: S, mechanical)_

- [ ] **goal-shaped-app consumer-value tail (2026-09-20 harvest)** — adopt
      `DomainConfig.Events` + `.On` chaining (unblocked NOW by system v4.8.0;
      bump the example pin first); a real postgres e2e leg already listed above;
      an examples CI test leg (all six examples carry tests but CI is
      build-only); scenario-based Given/When/Then test for the task flow;
      snapshot story demo ("snapshots are a worry the library manages"). —
      source: archived 10-25 §f21-26 _(Effort: M total, sliceable)_

---

## 92-tag release-train tail (2026-09-20 harvest)

> Harvested from the train execution report's §f (`docs/status/archived/2026-09-20_10-25_tag-wave-execution-92-tags-ci-templ-triage.md`)
> and the verify-arc reports; items since done are struck inline there. The
> owner questions ride the W3 bundle section below.

- [x] ~~🔥 **`.golangci.yml` hash-golden drift guard (M16)** — the curated config
      was mangled 10+ times (BuildFlow auto-configure + the 2026-09-20 10:31
      concurrent-corruption incident deleted the depguard allow-list
      mid-verify); `#check-lint-config` only helps when someone RUNS it. Wire a
      content hash-golden into `#verify`/pre-commit so the mangle fails loudly
      at the next gate touch.~~ **DONE 2026-09-20 (M02)** —
      `scripts/check-golangci-hash.sh` (sha256 golden
      `scripts/golangci-config-hash.golden.txt`, `--update` re-pin gated on the
      shape gates passing so corruption can never be pinned) wired into
      `#check-lint-config` (→ `#verify`), the pre-commit `.golangci.yml`
      trigger, and `#check-release-scripts` (4-leg self-test + mutation-proof).
      Caught live during rollout: incident #11 (auto-commit `96dc20986` 17:00 —
      go downgrade + jsonv2 tag + depguard deletion + gci re-enable + rationale
      stripping in one 111-file wave) repaired from the last-good commit
      (`9212c8408`) and the golden pinned over the HEALTHY config.
      Gotcha: docs/agents/gotchas-tooling-build.md "Incidents 8-11".
      _(Effort: S)_
- [x] ~~**Verify-launcher ergonomics** — (a) `can-run-composed-gate.sh --wait-loop`
      (the gate pair is two one-shot scripts; a load rebound between them
      aborts — the 16:39 session's retry-loop supervisor is the proven
      composition); (b) a pre-flight wrapper that runs the cheap composed-gate
      phases (templ, bench-gate, coverage, api-stability, duplication — each
      <5 min) before burning a 12–46 min verify attempt (attempts 7–9 of the
      S03 arc were each a skipped-pre-flight cost); (c) an in-`#verify` load
      threshold guard (refuse >N loadavg with a retry message, M15).~~
      **DONE 2026-09-20 (M03–M05):** (a) `--wait-loop` with `--max-wait`/
      `--retry-interval`, rebound-recovery + timeout self-test legs;
      (b) `scripts/preflight-composed.sh` (6 phases, stop-at-first-red with
      remedies, `PREFLIGHT_ONLY/SKIP` env hooks, fixture self-test) — caught a
      REAL templ FileName drift on its first live run (cwd-corrupted codegen
      committed; regenerated); (c) `scripts/verify-load-guard.sh` in the
      `#verify` head: refuse + retry recipe + `VERIFY_FORCE=1` override.
      Launch recipe: `bash scripts/preflight-composed.sh && nix run
      .#can-run-composed-gate -- --wait-loop && nix run .#verify`.
      — source: archived 16-39 §e/16-43 §e/f11+f14, 23-21 §f14 _(Effort: S/M)_
- [ ] **CI tail from the train (six legs, all root-caused)** — (a) benchkit
      load-gate fixture env propagation (actionlint+shellcheck leg red on GH
      runners: fixture expects the planted-loadavg PASS message, runner output
      shape differs; 12/12 pass locally); (b) coverage-gate pinned/setup Go
      toolchain (three consecutive deaths on `go: downloading go1.27.1` — the
      job has no setup-go step); (c) auto-retry-once for cancelled/failed infra
      legs (CGo/coverage transient class); (d) triage the 04:12 nightly-gates
      failure; (e) Module Isolation Build leg-set instability; (f) one clean
      post-fetch-depth-fix run confirming `TestTagContentMatchesChangelog`.
      Remote confirmation of the whole set is billing-gated (row above). —
      source: archived 10-25 §b1/§f5-10 _(Effort: M total)_
- [x] ~~**Wire the check-readme-* gates into a push leg** (harvested 2026-09-20) —
      both README gates currently run nightly-only, so a broken link or
      deprecated citation can land unnoticed for up to a day; the
      self-test-then-gate pattern in nightly-gates.yml is ready to copy.~~
      **DONE 2026-09-20 (M09):** both gates (self-test-then-gate) added to the
      `lint-scripts` job in ci.yml — every push, not nightly-only; gates green,
      actionlint clean. — source: archived 16-43 §f18 _(Effort: XS)_
- [x] ~~**gotchas-testing.md: record the GracefulClose select-race pattern**
      (harvested 2026-09-20) — a pre-cancelled context racing an instant Close
      loses ~50% to Go's uniform select pick under parallel load; the fix
      shipped 2026-09-20 (done-win re-checks `ctx.Err()`), and the reusable
      lesson is the testing pattern, not the patch.~~ **DONE 2026-09-20:**
      recorded in gotchas-testing.md (select-uniform-pick lesson + the
      done-win-re-checks-ctx.Err() shape, `system/system.go:322`). The
      leaked-QEMU-33070/slirp-RST diagnosis was already recorded there
      (2026-09-19 entries). — source: archived 16-39 §a2/16-43 §f20 _(Effort: XS)_
- [ ] [BLOCKED] **Upstream filings (owner approval; verify-before-filing
      first)** — (a) turso-go native-lib hash-mismatch + lazy-init failure
      family (`TestBackend_LazyInit_Concurrent` / `TestVectorSearch_LibSQLPushdown`
      red-ing 1–3 legs intermittently; bump-or-file decision rides the owner);
      (b) exhaustruct_v5 `skippedNamed` panic (repro ready); (c) go/types +
      x/tools parallel-check race. Draft in Lars's voice via github-voice after
      verification. — source: archived 10-25 §f3-4/§f20, 15-37 M14 _(Effort: S each + approval)_
- [ ] **Release-tooling polish tail (sliceable)** — smoke-all: per-module
      timing, `--resume` checkpoint, artifacts under `~/.cache/` never /tmp;
      `check-templ` leg-first error summary; `batch-release.sh --from-manifest`;
      `pin-sweep --dry-run` lists its standalone-verifies;
      `TestTagContentMatchesChangelog` train-section threshold (>=10 tags) as
      an error; document `SOAK_SKIP_BOLT=1` for the per-module loop;
      cqrs-upgrade dogfood sentinel on goal-shaped-app; api-stability's four
      exclusion maps unified into one table; the 6-place module registration
      consolidated (go.work, flake testModules/examplePaths, layer script, api
      exclusions, cqrs-lint catalog) or a `new-module` scaffold;
      verification-ladder doc (meta-tests → touched-lint → verify-fast →
      verify) in AGENTS testing gotchas; LSP/gopls `GOTOOLCHAIN=auto` config
      (kills ~95 noise diagnostics per session); templ-generate canonical-cwd
      contract in the docserver README/gate echo; scheduler-otel-status row in
      core.md §9; `check-example-standalone.sh` (example require audit — would
      have caught the B6f forward-pin abort). — source: archived 10-25 §f13-14/§f31-38/§f41-50 _(Effort: M total)_
- [ ] [BLOCKED] **Daemon pre-commit sanity gate** (three sessions asked) — the
      auto-commit daemon absorbs red/corrupt mid-edit states (6 authored
      commits + the 10:31 corruption landed as `chore:` blobs); a sanity check
      before heuristic commits would end the class. Infra/owner call. —
      source: archived 10-25 §f17/§d6, 09-40 §e1 _(Effort: M, owner-gated)_
- [ ] [BLOCKED] **BuildFlow templ-generate cwd fix** (external repo) — the
      pre-commit templ step runs from the repo root and re-corrupted a correct
      regeneration ( FileName paths baked in); file upstream, until then the
      documented regenerate-from-`catalog/docserver/` + `--no-verify` pattern
      stands. — source: archived 10-25 §d3/§f16 _(Effort: S, external)_

---

## Owner decisions — W3 bundle (2026-09-20)

> **RESOLVED 2026-09-20 (evening)** — all six rulings received and applied.
> Consolidation sheet:
> [`docs/status/2026-09-20_11-36_owner-bundle-w3.md`](docs/status/2026-09-20_11-36_owner-bundle-w3.md).

- [x] ~~**Q1: `cmd/api-stability/readme_claims_test.go` ownership**~~
      **RESOLVED 2026-09-20:** owner delegated ("do what makes the most
      sense") → kept as-is under api-stability; ownership note added to the
      module-map row (load-bearing README truth, green).
- [x] ~~**Q3: ratify the 10:31 repair ruling**~~ **RATIFIED 2026-09-20** —
      the go 1.27.1 contract restore + the formatter-consistent markdown
      reformats both stand; row closed.
- [x] ~~**Q4: flake.nix go pin vs nixpkgs lag**~~ **RESOLVED 2026-09-20
      (yes)** — env chain documented as the contract
      ([gowork-modes.md](docs/agents/gowork-modes.md)) and now mechanically
      enforced: `scripts/check-go-version.sh` (flake app `#check-go-version`,
      stub-fixture self-test) wired into the `#verify` head + the nightly
      (`Go version contract` step). Explicit pin when nixpkgs ships 1.27.1.
- [x] ~~**Q5: verify-window advisory flock**~~ **RESOLVED 2026-09-20**
      (owner unsure → advisory default with escape hatch):
      `scripts/lib/verify-lock.sh` (flock at a machine-level path, holder-PID
      diagnosis, `VERIFY_LOCK=0` opt-out) taken by `#verify`, pre-commit
      (60s wait, warn-only), `tag-release.sh` + `batch-release.sh`
      (write windows only — `--audit` is read-only and lock-free; flock is
      per open-file-description, the batch→tag exec re-acquire proved it).
- [x] ~~**Stale `example/taskmanager/v4.*` remote tags**~~ **DELETED
      2026-09-20** per ruling: `v4.0.0`/`v4.0.1`/`v4.1.0` removed from the
      remote (and local); the three dead-path lines dropped from
      `scripts/audit-tag-baseline.txt`; `check-release-scripts` green after.
      v0.2.x remains the served line; v3 lines stay baselined (2026-09-19
      decision).
- [x] ~~**MySQL leg vehicle**~~ **RULED 2026-09-20: keep the VM** — QEMU VM
      leg stays the canonical local vehicle; native-MariaDB productization
      closed; the codification row above became the VM-hardening row.

---

## Declined / Rejected (do not re-litigate)

> Guard list, not a backlog: these were investigated and closed with rationale.
> Re-open only with new evidence or an explicit owner decision.

- **command.Bus / MemoryBus removal (v5 candidate)** — DECLINED 2026-08-29:
  the in-process bus is 47 lines, delivers the documented saga/example flows,
  and complements `watermill/`. Re-evaluate only if the saga pattern moves to
  a dedicated orchestration module.
- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
- **Rewrite `check-module-layers.sh` as Go NOW** — deferred. The script is
  stable (348 lines). Revisit when complexity grows significantly.
- **Fix LogBackend same-nanosecond collision** — atomic-counter cost not
  worth the theoretical correctness gain; a counter may be off by 1.
- **Migrate F001/F002/F005/F014 to per-module coaching** — workspace-global
  by design; low leakage risk.
- **`System.WaitReady(ctx)` API** — declined in the file-renamer TOCTOU
  review: the catch-up drain closes the race; a readiness API would be a
  false guarantee. See
  `docs/feedback/archived/2026-08-13_file-renamer_drain-live-toctou-race-review.md`.
- **file-renamer circuitbreaker/dlq modules** — rejected; failsafe-go + the
  FAQ pointer cover it. See
  `docs/feedback/archived/2026-08-13_file-renamer_extract-circuitbreaker-and-dlq-review.md`.
- **Deep per-item annotation of archived reports (V3 T42)** — declined per
  the 2026-08-29 audit precedent ("So what?" test): archived location +
  claim-level corrections carry the historical signal; mass-striking
  wishlist tails is noise. Revisit only if a specific archived report
  misleads.
- **Benchkit "Phases 6/7 remain" flag** — stale: production replay +
  `benchtest.RunSuite` shipped (ROADMAP Theme 2). Do not harvest.
- **`metaengine.PlanFromSQLite(dsn, ...)` convenience API** — declined
  2026-09-06: comment rot fixed to reference real helpers; add the API only
  if a consumer asks. — source: 07-42 §f30
- **Ack-window pipelining for CatchUpSubscriber** — deferred by design:
  ~160-280K ev/s ceiling with a documented 10× degradation trigger. — source:
  07-42 §c5
- **Per-engine high-water marks for CatchUpSubscriber tail-only re-catch-up** —
  DECIDED AGAINST 2026-09-13: full rebuild is idempotent by construction (engine
  is Reset first), while tail-only replay must assume engine state matches a
  persisted watermark — the exact stale-state class the catch-up race fix
  closed; watermark bookkeeping also needs ADR-0136 reset semantics. Revisit
  only if `Replayed`/`CompletedAt` observability shows rebuild latency hurting
  failover SLOs.
- **Additive `CatchUpEngineWithResult` API** — DECIDED AGAINST 2026-09-13: the
  result is already observable via `CatchUpSnapshot`; the breaking `ResetResult`
  return waits for v5 where signature changes are free.
- **KeyProvider tier (env/file composite provider)** — deferred to ROADMAP;
  the bank-sync ask is closed by the shipped helpers. — source: 08-26 §f14
