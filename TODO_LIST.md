# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, 2026-09-08, and 2026-09-11). The Declined section at the
bottom is a do-not-re-litigate guard, not a backlog.

> **Prioritized execution plan (2026-09-08):**
> [`docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md`](docs/planning/archived/2026-09-08_17-45_SUPERB-pareto-execution-plan.md)
> ranked the then-list into Pareto waves (W0 release train → W1 trust →
> W2 efficiency → W3 v5 train) and was EXECUTED through 2026-09-11 (W3's v5
> items live in the v5 section below; the user-gated P22 halves remain in the
> Turso section). This file remains the living source of truth.

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
- [ ] **Code guard follow-up: make grouped-spec safety mechanical** — today the danger is advisory-only (Doctor WARN + docs). Options: `MaterializedViewSpec` validation refusing `GroupBy` on turso-go ≤ v0.8.0-pre.10 (breaking for legitimate small deployments) vs a config flag (`AllowGroupedViews`) vs silent status. Decide + implement once the upstream timeline is known (still unknown: PR #8257 unanswered, defect re-verified live on pre.10 2026-09-11). The mechanical flip point now exists: `TestTursoMatView_GroupedSumDefectAEnvelopeGuard` + `TURSO_IVM_ENFORCE_FIX=1` asserts exactness at the 2k-row repro shape the day upstream fixes it. _(Effort: S)_
- [ ] **Matview v2 feature surface** — planned-table matviews (ordered with `ApplyLayout` + backfill), filtered-view spec variants, multi-aggregate/DISTINCT serving, `DropMaterializedView` off-boarding, per-view IVM write-amp otel counter, `system.Introspection()` surface, cqrs-lint rules (matview-on-unsupported-driver; matview-plus-planned-table staleness trap), `example/materialized-views/`. Route individually when a consumer asks. — source: archived 19-25 §f23-35, 05-33 §f29-35
      _(Effort: M/L each)_
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). NEXT STEP (SUPERB S28): design one-pager for `AggregateOn(fn, column, group)` on `QueryDecl` — the declarative seam the planner can read — then routing v1: scalar-covered shapes price O(1) (matview-served), grouped shapes stay O(N) with a Doctor note (upstream defect A makes grouped routing unsafe). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A — scope the first cut to scalar-covered shapes only. — source: archived 19-25 §f29, 05-33 §f32, SUPERB S28/05-51 §f16-17
      _(Effort: M)_
- [ ] **Turso follow-ups from the IVM session (2026-09-11):** (a)
      `--self-test` mode for `scripts/check-turso-version.sh` (planted stale
      citation in a temp fixture) so fault-injection never mutates a live
      tracked file again; (b) run the `-tags ivmrepro` suite with `-race`
      once (24-round engine lifecycle + double-`t.Cleanup` Close);
      (c) clamp the suite's last chunk for non-multiple-of-1000
      `TURSO_IVM_REPRO_ROWS` values; (d) add the one-command repro check to
      `docs/release-checklist.md` (driver pin bumps always run it);
      (e) fold the session's three findings into the frozen upstream draft
      before filing (wall 25000-via-tursoengine vs 27000-raw is
      workload-dependent; zombie-tx readback artifact; poisoning is
      connection-state, not durable). — source: 05-21 §f2/§f4/§f7/§f8/§f12
      _(Effort: S total)_
- [ ] **Tag wave for the matview feature** — metaengine/sqliteengine/tursoengine/system carry sibling replaces for unpublished symbols (`MaterializedViewSpec` family); pins must be bumped and replaces stripped at the next release wave so consumers can use the feature from published tags. _(Effort: M — see AGENTS.md tag-wave procedure)_
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

- [ ] **Release-train note** — the `metaengine/projectionadapter` and
      `metaengine/irohengine` sibling replaces (unpublished `EngineResetter`
      symbols) + the `metaengine` pin bumps ride the next tag wave
      (`scripts/tag-release.sh` strips the replaces; smoke at cut time).

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
- [ ] **cqrs-lint cheap-fix follow-up tail (2026-09-11 session §f):** (a)
      validate the S001 URL/placeholder allowlist against real corpora
      (taskmanager scan + a probe project — prove no true positives killed);
      (b) D014/D015 registry-acceptance tests (the parity claimed for D016
      is untested on their side); (c) pin B008's non-bitshift Warning
      baseline (a global severity flip to Error would pass today's suite);
      (d) S001 selector-LHS receiver context in the message (03-44 #101
      second half) + golden impact check; (e) full-module `-race` for
      cmd/cqrs-lint (`./...`, not just `pkg/rules/...`); (f) extract the
      URL/placeholder value-classifier into lintutil BEFORE a second rule
      needs it (S001 split-brain prevention). — source: 05-12 §f1-5/§f26
      _(Effort: S each)_
- [ ] **Extend the error-taxonomy drift gate beyond its 5 modules** —
      gated+verified 2026-09-11: graph, storage/relational, projectionhost,
      middleware, transport/grpc (161 codes / 141 claims). Remaining
      sections: watermill, storage/pebble, core event/command/query,
      storage/view, stack, deriver, storage-facade. One `GATED_MODULES` line
      per module, each forcing that section's code inventory complete; add a
      per-module pool-size floor (scanner-break tripwire) and replace
      `rg … || true` with explicit extraction assertions while there. —
      source: 05-26 §b1/§f11-17, 05-51 §f30
      _(Effort: S/M)_
- [ ] **cqrs-upgrade strict-gate residual holes** — (a) `--strict` must
      FAIL when any module errored (rep.Error) — unscanned = unproven;
      (b) run the deprecation scan even for NoPins modules (indirect-only
      cqrs consumers currently escape); (c) make `bumps` always-present in
      `--json` (symmetry with `deprecations`); (d) consider a
      `schemaVersion` field for the `--json` wire; (e) E2E test of `run()`
      against a fixture module (flags→report→strict exit codes). — source:
      05-26 §e4/§f6-10
      _(Effort: S)_
- [ ] **Kill the self-lint false-green class at the root** — any path under
      `github.com/larsartmann/go-cqrs-lite/**` gets V007/F030 silently
      skipped (the prefix check); the consumer-copy workaround lives in a
      test. Fix `IsLibrarySelfLint`/presets to treat `example/*` as
      consumers, add an analyzed-assert (file count) wherever examples are
      scanned (the 02-47 lesson), then simplify `TestExamples_AreV5Clean`.
      V007 typed method detection (`types.Info.Selections`) is the bigger
      sibling — decide before the v5 cut. — source: 05-26 §e2/§f18-20
      _(Effort: M)_
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
      claiming), `watermill` (**v4.7.0 — the issue-#21 typed-causation wire
      protocol: `writeCausation`/`parseCausation` + legacy custom-mirror
      promotion, landed 2026-09-09 and STILL UNTAGGED**; go-localsync runs
      its documented workaround until this tag exists), `system` (v4.7.0: materialized views + the MV recipe's
      UNRELEASED marker flips when tagged). **Strip `storage/go.mod`'s two
      local replaces in the same wave.** Order constraints per CONTRIBUTING
      pre-tag checklist; cut→push→next interleave (GOPRIVATE resolves siblings
      via VCS). — source: 08-26 §c3, 15-09 §f47, SUPERB §f14-16
      _(Effort: M)_
- [ ] **Tag `cmd/cqrs-lint` v4.10.2 (ships the buildinfo version reporting)** —
      on master since 2026-09-11; v4.10.1 deliberately predates it. Verify the
      installed binary prints the real tag after `go install …@v4.10.2`. —
      source: 01-47 §b1/§f5
      _(Effort: S)_
- [ ] **`check-retracts-shipped.sh`** — fail when a master go.mod retract
      directive is absent from the module's newest tag (the inert-retract
      class: `retract v4.8.0` sat on master ~10 days before v4.10.1 shipped
      it). Acceptance test for every retract = clean-dir `go list -m
      module@latest`. — source: 01-47 §d3/§e1/§f6
      _(Effort: S)_
- [ ] **`tag-release.sh --audit --baseline` mode** — the one-shot audit found
      24 historical violations (1078 tags), all in dead paths that cannot be
      fixed; a known-violations baseline (art-dupl pattern) turns `--audit
      --check` into a CI leg gating NEW violations only. — source: 01-47
      §b3/§f7
      _(Effort: S/M)_
- [ ] **`scripts/smoke-probes.txt` + strengthen test-tag-release.sh Test 5** —
      per-binary probe command for the `--smoke` run check (`--help` exit
      semantics differ across CLIs); Test 5 covers the `--smoke` usage guard,
      not the no-main-package skip path. — source: 01-47 §b4/§b5/§f11/§f12
      _(Effort: S)_
- [ ] [BLOCKED] **Dead-path module/tag decisions (owner)** — (a)
      example/taskmanager + example/getting-started carry suffix-less module
      paths with permanently-invisible v3/v4 tags: re-path to /v4, delete, or
      document as v0-only; (b) `event/v4/eventtest`'s invisible v0.x tags:
      document as dead in modules.md + pin-sweep note. — source: 01-47 §c1/§f9/§f10
      _(Effort: M decision + S doc)_
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
- [ ] **Private-dep mechanical guard + visibility audit** — go-must (private)
      froze the proxy and broke every workspace-mode command until inlined.
      (a) `scripts/check-private-deps.sh` (+ flake app + CI leg): every
      `github.com/larsartmann/*` require in every go.mod must be
      proxy-servable (`@v/<version>.info` fetch); (b) audit sibling helper
      repo visibility (go-retry/go-codec/go-branded-id/go-sse/go-idempotency/
      go-flightrecorder) and record public/private in module-map.md; (c)
      owner policy: examples may only depend on public/proxy-servable
      modules. — source: 05-34 §e1/§f7-9
      _(Effort: S/M)_
- [ ] **Release-tooling follow-ups (post-hardening):** `--smoke-all` batch
      mode (push N tags → one command smoke-checks each); document the batch
      inter-module limitation (same-batch siblings resolve to the latest
      PUBLISHED tag); optional batch `--verify` full-pipeline dry-run;
      CONTRIBUTING.md references `batch-release.sh` +
      `nix run .#check-release-scripts` in the release process; extract
      `path_matches_major` into a sourced lib (two-copy lockstep risk);
      decide whether `check-release-scripts` also runs in `#verify` (~30s).
      — source: 05-34 §e2/§e6/§f22-27
      _(Effort: S)_
- [ ] **taskmanager tail from the go-must fix** — unit tests for the inlined
      `example/taskmanager/must.go` (a copy with zero tests); confirm what
      taskmanager's `go test` actually executes in 0.080s (which env markers
      skip); update module-map.md internal notes (no go-must anymore). —
      source: 05-34 §b3/§f5/§f6/§f31
      _(Effort: S)_

---

## Metaengine — follow-ups

- [ ] **Calibration provenance protocol + quiet-window re-runs** — protocol HALF DONE 2026-09-11 (later session), re-runs remain gated on a quiet window: (a) DONE — `scripts/calibration-gate.sh` asserts 1-min load < 5 (overridable `--max-load`/`CALIB_MAX_LOAD`; CI exempt) and aborts loudly — verified against a live compile storm (load 207 → hard abort); `calibration-drift.sh` runs it before benching; (b) DONE — protocol items 6-8 in `docs/benchmarks/calibration-2026-08-30.md` define the per-entry PROVENANCE line (store path + binary version output + uptime samples) and ban secondhand version citations; the 2026-09-11 SearchQuery entry now carries an explicit provenance-gap note; (c) MECHANISM DONE, RUN PENDING — `benchmark-regression.sh --save` writes a titled provenance header (fixture-tested, parser-safe), but the quiet-window count=5 SearchQuery re-run and the titled re-pin of `benchmarks/benchmark-baseline.txt` did NOT run (a 493-load storm held all session; gate correctly refuses); (d) PENDING — re-anchor ALL dgraph constants in one gate-passing window. Run when `scripts/calibration-gate.sh` passes: SearchQuery count=5 (supersede today's table if medians move >5%), then the benchmark-baseline re-pin, then the dgraph constant campaign. — source: 03-50 §b2/§b3/§f7/§f8/§f15/§f16, 02-48 §d3/§f8
      _(Effort: M)_

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

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
- [ ] 🔥 **Close the `CatchUpEngine` snapshot race** — the ONE known
      correctness hole in the ADR-0137 write-failover work: `CatchUpEngine`
      replays a once-taken `Events()` snapshot; concurrent applies DURING
      the replay window fold onto the failover engine only, so the
      reactivated engine silently misses that window (sequential tests
      cannot see it). Fix: loop reset+replay until the log length stops
      growing between snapshot and reactivation, or hold `s.mu` write-locked
      for the final stabilization pass; add a concurrent stress test (apply
      loop racing CatchUpEngine; assert post-reactivation reads see every
      event). Should land BEFORE the next tag wave. — source: 05-40 §d1/§e1/§f1
      _(Effort: M)_
- [ ] **Catch-up observability + tail replay** — surface catch-up state
      (running/failed/last-caught-up event id) in `Doctor` + `GetEngineStats`
      (today slog-only); per-engine catch-up high-water marks so a re-catch-up
      replays only the tail instead of the full journal; return `ResetResult`
      from `CatchUpEngine` (currently discarded); `Reset` docs should point
      at `CatchUpEngine` for the one-engine case (discovery). — source:
      05-40 §e5/§e6/§f9/§f10, §f35/§f36/§f45
      _(Effort: M)_
- [ ] **Doctor: per-entry-point synthetic-record feed counters** — the
      conformance sweep makes the entry-point record contract visible in
      TESTS; Doctor still cannot say WHICH entry point fed a synthetic
      (Type-only) record at runtime. — source: 03-50 §f17, 05-38 §f11
      _(Effort: M)_
- [ ] **Legacy-log-entry synthesis pin** — `replayShadows`/`applyReplay`
      synthesize a Type-only record ONLY for legacy `EventLog.Record()`
      entries (`Record.Type == ""`); the legacy path is asserted nowhere
      end-to-end — add one legacy case to the conformance sweep. — source:
      03-50 §f23, 05-38 §c1/§f2
      _(Effort: S)_
- [ ] **Conformance-sweep + hot-path tail** — (a) extend the sweep with an
      `ApplyIdempotent` dedup no-op second apply (advisory counts once, not
      twice); (b) micro-bench the `applyFold` raw-payload type-assertion
      overhead on the struct hot path (defend the encoded-apply fix with a
      number); (c) run the new PG/MySQL ClaimMetrics integration tests
      against live servers (`#integration-pg`, `#integration-mysql-nspawn`).
      — source: 05-38 §b2/§f3/§f4/§f10
      _(Effort: S/M)_
- [ ] **recipes.md: `ApplyEncodedRecord` snippet** — the projection.Projection
      adapter recipe for the encoded-record path; references are currently
      silent on it (only modules.md has the row). — source: 05-38 §c4/§f5
      _(Effort: XS)_
- [ ] **Calibration gate v2: sustained quiet** — `calibration-gate.sh`
      checks load1 only; a burst-draining host (load1=4, load5=30) passes
      and is still noisy. Require load1 AND load5 under the ceiling; also
      run `shellcheck` over it (never shellchecked). — source: 05-38 §e/§f1/§f9
      _(Effort: XS)_
- [ ] **scheduling/sqlstore hardening tail (carried from 03-50, untouched):**
      race-stress test (concurrent `Due` pollers vs `Metrics()` reader);
      counter-scope pin (`MarkFired`/`Schedule`/`Cancel` deliberately never
      touch claim counters); property test (counters never exceed committed
      polls); fuzz `decodeDueTimer` corrupt-payload path; worked
      `Metrics()` → `/status` example; runnable otel wiring example for the
      ClaimMetrics hooks; scheduler+claiming-store+Metrics e2e example;
      `RenewLease` ownership/claim tokens (code comment defers today);
      consider process-start timestamp on `ClaimMetricsSnapshot` for
      cross-restart rates. — source: 03-50 §f18-27, 05-38 §f19-27
      _(Effort: M, one rule per slice)_

---

## CI / Infrastructure

- [ ] **Zero the erraudit error-policy baseline** (precondition for the
      `error-audit` CI gate added 2026-09-11) — the job is wired
      (`.github/workflows/ci.yml`, same self-activating `ERRAUDIT_PAT`
      mechanism as go-codec) but stays dormant until (a) a secret exists and
      (b) these findings are zero. Baseline 2026-09-11: **253 findings across
      22 of 35 modules** under `erraudit lint ./... --enforce-go-error-family
      --type-aware` per module: storage 53, graph 46, event 25, encryption 14,
      command 13, decider 12, kv 11, stack/snapshot/benchkit 9 each, signing
      8, query/middleware/catalog 7 each, schema 6, metaengine 4,
      id/dispatcher 3 each, watermill/projectionhost/deriver/scheduling 1-2
      each. Recount with
      `for m in */; do [ -f "$m/go.mod" ] && (cd "$m" && GOEXPERIMENT=jsonv2 erraudit lint ./... --enforce-go-error-family --type-aware --format csv 2>/dev/null | tail -n +2 | grep -c . | xargs -I{} echo "$m {}"); done`.
      Do per-module batches (storage first — largest). _(Effort: L, 22
      modules; go-codec's ADR-0001 + docs/error-codes.md are the reference
      pattern)_
- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s; broken since ~2026-07-17. Local `nix run .#verify` remains the
      authoritative gate. _(Effort: S, user action)_
- [ ] [BLOCKED] **cqrs-lint Self-Lint credentials** — go-finding fetch fails
      under GOWORK=off (`git ls-remote` exit 128). _(Effort: S, user/creds)_
- [ ] **Calibration-drift gate redesign** — compare against a persisted
      CI-baseline artifact instead of absolute constants; nightly >100% rows
      are shared-runner noise. Add TMPDIR-filesystem detection (refuse to run
      on CoW). — source: archived/2026-09-04 §b2/§f16/§f18
      _(Effort: M)_
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
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_
- [ ] **Integration-tag lint as a first-class gate** — the gocognit finding
      was invisible to the official gate for ~10 days (the `lint-module` app
      hardcodes only `goexperiment.jsonv2`): give `lint-module` an optional
      build-tag argument and add a CI leg for modules shipping
      `*_integration_test.go`. — source: 01-38 §c1/§e2/§f4/§f5
      _(Effort: S/M)_
- [ ] 🔥 **Kill the missing-go.sum-hash class in CI** — `pin-sweep --check`
      cannot see missing go.sum hashes (the pgx v5.11.0 `/go.mod` hash class,
      found live 2026-09-11; the earlier 8-module cold-cache rot class is the
      same family; root-cause hole: tidy-under-warm-cache). Fix:
      `#verify-ci` gains a per-module `go mod download` + no-diff assertion
      (or a `check-modsums` flake app), plus a
      `TestEveryModulePassesStandaloneVet`-style repo-level meta-test. —
      source: 01-38 §c/§e3/§f6, 03-43 §f2/§f10, 08-41 §b5/§e6, 08-26 §f4
      _(Effort: M)_
- [ ] **Per-finding attribution for the sqlstore lint surface** — sqlclosecheck
      ×2 / QF1003 / wsl_v5: code-fixed since 09-06 or silenced by the
      `_test.go` exclusion block? 15-minute diff against the 09-06 pre-session
      worktree closes the item's story. Also: one canonical golangci-lint
      binary for ad-hoc runs (PATH v2.13.2 vs the nix pin). — source: 01-38
      §b1/§b2/§e5
      _(Effort: S)_
- [ ] **`aggregate_*` tripwire: permanent mutation fixture** — the 2026-09-11
      hand-planted mutation proof dies with its status report; a `testdata/`
      fixture containing a planted code + scanner self-assert makes it a CI
      fact. — source: 01-38 §e4/§f3
      _(Effort: S)_
- [ ] **Live-verify the MySQL shuffle rollout + `-race` the dgraph retry code**
      — vm-mysql.sh / vm-mysql-nspawn.sh carry `-shuffle=on` syntax-only
      (nspawn needs root; skip the ~131s VM run was a scope call); the new
      `retryOnContention` backoff has no race-detector coverage yet (three
      live seed runs + e2e ran without `-race`; the CI race leg skips without
      a server). — source: 02-16 §b1/§b2/§f2/§f3
      _(Effort: S)_
- [ ] **Document the shared-`dgraph.type` conflict domain + unit-pin
      `isContentionError`** — every SetJson mutation touches `dgraph.type`, so
      ALL parallel writers conflict on one Alpha (hard-won, exists nowhere in
      the docs — gotchas-language-footguns.md + dgraphengine README);
      error-class matching is currently only live-tested. — source: 02-16
      §e4/§f7/§f8
      _(Effort: S)_

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
- [ ] **`go mod tidy` in `integration/`** — gopls flags unused
      `google.golang.org/genproto/googleapis/rpc` (integration/go.mod:131;
      still flagged 2026-09-11). — source: 02-16 §f24
      _(Effort: XS)_
- [ ] **Unify ephemeral-script passthrough conventions** — ephemeral-pg.sh
      uses positional EXTRA_ARGS, ephemeral-dgraph.sh uses
      TEST_ARGS/TEST_ARGS2, redis/nats use raw passthrough; three
      conventions for the same job complicate evaluations. — source: 02-16
      §e5/§f25
      _(Effort: M)_
- [ ] **Skip-vs-fail classifier spread** — dgraph's live helpers now skip
      ONLY on server-unreachable and `t.Fatalf` otherwise (the honest-loud
      OQ-10 policy); the pg/mysql test helpers deserve the same classifier
      (same silent-skip class). — source: 05-51 §e6
      _(Effort: S)_
- [ ] **projectionhost integration-build compile check** — the goleak
      `TestMain` carries `//go:build !integration`; the two-TestMain clash
      risk is handled by the tag but never compile-checked WITH it. One
      command: `go vet -tags integration ./...` in projectionhost. — source:
      05-40 §b4/§f4
      _(Effort: XS)_
- [ ] **Watch dgraph + redis CI jobs (~10 shuffled runs)** — record any
      seed that fails; rare orderings WILL eventually appear in CI (that is
      the point of shuffling). — source: 02-16 §e7/§f10
      _(Effort: XS)_
- [ ] **Record shuffle seeds to a log for post-hoc replay** — ephemeral
      scripts echo the seed; persist it to a file so a failed CI seed can be
      replayed exactly (`-shuffle=N`). — source: 02-16 §f23
      _(Effort: XS)_
- [ ] [BLOCKED] **Quiet-window exclusive `nix run .#verify` composed GREEN**
      (supersedes the contention-fix verify item) — last composed GREEN was
      2026-09-09; three days of waves (Cordis, publish/reset/v5-train, both
      parallel sessions) are unverified as one chain, the dispatch-core
      fold-reroute refactor has never seen `-race`, and `scripts/verify-docs.sh`
      has never run end-to-end with its new tripwire. When the box is quiet:
      run `#verify`, then `-race` over `metaengine`, then `verify-docs.sh`;
      record date + commit + durations in TODO_LIST/plan (S03 acceptance).
      — source: 02-16 §c4/§f13, 05-40 §f2/§f3/§f8, SUPERB S03
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
      sessions). Also: dry-run the `benchmarks.yml` matview gate set's exact
      CI invocation shape (the actionlint job covers syntax; the relative
      `cd ../metaengine/tursoengine` hop is unproven). — source: run 34548534824, `gh run list`
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
>
> - [execution plan](docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md).
>   Owner decision 2026-08-22 (Appendix B): string `record.StreamRef` SURVIVES
>   v5 with a validating constructor; the struct `record.Stream` proposal is
>   rejected.

- [ ] **T23 — upstream skill-maintenance pass** (the plan's one open task):
      docs/reviews↔brainstorming divergence; read-prior-reports +
      copy-template steps in the review skills. Execute or decline at the next
      skill-maintenance window. _(Effort: S)_

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [ ] **Reconstruct the orphaned `cec9248da` work record** — tripwire +
      fix.go dedup + pg test helpers were daemon-absorbed with no authoring
      report; a short annotated report (what/where/verified-how) closes the
      provenance gap. — source: 04-35 §c1, 01-38 §f8
      _(Effort: XS)_
- [ ] **Skill references: reset recipe covers ALL engines** — SKILL.md +
      references still describe `Store.Reset` as memory-only; the ladder is
      12/12 now. Update the reset recipe + add the
      `WithContentionObserver` entry to recipes.md. — source: 05-51 §f33
      _(Effort: S)_
- [ ] **`docs/status/README.md` index upkeep** — the 2026-09-11 batch-day
      reports (8 files) need index entries before/after archiving; the index
      is the only map of the ~1500-file archive. — source: 05-34 §f35
      _(Effort: XS, recurring)_

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
