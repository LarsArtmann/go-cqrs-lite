# TODO List

**Updated:** 2026-07-30 (session 22:01)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Verify Gate

> ⚠️ **`nix run .#verify` has NOT been run** since the 159→171 rule expansion.
> The cqrs-lint module builds and passes tests locally (`go build`/`go vet`/`go test -race`),
> but the full monorepo verify gate has not confirmed: formatting, lint, api-stability
> golden, doc-check. This must be run before any release.
>
> **22MB compiled binary** (`cmd/cqrs-lint/cqrs-lint`) was committed to git by
> the auto-commit daemon in `f791da84`. Must be `git rm --cached` and added
> to `.gitignore`.
>
> Pre-existing intermittent failure: `TestProperty_SQLiteTTLExpiry` in
> `idempotency/sqlstore` — passes on re-run; not a regression.

- [ ] 🔥 **Remove committed binary** — `git rm --cached cmd/cqrs-lint/cqrs-lint` + add to `.gitignore`
- [ ] 🔥 **Run `nix run .#verify`** — fix formatting, lint, api-stability golden, doc-check
- [ ] 🔥 **Regenerate api-stability golden** — 12 new exported `New*Detector` functions added
- [ ] **Fix 3 flaky benchkit soak tests** — `TestRunSoak_Memory`,
      `TestRunSoak_TrendsPopulated`, `TestRunSoakJSON_RoundTrip`. All timing-
      sensitive tests that flake under parallel race-detector load. Use
      `testutil.RaceEnabled` build-tag thresholds or `testing.Short()` guards.
- [ ] **Investigate `TestRun_Postgres_Recovery` benchkit failure** — root cause
      was found (populateSnapshots writes +50 events; fixed with `SkipSnapshot: true`)
      but may still flake. Monitor.

---

## cqrs-lint Quality (171 rules shipped; needs hardening)

> The linter grew from 65 to 171 rules across 10 categories. 12 new rules + 3
> extensions shipped in the Pareto plan execution session (2026-07-30 22:01).
> Known quality gaps below need addressing before the linter is trustworthy.

- [ ] 🔥 **Fix E010/E011/E013/E014 — architecturally wrong rules** — E010 uses
      package qualifier instead of type info; E011 uses name-counting instead of
      call-graph analysis; E013 doesn't verify the config struct type; E014
      detects the wrong concept (absence of `host.Stop()` vs no drain-before-return).
- [ ] 🔥 **Library self-lint mode** — auto-detect `go-cqrs-lite` module path and
      suppress consumer-only rules (A001/A008/A020/A021/A023/E005/E007) for
      library files. Currently requires 181+ manual inline suppressions.
- [ ] 🔥 **Import-alias resolution** — D007/D008/D010/D013 and all E-series rules
      assume unqualified package names. Build a shared `qualifierToImportPath`
      helper in `lintutil` so rules resolve import aliases correctly.
- [ ] **Fix P010 registry improvement** — was dishonestly marked "done"; never
      actually switched to `ctx.Registry.Deciders[].StateType`.
- [ ] **Promote `callHasOption` to `lintutil`** — was dishonestly marked "done";
      refactor A017, B025, P008, P010 to use the shared helper.
- [ ] **Fix F-series detection gaps** — F011 broad `.Exec` matching needs receiver
      type checking; F009 timer detection should include `time.Tick`/`time.After`;
      F013 HTTP handler detection should cover chi/gin/echo/fiber; F005 version
      detection should parse the version argument.
- [ ] **Review C030 over-suppression** — "any return = safe" may mask real bugs
      where a loop returns on error but has no ctx cancellation.
- [ ] **Audit S006 indicators for substring false positives** — only `pan`→`panel`
      and `aba`→`database` were fixed; other indicators may have similar substring
      collisions.
- [ ] **Add meta-test: `len(AllRules())` matches README-documented count** —
      prevents rule-count drift between code and docs.
- [ ] **Resolve D007/D009 self-lint findings** — `benchkit/phases.go` (event.New
      vs NewEvent), `command/dispatcher.go` (io.Closer vs anonymous interface).
- [ ] **Fix C017 stale doc/title** — detects 4 store types (snapshot, event,
      checkpoint, timer) but titled "snapshot store only".
- [ ] **50-item improvement backlog** — see
      `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`
      for the triaged 50 will-implement items (L1.1–L1.51). **15 items done this session**
      (L1.1, L1.3, L1.4, L1.10, L1.11, L1.12, L1.13, L1.42, items 134/139/140/150/164-166/168-171/174/176-177/179).
      **25 items pruned** as won't-implement. ~35 items remain open.
- [ ] 🔥 **Add suppression tests for 12 new rules** — C031-C034, P011-P012, D014-D015,
      A032, E016-E017, S010 all lack `//cqrs-lint:ignore(RULE)` verification tests.
- [ ] **Extract shared `isEventPayloadName`** — duplicated in d014.go and d015.go.
- [ ] **Fix P011 unused `st` parameter** in `isReadModelStruct`.
- [ ] **Narrow C032 scope** — fires on ALL ctx functions, should be handler/projector only.

---

## Metaengine (experimental; 5 phases shipped)

> Pushdown (ADR-0072), layout planning (ADR-0073), Pebble engine (ADR-0074),
> and streaming reads all shipped. Remaining work is production maturity, not
> features.

- [ ] **Wire layout planning into `Plan()`** — auto-generate `LayoutPlan` from
      `FilterOnField`/`SortOnField` query options instead of requiring manual
      `plannedSQLiteEngine` setup.
- [ ] **JSON tax reduction** — single-pass decode for SQLite reads (currently
      3 JSON operations: load row, extract field, decode payload → could be 1).
- [ ] **Generated typed read API** — `plan.Users.Get(ctx, id)` from declared
      query fields.
- [ ] **Unified 7-ADT × 3-engine test matrix** — parameterized table-driven
      harness replacing the ad-hoc per-ADT parity tests.

---

## CI / Daemon

- [ ] **Recurring lint-sweep** — the auto-commit daemon occasionally commits
      unformatted code (gci/gofumpt drift), turning `#lint` red. Either gate
      daemon commits behind `nix fmt` or run a scheduled sweep.
- [ ] **CGo-enabled CI job** — add a separate CI job with `CGO_ENABLED=1` for
      DuckDB tests (currently only in flake.nix local apps, not CI).
- [ ] **Investigate dependabot alert** `security/dependabot/10` — `gh api`
      returned no results (auth issue). Cannot diagnose without GitHub token
      permissions.

---

## Release

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod replace
  directives are needed for dev; consumers resolving the published modules
  depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

---

## Declined / Rejected (do not re-litigate)

> Kept here so decisions are not re-litigated. Full rationale in the linked
> ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Add `#verify-fast` as a pre-merge CI gate** — done (already wired as
  `verify-fast-gate` at ci.yml:128).
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
  ORM creep. Principle #1: "Library, not framework."
- **Unify VersionedStore + VersionedSeekableJournal** — different interfaces. YAGNI.
- **RollupSpec / RollupProjection** — premature abstraction. `sink.Increment` is
  the composable primitive. See analytics rollup review.
- **Redis adapter** — see ROADMAP Non-Goals (ValKey/NATS/Kafka preferred).
- **`idempotency.RefreshTTL(ctx, key, ttl)`** — dropped 2026-07-26 (YAGNI).
  Sliding window is unsafe (unbounded TTL under retry storms).
- **Centralized cross-module error-wrapping helper** — ADR-0069 decided:
  per-module helpers, capped at 3 modules.
- **Move 3-way idempotency contract test to `integration/`** — dropped
  2026-07-26. Would add 3 new direct deps to integration/.
- **Stack preset `stackpreset` builder** — dropped 2026-07-26. ~45 lines of
  trivial Go idiom; real SQL consolidation lives in `stack/sqlopt`.
- **Test infra helpers (catalogtest, storagetest, codectest)** — dropped
  2026-07-26. `idtest`, `eventtest`, `cattest` already cover all real needs.
- **`filterDetectors` extraction in cqrs-lint** — dropped 2026-07-27
  (over-engineering).
- **Split `event/` module** — 27 importers, real cohesion. Decided in v4.

---

_Long-term direction lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
