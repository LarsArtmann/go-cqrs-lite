# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, and 2026-09-08). The Declined section at the
bottom is a do-not-re-litigate guard, not a backlog.

> **Prioritized execution plan (2026-09-08):**
> [`docs/planning/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`](docs/planning/2026-09-08_17-45_SUPERB-pareto-execution-plan.md)
> ranks this entire list into Pareto waves (W0 release train → W1 trust →
> W2 efficiency → W3 v5 train) with 27 medium tasks and a ≤12-minute micro
> breakdown. This file remains the living source of truth.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## Turso materialized views (ADR-0135) — upstream handoffs

> Created 2026-09-07 (matview operator option shipped; three upstream
> turso-go defects verified and documented — see
> `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`
> and the archived report
> `docs/status/archived/2026-09-07_19-25_turso-materialized-views-operator-option.md`).

- [BLOCKED] 🔥 **File the standalone upstream issue for the silent wrong-results bugs (defects A+B)** — grouped views diverge from the second transaction on and collapse at ~27k rows; draft is ready and fully verified in `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (everything below its first `---`). Blocked on user approval (external action). The COMMIT-abort half (defect C) is already reported: PR #8257 comment https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646. _(Effort: XS once approved)_
- [x] 🔥 ~~**Push the 3 unpushed commits, then edit the PR comment link to SHA `18b2c495c`**~~ DONE 2026-09-11 — the commits were already on `origin/master` (remote-tracking ref was stale); PR #8257 comment permalink updated `1c9f3bf` → `18b2c495c` and verified the blob renders.
- [x] ~~**Track turso-go releases for the IVM fixes** (2026-09-11 check)~~ — v0.8.0-pre.9 (09-08) and v0.8.0-pre.10 (09-09) released; repro re-run on pre.10: defect A reproduces with the IDENTICAL 430.50 delta at the 2k checkpoint, defect C's COMMIT abort still fires at the 27k chunk, scalar views exact. NOT fixed — warnings/bench-skips stay; all version citations refreshed to "≤ v0.8.0-pre.10" (Doctor WARN, recipes §2.29, readmodels caveat, AGENTS gotcha, bench doc, ADR-0135, FEATURES). PR #8257 still open, no maintainer response. Next check: on each new release tag. _(Effort: S per check)_
- [ ] **Code guard follow-up: make grouped-spec safety mechanical** — today the danger is advisory-only (Doctor WARN + docs). Options: `MaterializedViewSpec` validation refusing `GroupBy` on turso-go ≤ v0.8.0-pre.10 (breaking for legitimate small deployments) vs a config flag (`AllowGroupedViews`) vs silent status. Decide + implement once the upstream timeline is known (still unknown: PR #8257 unanswered, defect re-verified live on pre.10 2026-09-11). The mechanical flip point now exists: `TestTursoMatView_GroupedSumDefectAEnvelopeGuard` + `TURSO_IVM_ENFORCE_FIX=1` asserts exactness at the 2k-row repro shape the day upstream fixes it. _(Effort: S)_
- [x] ~~**Matview safety test tail**~~ DONE 2026-09-11 — (a) Doctor section tests existed (`metaengine/materialized_view_doctor_test.go`: Content/NoneBranch/GroupedWarnPin/NoGroupNoWarn); (b) `matViewDDL` golden existed (`metaengine/sqliteengine/matview_ddl_test.go`: fn × scalar/grouped + collection-quote escaping); (c) ADDED `metaengine/tursoengine/matview_property_test.go` `TestTursoMatView_PropertyServedMatchesBase` (rapid, 100 draws: random datasets × random 2-3 tx splits, scalar SUM/COUNT/AVG/MIN/MAX + grouped SUM served == in-memory expected, envelope ≤60 rows/≤8 groups — 900+ draws verified green); (d) small-scale pin existed (`TestTursoMatView_GroupedSumTwoTxDivergencePin`); ADDED `TestTursoMatView_GroupedSumDefectAEnvelopeGuard` — pins the ACTUAL 2k-row/316-group/2-tx defect-A repro shape, SKIPS while the defect is live, fails with the flip protocol (`TURSO_IVM_ENFORCE_FIX=1`) the day upstream fixes it. — source: archived 19-25 §f2/§f6/§f16-17, 05-33 §f11/§f16-18
- [x] ~~**Extend `scripts/benchmark-regression.sh` to the matview read bench (1k)**~~ DONE 2026-09-11 — the gate is now a multi-set allowlist (`GATE_SETS`, `DIR::REGEX`): stack/bench pipelines + `metaengine/tursoengine` `BenchmarkMatViewRead/agg=[A-Z]+/scale=1k` (14 names incl. grouped serving); `--bench`/`--dir` keep legacy single-set semantics; CI `benchmarks.yml` regression job runs the second set into the same `current.txt`; local baseline refreshed with matview entries; also fixed a latent locale bug (`comm` vs `LC_ALL=C sort` mis-sorting on `/`/`=` names). Verified: 16-stable self-compare, synthetic 2× regression → exit 1. — source: archived 19-25 §f5, 05-33 §f19
- [ ] **Matview v2 feature surface** — planned-table matviews (ordered with `ApplyLayout` + backfill), filtered-view spec variants, multi-aggregate/DISTINCT serving, `DropMaterializedView` off-boarding, per-view IVM write-amp otel counter, `system.Introspection()` surface, cqrs-lint rules (matview-on-unsupported-driver; matview-plus-planned-table staleness trap), `example/materialized-views/`. Route individually when a consumer asks. — source: archived 19-25 §f23-35, 05-33 §f29-35
      _(Effort: M/L each)_
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A (it would steer production aggregates at known-wrong results) — scope the first cut to scalar-covered shapes only. — source: archived 19-25 §f29, 05-33 §f32
      _(Effort: M)_
- [ ] **Tag wave for the matview feature** — metaengine/sqliteengine/tursoengine/system carry sibling replaces for unpublished symbols (`MaterializedViewSpec` family); pins must be bumped and replaces stripped at the next release wave so consumers can use the feature from published tags. _(Effort: M — see AGENTS.md tag-wave procedure)_

---

## Cordis spatiotemporal-composability follow-ups (2026-09-10)

> Source: [`docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md)
> (mapping report: `docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`).
> Operationalizes the Cordis learnings — revertible effects, reactive coeffects,
> observational equivalence — as correctness + trust wins **without breaking a
> v4 consumer**: every behavior change warns-first in v4.x, hard-errors at v5
> (rides the ADR-0123 wave).
>
> **ALL 27 tasks (M-01..M-27) DONE 2026-09-10** — shipped surface lives in
> CHANGELOG `[Unreleased]`: Reset warn-guard + projectionadapter `Resettable`
> + ADR-0136, the coeffect gate (`DomainConfig.Events` /
> `ErrDanglingEventSubscription`), cqrs-lint E018, `ValidateCoeffects` +
> `coeffects.md`, the arXiv grounding + figure recounts, ADR-0137 engine
> deactivation (quarantine/reroute/reprobe + Doctor/Stats health), and the
> equivalence tooling (`scenario.Interleaved` /
> `AssertObservationalEquivalence` + rapid property). Vocabulary stays
> internal-only until v5 (M-26 default held, verified leak-free). This
> section now carries only the follow-ups those shipped features surfaced.

- [ ] 🔥 **`sqliteengine.ResetEngine`** — implement `EngineResetter` on the
      production-default engine: 8 `meta_*` tables + planned tables +
      matviews + `multiSeq` state. Unblocks one-call revert on SQLite.
      Deliberately deferred out of the execution (risk surface; decided
      2026-09-10). _(Effort: M)_
- [ ] **EngineResetter on remaining persistent engines** — pebble, bbolt,
      badger, pg, mysql, turso, duckdb, dgraph, iroh (memory done; sqlite
      prioritized above). _(Effort: M each)_
- [ ] **Surface reset capability in `Doctor`/`GetEngineStats`** — operators
      should see which engines can reset before calling `Store.Reset`
      (ADR-0136 capability ladder). _(Effort: S)_
- [ ] **Fold-write failover for quarantined engines** — ADR-0137 currently
      reroutes reads only; writes to a quarantined engine's collections
      error loudly until reactivation/replan. Consider shadow-replication
      or write-reroute with catch-up. _(Effort: L)_
- [ ] **cqrs-lint E018 fold-case coverage** — the static rule flags
      projection subscriptions only; a fold case consuming a never-emitted
      type is caught by the runtime gate (`DomainConfig.Events`) but not
      statically (CollectFoldCaseStrings carries no position info). _(Effort: S)_
- [ ] **goleak for `metaengine` + `projectionhost` suites** — M-08 covered
      `system` only. _(Effort: S)_
- [ ] **`[Unreleased]`-position tripwire in `verify-docs.sh`** — the
      exactly-one check exists; add "must sit directly under the `#
      Changelog` header block" so a daemon-absorbed orphan fails at the
      next verify, not days later. _(Effort: XS)_
- [ ] **Release-train note** — the `metaengine/projectionadapter` sibling
      replace + the `metaengine` pin bump ride the next tag wave
      (`scripts/tag-release.sh` strips the replace; smoke at cut time).

---

## cqrs-lint

> Point-in-time execution plan (T01–T24 / F001–F096) with per-row resolution
> markers: `docs/planning/archived/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`.
> T01–T12, T20–T24, F089, F090(a+b), F091 Tiers 1–3 (incl. P014 ApplyLayout)
> and the 2026-09-08 hardening batch are DONE (CHANGELOG `[Unreleased]`); this
> section carries the living remainder.

- [x] **T13–T19 — exhaustive rule audit batches.** DONE 2026-09-11:
      every family audited per-rule (A001–A034 in 2 waves, B001–B031,
      D001–D019, E001–E018, V/T/F families, S001/rules.go line-by-line;
      C-family sampled 2026-09-06, S002–S011 2026-09-07). Real defects fixed
      (V006 semver sort, E017 dead suppression, D001/D005, S001 coverage,
      A013 embed FN, alias-blindness across A002/A003/A022/A024/A027/A030/
      D011 via `lintutil.QualifierTargetsModule`, B021 parity, T004 dead
      disjunct, 9 comment drifts) — see CHANGELOG `[Unreleased]` and the
      deferred follow-ups below. — source: archived/2026-09-06_02-40 §c,
      05-31 §f16-22
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
      the strong/weak split; F009/F010 pattern tokens; F018/F020
      mixed-confidence unpinned; V002/V003/V006 root-go.mod-only scope;
      b022_b025.go (495) and a020_a021_a022_a023.go (~357) over the 350-line
      convention — bundle with the file-size-gate policy decision.
- [x] 🔥 **F091 Tiers 2–3 + F090(b)** — ALL DONE: Tier-2 core + F090(b)
      2026-09-08 (`--typed-info` flag, typed dot-import attribution, C008
      usage-confirmation); Tier-3 remainder 2026-09-11 (C035/C013
      payload-shape confirmation under the same gate + the `&T{}` payload
      capture fix that makes the evidence channel see real emissions).
      — source: 05-31 §b4
- [x] **ApplyLayout rule** — DONE 2026-09-09 as P014
      `applylayout-bypasses-plan-path` (detection pair corrected to the
      `ApplyLayout` call + `ApplyLayoutPlan` method shape; see CHANGELOG
      `[Unreleased]`). — source: session-4 retro §f25
- [x] **`IsQualifierFor` adoption sweep** — DONE 2026-09-08 (Pareto P12):
      scanCallExpr + D018/D019 catalog-builder + performance JSON-codec
      heuristic all resolve via `IsQualifierFor`; alias-blindness class dead.
      — source: 05-31 §f1/§e8
- [x] **Replace-based end-to-end fixture module for typed-path rules** — DONE
      2026-09-08: `cmd/cqrs-lint/testdata/typedfixture` (schema/v4 via relative
      replace, excluded from api-stability/testModules meta-tests) drives the
      F090(b) tests in CI. — source: 05-31 §f7
- [x] **Extend the completeness-meta-test pattern** — DONE 2026-09-08 (P12):
      `consumerOnlyRules`, preset disable/override IDs, and the --preset help
      text are all meta-tested (`filters_meta_test.go`). — source:
      23-10 §e3, 05-31 §f9
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
- [ ] 🔥 **350-line limit: gate red repo-wide — split waves + gate-policy
      decision.** VERIFIED 2026-09-06, recounted 2026-09-11: 58 non-test
      files exceed the limit. The gate IS wired (CI `file-size-gate`
      + `nix run .#check-file-size`), red
      since ≈2026-08-08 — unnoticed because red non-required jobs don't block
      direct pushes (F040). DONE: the two worst table-catalog offenders split
      into 12 per-family files (largest 294); feature_profile split (594→3
      files); 2026-09-11: `storage/view/store.go` 358→276+`mapper.go` 83 and
      `stack/bundle.go` 363→253+`lifecycle.go` 117 (behavior-preserving
      same-package splits, build/vet/test green). Wiring the gate into
      `#verify` stays MOOT until the waves land — it would hold every verify
      run permanently red. REMAINING: owner picks the policy — full split vs
      baseline ratchet (no file grows, no new offender) vs
      table-catalog/harness exemptions — then the code-file split waves
      (typed_reader 1127, adttest/harness 953, metaengine/store 935,
      enginetest 935, execute 778, engines 725/722/663,
      architecture/helpers 627, suppression/parser 540, explain 516, …). —
      source: 06-56 §a9/§d1, recounted 2026-09-11 (`nix run
      .#check-file-size`)
      _(Effort: L, multi-session)_

---

## Release / Tagging

> The full 39-tag v4 wave (B1–B7) was cut, pushed, and verify-ci-green on
> 2026-08-29. The SUPERB wave tagged `otel/v4.4.0` + `cmd/cqrs-upgrade/v4.0.0`
> (2026-09-07) and a coordinated release re-tagged 15 modules (2026-09-08).
> Zero local `=> ../` replaces remain EXCEPT `storage/go.mod` (`=> ../encryption`,
> `=> ../snapshot` — the documented unpublished-sibling pattern).

- [ ] 🔥 **Repair the iroh standalone pin break (verify-ci RED risk)** —
      `metaengine/irohengine/loopback/go.mod` pins published `irohengine/v4
      v4.1.0`, which predates graph-op replication; the 2026-09-08
      int-endpoint convergence tests need HEAD. Workspace gates pass (go.work
      resolves the local sibling) but GOWORK=off per-module builds
      (`#verify-ci`, the CI matrix) resolve v4.1.0 and go red. Fix: tag
      `irohengine/v4.2.0` (graph WriteOp convergence + capability conformance)
      and bump loopback/quic pins (quic already carries `replace => ../`), or
      capability-probe skip-guard the two tests (weaker). — source:
      archived 04-35 §b1/§f1
      _(Effort: S/M)_
- [ ] **Cut `stack/sqlite/v4.3.1`** — published `v4.3.0` pins a broken stack
      pseudo-version (`stack/v4 v4.2.1-0.20260807…` whose `sqlopt` needs
      `storage.SQLiteSetSynchronous` — unresolvable standalone). Discovered by
      the cqrs-upgrade smoke; in-tree go.mod already fixed, the TAG still
      serves the broken pin to fresh consumers. — source: SUPERB §a6
      _(Effort: XS)_
- [ ] [BLOCKED] 🔥 **Next v4 tag wave** — substantial unpublished surfaces on
      master: `encryption` (key helpers + envelope v2), `snapshot`
      (`NewRewritingTransformedStore` + wire tags), `storage`
      (`MigrateSnapshotColumnsToStream`, EventSchema re-exports, bytea fix),
      `cmd/cqrs-lint` (working `--fix`, C005, RULES.md, doctor JSON, scorecard
      panel, preset policy), `cmd/api-stability` (sub-package golden),
      `catalog`, `benchkit` (system harness), `metaengine` (planner capability
      partition, record context, `SortPaginate[T]`, planned-table parity,
      matview family) + engines (irohengine v4.2.0 for the pin repair,
      mysqlengine, pgengine, sqliteengine, duckdbengine, dgraphengine
      recalibration, badgerengine — now consumes new `metaengine.SortPaginate`,
      pin bump + replace-strip REQUIRED), `scheduling/sqlstore` (MySQL
      claiming), `system` (v4.7.0: materialized views + the MV recipe's
      UNRELEASED marker flips when tagged). **Strip `storage/go.mod`'s two
      local replaces in the same wave.** Order constraints per CONTRIBUTING
      pre-tag checklist; cut→push→next interleave (GOPRIVATE resolves siblings
      via VCS). — source: 08-26 §c3, 15-09 §f47, SUPERB §f14-16
      _(Effort: M)_
- [ ] **Create GitHub Releases** for the outstanding tags (only storage/v4.7.1
      ever got one). `gh` auth VERIFIED working; script exists
      (`scripts/create-github-releases.sh`) — remaining work is running it per
      tag. — source: 05-00 §f12
      _(Effort: S)_
- [ ] **Consolidate indirect dep references** — the transitive
      `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}/v4` indirect deps
      in ~49 consumer go.mod files clean up after new tags publish. Track and
      verify. _(Effort: M)_
- [ ] **Run `scripts/pin-sweep.sh --check` as a standing post-release step** —
      proven 2026-09-08: a coordinated release that EXCLUDES a module can
      still break that module standalone (`storage` went red; whack-a-mole
      tidy was not a census). Also verify `storage/eventstore` pin health with
      evidence. — source: archived 07-48 §b2/§f2
      _(Effort: S)_
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms (worst-of-30 sample inflates under gate load). Shipped + gated
      green; keep or revisit. _(Effort: XS)_
- [x] **cqrs-bench deprecation stub** — DONE 2026-09-11: one-off stub commit
      on a detached worktree (never merged), tagged `cmd/cqrs-bench/v0.1.1`
      — same treatment as `cmd/cqrs-lint/v0.2.1`: suffix-less go.mod, zero
      requires, loud failure pointing at the `/v4` install command. Verified
      it builds standalone and exits 1. Push `cmd/cqrs-bench/v0.1.1` to make
      it proxy-effective, then `tag-release.sh --smoke cmd/cqrs-bench
      v0.1.1` is meaningless (stub has no proxy version to serve beyond
      @latest) — just `go install …cmd/cqrs-bench@latest` once pushed and
      confirm the stub runs. — source: archived/2026-09-01_21-37 §c1
      _(Effort: XS)_
- [x] **`retract cmd/cqrs-lint/v4.8.0`** — DONE 2026-09-11: the directive
      was sitting unreleased on master (a retract only exists for consumers
      once a TAG carries it); cut `cmd/cqrs-lint/v4.10.1` from a detached
      worktree via `tag-release.sh` (const bumped to 4.10.1 inside the tag,
      standalone build verified, `retract v4.8.0` confirmed in the tagged
      go.mod). Push the tag to make it proxy-effective, then `--smoke`.
      — source: archived/2026-09-01_21-37 §c6
      _(Effort: XS)_
- [x] **tag-release.sh hardening** — DONE 2026-09-11: (a) `--smoke` now
      follows the proxy check with a clean-dir `go install module@version`
      + `--help` run for main-package modules (hard gate on install, warning
      on odd help-exit codes) — the probe class that catches a poisoned
      tag; (b) new `--audit` replays the path-vs-tag guard over every tag
      of every module — one-shot run found 24 historical violations (1078
      tags checked), all in known-dead paths (cmd/cqrs-lint v4.2.0–v4.7.0,
      cmd/cqrs-bench v4.2.0, example/* v3/v4, eventtest v0.x); (c)
      pre-flight runs non-mutating `pin-sweep --check` (advisory — a full
      sweep would mutate 80+ unrelated go.mods, the opposite of this
      script's single-module scoping; the standalone build gate stays the
      hard stop); (d) guard logic extracted into `path_matches_major`,
      shared by release flow and audit; (e) `scripts/test-tag-release.sh`
      pins it all with 10 fixture-repo smoke tests (audit FAIL/OK, guard
      rejection, pre-push --smoke error). — source: archived/2026-09-01_21-37
      §c2/c3/c5, 15-09 §f44
      _(Effort: S/M)_
- [x] **Version-reporting unification** — DECIDED 2026-09-11: buildinfo
      primary, const deleted. `resolvedVersion()` now reports the
      toolchain-embedded version (`go install …@vX.Y.Z` → the real tag),
      falls back to `dev-<sha>[-dirty]` from stamped VCS revision for
      in-repo builds, then `dev` (Nix/ldflags builds append the injected
      commit/date). Removes BOTH drift classes: the stranded const (v4.7.0)
      and the tagger sed that poisoned v4.8.0 — tag-release.sh's bump
      block is gone (cuts no longer mutate source), TestVersionMatchesLatestTag
      retired, bump-cqrs-lint.sh reduced to tidy + vendorHash. On master —
      rides the NEXT cqrs-lint tag (v4.10.1 was deliberately cut from the
      pre-refactor HEAD so the retract could ship immediately). — source: archived/2026-09-01_21-37 §c4
      _(Effort: S)_
- [x] **cqrs-upgrade growth** — DONE (staged across sessions): `--json`,
      `--workspace`, `--to`, `--strict`, and x/mod-based in-process go.mod
      editing all shipped earlier (see CHANGELOG 2026-09-07); the last
      piece — the self-upgrade CI dogfood job — landed 2026-09-11 as a
      nightly `sentinel.yml` job running `cqrs-upgrade --workspace
      --dry-run --strict` over this repo (v5-readiness gate: repo itself
      is v5-clean; the tool exercises its full pipeline against 84 real
      modules). — source: SUPERB §f18-23
      _(Effort: S/M each)_
- [x] **Badger data-loss exposure review** — RESOLVED 2026-09-11, not
      pre-adoption: badgerengine published v4.0.0–v4.2.0. Window bounded:
      v4.0.0–v4.1.0 seeded ONLY the log counter on restart (v4.1.0's
      comment already claimed all four — a lying comment), so reopen+
      append overwrote early stream/journal entries; v4.2.0 (2026-09-08)
      shipped full seeding. Consumer audit: zero repo-internal data-path
      consumers (analyzer catalogs only); external adoption unlikely
      (five weeks old, niche backend) but unprovable. Action: v4.0.0–
      v4.1.0 retracted with reason comment; `metaengine/badgerengine/
      v4.2.1` tagged (code identical to v4.2.0) to publish the retract;
      full retrospective in the ADR-0118 incident addendum. Push the tag
      to make it proxy-effective. — source: 15-09 §g1
      _(Effort: S)_

---

## Metaengine — follow-ups

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

- [x] **DSN secret-redaction audit for sibling engines** — DONE 2026-09-08
      (Pareto P10): pg/mysql audited (no DSN echo — plain `%w` wraps only);
      turso's `redactDSN` matcher fixed (exact-spelling matching leaked
      `auth_token`/`AUTH_TOKEN`; now containment on `token`/`key`) and pinned
      by adversarial per-shape leak tests + a preserves-non-secrets guard.
      Strict-vs-lenient typo'd-param rejection stays BLOCKED (owner ruling).
      — source: archived 20-18 §f25-26, 20-57 §f22-23/§f41
- [ ] **Turso encryption test breadth + reachability docs** — (a) document the
      DriverConfig/`system` reachability gap for `WithEncryption` in
      tursoengine README + FAQ (direct `New()` only today); (b) live `file:`
      DSN round-trip; (c) second-ADT round-trip (Counter or journal) under
      encryption; (d) matview aggregation actually SERVING from a view on an
      encrypted engine (coexistence test constructs but does not query); (e)
      cipher-size hint exact-text pin. — source: archived 20-57 §b1/§b3/§f1-7
      _(Effort: S/M)_
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
- [x] ~~**ClaimMetrics surfacing** — claim metrics hooks shipped 08-30 with zero
      consumers; surface `ClaimMetrics` in Doctor or a status endpoint. —
      source: archived 22-33 §f14~~ DONE 2026-09-11 — `ClaimingTimerStore`
      now maintains the counters itself (atomics on every committed Due poll
      and RenewLease, hooks unchanged) and exposes a JSON-ready
      `Metrics()` snapshot (`ClaimMetricsSnapshot`: batches incl. empty
      polls as heartbeat / timers / renewed / rejections) — a Doctor-style
      report or `/status` endpoint reads claim liveness with zero wiring.
      Pinned by `TestClaimingSQLite_MetricsSnapshot`; api golden regenerated.
- [x] ~~**Fold `BenchmarkCalibration_DgraphSearchQuery` results into the
      calibration baseline doc** (`docs/benchmarks/calibration-2026-08-30.md`
      protocol/recalibration sections; bench exists, doc not updated; also
      record the ADTMap=O1 decision there). — source: archived 22-33 §f11~~
      DONE 2026-09-11 — live run on ephemeral Dgraph 25.4.0 (count=3,
      benchtime=20x, load ~5): ~838µs/3.2ms/13.1ms at 100/1K/10K docs;
      marginal slope ~1_093 ns/row at scale (server-side anyofterms is the
      CHEAPER read vs client-filtered MapScan). Constants unchanged
      (`NsPerScan=2_200` mid-band prices ReadFullTextSearch; dedicated
      search-cost field deferred). Folded as "Dgraph SearchQuery baseline
      (2026-09-11)" + "ADTMap complexity decision (2026-09-07)" sections in
      the baseline doc.
- [x] ~~**Demote catch-up Record-context completeness** — verify Demote's
      catch-up path passes full records to record-aware folds (07-43 §f36).
      — source: archived 22-33 §f26~~ DONE 2026-09-11 — verification found a
      REAL gap: the re-routed leg (`applyReplay`) honored `EventInput.Record`
      but the mirror leg (`replayToShadow`) always synthesized a Type-only
      record, dropping StreamID/Version on the demoted engine. Fixed to pass
      the recorded record through (synthesize only for legacy log entries);
      pinned by `TestDemoteEngine_RecordContextReplay` (fails pre-fix with
      partial context on the mirror leg).
- [x] ~~**enginetest fakes contract note** — document next to
      `RunCapabilityConformance`: fake engines must satisfy
      `engineServesADTNatively` for every declared ADT (the honestMapMixin /
      nativeMapEngine precedent). — source: 04-35 §e5/§f20~~ DONE 2026-09-11
      — "Contract for FAKE engines" paragraph added to the
      `adttest.RunCapabilityConformance` doc (metaengine/adttest/conformance.go):
      implement the declared ADT's backend interface or declare it in
      DegradedADTs, with both precedents named.

---

## CI / Infrastructure

- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s; broken since ~2026-07-17. Local `nix run .#verify` remains the
      authoritative gate. _(Effort: S, user action)_
- [ ] [BLOCKED] **cqrs-lint Self-Lint credentials** — go-finding fetch fails
      under GOWORK=off (`git ls-remote` exit 128). _(Effort: S, user/creds)_
- [ ] **First post-push CI run triage** — the ~80-job per-module matrix first
      executed 2026-09-06 (master pushed + green); expect new failure classes
      (flaky tests, module-specific env) on later pushes. — source:
      archived/2026-09-04 §c6
      _(Effort: M)_
- [ ] **Calibration-drift gate redesign** — compare against a persisted
      CI-baseline artifact instead of absolute constants; nightly >100% rows
      are shared-runner noise. Add TMPDIR-filesystem detection (refuse to run
      on CoW). — source: archived/2026-09-04 §b2/§f16/§f18
      _(Effort: M)_
- [ ] **Fresh-GOMODCACHE go.sum check in CI** — the 8-module go.sum rot class
      (integration gates fail from a cold module cache) should die in CI once,
      not per future session; root-cause the holes (tidy-under-warm-cache
      suspect). Pair with a GOWORK=off standalone build matrix sweep. —
      source: 08-41 §b5/§e6, 08-26 §f4
      _(Effort: M)_
- [ ] **Wire `check-csp` into CI** (nix chromium, no npm network) and decide
      `check-eventcatalog` placement (needs npm — nightly candidate; commit a
      `package-lock.json` from the exporter first). — source: 08-26 §c1/§f7/§f8
      _(Effort: S/M)_
- [ ] **pin-sweep `--check` nag semantics** — the module-layers CI leg goes
      red on every push between a tag push and the follow-up sweep commit (by
      design). Keep blocking-on-every-push or move to tag-push/cron triggers?
      (15-09 §g2). Extras: `--dry-run`, `--remote` sanity, unit harness. —
      source: 15-09 §e9/§f18–22
      _(Effort: S)_
- [ ] **Cheap CI gates into pre-commit** — module-layers, version-drift,
      workspace-sync, replace-directives are plain bash; wire staged-aware
      into the hook. — source: archived/2026-09-04 §e6
      _(Effort: S)_
- [ ] **"Days-since-green" metric/alert** — 6-week red droughts normalized
      drift; a Gatus-style freshness check catches the class in days. Related:
      nightly "all CI jobs green or annotated" sentinel. — source:
      archived/2026-09-04 §e5, 06-56 §e1
      _(Effort: S)_
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_
- [ ] **`check-coverage.sh` nix wrapper runs without the cache env** and
      reports 0.0% DRIFT vacuously — make the app export the env itself or
      fail loudly. Also run it once for the 2026-09-07/08 waves (matview +
      hardening batches were never coverage-checked). — source:
      archived/2026-08-30_06-34 §f, 19-25 §f4, 05-31 §f11
      _(Effort: S)_
- [ ] **actionlint CI step** (exists in devShell since T37) + extend
      shfmt-drift job with shellcheck for `scripts/`. — source: 15-09 §f28/§f29
      _(Effort: S)_
- [ ] **GOWORK-mode decision table in AGENTS.md** — which gate runs workspace
      vs per-module; which modules resolve siblings via replace vs published
      pins; how loopback/quic/VM/protocol-bench runs differ (the recurring
      GOWORK foot-gun, most recently the iroh verify-ci break). — source:
      04-35 §e7/§f8
      _(Effort: S)_

---

## Code Quality

- [ ] **>350-line production files (~54, 2026-09-06 count)** — see the
      cqrs-lint section for the verified picture, gate-policy options, and the
      already-split offenders; the code-file split waves are a standalone
      multi-session program pending the policy decision. Decide
      harness-dir exemptions (adttest/enginetest are exported test harnesses)
      first. _(Effort: XL, multi-session)_
- [x] **Attribute + resolve the 5 pending clone groups** (check-duplication,
      verified foreign at 15-09, owners landed since): DONE 2026-09-09 —
      cqrs-lint `fix.go` ×2 killed at the root by extracting the shared
      `staleByFile`/`finalizeFixResult` helpers (commit cec9248da);
      the surviving duckdb+sqlite `planned_parity` sort clones (the reported
      "pg" member dissolved — pgengine never had a planned_parity file; that
      clone group vanished when the owner's work landed) and the
      csp_browser_test ↔ store_collaborators pair were attributed
      `//art-dupl:accept` with domain rationale (dep-isolated
      cross-engine pattern; unrelated mutex-guard idioms). Gate re-verified
      green 2026-09-11: 0 new clone groups (baseline 54). — source: 15-09
      §b2/§f3
      _(Effort: S)_
- [x] **Pre-existing scheduling/sqlstore lint findings** — DONE, surface
      re-verified clean 2026-09-11: gocognit fixed by extracting the
      `pollAssertingLeaseHeld`/`reclaimOnce` helpers from
      `TestClaimingPostgres_RenewVsClaimRace`; gosec G202 attributed
      `//nolint:gosec // placeholders only, ids bound` (claiming_mysql.go);
      lint with the integration tag AND canonical `#lint-module` both 0
      issues (gocognit/gosec/sqlclosecheck/staticcheck/wsl_v5 explicitly
      enabled). On-sight repair: `scheduling/sqlstore/go.sum` was missing the
      pgx v5.11.0 go.mod hash — GOWORK=off integration-tag lint/build failed
      standalone until added. — source: 15-09 §c2/§f4, archived 22-33
      addendum, 04-35 §f7
      _(Effort: S)_
- [x] **`aggregate_*` family-code tripwire test** — DONE 2026-09-09, landed
      as `cmd/api-stability/aggregate_code_tripwire_test.go`: exact-string
      table of all 17 renamed codes, walks every repo `.go` file (skips own
      table), fails with file:line pointers. Mutation-verified 2026-09-11:
      planting `event.aggregate_not_found` turns the test red, removal
      restores green; runs in CI via the cmd/api-stability `-race` test leg.
      Deliberately exact-string (not a broad `aggregate_` grep) so
      `listing.aggregate_projection` and SQL column names don't false-fire.
      — source: archived 07-48 §f4
      _(Effort: XS)_
- [x] 🔥 **json/v2 map-order determinism sweep** — DONE 2026-09-08 (Pareto
      P08): SARIF `run.properties` map → fixed-order struct + 50-render
      byte-compare pin; every consumer-visible `json.Marshal` in cqrs-lint
      passes `json.Deterministic(true)`; catalog/asyncapi already used
      Deterministic. — source: docs-health pass 2026-09-08 (e-6)
- [x] **`example/metaengine-quickstart/README.md`** — DONE 2026-09-09
      (6bb82f5b), re-verified 2026-09-11: README authored from the four demo
      sections (maps/graph/vector/cqrs.yaml); class pinned mechanically by
      `TestEveryExampleHasREADME` (cmd/api-stability, passes); example runs
      all 4 sections green. — source: 07-42 §b2/§f28
      _(Effort: M)_
- [x] **Example v5-policy audit** — DONE 2026-09-09, re-verified 2026-09-11:
      taskmanager + metaengine-quickstart audited via
      `cqrs-upgrade -dry-run -strict -no-build` — no v5-removed API usage,
      exit 0, all pins up-to-date (re-run confirms). — source: 07-42 §f8
      _(Effort: M)_
- [ ] [BLOCKED] **macOS verification of ephemeral PG** —
      `scripts/ephemeral-pg.sh` claims cross-platform but was only
      static-review-tested; a GitHub Actions macOS runner leg is the
      verification route. _(Effort: M)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) —
      userspace MariaDB coverage exists but not the full nspawn env. Now also
      covers the MySQL claiming integration tests. _(Effort: M)_
- [x] **Evaluate `-shuffle=on` for the dgraph suite specifically** — DONE
      2026-09-11: seed 42 exposed two real contention gaps (unguarded
      mutation aborts + schema-Alter "Pending transactions" rejections
      that silently skipped ADT subtests); fixed at the execution layer
      (dgraphengine retry-on-contention), then ADOPTED and rolled into
      the ephemeral-pg/dgraph/redis + vm-mysql/nspawn invocations.
      Post-fix: green on seeds 42/7/1234 + the default invocation.
      _(Effort: S)_
- [ ] **Contention-retry observability** — `retryOnContention` retries
      SILENTLY (correct for tests, hides production Alpha contention
      storms). Add an otel counter (e.g. `cqrs.dgraph.contention_retry`)
      via the `otel/` re-export module; NOTE: adding otel/ to dgraphengine's
      go.mod needs a `check-arch` dep-budget review first. — source: 02-16
      §e2/§f1
      _(Effort: S)_
- [ ] **Skip-vs-fail policy for live conformance engine construction** —
      `newDgraphEngineOrSkip` turns ANY construction failure into a SKIP
      (how 4 ADT subtests silently vanished pre-fix). Distinguish
      server-unavailable (skip) from contention-after-retry-exhaustion
      (fail loudly). Pick the policy (ROADMAP OQ #10), then implement. —
      source: 02-16 §e3/§f9/§f21
      _(Effort: S)_
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
- [ ] **Test `ensureEdgeSchema`'s in-tx Alter path** — Alter retries even
      inside RunInTx (txnScoped=false) are pinned only by inspection; add
      the unit pin next to `transaction_retry_test.go`. — source: 02-16 §f35
      _(Effort: S)_
- [ ] **Review `doWrite` response-returning callers** — `doMutate` was
      narrowed to error-only (response never consumed); check whether any
      `doWrite` caller consumes the response, and narrow the rest for
      symmetry. — source: 02-16 §f34
      _(Effort: XS)_
- [ ] **Modernize dgraphengine test-modernize hints** — 13× `b.Loop()` in
      `bench_test.go` + `atomic.Uint64` in `helper_test.go` (gopls hints;
      lint-clean today, 10-minute sweep). — source: 02-16 §e8/§f16
      _(Effort: S)_
- [ ] **`go mod tidy` in `integration/`** — gopls flags unused
      `google.golang.org/genproto/googleapis/rpc` (integration/go.mod:131).
      Mind parallel-session in-flight edits before sweeping. — source: 02-16
      §f24
      _(Effort: XS)_
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
- [ ] **Record shuffle seeds to a log for post-hoc replay** — ephemeral
      scripts echo the seed; persist it to a file so a failed CI seed can be
      replayed exactly (`-shuffle=N`). — source: 02-16 §f23
      _(Effort: XS)_
- [ ] [BLOCKED] **Full `nix run .#verify` gate for the contention fix** —
      blocked while a parallel session's files sit dirty in the tree
      (#verify exclusivity + `nix fmt` fail-on-change); dgraphengine and
      stack verified green per-module meanwhile (build/vet/test/lint). —
      source: 02-16 §c4/§f13
      _(Effort: M)_
- [ ] 🔥 **CI triage: master red across ~15+ jobs, no green run in the last
      30.** Classified 2026-09-11 (run 34548534824): (a) FIXED same-day —
      the Module-matrix go.sum class (missing `/go.mod` hashes after the
      v4.5/v4.6 pin wave: badgerengine, mysqlengine, projectionhost,
      stack/bench, stack/postgres, testutil/pgtestcontainer, plus
      idempotency/sqlstore found live on the mysql-vm leg); detection +
      repair recipe in gotchas-module-management.md. (b) REMAINING, undiagnosed:
      FlakeHub auth errors in job logs despite `use-flakehub: false`
      (possibly fatal in the ephemeral dgraph/pg/redis integration jobs,
      which are green locally), shellcheck SC2086 in
      `scripts/test-tag-release.sh` (`git $notag` is INTENTIONALLY unquoted
      — quoting changes semantics; needs a disable directive or
      restructure), Minimum Coverage, verify-fast, go.work sync check, Nix
      Flake Check, CGo build, Security Scan. (c) KNOWN/accepted: File Size
      Check (the split-waves policy item above). NOTE: failures predate
      2026-09-11 (they exist on commit 82d5218fc, before that day's
      sessions). — source: run 34548534824, `gh run list`
      _(Effort: M-L, multi-session)_

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
- [ ] **v5 ADR: encryption-at-rest configuration** — `metaengine.DriverConfig.Encryption`
      + `KeyProvider func(ctx) ([]byte, error)` (vs raw `key []byte` — the
      KeyProvider lean enables rotation/hot-reload and keeps keys out of
      config structs; sets THE precedent for pg/mysql passwords too) +
      `system/` DeploymentConfig key-reference slot (env/file/secret-manager
      ref, never the key). Engines fail construction loudly when unable to
      honor (precedent: `RejectDurabilityTier`, `MaterializedViews`). —
      source: 20-18 §f15-17/§f21, 20-57 §f9-10
      _(Effort: L)_
- [ ] **v6 deletion markers:** snapshot wire fallback shims + pebble
      legacy-row support window get a ROADMAP-visible deadline marker (one
      release cycle after v5) so the deletion wave can grep for it. — source:
      08-41 §f10/§f50
      _(Effort: XS)_
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
> + [execution plan](docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md).
> Owner decision 2026-08-22 (Appendix B): string `record.StreamRef` SURVIVES
> v5 with a validating constructor; the struct `record.Stream` proposal is
> rejected.

- [ ] **T23 — upstream skill-maintenance pass** (the plan's one open task):
      docs/reviews↔brainstorming divergence; read-prior-reports +
      copy-template steps in the review skills. Execute or decline at the next
      skill-maintenance window. _(Effort: S)_

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [x] **Skill-reference propagation wave** — DONE 2026-09-11: rotation
      write-back recipe landed (recipes.md §2.31: RotatingSnapshotStateCodec
      lazy path + manual rewrite path; §2.7 cross-references it); MySQL
      claiming matrix FIXED (§2.26 was stale — NewClaimingMySQLStore works on
      MySQL 8.0+/MariaDB 10.6+ via SKIP LOCKED, verified in
      scheduling/sqlstore/claiming_mysql.go); planned-table capability roster
      added (§2.27: pg/mysql/sqlite/duckdb all implement
      Apply/Evolve/PlannedTables, backfill pg+mysql only); doctor
      record-context + planned-tables sections, `--format json`, check-csp/
      check-eventcatalog, CALIB_DUMP, and the pre-v5 snapshot decode recipe
      (§2.30) were already propagated (verified present). doc-check green.
      — source: 07-43 §c4, 08-26 §b1, 08-41 §b6/§f24
      _(Effort: M)_
- [x] **encryption module docs** — DONE 2026-09-11 (mostly already shipped):
      wire-format goldens existed (`envelope_wire_golden_test.go` ×3) and the
      v1↔v2 decode-symmetry property test existed
      (`envelope_symmetry_test.go`); README gained the Loading & Validation
      section (LoadKeyFromEnv/LoadKeyFromFile/ValidateKey/Encode/DecodeKeyBase64
      + error-surface contract), doc.go gained Key Management Helpers,
      Envelope Format (v2), and Snapshot-State Key Rotation Write-Back
      sections. Module build + golden/symmetry tests green.
      — source: 08-26 §b9/§f11–13
      _(Effort: S)_
- [x] **`awaitAck`/`replayPhase` lying log line** — DONE (stale item, fix
      already shipped): `awaitAck` returns a distinct `ackInterrupted` outcome
      for ctx-cancel/Close (catchup_subscriber.go:282-285) and replayPhase
      reports the shutdown, not a nack (catchup_replay.go:91-94, "NOT a
      consumer nack"); pinned by TestCatchUpSubscriber_CloseWhileBlockedOnAck.
      — source: 07-42 §c2
      _(Effort: XS)_
- [x] **Benchmark auto-discovery check** — DONE 2026-09-11 (answer: no
      auto-discovery): the gate set is an explicit allowlist
      (`BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$`,
      `$`-anchored), so the load-sensitive watermill
      `BenchmarkCatchUp_ReplayThroughput` never runs in the gate; a comment
      in benchmark-regression.sh now documents the deliberate no-auto-discovery
      decision so it isn't "widened" later.
      — source: 07-42 §e4/§f2
      _(Effort: S)_
- [x] 🔥 **Benchkit full-suite flake hunt** — DONE 2026-09-11, root cause
      found + residual closed: the big window was setup burning the caller ctx
      before any phase (fixed upstream by the `benchkit.not_started` entry
      guard + load-scaled budgets in mustRun); the remaining TOCTOU — ctx
      expiring after the guard with zero work done, every phase silently
      skipping, Run returning `(partial, nil)` — is now closed: runPhases
      fails loudly with `benchkit.expired` when a caller-bound (non-Duration)
      ctx expires with `TotalEvents == 0` (Duration-bounded runs keep
      graceful-partial semantics). system_test outer deadline raised 30s →
      90s×load-factor (two 15s inner budgets + close/reopen need headroom).
      Verified: ClosedStore/ExpiredContext + ResetProjection_RestartAndReplay
      green under 64-way CPU-soak load 36-52 with -race.
      — source: SUPERB §d2/§f11, docs-pass verify runs 2026-09-08
      _(Effort: M)_
- [x] **Watermill catch-up tail:** DONE 2026-09-11. (1) Restart-recovery
      property test landed: TestCatchUpSubscriber_RestartRecoversSkewSuppressedEvent
      mints a zero-timestamp-ULID event appended AFTER the watermark, proves
      live suppression, then proves the restart re-delivers it (journal-order
      ReadFrom) — pins the documented self-healing claim exactly.
      (2) Broker-backed throughput variant landed:
      TestRedisStream_CatchUpReplayThroughput (real Redis Streams live side,
      1000 events, order+handoff pinned, throughput logged — 318k events/sec
      locally; run via `nix run .#integration-redis`). (3)
      CloseWhileBlockedOnFullBuffer RETIRED as unreachable fiction: replay
      forwards are serialized with awaitAck, so the 256-slot output buffer can
      never fill; replaced by TestCatchUpSubscriber_CloseWhileReplayParkedInJournal
      (gating journal, deterministic park inside ReadFrom, no sleep).
      — source: 07-42 §f11/§f17/§f18
      _(Effort: M)_
- [x] **README/AGENTS quick-reference rows** — DONE (stale item): AGENTS.md
      quick-reference already had both rows (`#check-csp`,
      `#check-eventcatalog`); README is the consumer sales page and has no
      internal flake-app table by design; the tooling surface is documented
      for contributors in references/advanced.md §7.
      — source: 08-26 §c2
      _(Effort: XS)_
- [x] **Social preview image + homepage URL** — DONE 2026-09-11 (automatable
      parts): homepage set to https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite
      via `gh repo edit` (the item's "CLI can't set them" was stale for the
      homepage); branded 1280×640 asset generated at
      `docs/assets/social-preview.{svg,png}`. The social-preview UPLOAD is
      UI-only (the undocumented API endpoint returns 404): owner pastes
      docs/assets/social-preview.png at repo Settings → Social preview.
      — source: 07-42 §b1
      _(Effort: S, manual)_
- [x] **doc-check tail:** DONE 2026-09-11 — `--json` (deterministic wire
      shape incl. `ambiguities` array) and the no-import-alias-maps-to-
      multiple-packages warning were already shipped (main.go jsonSummary,
      resolve.go verifyBlocks); the scoped `#doc-check` flake app landed
      (same corpus as the #verify leg). Release posture DECIDED: ship the
      strict block-scoped resolver as-is at the next cmd/doc-check tag — no
      `--legacy-union` flag (internal-grade tool, stricter = fewer false
      passes, transition flag would be permanent maintenance for a tiny
      audience).
      — source: 15-09 §f13–16
      _(Effort: S/M)_
- [x] **exhaustruct_v5 canary test** — DONE 2026-09-11:
      `scripts/test-exhaustruct-canary.sh` (wired into `#check-lint-config`)
      proves every ignore-pattern entry is present and its target type still
      exists (stack in-repo, bbolt via module cache), then behaviorally: a
      hermetic fixture with os/exec.Cmd ignored + an un-ignored control
      struct is flagged ONLY on the control with patterns and BOTH without —
      proving v5 full-name pattern matching under the installed golangci-lint
      (2.13.2). The deprecated-linter-name golden already existed
      (`scripts/check-linter-names.sh`, 109 names checked, green).
      — source: 15-09 §c3/§f5/§f30
      _(Effort: S)_
- [x] **templ tripwire script** — DONE (stale item):
      `scripts/check-templ-paths.sh` already existed and is wired as
      `#check-templ` (codegen drift + FileName cwd tripwire) and inside both
      #verify chains; repo-wide `_templ.go` scan via find; ran green
      (FileName values all cwd-clean).
      — source: 15-09 §f25/§f26
      _(Effort: S)_
- [x] **AGENTS.md indexed-split** — DONE 2026-09-08 (Pareto P15+P16): 92 KB →
      28 KB index + `docs/agents/gotchas-{tooling-build,module-management,
      language-footguns,testing}.md` + `gowork-modes.md` (THE decision table)
      + `module-map.md`; zero-content-loss verified by bullet/row counts.
      — source: 15-09 §f39, evening-pass §f46
- [x] **error-taxonomy.md completeness check** — DONE 2026-09-11: pebble
      verified accurate (4 sentinels + family split all match source); the
      watermill table had a LIE ("Metadata parse fails → Corruption" — every
      `watermill.parse_*` site is Rejection, verified per-site) now corrected,
      plus added the missing rows: malformed-metadata Corruption
      (corrupt_metadata, create/convert_event_failed), catch-up
      checkpoint/replay Infrastructure, and subscribe/publish/lifecycle
      Infrastructure codes.
      — source: archived 07-48 §f23
      _(Effort: S)_
- [x] **DOMAIN_LANGUAGE.md entries** — DONE 2026-09-11: added
      Materialized-View Acceleration, IVM, and View-Maintained Write rows to
      the Metaengine table (ADR-0135-grounded), plus the two-model encryption
      entry (At-Rest Encryption vs payload AEAD, tursoengine
      `WithEncryption`, Cloud-BYOK boundary) in Security — at-rest encryption
      IS a domain concept since the 2026-09-07 turso wave.
      — source: archived 19-25 §f42, 20-57 §f31
      _(Effort: XS)_

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
- **KeyProvider tier (env/file composite provider)** — deferred to ROADMAP;
  the bank-sync ask is closed by the shipped helpers. — source: 08-26 §f14
