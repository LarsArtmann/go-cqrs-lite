# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here. Historical session reports live under
`docs/status/archived/` (annotated + archived by the docs-health passes of
2026-08-29 and 2026-09-06).

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## cqrs-lint

> Prioritized execution plan (2026-09-06): the full Pareto breakdown of this
> section — 24 medium tasks / 96 micro-tasks — lives at
> `docs/planning/2026-09-06_00-31_cqrs-lint-v5-hardening-pareto-plan.md`
> (T01–T07 executed 2026-09-06, annotated inline in the plan). This section is
> the living source; the plan is its point-in-time snapshot.

- [x] ~~T08 — RULES.md stub + DocURL anchors~~ DONE by follow-up session 2026-09-06: `cmd/cqrs-lint/RULES.md` exists with 204 `<a id>` anchors.
- [x] ~~F031 — `cqrs-lint --fix` E2E is a no-op through the real pipeline~~ DONE by session 2026-09-06 (commit `e44da78fa`): root cause was NOT upstream go-finding — C003 anchored its Direct fix at the fold's function declaration while BeforeCode lived on the default-case return, so the occurrence-safe provider silently refused; the if-stmt variant was dead code (Direct without code data fails finding validation). Detector re-anchored at the return statement; `fix_e2e_test.go` green AND real-CLI `--fix` on the preserved fixture verified to mutate exactly the targeted occurrence.
- [x] ~~T09 — CI wiring~~ DONE 2026-09-06: `cqrs-lint-self-lint` (strict-load + stale-suppression), `cqrs-lint-examples` (V007-silence gate over all 4 examples), V007 demo (`cmd/cqrs-lint/V007-DEMO.md`), and `check-lint-config` all wired in ci.yml. Residue: marking them REQUIRED status checks needs branch protection — owner decision, deliberately not done (F040; direct-push workflow). — source: 2026-09-06_02-40 §c + plan execution log
- [x] ~~T10 — rule-ID gap documentation~~ DONE 2026-09-06: `cmd/cqrs-lint/README.md` §"Rule ID numbering gaps" documents the 8 reserved IDs (A028, A031, P002–P005, S004, D004) and that listing one in `rules.disable` is a silent no-op (verified in-tree).
- [x] ~~T11 — V007 overhead measurement~~ DONE by follow-up session 2026-09-06: `docs/benchmarks/2026-09-06_cqrs-lint-v007-walltime.md` (verdict: below measurement noise).
- [ ] **T13–T19 — rule audit batches:** RISK-BASED SAMPLE DONE 2026-09-06 (plan execution log): all C-family detector files read — one real bug found+fixed (C005 missed the `json.NewDecoder(bytes.NewReader(...))` idiom); empirical FP hunt over top-volume findings (C023/D006/D014/A032) — all correctly targeted. Severity/confidence meta-verified (T04); V007 coverage mechanized (T02 drift test). REMAINING (low-yield): exhaustive per-file checklist audits of A001–A034, B001–B031, D001–D019, E001–E017, S/T/V/F families. Only behind a green full gate. — source: 2026-09-06_02-40 §c
      _(Effort: M/L)_
- [ ] **T20–T24 — subsystem reviews + design passes:** PARTIALLY DONE 2026-09-06.
      DONE: T22 shuffle/race evals (3 shuffled seeds + `-race -count=3` on
      suppression/fix, green — independently confirmed by two sessions; one
      apparent failure was a mid-landing transient with `d341d95bd`, re-run
      green); T23 design passes recorded
      (`docs/planning/2026-09-06_cqrs-lint-t23-design-passes.md`: v5-ready
      preset, dot-import detection, typed-info integration) + preset
      deprecated-surface policy with V007/F030 pins locked by policy test;
      sibling link checks (F092, session-3 log: go-sse v0.3.0+ /
      cqrs-htmx v4.9.0+ path-accurate). REMAINING: scanner/feature_profile
      split (T20), CLI subsystem file reviews (T21: doctor/health/scorecard/
      output). — source: 2026-09-06_02-40 §c
      _(Effort: M)_
- [ ] **Release-policy Q3: severity tightening in a minor.** S008/S009 now
      emit `error` (were `warning`); consumers using `--min-severity error` or
      CI fail-on-error see new failures after ≥v4.9.0. Acceptable in a minor
      (documented in CHANGELOG Added), or gate severity changes behind a
      "Changed" section + dedicated minor (v4.10.0)? User decision. — source: 2026-09-06_02-40 §g3
- [ ] [BLOCKED] **Daemon Q2: `.golangci.yml` exclusion from the auto-commit formatter.**
      ROOT-CAUSED 2026-09-06 (plan execution log): the gci re-adds come from
      BuildFlow's built-in golangci defaults regenerating config at pre-commit;
      no user-facing knob found in `~/.config/buildflow`. The in-repo
      `scripts/check-formatters.sh` self-heal repaired every occurrence and is
      the durable defense. REMAINING DECISION: accept self-heal permanently, or
      exclude `.golangci.yml` from BuildFlow's formatter upstream. User decision.
      — source: 2026-09-06_02-40 §d1/§g2 + plan execution log
- [ ] **cqrs-lint rule: ApplyLayout on engines that also implement
      LayoutPlanApplier → prefer the plan path.** SCOPED 2026-08-30: the
      source-based analyzer cannot know a receiver implements
      LayoutPlanApplier; a name heuristic would false-positive. Needs a design
      pass (type-impl detection or a module-level capability registry fed from
      api-stability's scan) before implementation. — source: session-4 retro §f25
      _(Effort: M)_
- [ ] **350-line limit: enforced but silently red repo-wide — needs split
      waves + a gate-policy decision.** VERIFIED 2026-09-06: the gate IS wired
      (CI `file-size-gate` + `nix run .#check-file-size`) but 56 non-test
      files exceed it (metaengine ~15: typed_reader.go 1127, adttest 935,
      enginetest 935, store.go 898, execute.go 767, engines 724/722/694/650;
      cqrs-lint remainder: architecture/helpers.go 627, feature_profile.go
      587, suppression/parser.go 540, explain.go 516, b022_b025.go 493, …) and
      it has been red since ≈2026-08-08 — unnoticed because red non-required
      jobs don't block direct pushes (F040). DONE: the pure table-catalog
      offenders are split (`pkg/rules/catalog.go` + `catalog_extra.go` → 12
      per-family files, largest 294 lines; rule data unchanged, 204-count +
      RULES.md freshness tests green). REMAINING: code-file splits need careful
      manual extraction (NOT mechanical) — or the owner picks a gate policy:
      full split vs baseline ratchet (no file grows, no new offender) vs
      table-catalog exemption. — source: cqrs-lint deep review 2026-09-05 +
      2026-09-06 verification
      _(Effort: L)_

---

## Release / Tagging

> The full 39-tag v4 wave (B1–B7) was cut, pushed, and verify-ci-green on
> 2026-08-29 — see the CHANGELOG dated sections and
> `docs/planning/archived/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md` (wave
> record). Zero local `=> ../` replaces remain.

- [ ] [BLOCKED] 🔥 **Next v4 patch/minor wave** — modules with prod-code
      changes since their latest tags (enumerate per module via
      `git diff <tag>..HEAD -- '*.go'`): at minimum `metaengine` (session-7
      surface: `LayoutPlanEvolver`, `KeyScanBackend`/`BackfillPlannedCollection`,
      `PlannedTablesReporter`, duckdb MapScan visibility fix — metaengine is
      ≥1 minor ahead of v4.12.0), `pgengine`, `mysqlengine`,
      `scheduling/sqlstore`, plus anything the post-2026-09-06 waves touched.
      Order constraints per the CONTRIBUTING pre-tag checklist; cut→push
      interleave (GOPRIVATE resolves siblings via VCS). Dry-runs were GREEN
      for dgraphengine/sqliteengine/projectionhost on 2026-08-30. Do NOT tag
      `stack/*`, `storage/view`, `storage/relational` (v5 deletes them).
      _(Effort: M)_
- [ ] **Create GitHub Releases** for the outstanding tags (2026-08-16 chain +
      subsequent waves; only storage/v4.7.1 ever got one). `gh` auth VERIFIED
      working (LarsArtmann, repo scope, 2026-08-31);
      `scripts/create-github-releases.sh` generates changelog-extracted bodies
      — remaining work is running it per tag. _(Effort: S)_
- [ ] **Consolidate indirect dep references** — the transitive
      `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}/v4` indirect deps
      in ~49 consumer go.mod files clean up after new tags publish. Track and
      verify. _(Effort: M)_
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99 bound
      50→150ms (worst-of-30 sample inflates under gate load). Shipped + gated
      green; keep or revisit. _(Effort: XS)_
- [ ] **cqrs-bench deprecation stub** — the dead suffix-less
      `cmd/cqrs-bench` path silently serves `v0.1.0` via `@latest`; ship the
      same deprecation-stub treatment `cmd/cqrs-lint/v0.2.1` got. — source:
      docs/status/archived/2026-09-01_21-37 §c1
      _(Effort: XS)_
- [ ] **`retract cmd/cqrs-lint/v4.8.0`** — the poisoned (syntax-error) tag
      remains fetchable; v4.8.1 supersedes it but a retract stops fresh
      consumers from resolving it. — source: archived/2026-09-01_21-37 §c6
      _(Effort: XS)_
- [ ] **tag-release.sh hardening:** zero tests exist for the script; add a
      proxy smoke-check ("clean-dir install @latest + run") as a documented
      post-cut step; one-shot all-modules path-vs-tag audit (module path vs
      major-version suffix, the issue-#20 class) — source: archived/2026-09-01_21-37 §c2/c3/c5
      _(Effort: S/M)_
- [ ] **Version-reporting unification** — cqrs-lint hand-maintains a version
      string const (tag-release.sh bumps it); adopting debug/buildinfo would
      remove the drift class. Decide const-vs-buildinfo before v5. — source: archived/2026-09-01_21-37 §c4
      _(Effort: S)_

---

## Metaengine — correctness & routing leftovers

- [x] ~~dgraphengine graph parity (`GraphDepthBound`/`GraphDepth3Diamond`
      cross-engine divergence)~~ FIXED 2026-08-30 (`eef6fa85d` + `6e62312eb`):
      Dgraph `@recurse` counts node LEVELS, not hops — depth+1 requested for
      depth>1; pinned by `TestDgraphGraph_RecurseDepthCountsHops`. Live suite
      green. (Kept struck only to correct the previous stale open claim;
      removed at the next pass.)
- [ ] **Dgraph per-OP constants in per-ROW fields** — `NsPerFilteredScan`/
      `NsPerScan` hold one-RPC per-OP numbers in per-ROW fields; recalibrate
      only with result-size-scaled benches, never from a desk. (Flagged in
      `docs/benchmarks/calibration-2026-08-30.md` + AGENTS ReadCosts gotcha.)
      _(Effort: M)_
- [ ] **Engines over-declaring `Supports` produce execution-time hard errors**
      with no plan-time diagnostic or routing penalty (CapabilityAudit renders
      a banner but is not a rule). _(Effort: M)_
- [ ] **Graph BFS fallback dedups nodes by `fmt.Sprint`** — `int(1)` collides
      with `"1"` on mixed-type nodes. _(Effort: S)_
- [ ] **OnRecord folds returning Embedding/IndexedText/Point/MultiEntry/Append
      receive an always-zero Record silently.** _(Effort: S)_
- [ ] **Single-sourcing of calibrated constants** — expected per-pattern values
      live in FOUR places (engine.go constants, routing-regression pins,
      drift-script table, baseline doc); decide the canonical source (coupled
      to the drift-gate redesign below). — source: session-6 §B4
      _(Effort: S decision / M impl)_
- [ ] **NsPerWrite/NetworkRTT "provably dead field" audit** — apply the
      ReadCosts treatment (per-pattern benches or demote) to the write-side
      profile fields. — source: archived/2026-08-29_18-38 §f47
      _(Effort: M)_
- [ ] **DuckDB + sqliteengine planned-table capability parity** — adopt
      `KeyScanBackend` + backfill, `LayoutPlanEvolver` (sqlite: PRAGMA
      table_info path), `PlannedTablesReporter` (Doctor row counts currently
      pg/mysql only); matrix legs currently skip. — source: archived/2026-08-31_16-28 §f11–15
      _(Effort: M)_
- [ ] **MariaDB SKIP LOCKED re-evaluation** — MariaDB 11.8 status may have
      changed; verify before relying on `ErrClaimingUnsupported`. — source: archived/2026-08-31_16-28 §f24
      _(Effort: S)_
- [ ] **`errorfamily` code rename `aggregate_*` → `stream_*`** (v5 item) —
      with a dashboards/consumers note. — source: session-4 retro §f30, session-7 §f42
      _(Effort: M, v5)_

---

## CI / Infrastructure

- [ ] [BLOCKED] **Fix GitHub Actions billing** — every paid CI job fails in
      3–7s ("recent account payments have failed or your spending limit needs
      to be increased"); broken since ~2026-07-17. Local `nix run .#verify`
      remains the authoritative gate. _(Effort: S, user action)_
- [ ] [BLOCKED] **cqrs-lint Self-Lint credentials** — go-finding fetch fails
      under GOWORK=off (`git ls-remote` exit 128). _(Effort: S, user/creds)_
- [ ] **First post-push CI run triage** — the ~80-job per-module matrix has
      NEVER executed (discovery bug fixed 2026-09-03); expect new failures
      (flaky tests, module-specific env) on the first push. Pre-size the
      minutes impact before pushing if billing is tight. — source: archived/2026-09-04 §c6
      _(Effort: M)_
- [ ] **Calibration-drift gate redesign** — compare against a persisted
      CI-baseline artifact (same mechanism as the regression job) instead of
      absolute constants; nightly >100% rows are shared-runner noise (proven
      locally 2026-09-04). Also add TMPDIR-filesystem detection (refuse to run
      on CoW). — source: archived/2026-09-04 §b2/§f16/§f18
      _(Effort: M)_
- [ ] **bbolt-on-btrfs local trap** — bbolt suites/benches need
      `TMPDIR=/tmp` (tmpfs) on this machine class; document in AGENTS (the
      mmap+fsync workload times out 10min on CoW). — source: archived/2026-09-04 §b3/§f19
      _(Effort: XS)_
- [ ] **go.work.sum hygiene** — regenerate after the externals removal; add a
      drift check. — source: archived/2026-09-04 §c9
      _(Effort: XS)_
- [ ] **Cheap CI gates into pre-commit** — module-layers, version-drift,
      workspace-sync, replace-directives are plain bash; wire staged-aware
      into the hook. — source: archived/2026-09-04 §e6
      _(Effort: S)_
- [ ] **"Days-since-green" metric/alert** — 6-week red droughts (07-17→09-03)
      normalized drift; a Gatus-style freshness check catches the class in
      days. — source: archived/2026-09-04 §e5
      _(Effort: S)_
- [ ] **CV consumer bump (operator-gated)** — 8 go-cqrs-lite modules behind
      latest tags in the CV repo (event v4.7.0→v4.9.0, command v4.6.0→v4.8.1,
      metadata, query, record, snapshot, storage/memory, dispatcher) + nix
      `vendorHash` cascade + full CV verification. — source: archived/2026-09-04 §c2
      _(Effort: M)_
- [ ] **`check-coverage.sh` nix wrapper runs without the cache env** and
      reports 0.0% DRIFT for everything — make the app export the env itself
      or fail loudly. — source: archived/2026-08-30_06-34 §f
      _(Effort: S)_

---

## Code Quality

- [ ] **>350-line production files (29 by the CI job's count)** —
      catalog_extra.go 1207, typed_reader.go 1127, adttest/harness.go 967,
      enginetest.go 935, store.go 898, …. A standalone multi-session refactor
      wave; decide harness-dir exemptions (adttest/enginetest are exported
      test harnesses, not production logic) first. _(Effort: XL, multi-session)_
- [ ] [BLOCKED] **macOS verification of ephemeral PG** —
      `scripts/ephemeral-pg.sh` claims cross-platform but was only
      static-review-tested; a GitHub Actions macOS runner leg is the
      verification route (blocked on macOS runner, same constraint as D9).
      _(Effort: M)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) —
      the nspawn env runs the full app-level flow; userspace MariaDB coverage
      exists but not the full env. _(Effort: M)_
- [ ] **Evaluate `-shuffle=on` for the dgraph suite specifically** (adopted
      for pg/mysql/sqlite/duckdb 2026-08-31; dgraph still needs its own
      shuffled evaluation, then roll into the ephemeral-* app invocations).
      _(Effort: S)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1–7 done. Pre-cut deprecation markers shipped 2026-08-17; migration
> guide at `docs/V5-MIGRATION-GUIDE.md`. The deletions themselves are TODO.

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it. _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)** —
      multi-collection batch atomicity + auto-projection replaces them. _(Effort: M)_
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
- [ ] **Breaking `record.NewStreamRef` validation** — change to
      `NewStreamRef(streamType, entityID string) (StreamRef, error)` rejecting
      an empty entityID; migrate call sites. Owner-confirmed 2026-08-22
      (decision memo:
      `docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md`
      Appendix B — struct `record.Stream` proposal REJECTED). _(Effort: M)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) — the
      final v4.x tags exist (transport/http/v4.3.0, transport/grpc/v4.2.1,
      2026-08-29); remaining: drop from go.work/flake testModules/api-stability
      list, then delete at the cut. _(Effort: M)_
- [ ] **Delete deprecated tombstone metadata API (ADR-0114 completion)** —
      remove `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/
      `TombstoneStatus`/`Metadata.Tombstone`; pre-reqs: type-driven status in
      `listing` (replaces the DetectTombstone call at listing/in_memory.go:155),
      migrate `example/taskmanager` off `OnTombstone`, regen golden, execute
      the tombstone section of `docs/planning/v5-deprecation-sweep.md`. _(Effort: M)_
- [ ] **Honest snapshot wire tags at v5 (T18 audit)** — rename
      `aggregateId`/`aggregateType` JSON tags per-backend with dual-read
      fallbacks; SQL rename = ALTER TABLE + backfill per dialect
      (`storage/sql/migrations/` + `nix run .#integration-pg`). Details in the
      2026-08-22 T18 audit (CHANGELOG). _(Effort: M)_
- [ ] **v5 items from extended review** — E1 (event-envelope Encoding →
      `record.Encoding`), E7 (watermill/middleware RetryConfig collision),
      E8 (typed Message Kind enum), E11 (AdapterCore.Encode error return),
      E13 (SQLTimerStore phantom param), E15 (middleware signature
      unification). _(Effort: M)_
- [ ] **More extended-review follow-ups (harvested 2026-09-06)** — E3 (bbolt
      command/query bare `fmt.Errorf` → adopt the pebble error-family
      pattern), E4 (sentinel name↔code mismatch: `storage/sql/errors.go:22`
      `ErrStreamTypeMismatch` carries `storage.aggregate_type_mismatch`), E6
      (`middleware.Option` vs `BundleOption` — merge or bridge at v5), E9
      (turso Policy nil-write panics), E10 (ShutdownDependency name
      validation), E14 (eventstore ownership asymmetry). — source:
      docs/reviews/2026-08-22_extended-data-model-review.md
      _(Effort: M)_
- [ ] **Post-landing sweep for the data-model series** — api-stability
      meta-tests, doc-check over skill refs, consumer-pin sweep for `record/v4`
      consumers under GOWORK=off (MarshalBinary lesson). _(Effort: M)_
- [ ] **Expand V5-MIGRATION-GUIDE** with before/after examples per v1 tier
      (incl. `relational → metaengine`) at the cut; the asrecord/
      MIGRATION_TO_STACK/PRESETS guides teach the dying stack surface — sweep
      them once v5 nears. _(Effort: M)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate. _(Effort: M)_

---

## Core Data Model v4.x/v5 (2026-08-22 review + plan)

> Source: [core data-model review](docs/reviews/2026-08-22_core-data-model-review.html)
> + [execution plan](docs/planning/archived/2026-08-22_03-52_core-data-model-v5-execution-plan.md).
> Owner decision 2026-08-22 (Appendix B): string `record.StreamRef` SURVIVES
> v5 with a validating constructor; the struct `record.Stream` proposal is
> rejected. Decision/reference tasks T01–T03, T14, T02 are DONE. One open
> plan task: **T23** (upstream skill fixes: docs/reviews↔brainstorming
> divergence; read-prior-reports + copy-template steps in the review skills)
> — execute or decline at the next skill-maintenance pass.

---

## Docs / Tooling tail (harvested 2026-09-06)

> Forward-looking items surfaced by the docs-health pass over
> `docs/status/2026-08-27 → 2026-09-06` (those reports are now archived; the
> items live here).

**Watermill / catch-up**

- [ ] Regression tests: CatchUp Close-while-blocked + double-Subscribe
      (inspection → pinned property; grep-verified absent). — source:
      docs/status/archived/2026-08-28_04-55 §f30
- [ ] Catch-up throughput benchmark (+ ack-window pipelining if numbers are
      bad); watermark ULD-ordering vs cross-process skew — document or
      property-test. — source: archived/2026-08-28_04-55 §f33–34

**Docs honesty / README**

- [ ] `example/readme-quickstart` still uses the deprecated Execute/Load pair
      forms (main.go:51,65); the 2026-09-06 modernization covered
      getting-started only. — source: archived/2026-09-01_23-43 §f13
- [ ] `metaengine/dsl.go:17` comment references nonexistent
      `PlanFromSQLite` (comment rot, verified 2026-09-06). — source:
      archived/2026-09-01_23-43 §c5
- [ ] README unverified claims: "3 dependencies" for event, coverage
      percentages, exact "82 modules" phrase sweep; README-claims meta-test;
      homepage/social preview; `.github/` ISSUE_TEMPLATE +
      PULL_REQUEST_TEMPLATE missing. — source: archived/2026-09-01_23-43 §b3–b4/§f
- [ ] example/ README audit (4 files) + docs/ README audit (24 non-archive
      files). — source: archived/2026-09-01_23-43 §c1–c2
- [ ] `storage.EventSchema` symmetry — add re-export or drop mention. —
      source: archived/2026-09-01_23-43 §c6
- [ ] `watermill.WithBackend(pub, sub, client)` / `WithCommandBackend` arity
      signature-verification. — source: archived/2026-09-01_23-43 §c4

**Catalog**

- [ ] Golden-test the flattened output for the eventcatalog exporter
      (embedded fields change exporter output; downstream blast radius
      unverified) + check `cmd/cqrs-gen` + `catalog/eventcatalog` modules for
      embedded-flattening fallout. — source: archived/2026-08-29_20-23 §f30–31
- [ ] CSP support never browser-validated against the embedded
      Scalar/AsyncAPI bundles. — source: archived/2026-08-29_17-35 §b2
- [ ] EventCatalog render validation is a manual /tmp flow — make it a
      repeatable flake app (`check-eventcatalog`). — source: archived/2026-08-29_17-35 §b7
- [ ] api-stability golden: include the `catalog/v4/docserver` package
      (currently invisible to the golden). — source: archived/2026-08-29_18-38 §f16
- [ ] cqrs-lint T38 tail: C040 follow-up + `doctor --format json` /
      `--fix --dry-run` flags. — source: archived/2026-08-29_17-35 §b8

**Encryption / consumer asks**

- [ ] Key-management helpers (bank-sync ask): key-generation helper
      (`GenerateKey`) + key load/serialize from env/file — the envelope
      key-ID + StaticKeyResolver path shipped, these two helpers did not. —
      source: docs/feedback/archived/2026-07-17_bank-sync_encryption-key-management-standardization.md
- [ ] Snapshot-encryption PG/SQL store test (encrypted-at-rest column
      assertion against a real server); rotation write-back option
      (re-encrypt-on-read migration). — source: archived/2026-08-29_18-38 §f6–7
- [ ] go-retry `DoWithValue[T]`: committed in the external repo but
      push/tag state unverified — confirm release so consumers can adopt. —
      source: archived/2026-08-29_17-35 §b8

**Tooling / lint**

- [ ] `exhaustruct` → `exhaustruct_v5` migration (deprecation warning fires
      on every lint run today). — source: archived/2026-09-01_00-11 §f14
- [ ] Formatter-exclusion probe rewrite; drop the dead `event/eventtest/`
      formatter path; AGENTS note on golangci exclusion path-base + probe
      method. — source: archived/2026-09-01_00-11 §b3/§f3
- [ ] Pin-sweep script (sibling pins to latest + golden refresh built in) in
      `scripts/` + CI leg; record-pin sweep variant. — source:
      archived/2026-08-28_04-55 §f30, archived/2026-09-01_00-11 §f18
- [ ] AGENTS gotcha: cross-binary DB naming lesson (session-0 phase-0
      finding). — source: archived/2026-08-28_07-55 §f46
- [ ] doc-check: teach it to resolve `sqlstore.` aliases without a visible
      import (cost two sessions a cycle each). — source: archived/2026-08-31_16-28 §f48
- [ ] Wire check-apps (check-templ, check-bench-gate, check-lint-config)
      into `#verify`. — source: archived/2026-08-29_18-38 §f73
- [ ] LSP go-mod-tidy warnings (catalog, watermill dedup unused) —
      investigate/tidy. — source: archived/2026-09-01_00-11 §f16
- [ ] kvstore 3-way contract test: move to `integration/` to drop the sqlite
      dep from idempotency/kvstore (deferred twice). — source:
      docs/status/archived/release-fix-2026-07-25 §f4
- [ ] Engine restart-safety harness adoption tail: `RunRestartSafetyTest` for
      badgerengine (has stream_log, no restart_safety_test); consider
      sqliteengine/duckdbengine; extract badger↔pebble `sortAndPaginate`
      into a shared core helper (real logic duplication, currently
      accepted). — source: docs/status/archived/2026-09-02_15-24 §f14–16

---

## Metaengine — Universal ADT Coverage (Phase 7)

- [ ] **iroh graph `WriteOp` convergence** — edges do not replicate
      cross-peer yet; capability-conformance wiring under `#test-integration`.
      _(Effort: M)_

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
