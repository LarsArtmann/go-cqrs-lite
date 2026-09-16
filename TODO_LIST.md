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

## Durable Work Queue module (proposed 2026-09-13)

- [ ] 🔥 **Assemble the existing pieces into a `queue/` sibling module** — the claim core is EXTRACTED into `claiming/` (P0 done 2026-09-13: `Spec` + SKIP LOCKED PG / single-writer SQLite / MySQL two-statement claims, expiry reclaim, `RenewStmt`, `EnsureLeaseColumn`; `scheduling/sqlstore` delegates byte-identically and keeps `RenewLease` + `ClaimMetrics`), the read side in metaengine planned tables, the journal in `event`/`watermill`; what's missing is the task-store assembly: lifecycle (pending→running→completed/dead), attempts+backoff+DLQ at store level, priorities(+aging) in claim order, DAG dep gating, owner-bearing claims, dedup'd enqueue, same-tx journal option. Spec source of truth = go-taskqueue's production-proven `internal/queue.Store` contract (upstream the semantics, don't reinvent); conformance = one mirrored suite across dialects. Consumers: go-taskqueue (reference donor), PapDashboard (production worker pools today), `example/taskmanager` (demo→real). — source: [`docs/planning/2026-09-13_durable-work-queue-module.md`](docs/planning/2026-09-13_durable-work-queue-module.md) _(P0 done; Effort: P1 M lifecycle+DLQ, then S each)_

## Command-side domain depth (2026-09-13 plan)

> Prioritized execution plan:
> [`docs/planning/2026-09-13_11-45_SUPERB-command-side-depth.md`](docs/planning/2026-09-13_11-45_SUPERB-command-side-depth.md)
> — Pareto waves (W1 decider causation → W2 first-class command records → W3 lifecycle
> upcasting + docs parity → W4 gates/filing), derived from the three 2026-09-13 reviews
> (`docs/reviews/2026-09-13_*`). Additive-only v4.x; `decider` gains ZERO new deps
> (local `CausedCommand` capability interface). Guardrails: decider.go is 377/350
> baselined (new file required), api golden regen in same edit, CHANGELOG symbols gated.
>
> **Status 2026-09-13: W1–W3 EXECUTED** — `decider.ExecuteCommandRef` + `CausedCommand`
>
> - `CommandDecideFunc` shipped with BDD/property/example coverage (note: Go 1.26
>   forbids generic methods, so it is a package-level function — recorded in the plan's
>   D1 amendment); `command.AsRecordPersisted` shipped with fidelity tests + v5
>   deprecation note on the thin bridge; D3 upcast composition CONFIRMED and pinned by
>   `commandlifecycle/upcast_composition_test.go`; recipes §2.1b/§2.19b, core §3.8 +
>   cheat-sheet rows, faq command-pitfalls section all landed; goldens regenerated.

- [x] 🔥 **W1: `decider.ExecuteCommandRef` + causation stamping** — DONE 2026-09-13. Package-level generic function (Go 1.26 generic-method limit), stamps typed `Metadata.Causation` + compat keys, respects decide-set causation, skips zero-ID commands; BDD suite + rapid property + runnable example.
- [x] 🔥 **W2: `command.AsRecordPersisted(*PersistedCommand)`** — DONE 2026-09-13. Full-fidelity bridge (payload, StreamType, receive stamps); fidelity tests; v5 deprecation doc-note on `AsRecord(*BasicCommand)`; `//art-dupl:accept` twin annotation.
- [x] **W3: lifecycle upcast composition spike** — DONE 2026-09-13, hypothesis CONFIRMED: `DecorateStore(raw, nil, UpcastSourceTransform(u))` around the Recorder gives read-path-only evolution, write-path passthrough, current-version passthrough; permanent test `commandlifecycle/upcast_composition_test.go`; recipe recipes §2.19b.
- [x] **W3: command-pitfalls FAQ + docs parity** — DONE 2026-09-13. faq.md "Command-side pitfalls" section (causation-by-default, closure trap, evolving persisted lifecycle payloads); recipes §2.1b; core §3.8 note + cheat-sheet rows; doc-check 1123 refs zero-failure.
- [ ] **W4: gates** — per-module `GOWORK=off` tests (decider/command/commandlifecycle/schema), api-stability golden + `TestEvery`, CHANGELOG `pkg.Symbol` citations, doc-check zero-warning, `nix run .#verify`, `#check-arch` (decider: zero new deps), `#check-duplication` (0 new groups). _(Effort: M — in progress)_
- [BLOCKED] **Release train (user approval)** — tag waves decider/command/commandlifecycle once W1–W4 land. _(Effort: M — see AGENTS.md tag-wave procedure)_
- [BLOCKED] **ADR-0138: command sourcing draft (consumer demand)** — design doc only, builds on W2's bridge, reconciles ADR-0112's planned `CommandAwareFold`. _(Effort: M)_

## Go 1.27 upgrade wave (proposed 2026-09-13)

- [ ] 🔥 **Toolchain + go-directive wave to Go 1.27** — Go 1.27 (2026-08-19; 1.27.1 2026-09-01) graduated `encoding/json/v2` (v1 now backed by the v2 engine — the `-tags "goexperiment.jsonv2"` footgun dies repo-wide, clearing ~20 live gopls `stdversion` warnings) and legalized **generic methods** (method-level type params; interface methods still can't). Scope: bump all 85 `go.mod` `go` directives (language features are directive-gated), flake `goToolchain` pin to nixpkgs `go_1_27` (verify availability first), CI, AGENTS.md/docs command chains, then full `#verify` + integration suites + bench-regression sweep (v2 unmarshal is significantly faster — expect improvements), and a release train so consumers actually receive it. Consumer impact: `go` directive ≥ 1.27 forces toolchain download on older setups (`GOTOOLCHAIN=auto` mitigates). Sequel: revisit `decider.ExecuteCommandRef` as a true `Repository[State]` method (additive), plus the other option-func families. — evidence: go.dev/doc/go1.27 release notes, plan D1 amendment. _(Effort: L — own wave, do NOT fold into other plans)_

## Investigate: `TestSystem_ResetProjection_RestartAndReplay` contention stall (found 2026-09-13)

- [ ] 🔥 **Suspected replay-starvation race in system.Start's projection path under extreme parallel load** — during two full `#verify` runs (machine load 35–52 from concurrent builds), phase 2 (fresh `system.New` over the same SQLite journal, no checkpoint = replay-from-zero) showed `processed=0 errors=0` for 130s+ while the journal demonstrably held the event: the projection worker never folded anything, i.e. permanently missed the replay — the exact hazard signature of the subscribe-vs-drain ordering that `projectionhost` TOCTOU guard (recipes §2.23) was built to prevent. Passes 8+/8+ times module-isolated (even at load 35+) and full-module (`-count=1`, 0.39s); fails only inside the full-verify package storm; unrelated to the 2026-09-13 command-side changes (no causal import path). Mitigation applied: `waitForProjectionProcessed` base 15s→45s + outer ctx 90s→270s (proves it is a stall, not slowness — budgets burned with zero progress). Next: run the system suite with `-parallel` + synthetic CPU/IO soakers to reproduce deterministically, then trace `system.Start` → projectionhost subscribe/drain ordering; if the guard has a gap, fix at the projectionhost layer (ADR-0136 replay guarantee). **Second witness (2026-09-13, evening):** `TestEngineHealth_CatchUpUnderConcurrentApplies` failed once under the full metaengine package suite ("primary ticks = 2001, want exactly 2000"); passes 5/5 isolated (`GOWORK=off go test -tags goexperiment.jsonv2 -run TestEngineHealth_CatchUpUnderConcurrentApplies -count=5 .`) — same load-sensitivity class, unrelated to the reconciliation changes (no catch-up/failover code touched). _(Effort: M — needs a quiet or deliberately loaded machine)_

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
- [x] **Turso follow-ups from the IVM session (2026-09-11)** — DONE
      2026-09-13: (a) `check-turso-version.sh --self-test` (clean pass +
      planted stale citation caught, temp fixture); (b) `-tags ivmrepro`
      suite run with `-race` once (80.7s, green); (c) last chunk clamped to
      `TURSO_IVM_REPRO_ROWS` (no phantom tail rows); (d) one-command repro
      added to `docs/release-checklist.md` §6; (e) all three findings folded
      into the frozen upstream draft as a dated addendum. — source: 05-21
      §f2/§f4/§f7/§f8/§f12
      _(Effort: S total — done)_
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
- [x] **cqrs-lint cheap-fix follow-up tail (2026-09-11 session §f)** — DONE
      2026-09-13: (a) S001 allowlist corpus-validated both directions
      (`TestS001_AllowlistKeepsRealCredentials`: 6 real credential shapes
      still flagged; URLs/placeholders stay suppressed) + the existing
      taskmanager integration scan; (b) D014/D015 registry-acceptance tests
      shipped (parity with D016); (c) `TestB008_NonBitshiftStaysWarning`
      pins the non-escalation branch; (d) S001 selector-LHS messages now
      carry the receiver (`cfg.Password`) via `s001LHSDisplay`, no golden
      drift; (e) full-module `-race` green (18/18 packages — and it caught a
      REAL stale C009 golden: taskmanager grew 2 signing-key panics, honest
      re-pin 2→4); (f) `lintutil.IsURLOrPlaceholder` extracted with direct
      unit tests; S001 delegates. — source: 05-12 §f1-5/§f26
      _(Effort: S each — done)_
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
- [ ] **cqrs-upgrade strict-gate residual holes** — (a) run the
      deprecation scan even for NoPins modules (indirect-only cqrs consumers
      currently escape); (b) consider a `schemaVersion` field for the
      `--json` wire; (c) E2E test of `run()` against a fixture module
      (flags→report→strict exit codes). — source: 05-26 §e4/§f6-10
      (DONE 2026-09-13: --strict now fails on module errors — unscanned =
      unproven — and `bumps` is always-present in --json, symmetric with
      `deprecations`; both pinned by tests.)
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

- [ ] **Reconciliation-wave untagged surfaces (2026-09-13)** — `metaengine`
      (`Store.StreamCollection`), `commandlifecycle/projections`
      (`CommandsByActor` + query/result types), plus the regenerated API golden.
      Fold into the next tag wave when it is authorized; no release action
      before that. — source:
      [`docs/status/2026-09-13_18-35_…execution.md`](docs/status/2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md)
      _(Effort: XS note; M at tag time)_

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
- [x] **Private-dep mechanical guard + visibility audit** — DONE 2026-09-13:
      `scripts/check-private-deps.sh` (+ `nix run .#check-private-deps` + CI
      leg in lint-scripts) enforces: known-private blocklist (go-must),
      audited-public allowlist for every larsartmann require, examples
      public-only; `--audit` re-runs the live `gh` visibility sweep
      (2026-09-13 result: ALL 14 sibling repos PUBLIC — cmdguard,
      go-atomic-write, go-branded-id, go-codec, go-error-family, go-finding,
      go-flightrecorder, go-idempotency, go-ndjson, go-output, go-retry,
      go-sse, samber-do-auditlog, templ-components; 388 requires checked,
      0 violations; mutation-tested). Owner policy (c) stays open as policy,
      but the mechanical gate now enforces it. — source: 05-34 §e1/§f7-9
      _(Effort: S/M — done)_
- [x] **Release-tooling follow-ups (post-hardening)** — DONE 2026-09-13:
      `--smoke-all` (batch-release.sh, stop-on-first-failure); same-batch
      sibling limitation documented in the script header; `--verify`
      full-pipeline dry-run DECIDED AGAINST (documented in the header);
      CONTRIBUTING.md gained the batch-tagging section + retracts-gate ref;
      `path_matches_major` (plus `module_has_root_main`/`smoke_probe_args`)
      extracted to `scripts/lib/release_common.sh` (single implementation);
      BONUS: `check-retracts-shipped.sh` + acceptance tests,
      `tag-release --audit --baseline` + `scripts/audit-tag-baseline.txt`
      (24 known violations) + `check-tag-audit` CI leg,
      `scripts/smoke-probes.txt` explicit probes. All in
      `nix run .#check-release-scripts`. `check-release-scripts` in `#verify`
      NOT done (decide separately, ~30s cost). — source: 05-34 §e2/§e6/§f22-27
      _(Effort: S — done)_
- [x] **taskmanager tail from the go-must fix** — DONE 2026-09-13:
      `must_test.go` covers Must/Check happy+panic paths; census answered —
      the 0.08s run is 12 REAL in-memory tests, nothing skipped, no env
      markers; module-map.md example/* row records both facts. — source:
      05-34 §b3/§f5/§f6/§f31
      _(Effort: S — done)_

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
- [x] **Catch-up observability + tail replay** — SHIPPED 2026-09-13:
      `CatchUpState`/`Store.CatchUpSnapshot`/`EngineStats.CatchUp`/Doctor
      "--- Catch-Up ---" section cover the observability half; `Reset` doc
      comment now points at `CatchUpEngine` for the one-engine case.
      DECIDED against per-engine high-water marks (tail-only re-catch-up):
      full rebuild is idempotent by construction (engine is Reset first),
      while tail-only replay must assume engine state matches a persisted
      watermark — the exact stale-state class the 2026-09-13 catch-up race
      fix closed; watermark bookkeeping would also need ADR-0136 reset
      semantics ("reset the watermark too") and crash-recovery analysis.
      Revisit only if `Replayed`/`CompletedAt` observability shows rebuild
      latency hurting failover SLOs. DECIDED against an additive
      `CatchUpEngineWithResult`: the result is already observable via
      `CatchUpSnapshot`; the breaking `ResetResult` return waits for v5
      where signature changes are free. — source: 05-40 §e5/§e6/§f9/§f10,
      §f35/§f36/§f45; decisions 2026-09-13
      _(Effort: M — done)_
- [ ] **Doctor: per-entry-point synthetic-record feed counters** — the
      conformance sweep makes the entry-point record contract visible in
      TESTS; Doctor still cannot say WHICH entry point fed a synthetic
      (Type-only) record at runtime. — source: 03-50 §f17, 05-38 §f11
      _(Effort: M)_
- [x] **Legacy-log-entry synthesis pin** — DONE 2026-09-13: "Backfill legacy
      log entry" case added to the conformance sweep — a legacy
      `EventLog.Record()` entry replays to the synthetic Type-only view with
      the advisory at 0 (replays never count it). — source:
      03-50 §f23, 05-38 §c1/§f2
      _(Effort: S — done)_
- [ ] **Conformance-sweep + hot-path tail** — (a) DONE 2026-09-13: dedup
      no-op case shipped (`TestApplyIdempotent_DuplicateIsNoOp`: fold runs
      once, advisory counts once) PLUS the legacy `EventLog.Record()`
      synthesis case in the sweep table; remaining: (b) micro-bench the
      `applyFold` raw-payload type-assertion overhead on the struct hot path
      (defend the encoded-apply fix with a number); (c) run the new PG/MySQL
      ClaimMetrics integration tests against live servers
      (`#integration-pg`, `#integration-mysql-nspawn`).
      — source: 05-38 §b2/§f3/§f4/§f10
      _(Effort: S/M)_
- [ ] **scheduling/sqlstore hardening tail (carried from 03-50):**
      race-stress test DONE 2026-09-13 (`claim_race_stress_test.go`:
      4 pollers × 25 polls vs a Metrics reader, exact-once claim invariant,
      counters agree with observed claims; module green under `-race`);
      counter-scope pin DONE (`TestClaimingSQLite_CounterScope`:
      Schedule/Cancel/MarkFired never move counters). Remaining: property
      test (counters never exceed committed polls); fuzz `decodeDueTimer`
      corrupt-payload path; `RenewLease` ownership/claim tokens (code
      comment defers today).
      DONE 2026-09-13 (OTEL-OBSERVABILITY SUPERB): worked `Metrics()` →
      `/status` example + runnable OTel wiring for the ClaimMetrics hooks +
      process-start timestamp (`ClaimMetricsSnapshot.StartedAt`) — all in
      `example/scheduler-otel-status`. — source: 03-50 §f18-27, 05-38 §f19-27
      _(Effort: M, one rule per slice)_
- [ ] **storage/sql: dialect-aware `db.system` span attribute** — pebble and
      bbolt spans carry the OTel semconv `db.system` (via `cqrsotel.DBSystem`)
      since 2026-09-13; the SQL store needs its Dialect threaded into the
      package-level span helpers (`storage/sql/otel.go`) to stamp
      `sqlite`/`postgres`/`mysql`/`duckdb`. ~10 call sites.
      — source: OTEL-OBSERVABILITY plan M7 remainder
      _(Effort: S/M)_

---

## CI / Infrastructure

- [x] 🔥 **File-size ratchet RED** — RESOLVED 2026-09-15. The lintutil.go claim was
      STALE (file already back at its 453-line baseline); the live RED came from two
      NEW offenders, both split: `cmd/doc-check/recipes_catalog_meta.go` 433→335 (+new
      `recipes_catalog_meta2.go` 105; `recipeSpec` also moved out of the `_test.go`
      file so the module builds non-test again) and `queue/conformance/lifecycle.go`
      425→325 (+new `lifecycle_cancel.go` 110). `nix run .#check-file-size` GREEN.
- [x] **Zero the erraudit error-policy baseline** — DONE, verified live 2026-09-15:
      full per-module recount (`erraudit lint ./... --enforce-go-error-family
      --type-aware --format csv`) = **0 findings across all modules** (the 09-13
      CI-comment claim held; the 09-11 baseline of 253 was fully zeroed). The
      `error-audit` CI job's activation precondition (b) is met; (a) the
      `ERRAUDIT_PAT` secret remains a user action.
- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s; broken since ~2026-07-17. Local `nix run .#verify` remains the
      authoritative gate. _(Effort: S, user action)_
- [ ] [BLOCKED] **cqrs-lint Self-Lint credentials** — go-finding fetch fails
      under GOWORK=off (`git ls-remote` exit 128). _(Effort: S, user/creds)_
- [x] **Calibration-drift gate redesign** — DONE 2026-09-15. `calibration-drift.sh`
      gained `--baseline FILE` / `--write-baseline FILE` (CI compares apples-to-apples
      against a persisted `module|label|ns_per_unit` artifact from the same runner
      class — the benchmarks.yml baseline-artifact pattern — instead of failing on
      >100%-of-shipped-constant shared-runner noise) + TMPDIR filesystem detection
      (refuses btrfs/ZFS unless `CALIB_ALLOW_COW=1`; test hook
      `CALIB_FAKE_TMPFS_TYPE`). Also fixed a LATENT BUG: the constant lookup used a
      spaced assoc key (`CALIB[$mod | $label]`) that never matched, so the gate always
      exited 1 with "no shipped constant". Harness `test-calibration-drift.sh`
      (5 checks, incl. CoW refusal) wired into `check-release-scripts`. Remaining
      knob: wiring `--write-baseline`/`--baseline` into a nightly CI job that
      uploads/downloads the artifact. — source: archived/2026-09-04 §b2/§f16/§f18
- [ ] **pin-sweep `--check` nag semantics** — REMAINING: the trigger-policy DECISION
      only (keep blocking-on-every-push, or move to tag-push/cron?). Recommendation
      from 15-09 §g2 evidence: keep blocking-on-every-push (the nag is the sweep
      enforcement) and let cron report-only. DONE 2026-09-15: the extras —
      `--dry-run` (preview, mutates nothing), `--remote` (compares `git ls-remote
      --tags origin` instead of local refs; catches the tag-pushed-but-not-fetched
      blind spot where plain `--check` stays green), and fixture harness
      `test-pin-sweep.sh` (4 tests incl. the remote blind spot) wired into
      `check-release-scripts`.
- [x] **Cheap CI gates into pre-commit** — DONE 2026-09-15, plus the REAL bug found:
      `core.hooksPath=.githooks` is set while `.githooks/` DID NOT EXIST — every
      pre-commit gate was silently dead (git skips missing hooks), and
      `nix run .#install-hooks` wrote to the ignored `.git/hooks/`. Fixed the app to
      honor `core.hooksPath`, installed a live `.githooks/pre-commit`, and added the
      three staged-aware gates (version-drift + replace-directives on go.mod;
      module-layers on flake.nix/layer-script/module-add-delete — 0.01s/0.85s/7.9s).
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_
- [x] **Integration-tag lint as a first-class gate** — DONE 2026-09-15.
      `lint-module` takes an optional extra build-tag arg
      (`nix run .#lint-module -- <mod> integration`); new CI leg
      `integration-tag-lint` lints every module shipping `*_integration_test.go`
      WITH the tag (18 dirs → module roots resolved). First run immediately proved
      the point: queue/postgres findings hidden behind the tag surfaced (wrapcheck
      ×37 / wsl_v5 ×2 — owned by the queue session's in-flight work).
- [x] 🔥 **Kill the missing-go.sum-hash class in CI** — DONE 2026-09-15. The
      `check-modsums` flake app (`go mod tidy -diff`, no-write) already existed and
      was in `#verify`; added (1) the `modsums` CI job (plain setup-go, immune to
      nix-cache throttling) and (2) the repo-level meta-test
      `TestEveryModuleGoSumIsTidy` (cmd/api-stability; skipped under `-short`). The
      meta-test caught live drift on its FIRST run (a go.mod edited before its
      go.sum update — the exact class).
- [x] **Per-finding attribution for the sqlstore lint surface** — DONE 2026-09-15.
      Config for the sqlstore surface is UNCHANGED since the 09-06 pre-session commit
      (`f505ca1ed`): sqlclosecheck/QF1003 have no sqlstore exclusion at either point
      → those were CODE-FIXED (claiming.go/claiming_mysql.go refactors);
      wsl_v5's `_test.go` exclusion predates 09-06 (pre-existing policy, unchanged).
      Canonical binary: both gates use the nix pin (`${pkgs.golangci-lint}`);
      documented the rule (gotchas-tooling-build.md: `nix run .#lint-module` IS the
      ad-hoc canonical invocation). INVESTIGATION ALSO FOUND: `gci` had been
      silently RE-ADDED to formatters (auto-commit `7e711d32d`, 09-14) making the
      canonical lint gate red repo-wide while `nix fmt` was green — removed again
      (4th re-add of the §18 war class), and the depguard allow-list block silently
      DELETED (auto-commit `4a9855ed2`, 09-11, 127 files) — restored + re-verified
      against all 130 direct deps; `check-lint-config` green again. Both survived
      only because the gates that catch them had not run — new gotcha recorded:
      after any auto-commit wave touching `.golangci.yml`, run
      `nix run .#check-lint-config` before trusting lint results.
- [x] **`aggregate_*` tripwire: permanent mutation fixture** — DONE 2026-09-15.
      Fixture `cmd/api-stability/testdata/aggregatetripwire/planted.go` (planted
      `event.aggregate_not_found` + `storage.parse_aggregate_id` + negative control
      `listing.aggregate_projection`); new `TestAggregateTripwireScannerBites`
      asserts the scanner fires exactly on the planted pair and NOT on the control;
      shared the per-file scan logic with the repo-wide walk so they cannot drift.
      Mutation-verified: corrupting the fixture → red; restored → green.
- [x] **Live-verify the MySQL shuffle rollout + `-race` the dgraph retry code**
      — DONE 2026-09-15 with one documented caveat. Dgraph: full dgraphengine suite
      GREEN under `-race` (124 subtests, 0 races, 106.9s, seed 71958892287567) —
      `retryOnContention` now has race-detector coverage. MySQL: the VM shuffle
      rollout is verified LIVE (real seeds generated + logged + suites executed
      shuffled: seed 12105120945791 for stack/mysql), but a fully GREEN shuffled VM
      suite is still pending: both attempts died ~15s into the first suite with
      transport-level `invalid connection`/`connection reset` on DIFFERENT modules
      and DIFFERENT seeds (zero assertion failures) — the documented semi-dead-VM /
      host-contention class (gotchas-tooling-build.md), reproduced while a
      concurrent session's stalwart e2e VM + a my-run orphaned QEMU (killed,
      `vm-state-machine` cleaned, port 33070 freed) competed for the box. Rerun in
      the quiet window tracked by the [BLOCKED] `#verify` item.
- [x] **Document the shared-`dgraph.type` conflict domain + unit-pin
      `isContentionError`** — verified ALREADY DONE 2026-09-15 (stale TODO):
      gotchas-language-footguns.md (SetJson/dgraph.type all-writers-conflict,
      retryOnContention backoff, RunInTx caller-retry contract) + dgraphengine
      README "Concurrency & contention" section + `TestIsContentionError`
      (transaction_retry_test.go, 10 table cases: verbatim/wrapped/negative/
      case-variants) — all present and green. — source: 02-16 §e4/§f7/§f8

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
- [ ] **Skip-vs-fail classifier spread** — dgraph's live helpers now skip
      ONLY on server-unreachable and `t.Fatalf` otherwise (the honest-loud
      OQ-10 policy); the pg/mysql test helpers deserve the same classifier
      (same silent-skip class). — source: 05-51 §e6
      _(Effort: S)_
- [ ] **Watch dgraph + redis CI jobs (~10 shuffled runs)** — record any
      seed that fails; rare orderings WILL eventually appear in CI (that is
      the point of shuffling). — source: 02-16 §e7/§f10
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
      (d) STILL OPEN: go.work sync check
      job, benchmarks.yml matview-gate dry-run (relative
      `cd ../metaengine/tursoengine` hop unproven). The test-tag-release.sh
      SC2086 item was already stale — the script uses the array form
      `git "${notag[@]}"` and shellcheck is clean (verified 2026-09-13). — source: run
      34548534824, run 34747274058, `gh run list`
      _(Effort: M-L, multi-session; the cache-backend migration is the
      single highest-leverage repair)_

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

- [ ] **T23 — upstream skill-maintenance pass** (the plan's one open task):
      docs/reviews↔brainstorming divergence; read-prior-reports +
      copy-template steps in the review skills. Execute or decline at the next
      skill-maintenance window. _(Effort: S)_

---

## Docs / consumer-surface truth

> Consumer-facing contracts that live only in CHANGELOG or doc comments are
> invisible to consumers reading the skill references.

- [ ] **Skill references: reset recipe covers ALL engines** — SKILL.md +
      references still describe `Store.Reset` as memory-only; the ladder is
      12/12 now. Update the reset recipe + add the
      `WithContentionObserver` entry to recipes.md. — source: 05-51 §f33
      _(Effort: S)_

---

## Event-Query-Model reconciliation follow-ups (2026-09-13)

> The 2026-07-23 design doc was reconciled against source (status banner + per-section addendum
>
> - coverage map); `StreamingScan` was wired (`Store.StreamCollection`) and the per-actor
>   lifecycle projection shipped the same day (see CHANGELOG). Carry-forward items below. Source:
>   [`plan`](docs/planning/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md).

- [ ] 🔥 **Distinct `command.rejected` event + errorfamily classification** — a business
      rejection currently surfaces as `command.failed` with error text; audit cannot tell
      "rejected by rule" from "broke". Needs a classification contract (which families count as
      rejection) + recorder method + middleware wiring. Payload capture stays opt-in/out.
      — source: [`T17 memo`](docs/planning/2026-09-13_T17-memo-command-log-audit-scope.md) _(Effort: M)_
- [BLOCKED] **Session-log boundary decision** — memo recommends sessions stay external
  (`cqrs-htmx/identity-model`) and NOT fold into the planned `queue/` module; revisit only on
  a concrete audit consumer. — source: [`T18 memo`](docs/planning/2026-09-13_T18-memo-session-log-boundary.md) _(Effort: XS decision)_
- [ ] **Verify Set-membership pushdown for SQL engines** — the doc's "UNIQUE index" Set claim was
      never source-verified (audit item 30). _(Effort: S)_
- [ ] **Verify graph traversal depth semantics** — the doc's `FriendsOf{Depth}` vs the shipped
      traversal implementation (audit item 31). _(Effort: S)_

---

## Quick-win batch follow-ups (2026-09-13)

> Tail of the 2026-09-13 ten-quick-win batch — small completeness gaps found
> by the session's own self-review. — source:
> [`docs/status/2026-09-13_08-52_quick-win-batch-self-review.md`](docs/status/2026-09-13_08-52_quick-win-batch-self-review.md) §b/§f

- [x] **calibration-gate.sh `--self-test` mode** — planted loadavg fixture
      (temp file, never a live tracked file); same class as the
      check-turso-version `--self-test` TODO. _(Effort: S)_
      2026-09-15: DONE — 8-check fault-injection suite (`CALIB_GATE_LOADAVG_FILE`
      env hook), wired as 4th leg of `nix run .#check-release-scripts` (CI
      `lint-scripts`); shellcheck clean.
- [x] **Recipes snippet compile harness** — snippets are reference-verified
      (doc-check) but not compile-verified; extract fenced Go blocks into a
      generated compile test (start with recipes.md). _(Effort: S/M)_
      2026-09-15: DONE for recipes.md — `cmd/doc-check` recipes harness:
      77/77 blocks classified (69 compile-scaffolded, 8 documented skips),
      full-coverage ratchet prevents new/rotting fences; caught 9 real doc
      lies (Plan variadic-spread, retry.Config Jitter, catalog exporter
      chains, typed Timer.Actor, BasicCommand embedding, …) — all fixed.
- [x] **Calibration-gate failure-message golden** — the operator-facing
      FAIL text is UX; pin its shape so refactors can't silently degrade it.
      _(Effort: XS)_
      2026-09-15: DONE — `scripts/testdata/calibration-gate-fail-message.golden`
      (uptime line normalized); mutation-tested (corrupt → self-test fails →
      restore → green).

---

## Skill-docs navigation hardening (2026-09-13)

> Fall-out of the 2026-09-13 skill-docs audit (14+9 defects fixed live; the
> validation logic below exists only as throwaway session scripts today).
> — source:
> [`docs/status/2026-09-13_08-47_skill-docs-audit-metaengine-goal-readiness.md`](docs/status/2026-09-13_08-47_skill-docs-audit-metaengine-goal-readiness.md) §e/§f

- [ ] **Anchor + § cross-ref validation in CI** — the audit found 2 silently
      broken TOC anchors (advanced §6.8, faq eventtest) and duplicate recipes
      section numbers (2× §2.13/§2.22/§2.23) that doc-check cannot see; port
      the session's GitHub-slugger + §-ref checker (Python, ~40 lines) into
      `scripts/check-doc-links.sh` or `cmd/doc-check`. Keep the slug rules
      GitHub-exact: underscores kept, punctuation stripped, inline-code
      content KEPT. _(Effort: S)_
- [ ] **doc-check arity spot-check** — both critical doc lies
      (`system.New(ctx, system.Deployment{…})`, `(ctx, deployment,
      domains...)`) passed symbol-level validation; parse fenced-Go call
      shapes for exported constructors and compare against go/doc arity.
      _(Effort: M)_
- [ ] **Consolidate the v5-deprecation story** — told in 6+ places (SKILL.md,
      core.md ×2, readmodels.md, faq.md, modules.md rows); one canonical
      block + pointers kills the next drift at the source. _(Effort: S)_
- [ ] **Discoverability: link `example/metaengine-quickstart`** from README.md
      and the metaengine module README (currently only reachable via the
      skill docs); it is the flagship "operator cqrs.yaml" goal demo and all
      three example binaries verified runnable 2026-09-13. _(Effort: XS)_
- [ ] **Decide `metaengine.Infer(samples…)` end-state** — docs steer to
      `OnRecord`/`AutoInsert` for production and call Infer prototyping-only;
      either deprecate at v5 (consistent with the steer) or promote it with a
      docs story for why it stays. _(Effort: XS decision, S if deprecated)_

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
