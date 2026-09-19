# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task finishes it moves to CHANGELOG
and its entry is deleted from this file. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29, 2026-09-06 ×2, 2026-09-08, 2026-09-11, and 2026-09-16). The Declined section at the
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

## Metaengine Universal Storage Substrate (proposed 2026-09-18)

> Owner directive 2026-09-18: metaengine becomes the ONE way data is stored/retrieved from disk.
> Full Pareto plan (23 tasks / 82 micro-tasks): [`docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md`](docs/planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md)

- [x] 🔥 **T01+T02: ADR-0142 + capability contracts** (the 1% → 51%) — DONE 2026-09-19: ADR-0142 written; `DueClaimer`/`DedupStore`/`FactSink` shipped as v4.x capability interfaces + `adttest` conformance + errorfamily codes + Supports entries (see CHANGELOG 2026-09-19 ADR-0142 entries).
- [x] 🔥 **T03–T08: conformance + sqlite/postgres/memory reference impls + `scheduling/engine` TimerStore facade + Scheduler wart fixes** (the 4% → 64%) — DONE 2026-09-19: claimkit ONE database/sql runtime (sqlite/pg/mysql/duckdb/turso native), Map runtimes (memory/pebble/bbolt/badger degraded), `scheduling/engine` facade with epoch-guarded MarkFired, family-aware scheduler retry; all conformance green incl. `-race`.
- [x] 🔥 **T09–T17: full absorption** (the 20% → 80%) — DONE 2026-09-19: queue engines register as drivers (sqlite/postgres/mysql — family complete), idempotency `NewFromEngine` facades, `system` TimerEngine/ManageTimers/Timers + persistent checkpoints, ALL engines implement-or-refuse (`RefusedADTs` universality rule: dgraph/bigtable/iroh-wrapper refuse with reasons — never silence), FactSink-in-tx as claimkit capability with same-tx conformance, Doctor/capability-audit rendering + audit rule 4, reset ladder covers claimkit collections (journal positions keep advancing — pinned per engine). Docs rows (modules.md/FEATURES/module-map) shipped.
- [x] **T17 remainder: SKILL recipes §2.x excerpt** — one engine-backed-timers/queue/dedup recipe block in recipes.md + recipes_catalog classification + doc-check green. DONE 2026-09-19: recipes.md §2.38 (3 compile-verified blocks, catalog 77→80, doc-check + TestRecipes green).
- [x] **T18a: claimkit micro-benches vs direct-SQL baseline** — DONE 2026-09-19: `metaengine/claimkit/bench_test.go` (claim steady-state, composite timer round-trip, dedup fresh/live-window; ClaimKit vs hand-composed direct SQL). Measured: the runtime is FASTER than the direct path on every pair (claim 63µs vs 82µs, dedup 11µs vs 19µs on the dev machine) — no abstraction tax. Four claimkit benches joined the `benchmark-regression.sh` gate set.
- [x] **T22: `example/taskmanager` on engine-backed queue** — DONE 2026-09-19: the deriver's auto-assign cascade rides `queue/sqlite` (dedup-keyed enqueue, lease-fenced worker, backoff retries, dead-lettering) instead of a fire-and-forget goroutine; restart-durability + end-to-end tests green `-race`, live demo run verified. Sibling replaces in the example go.mod ride until the family tag wave.
- [x] **T23: go-taskqueue semantic-diff probe** — DONE 2026-09-19: [`docs/research/2026-09-19_go-taskqueue-semantic-diff.md`](docs/research/2026-09-19_go-taskqueue-semantic-diff.md) — donor contract (read in full) vs library: core semantics 1:1, divergences are strengthenings (claim tokens, dep validation), genericity (typed payloads), or consumer-owned product surface. No silent drift; P5 input conclusion: upstreaming subtyped, not forked.
- [ ] **T18b: load-sweep + benchmark baseline regen under Go 1.27** — `#load-sweep` on timing paths and a `benchmark-regression.sh --save` refresh (the committed baseline predates the 1.27 toolchain AND now needs the new claimkit entries). Quiet-window gated: only meaningful on a machine under ~10 load; the concurrent-session reality has kept load at 28-74 all day. _(Effort: S once the window opens)_
- [x] **Lint debt → zero (ADR-0142 tail)** — DONE 2026-09-19: the real baseline was 47 findings + 6 unlintable modules (exhaustruct_v5 v5.0.3 panics on promoted-field literals — rewritten in keyed form; see CHANGELOG 2026-09-19). All 88 lint-gated modules re-verified 0 findings per-module with the canonical invocation; `.golangci.yml` dropped the graduated jsonv2 tag and aligned `run.go` to 1.27.1; traps (silent standalone-toolchain death, exhaustruct panic shape, multi-line nolint placement) documented in gotchas-tooling-build.md.
- [ ] **T19–T21 (v5-gated): fold capabilities into universal `Engine`, delete the duplicate SQL stacks, release train** — blocked on the v5 train per ADR-0142 §decision; DO NOT execute in v4.x (growing core interfaces is breaking, contract 21g discipline). The tag waves for claiming + the queue family are the separately-tracked item below. _(Effort: L; v5-gated)_
- [ ] **Go 1.27 follow-ups** — the 94-module `go 1.27.1` sweep completed the jsonv2 graduation (2026-09-19, un-broke every workspace-mode compile); the coordinated tag sweep (flake.nix/scripts/CI/workflows/Go-strings/docs → plain `go build`) landed 2026-09-19. Remaining: re-baseline load-sweep benchmarks under the 1.27 toolchain. _(Effort: S)_

## Durable Work Queue module (proposed 2026-09-13)

- [x] 🔥 **Assemble the existing pieces into a `queue/` sibling module** — DONE through M4 2026-09-19: P0 `claiming/` extraction (2026-09-13); M1–M3 contract + conformance suite + sqlite/postgres engines green `-race` (2026-09-14/15); **M4 (2026-09-19): T14 enqueue dep validation (`queue.ErrDanglingDep`, cycles unrepresentable by construction — store-minted IDs + existence check; no unblock-bump: claim-time gating + bounded aging cover it), T15 ADR-0134 claim tokens (`queue.Claim.Token` + `queue.NewClaimToken`, `lease_token` column + migrations both engines, finalize signatures take the token, theft = `ErrLeaseNotHeld`; ADR-0134 Accepted for queue/), T16 `queue.FactTx`/`FactSink` same-tx consumer fact appends + `Store.Watermarks` list, T17 `queue/mysql/v4` third engine (two-statement SKIP LOCKED, BIGINT-ms, InnoDB deadlock retry, nullable-unique dedup emulation) green on live MariaDB 11.4 incl. `-race -count=2` (gate: `MYSQL_TEST_DSN`)**. ALL THREE engines green on the shared suite incl. `-race`. Dedup'd enqueue + priorities/aging were already M1–M3. Spec source of truth = go-taskqueue's production-proven `internal/queue.Store` contract (upstreamed, not reinvented). Consumers: go-taskqueue (reference donor), PapDashboard (production worker pools today), `example/taskmanager` (demo→real). — source: [`docs/planning/2026-09-13_durable-work-queue-module.md`](docs/planning/2026-09-13_durable-work-queue-module.md) + queue-arc reports 14-14/14-46 + 2026-09-19 M4 report _(Effort: P1+M4 DONE)_
- [ ] **Tag the claiming + queue modules** — `claiming/v4.0.0` (dry-run READY, blocked on clean tree at the time; needs sqlstore replace pin + standalone build gate + proxy probe), then `queue`/`queue/sqlite`/`queue/postgres`/`queue/mysql` v4.0.0; also strips the `queue/{sqlite,postgres,mysql} => ../queue` sibling replaces. NOTE 2026-09-19: the T15 token change altered `queue.Store` finalize signatures pre-release — tag the WHOLE family in one wave so the golden and the modules move together. — source: queue-arc 14-14 §b2/§b3 _(Effort: S each once a wave is authorized — fold into the next tag wave)_
- [x] **Run `queue/postgres` conformance against live in-repo PG** — DONE 2026-09-16: `PG_MODULES="queue/postgres" TEST_TIMEOUT=420 nix run .#integration-pg` full suite PASS on the repo's own ephemeral PG (first run on this leg), incl. the `lifecycle_cancel.go` split end-to-end. — source: 2026-09-16 09-35 report §f2; 2026-09-16 15-02 report §c _(Effort: S)_
- [x] **Queue family docs + config tail** — DONE 2026-09-16, all four: (a) new `queue/README.md` (contract, module table, quickstarts; every claim source-verified before shipping); (b) `task.New`/`queue.Filter`/`facts.Fact` partial-literal semantics documented in that README (exhaustruct exemptions design-justified); (c) dead errcheck exclude-functions short forms replaced with fully-qualified `(*database/sql.DB|Rows|Stmt).Close` forms in `.golangci.yml` (probe found the short forms were dead); (d) `queue/mysql` doc mentions struck (`queue/store.go`, `queue/conformance/doc.go`) — the engine does not exist, so the mention lied. — source: 2026-09-16 09-35 report §e4/§f6-9 _(Effort: S each)_

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

- [ ] **W4: gates** — per-module `GOWORK=off` tests (decider/command/commandlifecycle/schema), api-stability golden + `TestEvery`, CHANGELOG `pkg.Symbol` citations, doc-check zero-warning, `nix run .#verify`, `#check-arch` (decider: zero new deps), `#check-duplication` (0 new groups). STATE 2026-09-16: every leg green EXCEPT the composed `nix run .#verify` (quiet-window-gated; load never dropped below ~30) — per-module tests, api golden (7,092 exports), TestEvery, check-changelog-symbols (86 citations), doc-check (1,142 refs), check-arch, check-duplication all re-run green. _(Effort: M — in progress)_
- [BLOCKED] **Release train (user approval)** — tag waves decider/command/commandlifecycle once W1–W4 land. _(Effort: M — see AGENTS.md tag-wave procedure)_
- [BLOCKED] **ADR-0138: command sourcing draft (consumer demand)** — design doc only, builds on W2's bridge, reconciles ADR-0112's planned `CommandAwareFold`. _(Effort: M)_

## Go 1.27 upgrade wave (proposed 2026-09-13)

- [ ] 🔥 **Toolchain + go-directive wave to Go 1.27** — Go 1.27 (2026-08-19; 1.27.1 2026-09-01) graduated `encoding/json/v2` (v1 now backed by the v2 engine — the `-tags "goexperiment.jsonv2"` footgun dies repo-wide, clearing ~20 live gopls `stdversion` warnings) and legalized **generic methods** (method-level type params; interface methods still can't). Scope: bump all 85 `go.mod` `go` directives (language features are directive-gated), flake `goToolchain` pin to nixpkgs `go_1_27` (verify availability first — CONFIRMED 2026-09-18: `nixpkgs#go_1_27` = 1.27.1; repo flake currently resolves default go = 1.26.7), CI, AGENTS.md/docs command chains, then full `#verify` + integration suites + bench-regression sweep (v2 unmarshal is significantly faster — expect improvements), and a release train so consumers actually receive it. Consumer impact: `go` directive ≥ 1.27 forces toolchain download on older setups (`GOTOOLCHAIN=auto` mitigates). Sequel: revisit `decider.ExecuteCommandRef` as a true `Repository[State]` method (additive), plus the other option-func families. — evidence: go.dev/doc/go1.27 release notes, plan D1 amendment. _(Effort: L — own wave, do NOT fold into other plans)_

## Investigate: `TestSystem_ResetProjection_RestartAndReplay` contention stall (found 2026-09-13)

- [x] 🔥 **Suspected replay-starvation race in system.Start's projection path under extreme parallel load — RESOLVED 2026-09-19, NOT a race**: root cause was `EngineResetter.ResetEngine` deleting the journal tables (ADR-0143 fix) — phase 1's `ResetProjection` wiped `meta_stream_log`, so phase 2's fresh system replayed from an empty journal (`journal holds 0 event(s) from sys2's view` was the literal smoking gun visible in every failure dump). It failed "only under full verify" because verify runs WORKSPACE mode (local engine trees with ResetEngine) while the passing module-isolated runs were `GOWORK=off` against PUBLISHED engine versions that predate ResetEngine — version-skew masking, not load. Fix: all engines' resets now preserve the journal (facts survive resets); every engine reset test pins journal survival. Lesson recorded in gotchas-testing: after touching cross-module contracts, run the module in BOTH workspace and `GOWORK=off` modes. The "second witness" (`TestEngineHealth_CatchUpUnderConcurrentApplies`, ticks 2001-vs-2000 under full-suite contention) remains a separate, genuinely load-sensitive timing test — not the same class.
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
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). NEXT STEP (SUPERB S28): design one-pager for `AggregateOn(fn, column, group)` on `QueryDecl` — the declarative seam the planner can read — then routing v1: scalar-covered shapes price O(1) (matview-served), grouped shapes stay O(N) with a Doctor note (upstream defect A makes grouped routing unsafe). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A — scope the first cut to scalar-covered shapes only. — source: archived 19-25 §f29, 05-33 §f32, SUPERB S28/05-51 §f16-17
      _(Effort: M)_
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
- [x] **Extend the error-taxonomy drift gate beyond its 5 modules** — DONE 2026-09-16
      (two waves): watermill, storage/pebble, core event/command/query landed
      earlier; this wave added storage/view (30 codes), stack incl. all 14
      preset prefixes (80), deriver (2), and storage (SQL facade) as FOUR
      extraction entries over the storage module's same-module subpackages
      root+sql+eventstore+readmodel (82 codes; `--max-depth 1` support added
      to the entry schema for the root scan). 18 modules / 519 codes / 450
      doc claims green + `--self-test` green. Real drift FOUND+fixed:
      `storage.scan_command`/`scan_query` are Infrastructure (doc claimed
      Corruption via `storage.scan_*`), and `storage.schedule_timer` was
      minted with TWO families — the marshal site now mints
      `storage.schedule_timer_marshal` (Corruption) so one code = one family;
      per-module pool-size floors wired (view 10, stack 50, deriver 2,
      facade 30/24/22/6); the floor check IS the explicit extraction
      assertion (0 codes ⇒ floor trip). —
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
- [x] **Kill the self-lint false-green class at the root** — DONE 2026-09-18:
      `analyzer.IsLibrarySelfLint` now treats `example/*` modules as consumers
      (new `IsExampleModulePath` guard; api golden +1 export); V007 and the
      F-family coaching rules run on examples in place. `TestExamples_AreV5Clean`
      simplified — the throwaway consumer-copy shim is DELETED, the scan runs
      against the real example dirs, and an analyzed-file-count assert (the
      02-47 lesson) fails loudly on a zero-file scan. First honest taskmanager
      profile: +E014/F004/F013/F021(×2)/F026/F028 coaching findings pinned in
      both goldens (no criticals); full cqrs-lint (19 pkgs) + cqrs-upgrade
      suites green; TestEvery green. The V007 typed-method-detection sibling
      (`types.Info.Selections`) remains a v5-cut decision (v007.go comment).
      — source: 05-26 §e2/§f18-20
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
      [`docs/status/archived/2026-09-13_18-35_…execution.md`](docs/status/archived/2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md)
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
      local replaces in the same wave.** **NEW 2026-09-16 preconditions:** (1)
      re-tag `otel/v4` carrying `DBSystem` BEFORE any storage/v4 tag — the
      published storage would otherwise reference an unpublished symbol
      (broken for consumers until otel re-tags); (2) the wave also strips the
      newer sibling replaces: `cmd/cqrs-bench => ../../benchkit`
      (statistical-rigor APIs: `RunRepeated`, `MetricVariation`,
      `WriteBenchstatRepeated`), `queue/sqlite` + `queue/postgres => ../queue`,
      `commandlifecycle/projections`, and bumps `metaengine`
      (`Store.StreamCollection`) + `commandlifecycle/projections`
      (`CommandsByActor`). Order constraints per CONTRIBUTING
      pre-tag checklist; cut→push→next interleave (GOPRIVATE resolves siblings
      via VCS). — source: 08-26 §c3, 15-09 §f47, SUPERB §f14-16; 08-04 §b5 +
      02-09 §f3 + 09-35 §f18 (2026-09-16 additions)
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

---

## Metaengine — follow-ups

- [ ] 🔥 **Planned-table filter pushdown silently returns empty for camelCase fields over snake_case json keys** — found 2026-09-18 in the Ledger CRM: a projection registered with `metaengine.FilterOnField[R]("ParentID", FilterEq)` gets a planned layout whose columns are named by the GO field names (`ParentID`, `DueAt`), but the planned-table writer populates those columns by extracting the stored JSON document with the same camelCase key — while the documents use the view's JSON TAGS (`parent_id`, `due_at`). Result: `ParentID`/`DueAt` columns stay NULL and every pushdown filter (`system.Where("ParentID", …)` → `WithFilter` → planned `WHERE "ParentID" = ?`) matches nothing, silently, no error. Repro: register a projection with a Filterable camelCase field, write a record, then `Find[R](Where("Field", v))` → 0 rows while unfiltered Find returns the row. Fix direction: planned-column extraction must use the same field-to-json-key mapping the encoder uses (or jsonPath must fall back field-name AND snake_case); add a parity test that every Filterable field round-trips through pushdown. CRM works around it today by filtering in Go (`tasksForParentID` in `internal/httpapi/helpers.go`) — delete that workaround once fixed. — source: 2026-09-18 CRM pareto-execution session, M9 (deal detail tasks panel); verified against live sqlite db: `meta_planned_task_views` row had `ParentID=NULL` while `value.parent_id` was set _(Effort: M — canonical key mapping, extraction fix, pushdown round-trip test)_

- [ ] 🔥 **Port ctx-scoped transactions from sqliteengine to pg/mysql/duckdb engines (same engine-global `activeTx` leak)** — found 2026-09-18 while fixing a production flake in the Ledger CRM (`sql: Rows are closed` on timeline loads): sqliteengine's `xc()/xd()` resolved the "active tx" from an engine-global `atomic.Pointer`, so any engine call from an unrelated goroutine landing inside another caller's `RunInTx` window was handed the foreign `*sql.Tx` — dirty reads of uncommitted writes, and readers dying with `sql: Rows are closed` when the foreign tx committed mid-iteration. FIXED in sqliteengine (commit `22ab7b218`): the tx now travels in the ctx under `txMarker{}` (value = `*txExecutor`), `xc(ctx)/xd(ctx)/txExec(ctx)` resolve affinity from ctx only, `activeTx` field deleted; regression-pinned by `metaengine/sqliteengine/tx_isolation_test.go` (`TestSQLiteEngine_TxIsolationFromForeignContext` — deterministic dirty-read guard, red before/green after — plus `TestSQLiteEngine_ConcurrentStreamReadVsAppendExpected`). pgengine (`engine.go:70`), mysqlengine (`engine.go:52`), and duckdbengine (`engine.go:55`) still use the identical engine-global pattern and have the same hazard; port the marker-carrying ctx + accessor refactor and the isolation test to each. NOTE for the CRM: its flake input pins go-cqrs-lite at `a0b5429a…` — the pin must be bumped to include `22ab7b218` (post-push) before the next `nix build` release binary. — source: 2026-09-18 CRM pareto-execution session, baseline-verified via worktree at `22ab7b218~1` _(Effort: M per engine — mechanical port + test)_
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

- [x] **`TestSystem_ResetProjection_RestartAndReplay` is load-fragile —
      RESOLVED AS SPECIFIED 2026-09-18: the structural fix was already in** —
      the shared-cache in-memory DSN hypothesis was stale: `5d66308c3`
      (2026-08-16, before the 09-13 failure observations) had already switched
      the test to a real temp-FILE DSN (`sqliteFileDSN`, `system/testdsn_test.go`),
      so the 09-13+ stalls were never the DB-destruction class. Two escalated
      storm repros on 2026-09-18 failed to reproduce: full system suite at
      load 38.6 (plain) and `-race -count=2` (338 passes) at load 60-111 —
      the stall needs the full-verify package storm (all-module + builds).
      Diagnostics added to the phase-2 failure path (full worker state:
      status/restarts/checkpoint/lastError + a direct `ReadFrom` against the
      event store) so the NEXT verify occurrence either passes or names the
      fork ("journal holds N events from sys2's view" vs worker idle). The
      residual subscribe-vs-drain stall risk lives in the 🔥 Investigate
      section above; budget margins deliberately NOT bumped further.
      _(was Effort: M, owner: system area)_

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
- [x] **pin-sweep `--check` nag semantics** — DECIDED 2026-09-18 per the
      15-09 §g2 recommendation: KEEP blocking-on-every-push (the nag is the
      sweep enforcement; `ci.yml` `pin-sweep.sh --check` unchanged) and let
      the nightly cron run REPORT-ONLY (`--check --remote`, warning on
      drift, in `nightly-gates.yml`). DONE 2026-09-15: the extras —
      `--dry-run` (preview, mutates nothing), `--remote` (compares `git ls-remote
      --tags origin` instead of local refs; catches the tag-pushed-but-not-fetched
      blind spot where plain `--check` stays green), and fixture harness
      `test-pin-sweep.sh` (4 tests incl. the remote blind spot) wired into
      `check-release-scripts`.
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo + nix `vendorHash` cascade + full CV
      verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_

- [x] **irohengine `/demo/` fmt.Printf pre-commit blocker — RESOLVED
      2026-09-18** by the canonical `scripts/pre-commit.sh` (TODO
      "pre-commit hook hardening"): the fmt.Printf gate now excludes
      `/demo/` paths alongside tests/examples/testdata/cmd. Verified
      empirically: the exact hook pipeline over the full tree returns zero
      violations with `metaengine/irohengine{,/quic}/demo/main.go` present
      (those files still contain `fmt.Printf`, by design). The old
      `--no-verify` workaround for docs-only commits touching them is dead.

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
- [x] 🔥 **Root-cause the `.golangci.yml` config-corruption loop** — CLOSED
      2026-09-18 with both halves done. (a) CULPRIT: the auto-commit daemon's
      fmt waves — every incident commit is `chore: auto-commit` (7e711d32d,
      d54cd38a7, c56d219a6, and c55e21fa8 caught live mid-session committing a
      header-only deletion); the inner tool is unconfirmed (BuildFlow
      auto-configure is the prime suspect — see gotchas-tooling-build.md for
      the full fingerprint). (b) SELF-HEAL SHIPPED: `scripts/restore-depguard.sh`
      + pinned golden `scripts/depguard-block.golden.yml` restore the block
      when it vanishes (restore-on-empty only; partial shrinkage fails loudly
      for a human; legit growth auto-refreshes the golden), wired into
      check-depguard → `check-lint-config`, plus a `.golangci.yml` staged
      trigger in the pre-commit hook that re-stages the repair. gci already
      self-healed via check-formatters.sh. Mutation-tested (corrupt → repair
      → byte-verify; shrinkage → exit 1; growth → golden refresh). Sixth AND
      seventh incidents (both 09-18) would now self-repair on the next gate
      run. — source: 2026-09-16 09-35 report §e1/§e2/§f4-5 _(Effort: M — DONE)_
- [x] **Pre-commit hook hardening batch** — ALL FOUR DONE 2026-09-18.
      (a) ONE canonical hook: `scripts/pre-commit.sh` is the source of truth,
      installed to `.githooks/pre-commit` (hooksPath) by `nix run
      .#install-hooks`, which now SETS `core.hooksPath .githooks` itself; the
      old BuildFlow heredoc writer is gone — BuildFlow is CHAINED inside the
      canonical hook (when the binary is present), so `buildflow precommit
      install` can no longer wipe the repo gates. The `/demo/` drift between
      source and installed copy is gone (reinstalled from source).
      (b) fresh clones boot: the `nix develop` shellHook installs the hook +
      sets hooksPath on first entry. (c) `.golangci.yml` staged trigger runs
      the self-heal pair and re-stages the repaired config. (d) the fmt gate
      is now staged-SCOPED and self-fixing (`nix fmt -- <staged files>` +
      re-stage) — honest commits in multi-writer trees no longer hit other
      sessions' in-flight files, and doc-only commits skip the code gates.
      — source: 08-05 §b6/§f8-10; 18-19 §e3 _(Effort: S/M — DONE)_
- [x] **Nightly gate cron + calibration-baseline CI wiring** — SHIPPED
      2026-09-18 as `.github/workflows/nightly-gates.yml` (cron 04:00 UTC +
      workflow_dispatch): `check-lint-config` (fails loudly + prints the diff
      when the self-heal had to repair, so the incident is visible within
      24h), `check-modsums`, `check-release-scripts` (script harnesses),
      report-only `pin-sweep --check --remote`, and the calibration artifact
      loop (rolling `actions/cache` baseline: night N compares `--baseline`,
      then refreshes via the new `nix run .#calibration-drift --
      --write-baseline` flake app). Deliberately NO nix cache action (the
      throttled dependency; see the CI triage item). REMAINING: first real
      run needs the Actions billing fix — until then it arms silently like
      the rest of CI. — source: 08-05 §b3/§e4/§f6/§f11 _(Effort: M — wiring DONE)_
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
- [x] **Pin the recipes-gate CI posture** — DECIDED 2026-09-18, one grep as
      promised: `cmd/doc-check` IS in `testModules` (flake.nix), and the CI
      `discover-modules` matrix runs `GOWORK=off go test ./... -count=1 -race`
      in EVERY go.mod module on every push — so `TestRecipesCompile` +
      `TestRecipesCatalogCoversFile` already run per-push cold-cache; the
      separate `check-recipes-compile` flake app would be duplication and is
      NOT added. Dev machines get the same gate inside `#verify`'s doc-check
      leg (warm ≈ 5-15 s; run `#verify` before tagging, per AGENTS.md). No
      code change needed — the posture was already correct. — source: 18-19
      §b1/§g1 _(Effort: XS decision + S — decision only)_

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
- [x] **Skip-vs-fail classifier spread** — DONE 2026-09-16: `pgSkipClass` /
      `mysqlSkipClass` (dgraph's `dgraphSkipClass` policy) now gate ALL
      pgengine construction helpers (testcontainer ×2, copy ×1) and
      mysqlengine sites (helper, layout ×2, planned-ops factory, internal
      graph helper): server-unreachable skips, everything else `t.Fatalf`
      "not a skip-class error". Partition pinned by unit tests in both
      packages (+ internal twin). — source: 05-51 §e6
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

- [x] **Skill references: reset recipe covers ALL engines** — DONE/STALE 2026-09-16:
      `readmodels.md` reset section (lines 319-337) ALREADY documents the 12/12
      `EngineResetter` ladder (no "memory-only" `Store.Reset` text exists
      anywhere in the references); the `WithContentionObserver` half shipped as
      recipes §2.36 "Watch Dgraph Contention Retries" with a compile-verified
      scaffold (`TestRecipes` green). — source: 05-51 §f33; 2026-09-16 15-02 report
      _(Effort: S)_

---

## Event-Query-Model reconciliation follow-ups (2026-09-13)

> The 2026-07-23 design doc was reconciled against source (status banner + per-section addendum
>
> - coverage map); `StreamingScan` was wired (`Store.StreamCollection`) and the per-actor
>   lifecycle projection shipped the same day (see CHANGELOG). Carry-forward items below. Source:
>   [`plan`](docs/planning/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md).

- [BLOCKED] **Session-log boundary decision** — memo recommends sessions stay external
  (`cqrs-htmx/identity-model`) and NOT fold into the planned `queue/` module; revisit only on
  a concrete audit consumer. — source: [`T18 memo`](docs/planning/2026-09-13_T18-memo-session-log-boundary.md) _(Effort: XS decision)_
- [x] **Verify Set-membership pushdown for SQL engines** — DONE, double-verified (2026-09-15 doc
      banner + independent re-check 2026-09-16): TRUE for SQLite, the only SQL engine with the Set
      ADT; `metaengine/engine.go:453` SetContains = `SELECT 1 FROM meta_set WHERE collection = ?
      AND key = ?` (sqliteengine/engine.go:130), uniqueness via the composite PRIMARY KEY
      (sqliteengine/engine.go:92-95), not a separate UNIQUE index; pg/mysql declare ADTSet
      degraded (no meta_set DDL). _(Effort: S)_
- [x] **Verify graph traversal depth semantics** — DONE, double-verified (2026-09-15 doc banner +
      independent re-check 2026-09-16): `FriendsOf{Depth}` ships as `Engine.GraphNeighbors` with
      within-≤depth-hops semantics on all four graph engines (sqlite CTE sqliteengine/graph.go:40-47,
      pgengine/graph.go:28-35, mysqlengine/graph.go:39-46, memory BFS memory_read.go:40-56), dedup,
      start excluded, negative = unlimited. _(Effort: S)_

---

## benchkit statistical-rigor tail (2026-09-16)

> The 02-09 session shipped P100 exactness, `RunRepeated`/`RepeatedResult`,
> per-metric `MetricVariation`, real benchstat samples, and load provenance —
> all verified green. The 09-35 session closed the two blockers (queue clones,
> soak gocyclo) and refreshed the FEATURES coverage line (151+43). What follows
> is the consolidated remainder. — source:
> [`docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md`](docs/status/archived/2026-09-16_02-09_benchmark-statistical-rigor.md)
> §b/§f, 09-35 §f P3

- [x] **`compare` + serialization tail** — DONE 2026-09-19: compare table
      Noisy column (`Result.NoisyMetricCount`) + `Variation:` footer /
      markdown variation summary (`PrintComparisonVariation`); manifest
      `runs[]` behind `--include-runs` (`WriteManifestRepeated`);
      `RepeatedResult.WriteRepeatedJSON`. CHANGELOG [Unreleased]
      2026-09-19 benchkit section.
- [x] **Benchstat CI workflow decision (owner Q3)** — RESOLVED 2026-09-19:
      per-metric CI gating chosen over A/B-by-revision benchstat (the median
      gate's weakness is a loud machine, not a missing A/B workflow —
      benchstat samples already work). benchmark-regression.sh gained a
      load gate (load1/load5 vs CPU count), a benchkit noise gate
      (headline-metric CoV threshold via `cqrs-bench --repeat 5 --format
      json`), 12 new fixture tests (mutation-tested), and the sqlite
      gate-set entry (`BenchmarkBenchkitSuite_SQLite$`, also wired into
      benchmarks.yml). The benchstat A/B shell remains a possible future
      convenience, not a gate.
- [x] **SDK polish batch** — DONE 2026-09-19: `LatencyStats.Min` exact
      fast-path; `Environment.LoadAvg1End` + mid-run drift warning;
      `Config.LoadWarnThreshold`; soak `ThroughputCoV`/`WriteP99CoV`;
      `Config.InterpolatedPercentiles` (+ `--interpolated-percentiles`);
      `MetricNames()` stable universe; zero-value audit warnings. Fresh
      backend-comparison capture (with variation output):
      [`docs/benchmarks/2026-09-19_backend-comparison-variation.md`](docs/benchmarks/2026-09-19_backend-comparison-variation.md)
      — captured on an OVERSUBSCRIBED shared host (load ~35-40 on 32
      CPUs); treat per-backend deltas as non-decision-grade and supersede
      on a calibration-gate PASS window (the capture's Variation footer
      shows exactly which metrics flagged NOISY).

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

- [ ] **Decide a first-class single-writer/lease story for engines** — CV's
      Phase-0 ADR conditions every library-store cutover on a CV-owned
      `metaengine.RegisterDriver` decorator wrapping their `<dsn>.lease`
      single-writer marker, because the library has NO engine/store-level
      lock (verified: only `queue/` has lease semantics — task claims, a
      different concept). Minimal shape: an engine open-mode/advisory lock
      option at `system` construction. Decide before v5 freezes engine
      construction surfaces. — source: reflection doc §4.2 _(Effort: M — design + ADR)_
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
- [ ] **Docs-truth tail (2026-09-16 plan-surfaced)** — readmodels.md
      Scan-limit note (the review doc's §4.1 promised it; godoc + FAQ already
      shipped); modules.md metaengine row mentions the Scan default;
      CHANGELOG `[Unreleased]` entry for the Scan/WithLimit doc fix; resolve
      the 3 pre-existing doc-check ambiguous-alias advisories
      (core.md:448, recipes.md:119, faq.md:233); embed the overflow probe
      source into the review doc (kill the `/tmp` citation).
      — source: status report 2026-09-16 21-02 §f-2/12/13; SUPERB plan T27/M104 _(Effort: S)_
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
- [ ] **Scan default v5 decision** — documented-100 (status quo, now loud in
      godoc+FAQ) vs unbounded default at the v5 cut. Survey consumers,
      decide, implement at the v5 branch. — G-T14 _(Effort: S decision + S impl)_
- [x] **`example/goal-shaped-app` + "The Goal in 5 minutes"** — DONE
      2026-09-19: `example/goal-shaped-app` (domain.go types only, zero
      engine/schema/registration/limit imports; app.go ONE Evolution with
      pure convention folds; operator `cqrs.yaml` + `CQRS_*` env swap
      sqlite→postgres; README walks real EXPLAIN/Doctor output; 4 tests
      incl. config-only swap + loud unknown-driver failure). core.md §0
      "The Goal in 5 minutes" section + §9 row; recipes.md §2.39
      compile-gated (catalog scaffold, TestRecipes green); wired into
      go.work + flake examplePaths. — G-T23 _(Effort: M)_
- [ ] **FEATURES maturity flip for the closed surface (🧪→✅)** — earned by
      the plan's gates (not declared): evidence links per row, CHANGELOG
      Goal-story entry, release notes. Final stamp of Goal closure. — G-T25
      _(Effort: S, gated on gates A–D)_

---

## Load-ordering test flakes (filed 2026-09-16)

- [x] **`system` hardening: `TestSystem_ResetProjection_RestartAndReplay`
      starves under full-repo parallel load** — ROOT CAUSE STILL OPEN, but
      the crime scene is now captured (2026-09-19): `waitForProjectionProcessed`
      dumps ALL goroutine stacks + the load factor into the test log on
      starvation expiry — every prior incident reported only
      `processed=0 errors=0`, so blocked-vs-exited-empty-vs-never-started
      was undiscoverable after the fact. 14+ standalone repro attempts
      across sessions (incl. a manufactured 60-binary storm at load ~100 on
      2026-09-19) all green — the flake fires only inside the composed
      `#verify-fast`. NEXT: the next composed failure reads the dump and
      fixes at the projectionhost layer (ADR-0136). Note: the composed gate
      is independently blocked at HEAD by the half-landed Go 1.27 directive
      wave (system/go.mod still `go 1.26.7` vs root go.mod/toolchain 1.27.1
      → workspace-mode system builds fail `json.Unmarshal requires
      go1.27`); go.work was bumped to 1.27.1 on 2026-09-19 to match the
      committed root go.mod + flake go_1_27 pin.
      — observed while gating the vector verification tail _(Effort: M —
      instrumentation done; root-cause fix pending one composed-run dump)_
- [x] **`queue/sqlite` conformance: `status_counts` leaks under load** —
      RESOLVED AT ROOT 2026-09-19, and it was NEVER a load flake: 20/100
      standalone failures. `task.NewID()` minted a crypto-random
      same-millisecond suffix while the claim SQL ties break with
      `created_at ASC, id ASC` — the ID doc PROMISES time-sorting for
      exactly this tie-break. Three enqueues land in one ms → random
      permutation → the claim could pick the test's `third` task →
      `Cancel` hit `running -> cancelled`. Fix: NewID now mints
      `ms(16hex) + per-ms random seed(12hex) + monotonic seq(8hex)` —
      strictly increasing per process, same 36-char shape, cross-process
      uniqueness via the seed; fixes all three engine dialects. Pinned by
      `queue/task/task_test.go` (monotonic + concurrent-unique +
      same-ms-burst). 100/100 subtest reruns green; full sqlite suite green.
      Remaining queue-family lint twins stay with the active workstream.
      — observed 2026-09-16 during `#verify-fast` _(Effort: S-M — done)_
- [x] **Repo-wide lint findings outside the vector-tail files** — SWEPT
      2026-09-19. All named modules lint clean: watermill, catalog/eventcatalog,
      otel/otlp, stack/sqlite, scheduling/sqlstore, integration,
      cmd/api-stability, cmd/doc-check (was 70; recipes_catalog table gets a
      commented exhaustruct_v5/gochecknoglobals/goconst exclusion — declarative
      classification table). Also restored `.golangci.yml` from the
      auto-commit corruption (depguard block deleted + gci resurrected —
      sixth corruption class incident; `check-lint-config` green again), which
      had been polluting every module scan with gci noise and silently
      disabling depguard. Fixes were real code (api-stability walk helpers,
      watermill metadata table, ctx threading in adttest claimDueT, prealloc,
      stale-nolint removal) plus documented scoped exclusions for narrative
      scenario/property tests. NOT swept: queue-family files (active
      parallel-session workstream) and metaengine core-module residue
      (maintidx/tparallel/revive in vector-era conformance harnesses —
      owner: the session that authored them).
      — observed 2026-09-16 during `#verify-fast` _(Effort: M — done for the
      named set)_

---

## Vector-search verification tail (2026-09-15)

> Vector ADT shipped on EVERY engine 2026-09-15 (brute-force + DuckDB/libSQL
> pushdown; see FEATURES Metaengine section). Report:
> [`docs/status/archived/2026-09-15_18-32_vector-search-every-engine.md`](docs/status/archived/2026-09-15_18-32_vector-search-every-engine.md)

- [x] **Verification gaps from the vector session** — DONE 2026-09-16 (all of
      (a)-(h)): (a) iroh passthrough verified; (b) system + quickstart green;
      (c) benchmarks captured in
      [`docs/benchmarks/2026-09-16_vector-search-paths.md`](docs/benchmarks/2026-09-16_vector-search-paths.md)
      (libSQL 0.76ms / sqlite Go-scan 0.97ms / DuckDB pushdown 1.22ms,
      quiet-machine medians; regression gate deliberately not wired);
      (d) `VectorPathReporter` surfaces the path in ExplainPlan + Doctor;
      (e) libSQL probe lazily cached via `sync.OnceValue`; (f) dimension lock
      (`CheckVectorDimension` + `ErrVectorDimensionMismatch` +
      `adttest.AssertVectorDimensionGuard`) green live on sqlite/turso/duckdb/
      mysql/pg/dgraph/bbolt/pebble/badger — live legs surfaced and fixed a
      broken dgraph probe (DQL root name) and MariaDB DECIMAL division;
      (g) vector legs in the restart-safety harness; (h) metaengine full
      non-short suite green (22.4s). DuckDB's committed construction bug
      (missing statement separator before `meta_vector` DDL, born broken
      2026-09-15 18:28) found and fixed; full cgo suite green. See the
      2026-09-16 Fixed CHANGELOG section. — source: 18-32 §b/§c/§f1-16
- [x] **Dgraph version floor decision** — DONE 2026-09-16: LAZY vector schema
      (first vector use, mirrors `ensureEdgeSchema`); Dgraph < v24 servers keep
      booting and serving every other ADT; first vector op fails with an
      actionable "vector predicates require Dgraph v24+" error. README documents
      the floor. Live-verified 7 consecutive green `#integration-dgraph` runs
      (incl. replay of a previously-hanging shuffle seed). The shared-server
      test flake class is RESOLVED beyond the original abort question: parallel
      `ResetEngine` tests were wiping the shared ephemeral server mid-run
      (reset tests now serial), construction Alters now retry
      `errIndexingInProgress` and every gRPC call is deadline-bounded. Any
      remaining "Transaction has been aborted" under extreme parallel load is
      retried by `retryOnContention` (cap raised to 2s). — source: 18-32 §f9-10/§g1
- [x] **ADR-0140 candidate: vector distance-semantics contract** — DONE 2026-09-16:
      shipped as [ADR-0140](docs/adr/0140-vector-distance-semantics-contract.md)
      (Status: Accepted): the distance table (cosine = `1-cosSim`, dot = NEGATED
      dot, euclidean = L2), ascending = nearest via `TopKNearest`,
      `VectorDistance` as the single Go scorer, engine-native parity pinned by
      adttest, zero-vector behavior, degrade-everywhere floor, and filtered
      k-NN pre-filter semantics. AGENTS.md contract #26 cites it.
      — source: 18-32 §f17 _(Effort: S)_

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
