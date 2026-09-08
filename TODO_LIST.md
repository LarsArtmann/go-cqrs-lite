# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 morning, and 2026-09-06 afternoon).

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
> and the status report
> `docs/status/2026-09-07_19-25_turso-materialized-views-operator-option.md`).

- [BLOCKED] 🔥 **File the standalone upstream issue for the silent wrong-results bugs (defects A+B)** — grouped views diverge from the second transaction on and collapse at ~27k rows; draft is ready and fully verified in `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (everything below its first `---`). Blocked on user approval (external action). The COMMIT-abort half (defect C) is already reported: PR #8257 comment https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646. _(Effort: XS once approved)_
- [ ] 🔥 **Push the 3 unpushed commits, then edit the PR comment link to SHA `18b2c495c`** — the posted comment's permalink points at `1c9f3bf` (last pushed SHA); the newer draft revision contains the full three-defect characterization. Comments are editable. Pushing also publishes the divergence research. _(Effort: XS)_
- [ ] **Track turso-go releases for the IVM fixes** — re-run the repro suite (`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`) on each new release; when green, remove the grouped-view warnings (Doctor WARN line, recipes §2.29 notice, AGENTS.md gotcha, bench doc warning) and un-skip the ≥10k matview bench cases. _(Effort: S per check)_
- [ ] **Code guard follow-up: make grouped-spec safety mechanical** — today the danger is advisory-only (Doctor WARN + docs). Options: `MaterializedViewSpec` validation refusing `GroupBy` on turso-go ≤ v0.8.0-pre.8 (breaking for legitimate small deployments) vs a config flag (`AllowGroupedViews`) vs silent status. Decide + implement once the upstream timeline is known. _(Effort: S)_
- [ ] **Add a regression test pinning the Doctor WARN line** for grouped specs (`metaengine/materialized_view_doctor.go`) — the warning was added 2026-09-07 without a dedicated test. _(Effort: XS)_
- [ ] **Tag wave for the matview feature** — metaengine/sqliteengine/tursoengine/system carry sibling replaces for unpublished symbols (`MaterializedViewSpec` family); pins must be bumped and replaces stripped at the next release wave so consumers can use the feature from published tags. _(Effort: M — see AGENTS.md tag-wave procedure)_

---

## cqrs-lint

> Point-in-time execution plan (T01–T24 / F001–F096) with per-row resolution
> markers: `docs/planning/archived/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`.
> T01–T12, T22–T24 and their fine tasks are resolved; this section carries the
> living remainder.

- [ ] **T13–T19 — exhaustive rule audit batches.** RISK-BASED SAMPLE DONE
      2026-09-06 (all C-family detectors read; C003 + C005 real bugs found and
      fixed; top-volume findings sampled zero-false; severity/confidence and
      V007 coverage mechanized by meta-tests). S-FAMILY BATCH DONE 2026-09-07
      (see addendum in
      `docs/status/2026-09-07_cqrs-lint-t20-t21-subsystem-reviews.md`:
      financialEscalatedRules comment drift + missing S011 escalation fixed,
      completeness meta-test added). REMAINING (explicitly low-yield, only
      behind a green full gate): per-file checklist audits of A001–A034,
      B001–B031, D001–D019, E001–E017, T/V/F families. — source:
      archived/2026-09-06_02-40 §c
      _(Effort: M/L)_
- [x] **T20–T21 — subsystem reviews.** DONE: feature_profile split (F073),
      scorecard deprecated panel (F082), health-policy test (F083), sibling
      link checks + docserver-CSS root cause (T24), and the line-by-line
      reviews of scanner*/feature_detect*/loader/registry/module_catalog*/
      upcaster (T20) and doctor/health/scorecard/output/explain CLI files
      (T21) — report:
      `docs/status/2026-09-07_cqrs-lint-t20-t21-subsystem-reviews.md`
      (no correctness bugs; 4 fixed in-pass — doctor severity-overrides
      rendering, 2 map-order nondeterminism bugs, Monetary override check;
      T20-1 store-detection gap + accepted heuristics recorded there).
      — source: archived/2026-09-06_02-40 §c + 06-58 §b3
- [x] **F089 — `rules.severity-overrides` + `v5-ready` preset** — DONE
      2026-09-07: `RulesConfig.SeverityOverrides` with catalog precedence
      (catalog → preset → parent → config → domain bias → min-severity),
      invalid severities dropped with a warning (never silently demoted to
      info), unknown rule IDs warned post-merge; `v5-ready` preset = sugar
      for `{"V007": "error"}`; init template, explain, doctor text+JSON,
      README preset table (test-locked) all wired. Design:
      `docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md`.
- [x] **F090(a) — dot-import flagging in V007** — DONE 2026-09-07: any
      dot-import of a go-cqrs-lite module fires V007 at the import position
      ("hides v5-removed-API usage — name the import"); non-CQRS dot-imports
      stay silent; fixture tests pin fire + silence. F090(b) (type-based
      attribution) is now implementable on the F091 Tier 1 machinery.
- [x] **F091 Tier 1 — typed qualifier resolution** — DONE 2026-09-07:
      `analyzer.ResolveQualifierTyped` (Uses[ident] → PkgName → import path)
      with authoritative semantics — typed "not a package" is final, string
      scan only on missing type info (broken-build survival). V007 adopted;
      the shadow false-positive class is dead (verified end-to-end: old
      binary double-fires a shadowed qualifier, new fires once). Wall-time
      gate PASSED (medians old 14.7s vs new 12.4s at load 65; loader has
      shipped NeedTypes since day one — the design's load-cost concern was
      already paid): `docs/benchmarks/2026-09-07_cqrs-lint-f091-tier1-typed-qualifier.md`.
      REMAINING (Tiers 2–3): C008 usage-confirmation + C035/C013
      payload-shape confirmation behind `--typed-info=auto`. The
      three-helper adoption (T20-8) is DONE 2026-09-08 — see the
      hardening-batch entry below.
- [x] **Self-lint finding delta after the collector unification** — DONE
      2026-09-07: full self-lint run = ZERO findings; the delta was one STALE
      C025 suppression at `cmd/cqrs-lint/run.go` (left over from the
      `31b779b1b` refactor that added `%w` to the stale-suppressions error —
      C025 correctly stopped firing). Directive removed; the CI
      `cqrs-lint-self-lint` job was RED on master and is green again.
      C038/C040: no findings on the repo corpus, nothing to triage.
- [x] **go-finding upstream issues (verify-before-filing first):** DONE
      2026-09-07 — both claims verified at source level
      (`pipeline/fix_engine.go:59-92` zero-edits-success-shape;
      `pipeline/fix_applier.go:140-167` RollbackAll(modified) spans earlier
      files) and filed: [#27](https://github.com/LarsArtmann/go-finding/issues/27)
      (zero edits indistinguishable from success),
      [#28](https://github.com/LarsArtmann/go-finding/issues/28)
      (resolveError rolls back all applied edits).
- [x] **Structural load-robustness for the two known flaky test classes** —
      DONE 2026-09-07: `benchkit.loadScaledCeiling` (3 hang ceilings scaled by
      ambient load1/GOMAXPROCS, cap 8x, go-test timeout stays the structural
      backstop) + `system_test.loadScaledDeadline` (catch-up poll deadlines
      8s/5s/15s scaled the same way). Full benchkit (152s) + system suites
      green under load 65. Both helpers are local copies (dep budgets) with
      `//art-dupl:accept`.
- [x] **T20 follow-up hardening batch** — DONE 2026-09-08 (gates: full
      cqrs-lint suite green, lint 0 issues, self-lint exit 0, api golden
      6735 exports):
      (1) store resolution deterministic — Pass 1 iterates packages and
      imports in sorted order with a first-wins guard (T20-3; pinned by a
      40-run stability test that fails on map-order code);
      (2) every shipped metaengine engine maps to a StoreKind — new
      `StoreBadger`/`StoreDgraph`/`StoreIroh` constants, engine→store
      table test (T20-1);
      (3) constructor-call handlers recorded in `analyzer.ConstructorHandlers`
      instead of call-text keys in CommandTypesRegistered (T20-4);
      (4) `CommandInfo.Embeds` split from Fields; B004 count semantics
      preserved (T20-5);
      (5) typed qualifier adoption shipped: `analyzer.IsQualifierFor` +
      `analyzer.IsEventTypeParam` — alias-fold-blindness for C038/C040 is
      dead; converted `capturePayloadTypeFromVar`, fold detection, and
      `IsInsideUpcasterClosure` (T20-8);
      (6) upcaster-closure ranges memoized per file — O(file) once per FILE,
      not per candidate call (T20-7);
      (7) doctor JSON fixed: `FeatureProfile` now carries json tags (the
      surface emitted Go-style keys), `modules` sorted by dir (was map
      order), shape golden locked (`TestDoctorJSONReport_Golden`,
      regen with UPDATE_GOLDEN=1);
      (8) RULES.md-vs-formatter ROOT CAUSE corrected: dprint's markdown
      plugin (NOT treefmt — it has no md formatter) re-pads the generator's
      tables; `**/RULES.md` excluded in dprint.json and the file
      regenerated — freshness test is stable now;
      (9) duplication gate: benchkit/system `//art-dupl:accept` placements
      VERIFIED live; the one new doctor.go idiom clone eliminated via
      slices.Sorted(maps.Keys); foreign post-baseline clone groups
      (per-engine calibration dumps, restart-safety tests, committed
      2026-09-08) absorbed by a baseline re-pin — 133 groups, gate green.
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
- [ ] **cqrs-lint rule: ApplyLayout on engines that also implement
      LayoutPlanApplier → prefer the plan path.** DESIGN PASS DONE
      2026-09-07 (appendix in
      `docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md`): selected
      structural method-shape detection (`ApplyLayoutPlan` +
      `BuildLayoutPlan` co-occurring on the receiver type — no metaengine
      import needed, no registry split-brain); fires behind F091 Tier-2
      `--typed-info=auto`, silent on the name-only fallback. REMAINING:
      implementation + fixtures (engine implementing both paths fires;
      plan-only and ApplyLayout-only stay silent). — source: session-4 retro
      §f25
      _(Effort: M — implementation now unblocked)_
- [ ] 🔥 **350-line limit: gate red repo-wide — split waves + gate-policy
      decision.** VERIFIED 2026-09-06: the gate IS wired (CI `file-size-gate`
      + `nix run .#check-file-size`) but ~54 non-test files exceed it, red
      since ≈2026-08-08 — unnoticed because red non-required jobs don't block
      direct pushes (F040). DONE: the two worst table-catalog offenders split
      into 12 per-family files (largest 294); feature_profile split (594→3
      files). REMAINING: owner picks the policy — full split vs baseline
      ratchet (no file grows, no new offender) vs table-catalog/harness
      exemptions — then the code-file split waves (typed_reader 1127, adttest
      952, enginetest 935, store 898, execute 767, engines 724/722/694/650,
      architecture/helpers 627, suppression/parser 540, explain 516, …). —
      source: 06-56 §a9/§d1
      _(Effort: L, multi-session)_

---

## Release / Tagging

> The full 39-tag v4 wave (B1–B7) was cut, pushed, and verify-ci-green on
> 2026-08-29. Master is pushed and in sync with origin as of 2026-09-06
> evening (repo-gate GREEN not re-claimed by the docs-health pass — CI
> billing is still broken, see below). Zero
> local `=> ../` replaces remain EXCEPT `storage/go.mod` (`=> ../encryption`,
> `=> ../snapshot` — the documented unpublished-sibling pattern).

- [ ] [BLOCKED] 🔥 **Next v4 tag wave** — substantial unpublished surfaces on
      master as of 2026-09-06: `encryption` (key helpers + envelope v2),
      `snapshot` (`NewRewritingTransformedStore` + wire tags), `storage`
      (`MigrateSnapshotColumnsToStream`, EventSchema re-exports, bytea fix),
      `cmd/cqrs-lint` (working `--fix`, C005, RULES.md, doctor JSON, scorecard
      panel, preset policy), `cmd/api-stability` (sub-package golden),
      `catalog`, `metaengine` (planner capability partition, record context,
      `SortPaginate[T]`, planned-table parity) + engines (irohengine OpGraph*
      consts, mysqlengine, pgengine, sqliteengine, duckdbengine, dgraphengine
      recalibration, badgerengine — now consumes new `metaengine.SortPaginate`,
      pin bump + replace-strip REQUIRED), `scheduling/sqlstore` (MySQL
      claiming). **Strip `storage/go.mod`'s two local replaces in the same
      wave.** Order constraints per CONTRIBUTING pre-tag checklist;
      cut→push→next interleave (GOPRIVATE resolves siblings via VCS). —
      source: 08-26 §c3, 15-09 §f47
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
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms (worst-of-30 sample inflates under gate load). Shipped + gated
      green; keep or revisit. _(Effort: XS)_
- [ ] **cqrs-bench deprecation stub** — the dead suffix-less `cmd/cqrs-bench`
      path silently serves `v0.1.0` via `@latest`; ship the deprecation-stub
      treatment `cmd/cqrs-lint/v0.2.1` got. — source: archived/2026-09-01_21-37
      §c1
      _(Effort: XS)_
- [ ] **`retract cmd/cqrs-lint/v4.8.0`** — the poisoned (syntax-error) tag
      remains fetchable; v4.8.1 supersedes it but a retract stops fresh
      consumers from resolving it. — source: archived/2026-09-01_21-37 §c6
      _(Effort: XS)_
- [ ] **tag-release.sh hardening:** zero tests exist; add a proxy smoke-check
      ("clean-dir install @latest + run") as a documented post-cut step;
      one-shot all-modules path-vs-tag audit (the issue-#20 class). Consider
      `pin-sweep --no-build` inside the pre-flight. — source:
      archived/2026-09-01_21-37 §c2/c3/c5, 15-09 §f44
      _(Effort: S/M)_
- [ ] **Version-reporting unification** — cqrs-lint hand-maintains a version
      string const (tag-release.sh bumps it); adopting debug/buildinfo would
      remove the drift class. Decide const-vs-buildinfo before v5. — source:
      archived/2026-09-01_21-37 §c4
      _(Effort: S)_
- [ ] **Badger data-loss exposure review** (user decision): the fixed
      restart-sequence bug means any badgerengine deployment that reopened a
      DB and appended overwrote early entries. Retrospective (bound the
      window, audit consumers) — or confirm badgerengine is pre-adoption and
      skip. — source: 15-09 §g1
      _(Effort: S)_

---

## Metaengine — correctness & verification follow-ups

> Harvested from the 2026-09-06 correctness wave (archived/07-43) + iroh
> phase-7 session (archived/07-04). The wave itself shipped: planner
> capability partition, record-context advisory, MariaDB claiming,
> single-sourced calibration constants, dgraph per-row recalibration,
> duckdb/sqlite planned-table parity, iroh graph WriteOp convergence.

- [x] **`ApplyBatch` drops `EventInput.Record`** — it routes through `Apply`,
      which synthesizes a Type-only Record; the field is dead on this path.
      Either honor it via `applyWithRecord` (preferred) or document it as
      replay-paths-only. — source: 07-43 §d1/§f1
      _(Effort: S)_
      ✅ Done 2026-09-07 — ApplyBatch calls applyWithRecord (empty
      Record.Type falls back to the event type); pinned by
      TestApplyBatch_HonorsRecord + synthetic-record counting test.
- [x] **`recordAwareEvents` cache has no invalidation hook** —
      runtime-registered queries (`RegisterQuery`) with new OnRecord folds are
      invisible to the advisory until restart; `Hooks.Logger` path untested. —
      source: 07-43 §b6/§f3, §f14
      _(Effort: S)_
      ✅ Done 2026-09-07 — RegisterQuery now invalidates the cache
      (recordAwareEvents.Store(nil)); Hooks.Logger advisory path pinned by
      TestSyntheticRecordAdvisory_LoggerPath.
- [x] **Observe-before-claim verification set:** Doctor planned-tables section
      on sqlite+duckdb (row counts actually render); adttest
      `RunPlannedOpsMatrix` legs for sqlite/duckdb; e2e
      `BackfillPlannedCollection` on both; lying-only-engine Apply hard error
      correlated with the plan WARN; Replan/CheckRouting under the new
      partition logic. — source: 07-43 §b2/b3, §f9–13
      _(Effort: M)_
      ✅ Done 2026-09-07 — sqlite+duckdb Doctor row-count tests,
      matrix/backfill legs, TestApply_LyingOnlyEngine hard-error
      correlation, TestReplan_KeepsExcluding + direct-call
      TestCheckRouting_NeverSuggests (default-deadband-proof pattern).
- [x] **MySQL/MariaDB claiming completion:** `TestClaimingMySQL_RenewLease`
      (mirror the PG contract); construction-time server-version probe
      (`SELECT VERSION()`, reject <10.6 MariaDB / <8.0 MySQL) or keep the
      documented fail-at-first-Due contract (user decision — 07-43 §g1); wire
      the integration tests into the nix ephemeral-mysql runners; update
      `scheduling/sqlstore/README.md` claiming support matrix. — source:
      07-43 §f15–19
      _(Effort: M)_
      ✅ Done 2026-09-07 — RenewLease test verified live on MariaDB
      :33061; DECISION: keep fail-at-first-Due (no version probe);
      claiming leg wired into both vm-mysql nix runners; README support
      matrix + version floors documented.
- [x] **Dgraph calibration completion:** skip-guarded dgraph ReadCosts pins in
      `TestRealProfiles_ReadCostsPinned`; decide the `NsPerPointLookup`
      OLogN-vs-one-RPC model mismatch (07-43 §g2 — routing semantics);
      bench SearchQuery separately; DSN-guarded remote dump tests so the drift
      script covers live windows. — source: 07-43 §f20–26
      _(Effort: M)_
      ✅ Done 2026-09-07 — skip-guarded pins (350k/2.2k/2.7k/2.2k +
      complexity); DECISION: ADTMap→O1 only, other OLogN ADTs deferred
      with comment; BenchmarkCalibration_DgraphSearchQuery added;
      CALIB_DUMP dump tests shipped per engine (drift script reads
      shipped profiles).
- [x] **Planner polish:** name the missing backend interface in the
      over-declaration diagnostic; thread `CapabilityGaps` through Plan so
      documented gaps suppress the new diagnostics; tie-break determinism
      test; document the capability-aware partition rule in planning docs. —
      source: 07-43 §f27–32
      _(Effort: M)_
      ✅ Done 2026-09-07 — diagnostic names the missing interface
      (e.g. metaengine.MapBackend); WithEngineCapabilityGaps threads
      through Plan AND Replan (Store-carried, disable-fix proven);
      TestPlan_EqualLatencyTieBreakIsDeterministic; rule documented in
      recipes §2.12.
- [x] **iroh test-coverage holes:** pin graphless `GraphRemoveEdge` sentinel;
      test `applyRemoteGraphRemove` record-but-skip path; non-string node
      endpoints over loopback+quic (normalizeAny divergence); loopback/quic
      convergence `-race -count=3`; extract `applyRemote` from engine.go
      proactively (334/350 lines — the next op kind busts the limit). —
      source: 07-04 §b2/b4, §f1–11
      _(Effort: M)_
      ✅ Done 2026-09-07 — sentinel pinned; record-but-skip internal
      test (disable-fix proven); int endpoints converge on BOTH loopback
      and quic; all three iroh modules -race -count=3 green; applyRemote
      extracted to engine_apply.go (engine.go 334→281). GOTCHA:
      loopback/quic module tests MUST run in workspace mode — GOWORK=off
      resolves published irohengine v4.1.0 which predates graph-op
      replication and fails the new tests environmentally.
- [x] **`metaengine.SortPaginate[T]`:** direct unit test (currently covered
      only via engine suites) + micro-benchmark pinning the zero-alloc
      closure contract. — source: 15-09 §f7/§f8
      _(Effort: S)_
      ✅ Done 2026-09-07 — 6 unit tests + alloc budget (truncation 0,
      sort ≤3, upper-bound style) + inlined-reference equivalence test +
      BenchmarkSortPaginate_1K.
- [x] **keycodec extraction:** badger's seq-seeding now mirrors pebble's
      `seq_seeding.go` semantically — extract the parse/seed helpers
      (`SplitGroupAndSeq`, `SeedSeqMax`) into `keycodec` + round-trip test
      pinning the 20-digit+NUL key layouts. — source: 15-09 §e10/§f9/§f10
      _(Effort: S)_
      ✅ Done 2026-09-07 — SplitGroupAndSeq/SeedSeqMax/SeqTailLen(21)
      in keycodec; badger + pebble call sites migrated (badger back under
      the 350-line limit); round-trip/rejects/tail-layout tests added.
- [x] **duckdbengine restart-safety adoption** — `RunRestartSafetyTest`
      harness applies mechanically (same shape as sqlite); was deferred while
      a concurrent session owned the module. Also confirm bboltengine
      coverage parity. — source: 15-09 §b1/§f11
      _(Effort: S)_
      ✅ Done 2026-09-07 — duckdb cgo-tagged adoption (with
      availability probe-skip) + bbolt parity confirmed; follow-on:
      enginetest.RunRestartSafetyFromDBTest extracted and badger/duckdb/
      sqlite FromDB tests consolidated onto it (killed the check-dupl
      clone group at the root).
- [x] **`errorfamily` code rename `aggregate_*` → `stream_*`** (v5 item) —
      with a dashboards/consumers note. — source: session-4 retro §f30
      _(Effort: M, v5)_
      ✅ Done 2026-09-08 — all 17 family codes renamed in one batch
      (incl. `watermill.parse_aggregate_id_failed` + deprecated
      transport/grpc codes missed by the 2026-08-22 census); full mapping +
      dashboards/consumers note in CHANGELOG `[Unreleased]`; sweep artifact
      §4 struck. Deviation: `storage.stream_by_aggregate` →
      `storage.read_stream`. Follow-on repairs in the same wave: stale
      `storage` go.mod pins (coordinated-release gap, standalone build was
      red at HEAD) + 3 stale T18 snapshot-schema goldens re-blessed.
      STILL OPEN separately: `listing.aggregate_projection` name, watermill
      metadata keys, events/commands SQL columns (sweep §4).

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
      fail loudly. — source: archived/2026-08-30_06-34 §f
      _(Effort: S)_
- [ ] **actionlint CI step** (exists in devShell since T37) + extend
      shfmt-drift job with shellcheck for `scripts/`. — source: 15-09 §f28/§f29
      _(Effort: S)_

---

## Code Quality

- [ ] **>350-line production files (~54, 2026-09-06 count)** — see the
      cqrs-lint section for the verified picture, gate-policy options, and the
      already-split offenders; the code-file split waves are a standalone
      multi-session program pending the policy decision. Decide
      harness-dir exemptions (adttest/enginetest are exported test harnesses)
      first. _(Effort: XL, multi-session)_
- [ ] **Attribute + resolve the 5 pending clone groups** (check-duplication,
      verified foreign at 15-09, owners landed since): cqrs-lint
      `pkg/suppression/fix.go` sortAuditEntries prologue ×2; the
      duckdb/pg/sqlite `planned_parity` sort.Slice trio; the csp_browser_test
      ↔ store_collaborators mutex-idiom pair. — source: 15-09 §b2/§f3
      _(Effort: S)_
- [ ] **Pre-existing scheduling/sqlstore lint findings** — gosec G202 (SQL
      concat), sqlclosecheck ×2, staticcheck QF1003, wsl_v5 (proven
      pre-session 2026-09-06). — source: 15-09 §c2/§f4
      _(Effort: S)_
- [ ] **`example/metaengine-quickstart/README.md` does not exist** — author it
      from its three demo files (docs/README.md links the directory; the
      copy-paste surface is missing its page). Consider a
      `TestEveryExampleHasREADME` meta-test so the class is caught
      mechanically. — source: 07-42 §b2/§f28
      _(Effort: M)_
- [ ] **Example v5-policy audit** — taskmanager + metaengine-quickstart not
      yet verified free of v5-removed APIs (getting-started + readme-quickstart
      verified 2026-09-06). — source: 07-42 §f8
      _(Effort: M)_
- [ ] [BLOCKED] **macOS verification of ephemeral PG** —
      `scripts/ephemeral-pg.sh` claims cross-platform but was only
      static-review-tested; a GitHub Actions macOS runner leg is the
      verification route. _(Effort: M)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) —
      userspace MariaDB coverage exists but not the full nspawn env. Now also
      covers the MySQL claiming integration tests. _(Effort: M)_
- [ ] **Evaluate `-shuffle=on` for the dgraph suite specifically** (adopted
      for pg/mysql/sqlite/duckdb; dgraph still needs its own evaluation, then
      roll into the ephemeral-* app invocations). _(Effort: S)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1–7 done. Pre-cut deprecation markers shipped 2026-08-17; migration
> guide at `docs/V5-MIGRATION-GUIDE.md`. Snapshot wire tags (T18) DONE
> 2026-09-06 — dual-read fallbacks in snapshot/pebble, SQL columns migrated by
> `MigrateSnapshotColumnsToStream` (auto-run by every InitSchema).

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
      error-code batch rename (~14 codes) with the dashboards note. Consider a
      central wire-key table doc (JSON/CBOR/SQL × backend × fallback status)
      rewriting sweep §4 as a table. — source: 08-41 §b1/§f1–11
      _(Effort: M)_
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
      `fmt.Errorf` → pebble error-family pattern), E4 (sentinel name↔code
      mismatch: `storage/sql/errors.go:22`), E6 (`middleware.Option` vs
      `BundleOption` merge/bridge), E9 (turso Policy nil-write panics), E10
      (ShutdownDependency name validation), E14 (eventstore ownership
      asymmetry). — source: `docs/reviews/2026-08-22_extended-data-model-review.md`
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
>   rejected. One open plan task: **T23** (upstream skill fixes:
>   docs/reviews↔brainstorming divergence; read-prior-reports +
>   copy-template steps in the review skills) — execute or decline at the next
>   skill-maintenance pass.

---

## Docs / consumer-surface truth

> Harvested 2026-09-06 (afternoon pass) from archived 07-42/08-26/08-41.
> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [ ] **Skill-reference propagation wave** (`references/*.md`): envelope v2 +
      rotation write-back recipe (recipes.md §2.7 extension); `doctor
      --format json` + `check-csp`/`check-eventcatalog` apps; MySQL claiming
      support matrix (10.6+ works); planned-table capability roster (sqlite +
      duckdb now qualify); Doctor's record-context + planned-tables sections;
      `CALIB_DUMP=1` usage; consumer recipe for decoding pre-v5 snapshots
      (JSON+CBOR fallback contract). — source: 07-43 §c4, 08-26 §b1,
      08-41 §b6/§f24
      _(Effort: M)_
- [ ] **encryption module docs** — README + doc.go still don't mention the
      key-management helpers or the v2 envelope format; add the wire-format
      golden (encrypt→Marshal output as a reviewed artifact) + v1↔v2 decode
      symmetry property test. — source: 08-26 §b9/§f11–13
      _(Effort: S)_
- [ ] **`awaitAck`/`replayPhase` lying log line** — on `Close()`, the log
      says `ERROR ... "consumer nacked replay event"` though the consumer
      never nacked. Distinguish Close from Nack. — source: 07-42 §c2
      _(Effort: XS)_
- [ ] **Benchmark auto-discovery check** — does
      `scripts/benchmark-regression.sh` auto-discover
      `BenchmarkCatchUp_ReplayThroughput` (load-sensitive)? Pin or exclude so
      it can't flake the CI regression gate. — source: 07-42 §e4/§f2
      _(Effort: S)_
- [ ] **Watermill catch-up tail:** restart-recovery property test (checkpoint
      behind a skew-suppressed event ⇒ replay re-delivers — pins the
      documented self-healing claim); broker-backed throughput variant via
      `ephemeral-redis.sh`; make `CloseWhileBlockedOnFullBuffer` deterministic
      (blocking-journal hook instead of the 100ms sleep). — source: 07-42
      §f11/§f17/§f18
      _(Effort: M)_
- [ ] **README/AGENTS quick-reference rows** for `check-csp` +
      `check-eventcatalog` (flake apps exist, not referenced in the quick-ref
      tables). — source: 08-26 §c2
      _(Effort: XS)_
- [ ] **Social preview image + homepage URL** — GitHub settings-UI fields
      (CLI can't set them); needs a generated asset + owner paste. — source:
      07-42 §b1
      _(Effort: S, manual)_
- [ ] **doc-check tail:** `--json` output for CI annotations; warn (don't
      silently union) when a no-import alias maps to multiple repo packages;
      scoped `#doc-check` flake app. Release posture decision for the stricter
      block-scoped resolver (ship as-is vs `--legacy-union` transition flag —
      15-09 §g3). — source: 15-09 §f13–16
      _(Effort: S/M)_
- [ ] **exhaustruct_v5 canary test** — prove each `ignore-patterns` entry
      still matches under v5 full-type-name semantics; plus a
      deprecated-linter-name golden for `.golangci.yml` (config verify catches
      schema, not deprecations). — source: 15-09 §c3/§f5/§f30
      _(Effort: S)_
- [ ] **templ tripwire script** — parse `_templ.go` FileName metadata and
      fail if paths aren't `catalog/docserver/`-relative (automates the
      cwd gotcha); consider scanning all templ dirs repo-wide. — source:
      15-09 §f25/§f26
      _(Effort: S)_

---

## Declined / Rejected (do not re-litigate)

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
