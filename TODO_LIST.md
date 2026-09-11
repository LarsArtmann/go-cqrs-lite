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
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). DESIGN FINDINGS 2026-09-11: there is no clean seam yet — the planner (`EngineProfile.ReadCosts` per-pattern, `ReadPattern=ReadAggregate`) never sees the aggregate SHAPE (fn/column/group live in opaque query closures), so coverage cannot influence plan cost without a new declarative surface (queries must carry their aggregate spec at plan time — v2-adjacent). Also: routing grouped shapes would be UNSAFE until upstream fixes defect A (it would steer production aggregates at known-wrong results) — scope the first cut to scalar-covered shapes only. — source: archived 19-25 §f29, 05-33 §f32
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
- [ ] **Audit `scripts/batch-release.sh` against the hardened tag-release.sh** —
      it may encode the pre-hardening flow (no path-vs-tag guard, no smoke
      probe, no audit). — source: 01-47 §f39
      _(Effort: M)_
- [ ] **Watch the first nightly `upgrade-dogfood` sentinel run** — the flake
      app + sentinel.yml job pass locally (4m21s, 83 modules, 0 findings);
      the first real CI run (network topology, GOPROXY, 20-min timeout) is
      unobserved. — source: 01-47 §b2/§f2
      _(Effort: XS)_
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

- [ ] 🔥 **Audit the encoded-apply record context + build the entry-point conformance sweep** — `metaengine/encoded.go:49` feeds OnRecord folds a `record.Record{Type: eventType}` on the encoded-apply path (outside `applyWithRecord`): no real record context, no Doctor synthetic-apply advisory — the same bug class the Demote mirror-leg gap proved real (fixed + pinned 2026-09-11, `TestDemoteEngine_RecordContextReplay`). Follow with ONE conformance-style test walking EVERY fold-dispatch entry point (Apply, ApplyBatch, ApplyRecord, Backfill/replayShadows, Verify, DemoteEngine ×2, encoded-apply, live replicator) asserting the record-context contract + advisory counting — kills the spot-test whack-a-mole. — source: 03-50 §c1/§f1/§f2, §e
      _(Effort: M)_
- [ ] **ClaimMetrics documentation + pin tail** — the shipped `Metrics()`/`ClaimMetricsSnapshot` surface (2026-09-11) still needs: (a) `scheduling/sqlstore/README.md` claiming docs, (b) a FEATURES.md row, (c) a JSON marshal pin test (tag stability is now public API), (d) consider a PG/MySQL live-window `Metrics()` integration test (snapshot pinned on SQLite only). — source: 03-50 §c2/§f3/§f4/§f5/§f9
      _(Effort: S)_
- [ ] **Calibration provenance protocol + quiet-window re-runs** — (a) script-side load gate (assert load < N before benching, abort loudly), (b) a provenance line per baseline entry (nix store path, binary version output, uptime samples) — the 2026-09-11 SearchQuery run shipped an unverified engine-version citation and a load-ramping window; (c) quiet-window count=5 SearchQuery re-run (supersede today's table if medians move >5%) and a titled re-pin of `benchmarks/benchmark-baseline.txt` (the 02:40 load-noise refresh); (d) re-anchor ALL dgraph constants in one quiet window (doc mixes 09-01/09-06/09-11 runs). — source: 03-50 §b2/§b3/§f7/§f8/§f15/§f16, 02-48 §d3/§f8
      _(Effort: M)_

> The 2026-09-07/08 correctness batch (ApplyBatch Record handling,
> record-aware cache invalidation, Doctor observations, MySQL claiming, dgraph
> calibration, planner polish, keycodec, restart harnesses) SHIPPED in full —
> see CHANGELOG `[Unreleased]`. What follows is the open tail.

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

---

## CI / Infrastructure

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
- [ ] **`check-coverage.sh` nix wrapper runs without the cache env** and
      reports 0.0% DRIFT vacuously — make the app export the env itself or
      fail loudly. Also run it once for the 2026-09-07/08 waves (matview +
      hardening batches were never coverage-checked). — source:
      archived/2026-08-30_06-34 §f, 19-25 §f4, 05-31 §f11
      _(Effort: S)_
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
