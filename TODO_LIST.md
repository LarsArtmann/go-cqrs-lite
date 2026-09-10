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
- [ ] 🔥 **Push the 3 unpushed commits, then edit the PR comment link to SHA `18b2c495c`** — the posted comment's permalink points at `1c9f3bf` (last pushed SHA); the newer draft revision contains the full three-defect characterization. Comments are editable. Pushing also publishes the divergence research. _(Effort: XS)_
- [ ] **Track turso-go releases for the IVM fixes** — re-run the repro suite (`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`) on each new release; when green, remove the grouped-view warnings (Doctor WARN line, recipes §2.29 notice, AGENTS.md gotcha, bench doc warning) and un-skip the ≥10k matview bench cases. Also watch PR #8257 for maintainer response. _(Effort: S per check)_
- [ ] **Code guard follow-up: make grouped-spec safety mechanical** — today the danger is advisory-only (Doctor WARN + docs). Options: `MaterializedViewSpec` validation refusing `GroupBy` on turso-go ≤ v0.8.0-pre.8 (breaking for legitimate small deployments) vs a config flag (`AllowGroupedViews`) vs silent status. Decide + implement once the upstream timeline is known. _(Effort: S)_
- [ ] **Matview safety test tail** — (a) regression test pinning the Doctor WARN line for grouped specs + the rendered "Materialized views" section incl. row counts and the "none" branch (`metaengine/materialized_view_doctor.go` has zero dedicated tests); (b) `matViewDDL` golden test (go-snaps) locking the SQL dialect per fn × scalar/grouped; (c) property test matview-served aggregate == base-table aggregate (with a second-tx case that would catch defect A the day upstream fixes it); (d) regression test asserting the 2-tx divergence so an upstream fix flips it loudly. — source: archived 19-25 §f2/§f6/§f16-17, 05-33 §f11/§f16-18
      _(Effort: M)_
- [ ] **Extend `scripts/benchmark-regression.sh` to the matview read bench (1k)** so the serving-path acceleration cannot silently rot. — source: archived 19-25 §f5, 05-33 §f19
      _(Effort: S)_
- [ ] **Matview v2 feature surface** — planned-table matviews (ordered with `ApplyLayout` + backfill), filtered-view spec variants, multi-aggregate/DISTINCT serving, `DropMaterializedView` off-boarding, per-view IVM write-amp otel counter, `system.Introspection()` surface, cqrs-lint rules (matview-on-unsupported-driver; matview-plus-planned-table staleness trap), `example/materialized-views/`. Route individually when a consumer asks. — source: archived 19-25 §f23-35, 05-33 §f29-35
      _(Effort: M/L each)_
- [ ] **Routing integration: teach the cost model matview-covered shapes are O(1)/O(groups)** so cross-engine routing prefers the Turso engine for covered aggregates (planner-side). — source: archived 19-25 §f29, 05-33 §f32
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
- [ ] **Wire `#check-file-size` into verify (or fix violations)** — the app
      exists but nothing runs it; `catalog/eventcatalog/exporter.go` is
      already over the 350-line limit. _(Effort: XS to wire, S to split)_
- [ ] **Release-train note** — the `metaengine/projectionadapter` sibling
      replace + the `metaengine` pin bump ride the next tag wave
      (`scripts/tag-release.sh` strips the replace; smoke at cut time).

---

## cqrs-lint

> Point-in-time execution plan (T01–T24 / F001–F096) with per-row resolution
> markers: `docs/planning/archived/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`.
> T01–T12, T20–T24, F089, F090(a), F091 Tier 1 and the 2026-09-08 hardening
> batch are DONE (CHANGELOG `[Unreleased]`); this section carries the living
> remainder.

- [ ] **T13–T19 — exhaustive rule audit batches.** RISK-BASED SAMPLE DONE
      2026-09-06 (C-family) + S-FAMILY DONE 2026-09-07 (financialEscalatedRules
      comment drift + missing S011 escalation fixed, completeness meta-test
      added). REMAINING (explicitly low-yield, only behind a green full gate):
      per-file checklist audits of A001–A034 (2 waves), B001–B031, D001–D019,
      E001–E017, V/T/F families, plus the S001/rules.go line-by-line remainder.
      — source: archived/2026-09-06_02-40 §c, 05-31 §f16-22
      _(Effort: M/L)_
- [ ] 🔥 **F091 Tiers 2–3 + F090(b)** — TIER-2 CORE + F090(b) DONE 2026-09-08
      (`--typed-info` flag plumbed, F090(b) typed dot-import attribution with
      committed fixture + tests, C008 usage-confirmation). REMAINING:
      C035/C013 payload-shape confirmation under the same gate.
      — source: 05-31 §b4
      _(Effort: M)_
- [ ] **ApplyLayout rule (design done, implement)** — structural method-shape
      detection (`ApplyLayoutPlan` + `BuildLayoutPlan` co-occurring on the
      receiver type); fires behind F091 Tier-2 `--typed-info=auto`, silent on
      the name-only fallback. Design: appendix in
      `docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md`. — source:
      session-4 retro §f25
      _(Effort: M)_
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
- [ ] **ClaimMetrics surfacing** — claim metrics hooks shipped 08-30 with zero
      consumers; surface `ClaimMetrics` in Doctor or a status endpoint. —
      source: archived 22-33 §f14
      _(Effort: S)_
- [ ] **Fold `BenchmarkCalibration_DgraphSearchQuery` results into the
      calibration baseline doc** (`docs/benchmarks/calibration-2026-08-30.md`
      protocol/recalibration sections; bench exists, doc not updated; also
      record the ADTMap=O1 decision there). — source: archived 22-33 §f11
      _(Effort: XS)_
- [ ] **Demote catch-up Record-context completeness** — verify Demote's
      catch-up path passes full records to record-aware folds (07-43 §f36).
      — source: archived 22-33 §f26
      _(Effort: S)_
- [ ] **enginetest fakes contract note** — document next to
      `RunCapabilityConformance`: fake engines must satisfy
      `engineServesADTNatively` for every declared ADT (the honestMapMixin /
      nativeMapEngine precedent). — source: 04-35 §e5/§f20
      _(Effort: XS)_

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
- [ ] **`example/metaengine-quickstart/README.md` does not exist** — author it
      from its four demo sections (docs/README.md links the directory; the
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

- [ ] **Skill-reference propagation wave** (`references/*.md`): envelope v2 +
      rotation write-back recipe (recipes.md §2.7 extension); `doctor
      --format json` + `check-csp`/`check-eventcatalog` apps; MySQL claiming
      support matrix (10.6+ works); planned-table capability roster (sqlite +
      duckdb now qualify); Doctor's record-context + planned-tables sections;
      `CALIB_DUMP=1` usage; consumer recipe for decoding pre-v5 snapshots
      (JSON+CBOR fallback contract). (modules.md cqrs-upgrade + tursoengine
      rows and the readmodels.md matview section landed 2026-09-08.) —
      source: 07-43 §c4, 08-26 §b1, 08-41 §b6/§f24
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
- [ ] 🔥 **Benchkit full-suite flake hunt (unexplained since 2026-09-07)** —
      `TestRun_ClosedStore`/`TestRun_ClosedStore_ErrorMessage` ("expected
      error from closed store, got nil" after ~26s) and
      `TestRun_Pebble`/`TestRun_Recovery_Pebble` ("checkpoint phase: context
      deadline exceeded" at 90s) FAIL under the full workspace suite
      (`#verify`/`#verify-fast`, `-race`, shared-host load 18-65) but PASS
      isolated (40s, no -race). Observed first by the SUPERB session (its §d2
      "unexplained, not explained") and reproduced by the 2026-09-08
      docs-health verify runs. Same class:
      `system.TestSystem_ResetProjection_RestartAndReplay` (snapshot-load
      deadline under load; 0.4s isolated). Either scale the internal
      deadlines like `loadScaledCeiling`/`loadScaledDeadline` (the proven
      pattern) or find the real race in the closed-store error path. —
      source: SUPERB §d2/§f11, docs-pass verify runs 2026-09-08
      _(Effort: M)_
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
- [x] **AGENTS.md indexed-split** — DONE 2026-09-08 (Pareto P15+P16): 92 KB →
      28 KB index + `docs/agents/gotchas-{tooling-build,module-management,
      language-footguns,testing}.md` + `gowork-modes.md` (THE decision table)
      + `module-map.md`; zero-content-loss verified by bullet/row counts.
      — source: 15-09 §f39, evening-pass §f46
- [ ] **error-taxonomy.md completeness check** — verify it covers the
      storage/pebble/watermill family codes at all; extend if the doc aspires
      to completeness (the 2026-09-08 rename made its stream-code table
      current). — source: archived 07-48 §f23
      _(Effort: S)_
- [ ] **DOMAIN_LANGUAGE.md entries** — "materialized view acceleration",
      "IVM", "view-maintained write" (+ the two-model encryption table if
      at-rest encryption becomes a domain concept). — source: archived 19-25
      §f42, 20-57 §f31
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
