# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, 2026-09-08, 2026-09-11, 2026-09-16, 2026-09-19,
2026-09-20, 2026-09-21 ×2 — the 8th pass struck 815+ verified-done items
inline across 57 reports; the 11th pass archived the v4.9.0-wave/M23/W0
cluster (7 files) and harvested the M23 + go-graph-rag + W0 §f tails below,
deleting every completed receipt row).
The Declined section at the bottom is a do-not-re-litigate guard, not a backlog.

> **Prioritized execution plan (2026-09-08):**
> [`docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`](docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md)
> ranked the then-list into Pareto waves (W0 release train → W1 trust →
> W2 efficiency → W3 v5 train) and was EXECUTED through 2026-09-11 (W3's v5
> items live in the v5 section below; the user-gated P22 halves remain in the
> Turso section). **Current plan (2026-09-22 01:25):**
> [`docs/planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`](docs/planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md)
> (T01–T27, all 25 sections mapped; 1% tier = restore the go 1.27.1 contract
>
> - drift gate + composed verify; predecessor:
>   [2026-09-20 17:40 owner-unblock plan](docs/planning/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md),
>   M-items folded into the new T-numbering). This file remains the living source of truth.

## Section index

[Legend](#legend) ·
[Metaengine Universal Storage Substrate](#metaengine-universal-storage-substrate-proposed-2026-09-18) ·
[Durable Work Queue](#durable-work-queue-module-proposed-2026-09-13) ·
[Command-side depth](#command-side-domain-depth-2026-09-13-plan) ·
[Reset-projection stall](#investigate-testsystem_resetprojection_restartandreplay-contention-stall-found-2026-09-13) ·
[Turso matviews](#turso-materialized-views-adr-0135--upstream-handoffs) ·
[Cordis follow-ups](#cordis-spatiotemporal-composability-follow-ups-2026-09-10) ·
[cqrs-lint](#cqrs-lint) ·
[Release / Tagging](#release--tagging) ·
[Metaengine follow-ups](#metaengine--follow-ups) ·
[CI / Infrastructure](#ci--infrastructure) ·
[Code Quality](#code-quality) ·
[v5 Unification](#v5-unification-phase-8-deletion--cut) ·
[Docs truth](#docs--consumer-surface-truth) ·
[benchkit tail](#benchkit-statistical-rigor-tail-2026-09-16) ·
[CV verdicts](#cv-consumer-verdict-follow-ups-2026-09-16) ·
[Goal-closure](#metaengine-goal-closure-follow-ups-2026-09-17) ·
[Vector-search tail](#vector-search-verification-tail-2026-09-15) ·
[Watermill skill](#watermill-sibling-skill-follow-through-2026-09-15) ·
[md-go-validator](#md-go-validator-ci-integration-from-2026-09-13-audit) ·
[Temporal cells](#temporal-versioned-cells--adr-0141-follow-ups-harvested-2026-09-18) ·
[go-graph-rag](#go-graph-rag-feedback-follow-ups-2026-09-15-triaged-2026-09-19) ·
[92-tag tail](#92-tag-release-train-tail-2026-09-20-harvest) ·
[Declined](#declined--rejected-do-not-re-litigate)

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## Metaengine Universal Storage Substrate (proposed 2026-09-18)

> Owner directive 2026-09-18: metaengine becomes the ONE way data is stored/retrieved from disk.
> Full Pareto plan (23 tasks / 82 micro-tasks): [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

- [ ] **T19–T21 (v5-gated): fold capabilities into universal `Engine`, delete the duplicate SQL stacks, release train** — blocked on the v5 train per ADR-0142 §decision; DO NOT execute in v4.x (growing core interfaces is breaking, contract 21g discipline). The tag waves for claiming + the queue family are the separately-tracked item below. _(Effort: L; v5-gated)_

- [ ] **T18b chain hardening: make the armed pipeline survive storms/reboots** —
      the closure/campaign/root-cause stages are detached bash pollers (`/var/tmp/t18b/`,
      pid-chained, 6h deadlines); a reboot or lapsed deadline strands the arc until a
      session re-arms by hand (re-arm one-liner preserved in the live 16-37 report).
      Convert to a supervised design (systemd user unit à la the nightly timer, or an
      idempotent cron re-arm line), results-file polling instead of pid-chaining,
      `PHASE-*.{RUNNING,DONE,FAILED}` watchdog markers, and settle the deadline-lapse
      policy (auto-re-arm vs one-shot — owner question). State 2026-09-21 16:37: chain
      ALIVE, quiet-window-gated (load peaked 863); the 1.27.1 baseline `a91e7cd90`
      stands — re-pin + p99 demotion + `--save` refusal + KNOWN-UNSTABLE suppression +
      per-suite `benchtime::count` + CI parity + widening candidate `100x::9` all
      landed and are production-validated (receipts: the six live T18b reports +
      CHANGELOG [Unreleased]); the completion watcher autonomously finishes
      re-pin→verify→green, then the campaign queue (SearchQuery + dgraph) and
      root-cause matrix fire. Also open: host benchmark-ceiling policy (owner
      UNDECIDED — strict <5 stands). — source: 16-37 §d2/§e2/§f4/§f20, 14-18 §d3 _(Effort: M)_
- [ ] **Go 1.27 follow-ups** — nothing remains: the 94-module `go 1.27.1` sweep
      completed the jsonv2 graduation (2026-09-19) and the load-sweep + baseline
      re-pin landed 2026-09-20 (`a91e7cd90`, T18b arc). Row kept as the section
      anchor only. _(Effort: —)_

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

- [ ] **Queue M4 verification tail (harvested 2026-09-21)** — sliceable:
      (a) unit-pin `deadlockBackoff` (bounds, exponential shape, jitter range,
      attempt cap) and exercise the ClaimDue retry loop under a forced real
      deadlock (fault-injected `claimOnce` or lock-order contention) — the
      shipped backoff path has zero executed coverage;
      (b) prove `testutil/mysqltestcontainer` skip paths (no Docker, `-short`)
      + a smoke test; confirm plain (non-integration-tag) queue/postgres skips
      cleanly without a TestMain;
      (c) clock seam for the conformance harness (ADR-0122 `WithClock` /
      lease-duration injection) to delete the fixed 500-650ms sleeps;
      (d) remove the vestigial `_ = subject` + unused `short` wart in
      conformance tokens.go/lifecycle.go;
      (e) sweep the shared-DB parallel-migrate class (t.Parallel + shared
      `POSTGRES_TEST_DSN` migrate) across metaengine/*engine + storage engine
      suites before CI `-count=2` legs multiply;
      (f) CI legs: queue/mysql via the container harness (runners have Docker)
      + an explicit queue/postgres matrix entry. —
      source: archived 22-01 §b/§e3-5/§f3-17 _(Effort: M total, sliceable)_

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

  done 2026-09-19 — superseded by `cmd/cqrs-lint/v4.12.0` in the 92-tag
  train (buildinfo version reporting rides it; smoke probe covers the
  installed binary).<br>**Original:** on master since 2026-09-11;
  v4.10.1 deliberately predates it. Verify the
  installed binary prints the real tag after `go install …@v4.10.2`. —
  source: 01-47 §b1/§f5 _(Effort: S)_
  2026-09-19 (5 modules with retracts checked); the clean-dir `go list -m
      module@latest` acceptance ran green for 10 key modules post-train.<br>**Original:**
  fail when a master go.mod retract
  directive is absent from the module's newest tag (the inert-retract
  class: `retract v4.8.0` sat on master ~10 days before v4.10.1 shipped
  it). Acceptance test for every retract = clean-dir `go list -m
      module@latest`. — source: 01-47 §d3/§e1/§f6
  _(Effort: S)_
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
      — 92 releases created for the train (152 repo total);
      `create-github-releases.sh` extended to match train-section headers
      (bounded-token match, newest section first, trimmed body + CHANGELOG
      pointer like the 09-08 precedent) with `--dry-run`; all 92 extracted,
      bogus tags skip. `gh` auth working. — source: 05-00 §f12
      _(Effort: S)_
      ADR-0128 extracted codec/retry/idempotency/flightrecorder to external
      repos and the 92-tag train repinned everything; zero
      `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}` references
      remain in any go.mod (verified by grep across all 95).<br>**Original:**
      the transitive
      `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}/v4` indirect deps
      in ~49 consumer go.mod files clean up after new tags publish. Track and
      verify. _(Effort: M)_
      done 2026-09-19 — the train's cut loop ran per-batch pin-sweeps (commits
      2a9ccb75a…38c3b4fd4) and the post-wave `--check --remote` is green
      (local + origin tag sources); `storage/eventstore` is a package inside
      `storage/v4`, so its pin health is the storage pin coherence the sweep
      already covers. — source: archived 07-48 §b2/§f2
      _(Effort: S)_
- [ ] **Release-train tail (post-v4.9.0 waves, queued in [Unreleased])** —
      metaengine wave (row above); queue/mysql + `testutil/mysqltestcontainer`
      tag pair; `scheduling/engine` for `ErrEngineNotDueClaimer`; encryption
      docs/wire-goldens entry; cqrs-lint typed-info tier (P014/F090/F091/
      C008/C013/C035 — the biggest single Unreleased item); cqrs-upgrade
      `--strict` growth; benchkit/cqrs-bench (row in its own section). After the
      queue/metaengine waves: drop taskmanager's four sibling replaces (queue,
      queue/sqlite, claiming, metaengine) — the same consumer-purity play as
      scheduler-otel-status. — source: closeout §f17-24 _(Effort: M each, wave mechanics)_
- [ ] [BLOCKED] **claiming V006 advisory decision (owner)** — claiming has no
      content since v4.0.0, so examples' V006 "same release" advisory is
      structural: content-identical `claiming/v4.0.1` re-tag, or teach V006 to
      skip pins at a module's newest existing tag (linter-semantics fix).
      — source: closeout §f10/§g2 _(Effort: XS + decision)_
- [ ] **Post-wave hygiene** — `scripts/pin-sweep.sh --check` pass (the cut-time
      advisory flagged stale sibling pins repo-wide); V007-gated
      `cqrs-lint-examples` loop over ALL six examples locally (CI ran 3);
      rename-guard check: V006 version-set goldens vs the new tag set
      (taskmanager golden pins the version list). — source: closeout §f15/§f16/§f25
      _(Effort: S total)_
      2026-09-22 (T07 partial): pin-sweep --check GREEN ("All sibling pins
      at their latest tags"); cqrs-lint over all six examples — zero
      error-severity findings after fixing taskmanager (C017 memory-DLQ →
      SQLiteDeadLetterStore on cfg.DatabasePath, S010 wire-vs-at-rest nolint
      with rationale, F031 explicit WithLimit(listPageSize)) and
      readme-quickstart (C028 ×2 discarded Dispatch/RegisterTyped errors
      handled); remaining WARNINGs are deliberate demo simplicity
      (branded-ID suggestions, must.go panics, version-pin mix advisory —
      dispatcher IS at its latest tag). V006 taskmanager golden: still open
      (needs the next tag wave's version set).
- [ ] **Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms (worst-of-30 sample inflates under gate load). Shipped + gated
      green; keep or revisit. _(Effort: XS)_

---

## Metaengine — follow-ups

- [x] **Turso grouped-materialized-views fail-closed (feedback #2)** — done
      2026-09-22 (T16): `tursoengine.New` refuses a `GroupBy` matview spec
      with `ErrGroupedViewBugRefused` unless `WithKnownGroupedViewBug()`
      acknowledges; pinned by
      `TestTursoMatView_GroupedSpecRefusedWithoutOptIn`; the
      defect-exercising repro/bench/property suites opt in explicitly;
      API golden regenerated; CHANGELOG + readmodels.md caveat updated
      (Doctor WARN retained for opted-in deployments). Original:
      `WithKnownGroupedViewBug`-style opt-in: grouped matviews diverge silently
      (upstream defect A, ADR-0135); construction-time refusal unless the caller
      acknowledges. Complements the existing Doctor WARN + envelope-guard test.
      — source: 23-24 followups §f18, feedback doc §3.2 _(Effort: S)_
- [ ] **Feedback #6: system test-mass gap** — (a) config-loader table tests +
      fuzz for `system` (koanf/YAML surfaces); (b) lifecycle/shutdown stress
      with real engines; (c) determinism test (same domain+deployment →
      identical wiring). The 2026-09-21 projectionhost double-apply find is
      evidence for its priority. — source: 23-24 followups §f19-21, feedback doc
      §4.6; execution sequencing: the archived SUPERB excellence plan P2 _(Effort: L each)_
- [ ] **Post-v4.9.0 metaengine tag wave** — publish the [Unreleased]
      metaengine surface: G-T13 ADTSet parity (pg/mysql, mysql VM leg still
      pending a quiet window), G-T12 `BackfillPlannedTables`,
      `ScanScoredVector`/`RowScanner`, adttest helpers (`AssertTxIsolationFromForeignContext`).
      — source: closeout §f17/§c2 _(Effort: M — tag-wave mechanics)_
- [ ] **Calibration provenance protocol + quiet-window re-runs** — protocol HALF DONE 2026-09-11 (later session), re-runs remain gated on a quiet window: (a) DONE — `scripts/calibration-gate.sh` asserts 1-min load < 5 (overridable `--max-load`/`CALIB_MAX_LOAD`; CI exempt) and aborts loudly — verified against a live compile storm (load 207 → hard abort); `calibration-drift.sh` runs it before benching; (b) DONE — protocol items 6-8 in `docs/benchmarks/calibration-2026-08-30.md` define the per-entry PROVENANCE line (store path + binary version output + uptime samples) and ban secondhand version citations; the 2026-09-11 SearchQuery entry now carries an explicit provenance-gap note; (c) MECHANISM DONE, RUN PARTIAL — `benchmark-regression.sh --save` writes a titled provenance header (fixture-tested, parser-safe); the titled re-pin of `benchmarks/benchmark-baseline.txt` **DID run 2026-09-20 17:12 UTC** (T18b row above: noise-clean save, go1.27.1 provenance, claimkit/SQLite entries, 0 regressions vs the 2026-09-11 baseline); the quiet-window count=5 SearchQuery re-run remains pending (a 493-load storm held the 2026-09-11 session; gate correctly refuses); (d) PENDING — re-anchor ALL dgraph constants in one gate-passing window. Run when `scripts/calibration-gate.sh` passes: SearchQuery count=5 (supersede today's table if medians move >5%), then the benchmark-baseline re-pin, then the dgraph constant campaign. — source: 03-50 §b2/§b3/§f7/§f8/§f15/§f16, 02-48 §d3/§f8
      _(Effort: M)_

- [ ] **M20 design-ratification follow-ups (one-pagers delivered 2026-09-21, awaiting owner)** —
      (a) **ADR-0146 candidate: `EngineConfig.SingleWriter`** advisory lease —
      `<dsn>.cqrs-lease` flock, fail-loud default-off, one shared Tier-0-style
      helper (lease semantics today exist only in `queue/`+`claiming/` task
      claims); one-pager:
      [`docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md`](docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md).
      (b) **AggregateOn first cut** — `AggregateSpec` QueryOption on `QueryDecl` +
      construction-time validation + `MatViewSpecReporter` capability + planner
      O(1) pricing for scalar-covered shapes (grouped routing stays unsafe until
      turso-go defect A is fixed upstream + the flip-runbook gate passes);
      one-pager:
      [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md).
      (c) **Routing integration v1** after (b): scalar-covered shapes price O(1)
      and route to the matview engine; Doctor INFO for uncovered shapes.
      (d) Scan-default v5 survey feeding G-T14:
      [`docs/planning/2026-09-21_scan-default-v5-survey.md`](docs/planning/2026-09-21_scan-default-v5-survey.md)
      (row in Goal-closure section). — source: archived 15-34 §a1-3/§f28-32 _(Effort: M each, ratification-gated)_

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

done 2026-09-21 (M16) — every §2.11 claim re-verified against source
(probe interval 1s `probe.go:89-115`, timeout 5s, jitter 0.2,
`DefaultRoutingHysteresis` 0.20 `store_routing.go:17-24`,
`StartAutoReplan` stop-func shape, `Replan`, `GetEngineStats`,
`FormatLiveLatency`): section accurate as-written, no edits needed. —
evidence: archived 15-34 §a9.<br>**Original:** recipes §2.11
(`ProbeEngine`/`LatencyTracker`/`Calibration` surface) had never been
drift-checked against the shipped code. — source: archived 16-43 §f16 _(Effort: S)_

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

`scripts/wait-for-quiet.sh` (1-min AND 5-min ceilings, self-tested),
`scripts/can-run-composed-gate.sh` (no-release-procs + tree-stability +
load assert), and the `#verify` `-p` parallelism cap (`VERIFY_TEST_P`, set
to 4) all shipped in T26 (09-40 §a4, now archived); composed-`#verify`
went GREEN the same day (S03). Remaining launcher ergonomics live in the
release-train tail row below. — source: archived 06-47 §f6-8, 12-02 §f9/11/17, 18-11 §f11

- [ ] [BLOCKED] 🔥 **Push decision (owner)** — 30+ commits from ≥3 sessions sit
      unpushed on master (v4.9.0 wave, go.work fix, guard wave, md-go gate);
      ALL remote CI evidence is gated on it (M23's ci.yml leg, the `Examples
      Test` job, the guard-wave legs, release.yml runs). Also rules the push
      cadence going forward (batch vs phase-boundary). — source: delta §f1,
      18-19 §f38, closeout §f2/§g1 _(Effort: XS — owner)_
- [ ] 🔥 **go.work drift gate** — extend `check-go-version.sh` to assert
      `go.work go >= max(module go directives)`; the downgrade class struck a
      THIRD time (140-file auto-commit `4a540b02c` 2026-09-21 23:33 downgraded
      go.work + all 96 go.mods to `go 1.27`, minutes after the 23:24 session
      had restored 1.27.1 — workspace builds and `#test-examples` are RED
      until re-restored; earlier waves: `27093331c` 18:38, `96dc20986` #11).
      Vigilance failed three times — the gate is the only durable answer.
      — source: closeout §f8/§e1/§d1, verified live by the 11th docs pass _(Effort: S; re-restore = S mechanical)_
      RESOLVED 2026-09-22 ~02:45: ROOT CAUSE FOUND — BuildFlow's
      `go-version-auto-configure` auto-fix (pre-commit hook) canonicalizes go
      directives to major.minor (`1.27.1`→`1.27`); every wave was an authored
      commit's hook run + daemon absorption (2 more waves: ~02:39 + the
      02:41 failed-commit replay). FIX SHIPPED: (1) `.buildflow.yml`
      `skip_steps: [go-version-auto-configure]` (dry-run-verified effective);
      (2) gate extended with floor + full lockstep equality + CI=true leg
      (self-test 9/9 green; wiring inherited via #verify head + nightly);
      (3) upstream BuildFlow patch-floor fix → F154. Waves 5 restores landed
      with the skip in place — no recurrence since.
- [ ] **F154: BuildFlow upstream — go-version-auto-configure must respect
      dependency-driven patch floors** — the step canonicalizes go directives
      to major.minor, silently downgrading modules whose deps require the
      patch (this repo: 5 waves, hours lost). File upstream (owner-approved
      class) with the repro: module with `go 1.27.1` + dep requiring >=1.27.1,
      `buildflow -s go-version-auto-configure --fix` → directive becomes
      `go 1.27`, build breaks. Fleet-wide blast radius — every LarsArtmann
      repo with a patch-qualified contract is exposed. — source: 2026-09-22
      root-cause session _(Effort: S filing)_
- [ ] **F153: pkg.go.dev license detection — "License: UNKNOWN"** hides all
      module docs (license-redistribution gate) for system/v4@v4.9.0 — and
      possibly every module: no LICENSE file at module subdirectory roots?
      Verify whether the repo-root LICENSE propagates to submodules on
      pkg.go.dev; if not, decide per-module LICENSE files or accept hidden
      docs. Consumer-trust blocker for the public surface. — source: T03
      post-wave verification 2026-09-22 _(Effort: S verify, M if per-module
      LICENSE files needed)_
- [ ] **Pre-commit hook env hygiene** — the hook's appended workspace build and
      govulncheck step run on the ambient toolchain (host go 1.26.7 → garbage
      errors, the mid-wave `--no-verify` workaround); inject the documented env
      chain or build per-module GOWORK=off. Overlaps the scoped-gates row above.
      — source: closeout §f9/§f33, followups §f12 _(Effort: M)_
- [ ] **`/mnt/buildcache` capacity monitoring** — hit 100% mid-gate on
      2026-09-21 (an 18G shared-go-cache clear forced rebuilds on other
      builders); 80% warning + a bounded `go clean` policy. RECURRED
      2026-09-22 02:49 — 100% full again (208G/220G, 0 avail; ambient-
      GOMODCACHE toolchain unzips die "no space left on device"; the go-mod
      subdir is only 5.6G, so ~200G lives elsewhere on the mount — needs a
      `du` breakdown before any clear policy).
      — source: followups §f13/§e8 _(Effort: S)_
- [ ] **Watch the first real CI runs (push-gated)** — `Examples Test` job
      (nix eval, 10m timeout, DB-skip env), the md-go-validator ci.yml leg
      (cold build ~1-2 min), the nightly `Go version contract` step, and the
      README push-leg timeout. — source: followups §f2, 18-19 §f3, W0 §f28/§f31
      _(Effort: S, observe)_
- [ ] **W0 verification tail (sliceable)** — (a) `CI=true` self-test-leg
      audit across every gate script lacking one; (b) empty-`go list` fiction
      sweep across the remaining plain-go CI jobs (coverage-gate class); (c)
      `check-go-version` into `verify-ci`/`verify-parallel` heads; (d) grow
      `preflight-composed.sh` (`check-turso-version`, error-taxonomy) if <5min;
      (e) verify-lock consumers audit (smoke-all, load-sweep chains); (f)
      coverage-gate Tier-2 floor (schema/snapshot/projection) after two green
      weeks; (g) document verify-lock semantics in AGENTS; (h) load-guard (10)
      vs calibration (5) two-tier ceiling intent in gowork-modes; (i)
      api-stability golden spot-verify (`WithMaxOpenConns`/`WithMaxIdleConns`);
      (j) triage the 18:38 112-file go-directive downgrade (incident #12?
      closeout §d1 points at the pre-commit hole — confirm and close).
      — source: archived 14-12 §f21-31/§f33-45 _(Effort: M total, sliceable)_
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
- [ ] **Run the real `#integration-mysql-vm` leg through the hardened
      `vm-mysql.sh` in a quiet window (load1<5)**, then update the F52 AGENTS
      integration rows + strike with evidence. The hardening is self-tested;
      the live proof run never happened (load 8–72 all day 2026-09-21).
      Also consider the same stale-port pre-flight for `vm-mysql-nspawn.sh`
      (cheap insurance). — source: archived 14-12 §b1/§f2/§f25, 15-34 §b1 _(Effort: M, quiet-window)_
- [ ] **`scripts/go-env.sh` env-chain helper** — `GOTOOLCHAIN=auto` + the
      cache env chain in one sourced file, adopted by gate scripts, flake apps
      (the `#check-coverage` ambient-PATH fragility), and session tooling;
      generalizes the GOTOOLCHAIN=local incident class (the 14:18 cost-pass
      burn) and the gowork-modes contract. Requested by 4 sessions. — source: archived 14-18 §e1/§f2, 14-52 §f8, 16-37 §e4/§f5 _(Effort: S)_
- [ ] **Wire `quiet-window-run.sh --self-test` + the benchmark gate scripts
      into `check-release-scripts`** (CI-covered shellcheck + self-test for
      `quiet-window-run`, `nightly-bench`, `benchmark-regression`); add the
      "assert the mangle landed" assertion to the check-golangci-hash and
      restore-depguard mutation fixtures while there. — source: archived
      12-38 §f3, 14-18 §f4/§f19, 14-52 §f7, 14-12 §f22/§f24 _(Effort: S)_
- [ ] **Composed `#verify` re-record (W1 sibling)** — the S03 green
      (2026-09-20 15:04) now predates the guard chain (hash-golden,
      wait-loop, load guard, go-version gate, md-go-validator insertions),
      the v4.9.0 4-tag wave + example pin bumps, the go.work 1.27.1
      restoration, and the benchkit/cqrs-bench polish wave; one clean composed
      run re-proves the chain end-to-end (also closes the `#verify-fast`
      execution-unverified insertions). Recipe:
      `bash scripts/preflight-composed.sh && nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify`.
      — source: archived 16-37 §f10, 15-34 §f20, 15-57 §b1 _(Effort: M, quiet-window)_
- [ ] **Verify the nightly weekly load-sweep leg fires** (Sundays-only, first
      real run) and logs cleanly. — source: 14-52 addendum 2, 16-37 §f31 _(Effort: XS, observe)_

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
- [ ] **Rest of sweep §4 (wire-vocabulary renames):** REMAINING (2026-09-22
      pass): (a) SQL `events`/`commands` column renames + migrations —
      recommendation on the table: v5.x expand-contract, NOT the v5.0 cut
      (assessment in `docs/WIRE-FORMAT-KEYS.md`; owner ruling pending);
      (b) consumer grep for old code strings in sibling alert/dashboard
      configs at the cut; (c) `listing.aggregate_projection` collection-name
      rename (TBD). DONE in this pass + earlier waves: pebble event rows
      (`aggregate_*` → `stream_*` with decode-only legacy fallback — the
      last binary surface; census gap closed in WIRE-FORMAT-KEYS), watermill
      dual-read/dual-write, bbolt/pebble command + snapshot rows, benchkit
      keys, pebble slog keys, error-family codes, E1 encoding stamps typed
      as `codec.Encoding`, and the central wire-key table doc itself
      (`docs/WIRE-FORMAT-KEYS.md`). — source: 08-41 §b1/§f1–11, 07-48 §f5-13
      _(Effort: S remaining)_
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
- [ ] **v5 items from extended review — EXECUTED 2026-09-22** — E1 (event
      envelope Encoding typed as `codec.Encoding`; `record.Encoding`
      rejected — its closed enum would drop custom codec stamps), E7
      (`HandlerRetryConfig` rename + deprecated aliases), E8 (typed
      `middleware.Kind`), E11 (`AdapterCore.Encode` error return), E13
      (phantom param documented: Go has no generic methods — the honest
      resolution), E15 (`dispatcher.Middleware[H]` alias unification).
      E3/E9/E10/E14 verified already-done in earlier waves (bbolt
      errorfamily, turso Policy write guards, ShutdownDependency
      validation, OwnedDBHandle type split). Remaining: golden/meta-tests
      pass + cut. _(Effort: — )_
- [ ] **More extended-review follow-ups — DONE (verified 2026-09-22)** — E3
      (bbolt errorfamily — landed), E6 (`BundleOption` → deprecated alias
      of `Option`), E9 (turso Policy nil-write guards — landed), E10
      (ShutdownDependency validation incl. unknown-engine rejection —
      landed), E14 (`OwnedDBHandle` vs `DBHandle` type split — landed).
      (E4 was resolved by the 2026-09-08 stream-code rename.) — source:
      `docs/reviews/2026-08-22_extended-data-model-review.md` _(Effort: — )_
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
- [ ] **Feedback #4: split `system`'s engine requires (`systemtest`)** ahead of
      v5 — "system is a category error for a library" was the consumer's core
      verdict; the go-graph-rag evaluation routed it here. — source: 23-24
      followups §f22, feedback doc §4.4 _(Effort: L)_
- [x] **`metaengine.DeferClose` engine-twin deprecation note** — done
      2026-09-22: `Deprecated` doc note added (record.DeferClose is the
      canonical home since record/v4.6.0; twin kept through v5 for the
      sibling-replace family, removed at the v6 API train) — ADR-0144's
      deliberate-keep stance preserved. Original: every consumer can reach
      the Tier-0 `record.DeferClose` since record/v4.6.0; add the v5-list
      note to deprecate the self-contained twin (ADR-0144 kept it
      deliberately — revisit at the v5 API train). — source: closeout §f45
      _(Effort: XS)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate. _(Effort: M)_

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [ ] **README review deep-read tail (2026-09-13 cluster, 8th-pass harvest)** —
      (b) add READMEs to the doc-check gate (flake app/CI); (d) quick-start
      drift-guard tests for stack/sqlite, storage/memory, decider, scheduling,
      projectionhost. [(a) doc-check repoRoot fix, (c) T37 deep-reads, (e)
      deprecated-symbol gate, (f) link checker — all done 2026-09-20.]
      — source: archived 12-16 §f (150-154, 158-161, 174, 179) _(Effort: M total,
      sliceable)_
- [x] 🔥 **Post-v4.9.0 skill-reference sweep** — done 2026-09-22 (T14):
      core.md + recipes.md pyramids rewritten to the fluent
      `Evolve(...).On(...).Done()` chain (recipes catalog trailer updated,
      TestRecipes green); FAQ "How do I write a minimal third-party engine"
      entry added (AtomicAppender/Transactional/ErrRacySaveRefused honesty);
      F151 rule-count drift fixed (206→207 in FEATURES/ROADMAP/cqrs-lint
      README); at-least-once contract block in readmodels.md + dedup routes;
      getting-started counter canary (README note + seam-naming failure
      messages); goal-shaped README fence verified green
      (F152). doc-check 1200 refs + TestRecipes + example tests all green.
      Original: grep SKILL.md +
      `references/*.md` + example READMEs for the nested `OnEvolution` pyramid
      (still shown in core.md:136, recipes.md:2257/2600, goal-shaped README)
      vs the now-blessed fluent `Evolve(...).On(...).Done()` chain; adopt `.On`
      where shown (recipes catalog entries + compile-tests for changed fences);
      add the FAQ "writing a minimal third-party engine" entry
      (`AtomicAppender`/`Transactional` honestly; racy fallback now refused) +
      the fail-closed registration recipe. — source: closeout §f4/§f6/§f14,
      followups §f14-16/§f48 _(Effort: M)_
- [x] **Post-wave release verification** — done 2026-09-22 (T03): 3 of 4
      Releases had rendered (system/v4.9.0, record/v4.6.0,
      projectionhost/v4.5.1); `scheduling/sqlstore/v4.1.1`'s tag was pushed
      but release.yml never triggered for it — release created manually
      with a provenance note. system/v4.9.0 notes curated (headline: fluent
      `.On` chain + fail-closed racy Save). pkg.go.dev indexes system/v4
      @v4.9.0; `On`/`WithRacySave` render, `ErrRacySaveRefused` defined at
      system/errors.go:24 and tagged — BUT pkg.go.dev shows "License:
      UNKNOWN" and hides docs (license-redistribution gate) → folded into
      F153. Original: confirm the 4 GitHub Releases
      rendered (release.yml on the v4.9.0-wave tags) + curate system/v4.9.0's
      notes; pkg.go.dev spot-check that `On`/`ErrRacySaveRefused`/`WithRacySave`
      render. Push-gated. — source: closeout §f3/§f27 _(Effort: S)_
- [ ] **Canonical T18b record + gate-semantics ADR** (replaces the six-report
      narrative series) — RECORD DONE 2026-09-22 (T17):
      `docs/benchmarks/2026-09-20-21_t18b-record.md` written (closure
      receipt, the four gate-semantics changes with evidence, widening rule,
      KNOWN-UNSTABLE table, GOTOOLCHAIN + load-storm + reboot incidents);
      `/var/tmp/t18b` retired (trashed; chain had landed green 2026-09-21
      18:14 UTC per closure-completion.log). REMAINING: the gate-semantics
      ADR document + the calibration case-study appendix in
      `docs/benchmarks/calibration-2026-08-30.md` (storm/reboot/p99/bimodal/
      GOTOOLCHAIN/863 series as the "why the gates exist" record — source
      material now one click away in the canonical record). Original: one
- [ ] **Stale-reference sweep for the bench-gate contract changes** — old
      noise-headline list, unconditional `--save` mentions, matview
      benchtime/count mentions across README, `cmd/cqrs-bench/README.md`,
      docs/benchmarks, and workflows (five consecutive sessions flagged it).
      — source: archived 10-30 §f3, 12-38 §f5, 14-18 §f5, 14-52 §f6, 16-37
      §f16 _(Effort: S)_
- [ ] **M13 tail: per-module fresh-run last-verified stamps + script-derived
      counts** — FEATURES guarantee rows carry 09-21 doc-gate stamps, but
      per-module fresh-run verification stamps need quiet CPU to be honest;
      extend `check-canonical-facts.sh` to derive the go.mod count into
      FEATURES too (F69 overlap, kills the last hand-maintained count). —
      source: archived 15-34 §b4/§f5, 14-12 §f23 _(Effort: M, quiet-CPU)_
- [ ] **doc-check `--list-all-ambiguous` mode** — emit EVERY instance per
      ambiguous alias, not just the first (the M16 alias bundle took 5
      iterate-and-unmask rounds; one sweep grep would have collapsed them).
      Candidate for the md-go-validator-adjacent tooling wave. — source:
      archived 15-05 §e, 15-34 §c/§f8 _(Effort: S)_
- [ ] **docs-health pass hygiene (10th-pass §e/§f15-20)** — (a) index-vs-disk
      gate: extend `check-canonical-facts.sh` (or a sibling) to derive
      live-report count vs the README table, archived count vs the day-table
      sum, and day-row presence per archived day (the 9th AND 10th passes
      almost shipped index rot); (b) mechanical harvest ledger (per-report
      item → new-row/existing/declined table) as a pass artifact; (c) weekly
      docs-health cadence decision (owner); (d) pass checklist (index update
      step + banner vocabulary pointer) — suggest upstream to the crush-config
      skill repo. — source: archived 23-31 §e1-3/§f15-20 _(Effort: S gate + XS conventions)_
- [ ] **Bench-gate/tooling single-mention tail (16-37 §f, never harvested)** —
      `--explain <bench>` per-sample diagnostics; `--json` evidence output for
      quiet-window-run; deep-quiet window probe/logger; variance-aware
      stability probe (two-axes verdicts); CI baseline-artifact 5-new-entries
      confirm; fragile p99/max threshold sweep across gates;
      `SUM_VIA_GROUPED/baseline` +34.5% one-off investigation; auto-embed
      noise verdict + CoV + GOVERSION into `--save` headers; DirectSQL A/B
      gate-set decision (dep-budget review first); README ops section for
      `quiet-window-run`/`nightly-bench`. — source: archived 16-37 §f14-26/§f35
      _(Effort: S-M each, sliceable)_

---

## benchkit statistical-rigor tail (2026-09-16)

> The 02-09 session shipped P100 exactness, `RunRepeated`/`RepeatedResult`,
> per-metric `MetricVariation`, real benchstat samples, and load provenance —
> all verified green. The 09-35 session closed the two blockers (queue clones,
> soak gocyclo) and refreshed the FEATURES coverage line (151+43). What follows
> is the consolidated remainder. — source:
> [`docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md`](docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md)
> §b/§f, 09-35 §f P3

- [BLOCKED] **Supersede-note on the oversubscribed 2026-09-19 capture** —
  annotate `docs/benchmarks/2026-09-19_backend-comparison-variation.md` as
  superseded (keeping it as the what-noisy-looks-like example) once a
  quiet-window capture exists. Blocked on machine quietness: load was
  35.6/32 CPUs at the 2026-09-21 attempt. Protocol: wait for
  `scripts/calibration-gate.sh` PASS, re-run the compare command from the
  capture header, then annotate. — source: archived 15-37 §f3/§f30

- [ ] **Benchkit polish-tail verification debts (harvested 2026-09-21)** —
      (a) test for `RunSuiteRepeated` (the one new export with zero direct
      coverage: tiny profile × 2 repeats, assert `<metric>_cov%` metrics +
      NOISY Logf); (b) verify `<metric>_cov%` through real benchstat output;
      (c) `startProfiling` teardown-order test; (d) drift-tripwire pinning
      script `NOISE_HEADLINE` == `benchkit.HeadlineMetricNames()` (the
      split-brain is comment-enforced today); (e) list-phases metric-map
      test (every non-`report:` name must exist in `benchkit.MetricNames()`);
      (f) fix the README `--progress` default row (says 0, flags.go says 5s);
      (g) sync benchkit/README.md + doc.go API tours with the new exports
      (RunSuiteRepeated, HeadlineMetricNames, constants, ReservoirSize);
      (h) tighten `noise_target_guard` to identifier-grade matching; (i)
      investigate the testcontainers teardown noise (`🚫 Container terminated`
      during the race repro — leak or expected cleanup?). — source: archived
      15-57-benchkit §b/§d/§e1-2/§f1-10 _(Effort: M total, sliceable)_
- [ ] [BLOCKED] **Benchkit tag wave (owner go-ahead)** — cut benchkit with
      the statistical-rigor + polish-tail APIs (~+17 untagged exports deep),
      bump `cmd/cqrs-bench` pin, strip the sibling replace; batch with the
      next queue/system release or cut now — owner timing call. — source:
      archived 15-57-benchkit §f15/§g1 _(Effort: M, owner-gated)_

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
> [`docs/planning/archived/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md`](docs/planning/archived/2026-09-16_21-05_SUPERB-metaengine-system-excellence-pareto-plan.md)**
> (P0: tag wave + replay-starvation fix + ApplyBatch atomicity; P1: lease +
> FilterContains + Forever + E9/E10 + matview guard; P2: v5 deletions + E-items +
> AggregateOn seam; P3: proof + docs + v5.0.0 cut).

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
      **G-T01 DELIVERED 2026-09-21:**
      [`docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md`](docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md)
      (R01 Infer coverage inventory · R02 sanctioned-surface coverage matrix ·
      R03 consumer shapes · 3 options + hybrid recommendation · per-outcome
      XS session script; ruling lands as **ADR-0146** — the ADR-0141 slot this
      row named was taken by temporal cells). REMAINING: G-T02 owner ruling
      (+ G-T03 one-pager if revive/hybrid).
      — G-T01/G-T02/G-T03 _(Effort: S memo + XS ruling; M if revive)_
- [ ] **Scan default v5 decision** — SURVEY DELIVERED 2026-09-21:
      [`docs/planning/2026-09-21_scan-default-v5-survey.md`](docs/planning/2026-09-21_scan-default-v5-survey.md)
      (consumer census incl. `system.Find` inheriting the cap + recommendation:
      flip to unbounded at the v5 cut + cqrs-lint nudge + optional operator
      ceiling). Documented-100 is the status quo, loud in godoc+FAQ. REMAINING:
      owner decision, implement at the v5 branch. — G-T14 _(Effort: S decision + S impl)_
      **RULED 2026-09-21 (owner): Option C** — unbounded at the v5 cut +
      cqrs-lint nudge + operator ceiling. The v4-safe add-ons LANDED:
      `metaengine.WithDefaultLimit(n)` plan option (operator ceiling for
      un-limited scans; explicit `WithLimit` wins; survives Replan) with tests,
      and cqrs-lint **F031** `scan-without-limit` (warn/low, suppressed by
      `WithDefaultLimit`, disabled in library presets). The remaining step —
      flipping the built-in 100 to unbounded — executes ON THE V5 BRANCH per
      the ADR-0123 cut plan.
- [ ] **FEATURES maturity flip for the closed surface (🧪→✅)** — earned by
      the plan's gates (not declared): evidence links per row, CHANGELOG
      Goal-story entry, release notes. Final stamp of Goal closure. — G-T25
      _(Effort: S, gated on gates A–D)_
      **GATE STATUS (verified 2026-09-21, still BLOCKED — do not flip):**
      Gate A needs the owner Direction Ruling recorded as an ADR
      (G-T01/G-T02 — the 🔥 row above, unanswered; ADR-0141 slot was taken
      by temporal cells) plus routing integration tests across ≥2 engines.
      Gate B needs the G-T16 parity benchmark + new-surface conformance
      sweep. Gate C/D need the release wave (W0 ~90-tag publish) and the
      example-green-under-two-engines proof (sqlite ✓; postgres leg landed
      2026-09-21, runs under `#integration-pg`). This row executes only
      after ALL of those are earned.

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
- [ ] **Skill-quality tail — verify upstream latests for the
      redisstream/kafka/amqp/sql watermill plugins** (trigger-eval prompts,
      `references/advanced.md`, both cross-links, and the claims-checklist rule
      all landed 2026-09-21). — source: 19-45 §f1/§f3-5/§f8; 19-57 §f3/§f8
      _(Effort: S)_

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

## md-go-validator CI integration (from 2026-09-13 audit)

> The gate SHIPPED 2026-09-21 (M23): `#check-md-go` green on the 1,642-block
> corpus (1461 valid / 78 skipped / 103 baselined archived), wired into
> `#verify`, `#verify-fast`, and ci.yml; P2+P3 swept (67 `// skip-validate`
> insertions across 34 files); P4 policy = archived-only baseline with
> inert-shrink ratchet. Build narrative: the archived 18-19 + 23-24 delta
> reports. The harvested open tail: — source: 18-19 §f, 23-24-delta §f

- [x] 🔥 **`--self-test` for `scripts/check-md-go.sh`** — done (2026-09-22
      verified 4/4 PASS): planted fixture repo + PATH-stubbed binary
      (flag-file controlled), pins green-pass / new-error refusal /
      live-path-baseline refusal (ARCHIVE_SEGMENT mutation leg) /
      uncommitted-baseline refusal; wired into `#check-release-scripts`
      (flake:1047) + ci.yml:85. FEATURES gates row +
      `docs/release-checklist.md` mention added 2026-09-22. Original:
      planted fixture tree
      (PATH-stubbed binary or `--config` override), golden message shapes in
      `scripts/testdata/`, mutation-tested goldens; the four gate behaviors are
      currently session-only memories (repo convention: CI-gating scripts ship
      with self-tests). _(Effort: M)_
- [ ] **Explain + verify the 11 tool-heuristic auto-skips, then decide
      `--fail-on-skipped`** (strict vs tolerant) — open since the 09-13 audit
      (§b5). _(Effort: S + XS decision)_
- [ ] **Baseline-bump ritual + stale-entry ratchet** — pin the flake input to
      tag/rev OR encode the master+lock bump ritual as a script (gotchas prose
      exists); meta-check that fails when the baseline references files that no
      longer exist (may only shrink). _(Effort: S)_
- [ ] **`docs/status/` + planning authoring convention** — pseudo-Go fences in
      new reports get `// skip-validate` at WRITE time (prevents surprise CI
      failures); one convention line in `docs/status/README.md` + a CONTRIBUTING
      paragraph for doc authors (fence-tag + regenerate command).
      — source: 18-19 §f10/§f20 _(Effort: S)_
- [ ] [BLOCKED] **Upstream md-go-validator (owner repo; verify-before-filing)** —
      (a) relative-path baseline mode (deletes every consumer's sed
      re-absolutization layer); (b) `--save-baseline` exits 0 when the save
      succeeds (wrappers should not need `|| true`). _(Effort: M + XS)_
- [ ] **Gate wiring tail** — `check-md-go` into nightly-gates.yml; FEATURES
      gates-inventory row + `docs/release-checklist.md` mention; mutation-test
      the `ARCHIVE_SEGMENT` regex in the script. — source: 18-19 §f15/§f16/§f37
      _(Effort: S total, sliceable)_
- [ ] **Version stamp + fleet pins** — packaged `--version` prints `dev` (VCS
      stamping stripped by buildGoModule/proxyVendor?); host-binary catch-up
      (SystemNix relock so bare runs agree with the app); consider one documented
      pin cadence across the gogenfilter/SystemNix tool builds.
      — source: 18-19 §f13/§f18/§f26 _(Effort: S/M)_
- [ ] **Investigate the 5h endurance green window** — real annotation discipline
      or zero exposure (did any concurrent doc carry go fences at all)? One jq
      diff over the gap's doc commits. — source: delta §f6 _(Effort: S)_

## go-graph-rag feedback follow-ups (2026-09-15, triaged 2026-09-19)

> Consumer evaluation of `metaengine/v4.13` + `system/v4.7` (adopted neither —
> "system is a category error for a library, metaengine wrong-shaped for
> GraphRAG"). Feedback #3 (fail-closed Save) and #5 (doc.go stamps) are FIXED
> and released in the 2026-09-21 v4.9.0 wave; #2 → Turso §, #4 → v5 §,
> #6 → metaengine rows, #1/#7/#8 → ROADMAP. Source:
> [`docs/feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md`](docs/feedback/reviewed/archived/2026-09-15_go-graph-rag_metaengine-system-evaluation-feedback.md)

- [ ] [BLOCKED] **Update the go-graph-rag consumer** — feedback #3 + #5 are
      fixed and released (`system/v4.9.0` `ErrRacySaveRefused`/`WithRacySave`/
      `ErrEventSaveNotAtomic` + the 17 `# Experimental` doc.go stamps); nobody
      has told the consumer. Needs owner voice (`github-voice`); consider
      inviting a re-test on v4.9.0. — source: 23-24 followups §f5/§f40 _(Effort: S)_
- [ ] **At-least-once projection-fold contract** — document the dedup recipe +
      the at-least-once delivery contract in the skill references
      (`recipes.md`/`readmodels.md`); mark getting-started's counter test as
      the at-least-once canary in its README (purpose + what a failure means);
      make the convergence test's failure messages actionable (name the seam:
      drain/live overlap, pin drift, or load). — source: 23-24 followups §f6-8
      _(Effort: M)_
- [x] **scheduler-otel-status test suite** — done 2026-09-22 (T15):
      `main_test.go` added — claim-flow test (schedule → Due-claim →
      MarkFired → Metrics() counts exactly that → second poll finds
      nothing, race-clean) + the /status rate-math test. flake.nix comment
      flipped back to "all six carry suites"; CHANGELOG's "all six" wording
      is now TRUE again. Original: the one example with NO test files
      (`go test` → `[no test files]`); the flake comment said "all six carry
      suites" — comment corrected 2026-09-22, the suite is still missing (or
      correct the claim everywhere + drop it from the CHANGELOG wording).
      — source: 23-24 followups §f9/§f10, closeout §f12/§f13 _(Effort: S)_
- [ ] **Naming: `evolutionBuilder.On` vs `lookupBuilder.On`** — same method
      name, subtly different semantics (fold registration vs sample
      registration); document or align. Also watch the line-count ratchet:
      `projectionhost/worker_drain.go` 329, `system/evolutions.go` 320 (350
      cap). — source: 23-24 followups §f30/§f31 _(Effort: S/L)_
- [ ] [BLOCKED] **goal-shaped-app postgres e2e CI leg** — the test exists
      (DSN-gated, runs under `#integration-pg`); the deferred slice is the
      env-provisioned CI leg (ephemeral PG in the examples job vs nightly app —
      cost/queue owner call). — source: closeout §f11/§g3 _(Effort: S + decision)_
- [ ] [BLOCKED] **Does `#test-examples` join blocking `#verify`?** — built
      CI-only (fast local loops); the projectionhost double-apply bug lived
      precisely in the build-vs-tested gap. Owner call on gate ownership.
      — source: 23-24 followups §f11/§g2 _(Effort: XS decision)_

---

## 92-tag release-train tail (2026-09-20 harvest)

> Harvested from the train execution report's §f (`docs/status/archived/2026-09-20_10-25_tag-wave-execution-92-tags-ci-templ-triage.md`)
> and the verify-arc reports; items since done are struck inline there. The
> owner questions ride the W3 bundle section below.

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
- [ ] [BLOCKED] **Upstream filings (owner approval; verify-before-filing
      first)** — (a) turso-go native-lib hash-mismatch + lazy-init failure
      family (`TestBackend_LazyInit_Concurrent` / `TestVectorSearch_LibSQLPushdown`
      red-ing 1–3 legs intermittently; bump-or-file decision rides the owner);
      (b) exhaustruct_v5 `skippedNamed` panic (repro ready); (c) go/types +
      x/tools parallel-check race. Draft in Lars's voice via github-voice after
      verification. — source: archived 10-25 §f3-4/§f20, 15-37 M14 _(Effort: S each + approval)_
      2026-09-22 repro-prep session findings on (b): the upstream module is
      `dev.gaijin.team/go/exhaustruct/v5` (the gaijin fork golangci wraps as
      exhaustruct_v5) — `github.com/4meepo/exhaustruct` is 404-GONE, file at
      the Gaijin forge. A singlechecker harness built + ran v5.0.3 (kept at
      `/tmp/exhaustruct-repro`, rebuild: go get
      dev.gaijin.team/go/exhaustruct/v5/analyzer@v5.0.3); minimal synthetic
      shapes (same-pkg + cross-pkg embedded, unkeyed-element and keyed-field
      forms) do NOT trigger the panic — the trigger is subtler than the 16-51
      report's one-liner; the definitive repro is running the analyzer over
      the historical pre-fix tree (refs around `eea1c3c66^`, storage/pebble +
      stack presets still carried the old literal forms). Isolate before
      filing; check the fork's latest version for a fix first (Gate 5).
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

## Upstream asks from cqrs-htmx (harvested 2026-09-22, docs-health D1)

- **Upstream `requestContextEnricher` into `event/`** — cqrs-htmx's usermgmt carries a local copy (correlation/request-ID metadata enricher, `audit_context.go`) because no upstream enricher covers it. Upstreaming it lets the local copy drop at the next family train. Source: cqrs-htmx TODO_LIST P3 ask (3); verify pass there 2026-09-22.
- **`system.New` checkpoint/DLQ store options** — the declarative composition root uses an internal in-memory checkpoint store, so consumers needing durable checkpoints or dead letters cannot use it (ADR-0051's accepted limitation keeps cqrs-htmx's `NewProjectionLayer` consumers pinned until this lands). Source: cqrs-htmx ADR-0051 + TODO_LIST P3 ask (4).
- **`System.Explain`: include per-query Volume/placement in the topology view** — Explain currently shows drivers/engines/collection counts; the metaengine cost-based planner's Volume hints (which cqrs-htmx's systemadapter declarations all carry) are invisible for introspection. Verified absent against system/v4.9.0 on 2026-09-22 (empirical run: topology prints collections count only).
