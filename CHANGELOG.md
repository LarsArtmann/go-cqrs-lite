# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### cqrs-lint v0.2.2 — loader error surfacing + --strict flag

**Fixed:**

- **Silent "Nothing to lint" on broken builds** — when `go/packages` failed to
  load packages (e.g., unresolvable dependencies from the go-cqrs-lite v4.0.0
  publish bug), the loader silently `continue`d past errors and produced an
  empty `AnalysisContext`. cqrs-lint then reported "No Go files importing
  go-cqrs-lite found. Nothing to lint." — a clean bill of health on a broken
  project. Now `BuildContext` collects per-package and per-module errors into
  `AnalysisContext.LoadErrors` and the main `run()` function surfaces them
  with a clear diagnostic and a non-zero exit code.
- **Doctor/lint split-brain on errored packages** — `doctor` read
  `ctx.Packages` (which includes errored packages) for feature detection while
  `lint` read `ctx.GoFiles` (which excludes them). The two commands could
  disagree on the project's feature profile. Now `DetectFeatures` skips
  packages with errors, making `lint` and `doctor` read the same data.
- **Doctor silently ignored `BuildContext` errors** — the `doctor` command
  called `BuildContext` but discarded the `LoadErrors` field. Now prints a
  "WARNING: package loading was partial" block with per-package error details.
- **Feature detection used unreliable import metadata** — errored packages
  may have incomplete `Imports` slices. `DetectFeatures` now skips packages
  with `len(pkg.Errors) > 0` in its import-based detection pass.

**Added:**

- **`--strict-load` flag** — exits non-zero if any packages failed to load, even
  when some packages were analyzed successfully. Without `--strict-load`, cqrs-lint
  proceeds with partial analysis and prints a warning.
- **Partial analysis warning** — in non-strict mode, when some packages loaded
  with errors but others succeeded, cqrs-lint prints a warning with the error
  count and suggests `--verbose` or `--strict-load`.
- **Message split for empty analysis** — "No Go files found. Nothing to lint."
  (no `.go` files at all) vs "Found Go files but none import go-cqrs-lite.
  Nothing to lint." (packages loaded, none import go-cqrs-lite). The old
  message conflated both cases.
- **`--verbose` load error display** — the `--verbose` output now includes a
  "Load errors (N):" section with per-package details.
- **Integration tests for loader error handling** — tests for broken modules,
  clean modules, syntax errors, strict mode, message split, and exit codes.

**Changed:**

- **`AnalysisContext.LoadErrors` field** — new `[]PackageLoadError` field holds
  per-package load errors collected during `BuildContext`. The
  `PackageLoadError` struct carries `Module`, `PkgPath`, and `Errors` fields.
- **`printLoadErrors` helper** — shared between `run()` and `doctor` for
  consistent error formatting.

### cqrs-lint v0.2.1 — health score correctness + show-suppressed

**Fixed:**

- **Health score computed on filtered findings** — the health score was
  computed on display-filtered findings (post severity/confidence filtering)
  instead of all unsuppressed findings. This meant `--min-severity`,
  `--min-confidence`, and `--fp-suspects` would change the health score,
  making it an unreliable project-health metric. Now computed on all
  unsuppressed findings.
- **InfoCapped display showed wrong cap value** — the health score display
  hardcoded the default cap (20) instead of showing the actual configured cap
  from `{"health": {"info-cap": N}}`. Added `InfoCapApplied` to HealthScore.
- **Suppressed findings shown in output** — `//cqrs-lint:ignore(RULE)`
  comments marked findings as suppressed but they still appeared in all
  output formats. Now properly filtered from output, health score, and the
  error-exit check. A summary count is printed to stderr.
- **scanner.Err() unchecked** — the suppression parser's `bufio.Scanner`
  errors (e.g., buffer overflow on >1MB lines) were silently dropped. Now
  logged to stderr as a warning.

**Added:**

- **`--show-suppressed` flag** — lists suppressed findings with their file
  location, rule ID, and suppression reason. An audit view beyond the
  counts-only `cqrs-lint doctor`.
- **`--fp-suspects` flag** — surfaces only low-confidence findings (below
  Medium confidence), which are the most likely false positives. Advisory
  mode: exit code is always 0.
- **Suppression count in output** — the main lint run now reports how many
  findings were suppressed by inline comments.

**Changed:**

- **`filterSuppressed` returns both active and suppressed slices** — enables
  the `--show-suppressed` feature and eliminates the need for a second pass.
- **Pre-allocated filter result slices** — `filterBySeverity`,
  `filterByConfidence`, `filterSuppressed`, `filterFPSuspects`, and
  `collectFindings` dedup now pre-allocate to avoid reallocation.
- **Extracted `shouldExitWithError`** — the exit-code decision is now a
  testable function instead of inline logic in `run()`.

### Documentation Health

- **Historical artifact banners** — All 41 `docs/*2026-07-1*.md` session reports
  now carry a banner at the top stating they are point-in-time snapshots, with
  links to this CHANGELOG and TODO_LIST.md for current status. This prevents
  readers from acting on stale TODO/Open/Not Started items in old reports.

### Fixed — Documentation Health Audit (2026-07-16)

- **README.md license section** — Said "MIT" but actual LICENSE file is
  PROPRIETARY. Corrected to match reality. This was a Critical documentation
  lie — consumers could have assumed MIT when the code is proprietary.
- **Module count across all docs** — AGENTS.md, README.md, FEATURES.md,
  CONTRIBUTING.md, docs/README.md, docs/v4-WISHLIST.md said "48" or "49"
  but the actual count is 52 `go.mod` files. All references now say 52 with
  a verify command (`find . -name go.mod -not -path './vendor/*' | wc -l`).
- **ROADMAP.md** — Was frozen at v3.6.0 ("Current State: v3.6.0 released")
  despite v4.0.0 being shipped. Rebuilt from scratch: current state reflects
  v4.0.0, release history table added, Long Term Vision cleaned (completed
  items removed — they belong in CHANGELOG, not ROADMAP).
- **TODO_LIST.md** — 13 completed items were still listed as open (middleware
  ordering guide, SQL TimerStore, SQL AggregateReader, lint-clean scheduling
  and scenario, ADR numbering fix, CONTRIBUTING agent rules, DeadLetterStoreAdmin
  docs, per-projection lag, session archiving, module graph, dependency model
  consolidation, event go.mod tidy). Removed; remaining items are genuinely open.
- **README.md migration reference** — Said "Migrating from v2?" but the current
  version is v4. Updated to reference both v3 and v4 migration guides.
- **FEATURES.md missing cqrs-lint** — Shipped feature (60 rules, 159+ tests)
  absent from feature inventory. Added full feature table.
- **FEATURES.md missing SQLTimerStore** — `storage.SQLTimerStore[T]` shipped
  but only MemoryTimerStore listed. Added row.
- **docs/v4-WISHLIST.md** — Said "PREP COMPLETE" and "Ready to cut" despite
  v4.0.0 already shipped. Updated to SHIPPED with correct module count.
- **docs/README.md module count** — Said "28 modules" (very stale). Updated
  to 52.

### Resolved During July 2026 Sessions

> The items below were flagged as TODO/Open/Not Started/Broken in the
> `docs/*2026-07-1*.md` session reports. They have ALL been resolved.
> This section exists so readers of those historical reports can verify
> resolution without grepping through code.

**v4 Release (2026-07-11):**

- ✅ Module path migration `/v3` → `/v4` (49 go.mod, ~750 .go files)
- ✅ Codec defaults flipped to CBOR (event, kv, snapshot, command, query)
- ✅ Deprecated alias removal (8 event/ aliases, WithNewCodec, WithReplay)
- ✅ BackfillHandler consolidation (`BackfillHandler(*SSEBroker)`)
- ✅ Storage split (eventstore/, readmodel/, sql/, relational/, view/, migrations/)
- ✅ HealthCheck on OwnedDBHandle (all SQL stores inherit)
- ✅ WithShutdownDependency (topological close ordering)
- ✅ ADR-0044 blind store envelopes (WrapEncode/UnwrapDecode)
- ✅ JSON quality audit (Deterministic + MatchCaseInsensitiveNames on all calls)
- ✅ ADRs 0047-0054 written (json/v2, transport codec, dispatch middleware, etc.)
- ✅ eventtest tag created locally (`event/v4/eventtest/v0.1.0`)
- ✅ v3 git tag backfill (v3.0.0–v3.7.1)

**DiscordSync Feedback Gaps (2026-07-10/11):**

- ✅ Gap 1: `schema.VersionedSeekableJournal` (upcasters for projectionhost)
- ✅ Gap 2: `transport/http.WithPayloadTransform` (all 3 SSE paths: live, replay, backfill)
- ✅ Gap 3: `projectionhost.SQLiteDeadLetterStore` (persistent DLQ)
- ✅ Gap 4: `prometheus.WithViews` (CQRS histogram boundaries applied)
- ✅ `DeadLetterStoreAdmin` interface (Count, ListPaged, PurgeBefore)
- ✅ `BackfillHandlerWithTransform` → consolidated into BackfillHandler(broker)

**Post-v4 Cleanup (2026-07-12):**

- ✅ 287 session artifacts archived to `docs/*/archive/`
- ✅ `docs/getting-started.md` rewritten + compile-verified
- ✅ ADR index regenerated (0032 → 0054)
- ✅ Module dependency graph (mermaid) in README
- ✅ `docs/middleware-ordering.md` (recommended order + rationale)
- ✅ CONTRIBUTING.md: Four-Tier Model (replaced stale 7-layer)
- ✅ CONTRIBUTING.md: AI Agent safety rules
- ✅ ADR numbering fix (duplicate 0047 → 0054)
- ✅ Lint-clean scheduling (11→0) and scenario (4→0)
- ✅ DiscordSync feedback doc reconciled (Gaps 3-5: REJECTED → SHIPPED)
- ✅ API surface regenerated (2209 exports)
- ✅ `fire_at` → `fireAt` JSON tag (tagliatelle compliance)
- ✅ `nix fmt` clean, `nix run .#lint` clean (0 issues)
- ✅ Doc-check passed (880+ references valid)

**cqrs-lint (2026-07-16) — Built from scratch across 6 sessions:**

- ✅ 60 rules with real detectors (correctness 12, API 19, boilerplate 15, consistency 4, architecture 7, security 3)
- ✅ 159+ tests + 3 benchmarks, 0 lint issues
- ✅ Scanner accuracy fixes (6 critical bugs: capturePayloadType, detectFoldFunc, isLikelyDecider, isOOAggregate, C004 dead rule, Type/AggregateID scanning)
- ✅ Source snippets on 34/60 detectors
- ✅ CLI overhaul: fang, go-output tables, monorepo support, module grouping
- ✅ Dead code eliminated (13 items: TypeResolver, HandlerInfo, nodeString, etc.)
- ✅ Severity filter bug fixed (alphabetical comparison → Severity.Compare)
- ✅ Catalog consolidated (3 files → 2, organized by category)
- ✅ Finding location improvements (go.mod:1:1 → real source lines on 7 rules)
- ✅ SARIF golden file test
- ✅ JSON v2 test determinism (10 non-deterministic tests fixed)
- ✅ Race detection verified (zero data races across all 50+ modules)
- ✅ Pipeline metrics wired (--verbose shows per-detector timing)
- ✅ CONTRIBUTING.md rule development guide

**Documentation Health Audit (2026-07-16):**

- ✅ README.md license corrected (MIT → PROPRIETARY)
- ✅ Module count corrected across all docs (48 → 52)
- ✅ ROADMAP.md rebuilt (v3.6.0 → v4.0.0)
- ✅ TODO_LIST.md cleaned (13 stale items removed)
- ✅ FEATURES.md updated (cqrs-lint + SQLTimerStore added)
- ✅ Historical artifact banners added to all session reports

### Fixed — Prior Session Fixes

- **README.md and docs/getting-started.md code examples** — Command examples
  were missing the required `ID()` method (inherited via `*command.BasicCommand`
  embedding). Getting-started also used `event.NewEvent` instead of `event.New`
  for typed payloads, referenced nonexistent `event.NewMemoryBus()`, and used
  `Fold` instead of `Apply` on `decider.Decider`. All examples now compile-verified.
- **api-stability golden file** — `docs/api_surface.txt` was stale (missing
  `kv.OpIsNull`, `kv.OpIsNotNull`, `kv.ViewUpdater` shipped in v4.0.0). Regenerated.
- **kv/benchmark_test.go** — Replaced `[]byte(fmt.Sprintf(...))` with
  `fmt.Appendf(nil, ...)` to clear `fmtappendf` diagnostics.

### Changed

- **scheduling: JSON tag `fire_at` → `fireAt`** — Tagliatelle (camelCase)
  compliance. The scheduling module is new in v4; no pre-v4 data exists to break.
- **scheduling: Magic numbers extracted to named constants** —
  `defaultPollInterval`, `defaultMaxRetries`, `defaultRetryDelay`,
  `jitterHalfDivisor`. All `ctx.Err()` returns wrapped per wrapcheck.
- **scenario: Lint cleanup** — `exhaustruct` nolints on builder-pattern structs,
  `errname`/`varnamelen` fixes.

### Documentation

- **287 session artifacts archived** — `docs/{status,planning,research,reviews,
quality,architecture-understanding,brainstorming,modularization}/` timestamped
  files moved to `archive/` subdirectories with explanatory READMEs.
- **Feedback doc reconciled** — DiscordSync round 3 appendix Gaps 3-5 changed
  from "REJECTED" to "SHIPPED" to match actual code.
- **ADR index extended** — From 0032 to 0054 (21 new entries, duplicate 0047
  renumbered to 0054).
- **CONTRIBUTING.md** — 7-layer model replaced with Four-Tier Model (ADR-0046).
  Added "Working with AI Agents" section.
- **New docs/middleware-ordering.md** — Recommended middleware application order
  for all 30+ middlewares.

## [cmd/cqrs-lint/v0.2.0] - 2026-07-17

### cqrs-lint — DiscordSync feedback: false-positive reduction & fairer scoring

These changes close the feedback loop on the DiscordSync cqrs-lint report
(18 of 34 findings were D002 false positives). Rule heuristics are now shaped
by what real consumers do, and the health score no longer lets info-level noise
drown real bugs.

**Fixed — rule false positives (prior session, now changelogged):**

- **C001 closure-escape** — `txVarEscapesToArg` now skips the false positive
  where the tx variable is passed to a callback that contractually owns the
  commit (suggesting `return tx.Commit()` there would double-commit). The old
  test encoded the false positive; replaced with a genuine missing-commit case.
- **C008 money corroboration** — money field names split into strong
  (amount/price/cost/balance/fee — fire alone) and weak
  (value/total/charge/payment/salary — need a money struct/package name).
  Eliminates the observability/metrics false positives.
- **D005 version-token regex** — `HasPrefix("v") && len >= 3` replaced with
  `^v\d+\.\d+`, rejecting prose words ("via", "version") and bare major
  versions ("v3") while keeping real semver references.
- **A005 broadcast vs projection** — `classifyCallbackBody` inspects the
  SubscribeAll callback and suppresses fire-and-forget fan-out (SSE broadcasters,
  stats notifiers) that has broadcast calls and no persistence calls.

**Added — D002 external-API opt-out (biggest remaining false-positive source):**

- D002 now excludes structs that mirror an external API (Discord/Stripe/GitHub),
  whose snake_case JSON tags are dictated upstream and aren't a local style
  choice. Two complementary opt-outs:
  - **Config** (`.cqrs-lint.json`): `"rules": {"external-api-struct-prefixes": ["Discord", "Stripe"]}`
    marks every struct whose name starts with a listed prefix.
  - **In-source marker**: `//cqrs-lint:external-api` on a struct's doc comment
    marks a single struct (works on both single `type Foo struct{}` and grouped
    `type ( ... )` blocks).
- `cqrs-lint doctor` now prints the loaded `rules` overrides so consumers can
  verify their prefix list was picked up.

**Improved — recall & context-awareness:**

- **C001 tx-use signal** — a function that uses the tx (`tx.Exec`, `tx.QueryRow`,
  any non-lifecycle method) now flags even without a bare `return nil`. tx usage
  is a stronger bug signal than the return shape; the old gate missed functions
  that return a sentinel/wrapped error after using the tx.
- **A005 widened broadcast signals** — added `Publish`, `Emit`, `Forward`,
  `Dispatch`, `WriteTo`, `Flush` to the fan-out detector (safe: a callback that
  both broadcasts and persists still flags). Catches deriver/republish patterns
  that aren't projections.
- **C008 project-aware downgrade** — when no package or struct anywhere in the
  project looks monetary, strong-field findings downgrade to Info/Low
  ("maybe money") instead of Warning/Medium. Non-payments codebases no longer
  get full-severity money warnings on coincidental `amount`/`balance` fields.

**Changed — fairer health score:**

- **Confidence weighting** — each finding's deduction is scaled by its
  confidence: High/Full = full deduction, Medium = 75%, Low = 50%. A flood of
  Low-confidence heuristic matches no longer costs the same as confirmed bugs.
  (No-confidence findings keep full weight, preserving prior behavior.)
- **Info cap** — total Info-severity deductions are capped at 20 points so a
  chatty style rule can't outweigh a Critical correctness bug.

> **Migration impact:** both scoring changes shift health scores on re-run.
> Projects that previously lost many points to info-level findings (especially
> D002 mixed-casing) will see their score rise. The relative ranking of findings
> is unchanged; only the aggregate penalty is fairer. Pin to `v0.1.0` if you
> depend on the old absolute score values.

## [cmd/cqrs-lint/v0.1.0] - 2026-07-16

First release of the domain-aware linter for CQRS/ES Go projects.

### Added

- **60 rules across 6 categories** — correctness (12), API misuse (19), boilerplate
  (15), consistency (4), architecture (7), security (3). Each rule has a real detector
  backed by AST analysis.
- **CLI** — struct-tag flags, `--min-confidence`, `--health-score`, `--verbose`,
  `--color`, `--format` (text/json/sarif/markdown). Monorepo support with per-module
  output grouping.
- **Config file** — `.cqrs-lint.json` via cmdguard for project-specific rule
  configuration, exclusions, and severity overrides.
- **Suppression comments** — `//cqrs-lint:ignore(rule-id) reason` for inline false-positive
  suppression.
- **Health score** — `--health-score` computes a 0–100 code health metric weighted by
  finding severity and category.
- **Source snippets** — 34/60 detectors include the offending source line in findings.
- **SARIF output** — GitHub Code Scanning integration via `--format sarif`.
- **165 tests** — positive tests (rule fires), negative tests (rule doesn't fire),
  scanner accuracy tests, CLI output tests, SARIF golden file test.
- **Auto-fix** — 4 rules support `--fix` for automatic remediation.

### Fixed

- **A011 detector compile bug** — `slices.Contains()` was called with zero arguments
  (incomplete refactor in `b3931503`). Fixed to `slices.ContainsFunc` with a suffix
  predicate. Detector now correctly identifies event payload structs.

## [retry/v4.0.0] - 2026-07-16

First release of the zero-dependency retry module.

### Added

- **`retry.Do(ctx, fn)`** — executes `fn` with exponential backoff and jitter until
  success, context cancellation, or max retries exhausted.
- **`retry.Config`** — configurable max attempts, initial delay, max delay, multiplier,
  and jitter factor. Sensible defaults via `retry.DefaultConfig`.
- **`retry.ErrExhausted`** — sentinel error returned when all retries are spent.
- **`retry.ErrCanceled`** — sentinel error returned when context is cancelled mid-retry.
- **Zero dependencies** — no CQRS, no OTel, no external imports. Pure Go stdlib.

## [idempotency/kvstore/v4.0.0] - 2026-07-16

First release of the KV-backed idempotency subpackage.

### Added

- **`kvstore.KVStore`** — idempotency store backed by the `kv.Store` interface.
  Works with any KV implementation (memory, Pebble, SQL-backed).
- **`kvstore.KVBackend`** — interface for pluggable KV backends.
- **Atomic check-and-set** — prevents duplicate processing under concurrent access.

## [v4.0.1 patches] - 2026-07-16

Per-module patch releases for bug fixes and additive features since v4.0.0.

### Fixed (projectionhost/v4.0.1)

- **Stagger-shutdown leak** — `defer wg.Done()` moved to `Start()` goroutine to
  prevent WaitGroup underflow during graceful shutdown.
- **Status() non-deterministic ordering** — `Host.Status()` output now sorted by
  worker name for deterministic test assertions and dashboard output.
- **purgeDLQ field initialization** — `purgeDLQ` explicitly initialized in constructor
  to prevent zero-value ambiguity.

### Fixed (watermill/v4.0.1)

- **EventBus dispatch deadlock** — `Close()` now releases `b.mu` before calling
  `backend.Close()`, preventing deadlock with background dispatch goroutine.

### Added (storage/v4.0.1, kv/v4.0.1)

- **IS NULL / IS NOT NULL operators** — `kv.OpIsNull`, `kv.OpIsNotNull` for NULL-aware
  view queries. Supported in `SQLViewStore.Query` and `Count`.
- **`RawWhere` escape hatch** — raw SQL WHERE clause injection for complex predicates
  not expressible via the `Condition` struct.
- **`ViewUpdater` interface** — incremental view updates (read-modify-write) for
  projections that need to merge event data with existing row state.
- **BLOB support** — `ViewColumn` now supports `BLOB` type for binary payload storage.

### Changed (metadata normalization batch)

- Module dependency references normalized across 15 modules (event, decider, middleware,
  signing, encryption, listing, codec, otel, schema, snapshot, id, graph, scenario,
  scheduling, transport/http). `metadata/v4` pinned to v4.0.0 in all go.mod files.
  No API changes — internal go.mod housekeeping only.

## [4.0.0] - 2026-07-11

**Major version cut — CBOR defaults, API cleanup, BackfillHandler consolidation.**

This release flips all codec defaults from JSON to CBOR (with full backward
compatibility via envelope wrapping), removes deprecated APIs, consolidates
the SSE backfill API, and migrates module paths to `/v4`.

See [`docs/migration/MIGRATION-GUIDE.md`](docs/migration/MIGRATION-GUIDE.md)
for step-by-step upgrade instructions.

### Breaking Changes

1. **Module path migration `/v3` → `/v4`** — All 49 `go.mod` files and every
   import path updated. Consumers must update `go.mod` require directives and
   all import statements. `go mod tidy` resolves most of this automatically.

2. **Codec defaults flipped to CBOR** — `event.DefaultCodec`, `kv.NewTypedStore`,
   `snapshot.NewTypedStore`, `command.NewTypedStore`, `query.NewTypedStore`,
   and `stack.ReadModel`/`Materialize` all default to `CBORCodec` instead of
   `JSONCodec`. **No data migration required** — envelope wrapping (ADR-0044)
   stamps the encoding on every write and auto-detects on read. Old JSON data
   is transparently handled via the permanent JSONCodec fallback (ADR-0050).
   See ADR-0053 for the full rationale.

3. **Deprecated aliases removed** — 8 event/+schema/ aliases deleted
   (`AggregateRef`, `Tracing`, `CustomData`, etc.). `event.WithNewCodec`
   removed (use `WithCodec`). `event.WithReplay` removed (use
   `WithProcessingMode`). `query.Handler` deprecation notice removed.

4. **`BackfillHandler` signature changed** — Now takes `*SSEBroker` instead of
   `event.SeekableJournal`. The broker's journal and payload transform are used
   directly, unifying SSE and REST backfill under a single codec configuration.
   `BackfillHandlerWithTransform` removed (consolidated — configure the
   transform on the broker via `WithPayloadTransform`).

### Migration

```go
// Before (v3):
import "github.com/larsartmann/go-cqrs-lite/event/v3"
handler := http.BackfillHandler(journal)

// After (v4):
import "github.com/larsartmann/go-cqrs-lite/event/v4"
handler := http.BackfillHandler(broker) // broker must have WithReconnectJournal
```

Codec defaults: no action needed. Old data reads correctly. New data is CBOR.
To revert process-wide: `event.DefaultCodec = codec.JSONCodec{}`.

### Added

- **`HealthCheck` on `OwnedDBHandle`** — all SQL stores (event, snapshot,
  checkpoint, command, query) now inherit `HealthCheck(ctx)` via embedding.
  Previously only `*SQLEventStore` implemented it.
- **`SSEBroker.Journal()` and `SSEBroker.PayloadTransform()` accessors** —
  exposed for `BackfillHandler` and consumer introspection.
- **ADR-0053** — Unified codec default flip rationale and backward-compat
  guarantees.
- **Envelope backward-compat integration tests** — `kv.TestTypedStore_Migration_*`
  verify old raw JSON data reads through new CBOR-default stores, and mixed
  old+new data coexists correctly.
- **`storage/eventstore/` sub-package** — SQLEventStore, SQLSnapshotStore,
  SQLCheckpointStore extracted into focused package. Full backward compat via
  type aliases and constructor re-exports in `storage/`.
- **`storage/readmodel/` sub-package** — SQLKVStore extracted into focused
  package. Full backward compat via type aliases and constructor re-exports.
- **`WithShutdownDependency` integration tests** — now tested through real
  `stack.New()` constructor path with close-order tracking, not just struct
  literals.

#### Dead Code Cleanup

- **Deleted `event/arena_experiment.go`** — 36-line stub with zero consumers, no tests,
  and no real GC benefit (arena-allocating a struct header while its fields remain
  heap-allocated saves nothing). Removed `goexperiment.arenas` from `flake.nix`,
  `scripts/check-module-isolation.sh`, and all living docs.

#### Projectionhost Observability

- **`Host.LagPerProjection() map[string]time.Duration`** — per-worker lag keyed by
  projection name, for Prometheus dashboards with `WithLabelValues`. Returns 0 for
  workers that haven't processed any event yet (item 38).
- **`WorkerState.Lag` field** — `Lag time.Duration` populated in `snapshot()` via the
  new `worker.lagDuration()` method. Previously only available via the aggregate
  `Host.LagDuration()` (item 39).
- **`Reset(ctx, name, opts...)` with `WithPurgeDeadLetters()`** — projection reset
  now optionally purges dead-letter entries from the configured `DeadLetterStore`.
  Backward compatible: `Reset(ctx, name)` still works without purging (item 46).
- **`Host.LagDuration()` refactored** — now delegates to `worker.lagDuration()` for
  consistency (returns max lag across all workers).
- **6 new tests** — lag before/after processing, per-projection map, DLQ purge
  with/without flag, WorkerState.Lag in Status().

#### Scenario Projection Tests

- **`scenario.GivenProjection` tests** — added `ThenError`, multiple-events, and
  empty-events tests covering the projection DSL more thoroughly (item 48).

#### Race Detector Coverage

- **Full `-race` suite** — all 48 modules pass with race detector. Only
  `cmd/api-stability` fails (pre-existing: subprocess doesn't inherit
  `goexperiment.jsonv2` tags) (item 13).

#### Documentation

- **ADR-0043 Part B** — consumer operational guide for the two `DeadLetterEntry`
  types: decision tree, code examples for dispatch-side vs projection-side DLQ,
  and structural comparison table explaining why they can't merge (item 45).
- **README docs freshness** — fixed stale `testutil` API references
  (`MustNewCmd`→`NewCmd`, removed `ParseAggID`, `NoopCommandHandler{}`→`NoopCommandHandler()`)
  and `v2`→`v3` paths in `testutil/README.md`.
- **AGENTS.md** — updated projectionhost key patterns with `LagPerProjection()`,
  `WorkerState.Lag`, and `WithPurgeDeadLetters()` examples.
- **`projectionhost/README.md`** — added Status & Lag section with dashboard examples,
  Reset section with purge option.

#### P3 Polish & Cleanup

- **Restored bundle.go architectural comment** — documented the Bundle↔CatchUpSubscriber
  relationship (SeekableJournal + Subscriber + CheckpointStore fields compose into the
  replay-then-live projection pipeline) after dead `var _` code was removed.
- **Fixed histogram test hard-coded values** — `prometheus/exporter_test.go` now references
  `cqrsotel.CQRSHistogramBoundaries` directly instead of duplicating the literal. If boundaries
  change in `otel/`, the test tracks the real value.
- **Verified `nix flake check`** — passes after `scripts/check-module-layers.sh` changes.
- **Race detector verified** on `stack/` and `example/taskmanager/` — both pass with `-race`.
- **CBOR→JSON SSE e2e test** — `TestSSEHandler_PayloadTransform_CBOR_ToJSON_BrowserFlow`
  in `transport/http/sse_options_test.go` verifies CBOR events transform to JSON for browser
  consumption across all SSE delivery paths.
- **Fixed taskmanager integration test failures** — `example/taskmanager` now uses JSON codec
  (`event.DefaultCodec = codec.JSONCodec{}`) via `codec_init.go` to fix CBOR decode failures
  in the projection pipeline. Events are also human-readable in the database and SSE stream.

#### DLQ Admin Operations & SQLite Dead-Letter Store (`projectionhost`)

- **`DeadLetterStoreAdmin` interface** — production management operations for dead-letter stores:
  `Count(ctx) (int64, error)`, `ListPaged(ctx, projectionName, offset, limit)`,
  `PurgeBefore(ctx, before time.Time) (int64, error)`.
- **`SQLiteDeadLetterStore`** — persistent SQLite-backed dead-letter store (survives restarts).
  Full column layout, index strategy, and reconstruction docs in `projectionhost/doc.go`.
- **DLQ index optimization** — replaced redundant `idx_pdl_projection` with
  `idx_pdl_projection_time(projection_name, failed_at)` (covers List + pagination + ORDER BY)
  and `idx_pdl_failed_at(failed_at)` (covers List-all + PurgeBefore).
- **DLQ test coverage** — stress test (10k entries: Count, ListPaged, PurgeBefore), concurrent
  store test (20 goroutines × 50 entries = 1000 writes), corrupt-payload test (surfaces error
  with event ID, no panic).

#### VersionedSeekableJournal (`schema`)

- **`schema.VersionedSeekableJournal`** — wraps `event.SeekableJournal` with upcaster chains,
  enabling schema evolution for `projectionhost.New()` (which requires `SeekableJournal`).
  Cross-module integration test with `projectionhost.New()` included.
- **Property tests** (rapid, 100 iterations each) — upcaster chain (random depth + events),
  passthrough (unregistered types), ReadFrom (position-based seek with upcasting).
- **Mid-stream upcast error test** — 10 events, upcaster fails on event 5, error propagates
  from both ReadAll and ReadFrom (no panic, no partial results).
- **Benchmarks** — ReadAll no-upcasters (140µs), ReadAll 3-chain (7.5ms), ReadFrom 3-chain
  500 events (536µs).

#### SSE Transform & Replay Safety (`transport/http`)

- **`WithPayloadTransform`** — wire-format transcoding (e.g., CBOR→JSON for browsers) applied
  uniformly across all three SSE paths: live, replay, and backfill.
- **`BackfillHandlerWithTransform`** — REST backfill endpoint with the same payload transform.
- **`SSEReplayBudgetDisabled = -1`** sentinel — `WithReplayByteBudget(0)` now auto-defaults to
  the 8MB safety budget; pass -1 to explicitly disable budgeting.
- **Large-payload byte-budget test** — 100KB × 5 events under 250KB budget boundary verification.

#### Blind Store Encoding Envelopes (`codec`, `kv`, `snapshot`, `command`, `query`)

- **`codec.WrapEncode` / `codec.UnwrapDecode`** — ADR-0044 encoding stamps on blind stores.
  All four blind stores (kv, snapshot, command, query) are now self-describing: the codec is
  stamped on write and auto-detected on read. `UnwrapDecode` falls back to JSONCodec for
  backward compat with pre-envelope data.

#### Prometheus Custom Views (`prometheus`)

- **`WithViews(views ...metric.View) Option`** — custom metric views for the Prometheus exporter.
  Compose with `cqrsotel.NewCQRSViews()` to apply CQRS histogram boundaries.

#### Stack Health Checks & Shutdown Ordering (`stack`)

- **`HealthChecker` interface + `Bundle.HealthCheck(ctx)`** — pings the database and calls
  `HealthCheck` on every registered closer that implements the interface. Enables Kubernetes
  liveness/readiness probes.
- **`WithShutdownDependency(before, after string) Option`** — topological sort (Kahn's algorithm)
  for close-time dependency ordering. Projections drain before the event store closes. Cycles
  fall back to registration order.

#### Decider Hot-State Cache (`decider`)

- **`StateCache[State]` interface + LRU implementation** — incremental loads: on cache hit,
  `LoadFromVersion(cachedVer)` + fold delta → O(new events) instead of O(total events).
  `WithStateCache[State]` option enables it. Cache updated on every Execute, invalidated on
  fold/store errors. Benchmark: 7.4x faster Load (2090→283 ns/op) with 500-event history.
  Process-local, best-effort, zero new dependencies.

#### Read-Pressure Snapshot Strategy (`snapshot`)

- **`ReadPressure` strategy** — triggers snapshots based on read count (hot-read, cold-write
  aggregates). `AggregateAwareStrategy` and `ReadTracker` optional interfaces.
  Composable with `EveryNEvents` via `WithInnerStrategy`. Wired into decider Repository via
  optional interface checks. Fully backward compatible.

#### id/ + metadata/ Package Extraction

- **`id/` package** — branded IDs (`AggregateRef`, `EventID`, markers) extracted from `event/`
  into a standalone, zero-event-dependency module.
- **`metadata/` package** — `Tracing`, `CustomData[K]`, and shared metadata types extracted from
  `event/` for cross-module reuse (command, query, event).

#### SQL Error Classification Auto-Registration (`storage/sql`)

- **`errorfamily.RegisterStdlibDefaults()`** called via `init()` — registers stdlib error
  classifications automatically on import.
- **Database driver classifiers** — SQLite BUSY/LOCKED→Transient, CONSTRAINT→Conflict;
  Postgres SQLSTATE class mappings. Registered via `init()` in `storage/sql/classify_init.go`.

#### Idempotency Middleware — Generic Factory (`middleware/v4`)

- **`middleware.NewIdempotency[M]`** — generic idempotency middleware factory following the
  `NewValidation[M]` / `NewTracing[M]` pattern. Works for all 3 CQRS message types:
  - **`middleware.CommandIdempotency(store, ttl, keyExtractor)`** — command dedup using the
    command's minted ID by default (pass `nil` for keyExtractor).
  - **`middleware.EventIdempotency(store, ttl, keyExtractor)`** — event dedup using the event's
    minted ID by default (pass `nil` for keyExtractor). For ordered event consumption (projections),
    checkpoint-based dedup (`projectionhost`) is structurally stronger — use this when you don't own
    the checkpoint (webhooks, external sinks, cross-system delivery).
  - **`middleware.QueryIdempotency(store, ttl, keyExtractor)`** — query dedup. Requires a non-nil
    keyExtractor (queries have no built-in identity). Panics at construction if nil.
- Store errors are classified as `Transient` via `errorfamily.Wrapf`. Duplicate keys return
  `idempotency.ErrDuplicate` (a `Conflict` family error).

#### Documentation & ADRs

- **ADR-0043** — Dead-letter store design (dispatch-side vs projection poison entries).
- **ADR-0044** — Blind store encoding stamps (envelope wrapper).
- **ADR-0047** — json/v2 case-insensitive decode.
- **ADR-0048** — Deterministic encoding.
- **ADR-0049** — Dispatch-time middleware ordering.
- **SECURITY.md** — vulnerability reporting process.
- **Consumer migration guide** — `docs/migration/MIGRATION-GUIDE.md` for id/ + metadata/ extraction.
- **SKILL.md** updated — `VersionedSeekableJournal`, `BackfillHandlerWithTransform`, `WithViews`
  added to decision matrix + cheat sheet. doc-check passes (868 refs).
- **metadata/ + id/** added to AGENTS.md module table.
- **Deprecated alias cleanup** — 8 deprecated aliases deleted from `event/` + `schema/`
  (AggregateRef, Tracing, CustomData, etc.). Internal usage migrated to `id.` and `metadata.`.
  `event.WithNewCodec` removed (use `WithCodec`). `event.WithReplay` removed (use `WithProcessingMode`).
  `query.Handler` deprecation notice removed — it is the dispatch core, not deprecated.

### Changed

#### CBOR is the Default Codec (`event`, `codec`, `stack`)

- **`event.DefaultCodec`** is now `codec.CBORCodec{}` (was `JSONCodec{}`). Events are
  self-describing (`evt.Encoding()` stamp on every event), so mixed JSON+CBOR streams decode
  correctly via `DecodePayloadAuto`. Blind stores are self-describing via ADR-0044 envelopes.
  Blind store defaults (kv, snapshot, command, query) also flipped to CBOR.

#### Deprecated Alias Cleanup

- **~200 usages across 42 files** updated from `event.AggregateRef` → `id.AggregateRef`,
  `event.Tracing` → `metadata.Tracing`, etc. All internal code now uses `id.` and `metadata.`
  directly. SA1019 deprecated alias warnings eliminated across all modules.

#### JSON Quality Audit

- **`Deterministic(true)`** added to all `Marshal` calls in signing, encryption, event, storage,
  transport, listing, catalog.
- **`MatchCaseInsensitiveNames(true)`** added to all `Unmarshal` calls across all modules.
  Implements ADR-0047 (case-insensitive decode) and ADR-0048 (deterministic encoding).

#### errorfamily.HTTPStatus() Adoption (`example/taskmanager`)

- **`writeCQRSError`** simplified from 15-line switch statement to a 1-line
  `errorfamily.HTTPStatus(err)` call.

#### Dispatcher Middleware-at-Dispatch-Time Fix (`dispatcher`)

- Middleware can now be added in any order — the chain is rebuilt at dispatch time, not
  construction time. Documented in `dispatcher/doc.go`.

#### CI Sync Scripts

- **`scripts/check-workspace-sync.sh`** — verifies go.work ↔ flake.nix module sync. 8 missing
  modules added to flake.nix testModules.
- **`scripts/check-api-stability-sync.sh`** — verifies go.work ↔ api-stability tracking sync.
  12 missing modules added to api-stability tracking.
- **`scripts/check-module-layers.sh`** — dependency budget violations fixed (deriver=4,
  stack=14). projectionhost raised 7→9, watermill raised 8→9 (SQLite DLQ + metadata extraction).

#### Idempotency Module Slimmed Down (`idempotency/v4`)

- Removed `idempotency.CommandIdempotency`, `idempotency.KeyExtractor`, and
  `idempotency.CommandIDKey` — replaced by the generic `middleware.CommandIdempotency` factory.
- Module dependencies reduced: `command/v4` and `id/v4` dropped from direct deps. Now depends on
  `kv/v4` + `go-error-family` only.
- Layer changed from Layer 2 (→command, event, id, kv) to Layer 1 (→kv).
- Added to `flake.nix` testModules and `cmd/api-stability` module tracking (was missing from both
  since module creation).
- Pre-existing lint issues fixed: `exhaustruct`, `nestif`, `revive` (unused ctx), `wrapcheck`.

### Fixed

- **`WithReplayByteBudget(0)` semantics** — 0 now auto-defaults to the 8MB safety budget;
  `SSEReplayBudgetDisabled = -1` explicitly disables budgeting.
- **`api_surface.txt`** — removed dead `JSONCodecV2` entry. Regenerated golden with all new
  modules tracked (2212 exports).
- **File-size violations** — 3 production files split under the 350-line CI limit:
  `signing/cose.go` → `cose_sign1.go`, `cmd/doc-check/main.go` → `exports.go`,
  `catalog/eventcatalog/frontmatter_render.go` → `frontmatter_convert.go`.
- **Dead code removed** — `codec/jsonv2_experiment.go` (dead Go experiment tag gated zero files).
  All 4 `var _ =` hacks removed (`sse_backfill.go`, `example/taskmanager/http.go`,
  `stack/bundle.go`, `example/taskmanager/setup.go`).

### Security

- **SECURITY.md** — documents the vulnerability reporting process.

## [3.7.1] - 2026-07-07

**Release documentation completeness — all 48 modules synced to v3.7.1.**

v3.7.0 was published with 46 modules tagged (otel skipped as unchanged). This
patch releases all 48 modules at a uniform version for consumer dependency
alignment, and adds the CHANGELOG/version-string updates that v3.7.0 shipped
without.

### Fixed

- **CHANGELOG.md** — added [3.7.0] section (was missing from the v3.7.0 release).
- **flake.nix** — package version bumped to 3.7.0 (was stale at 3.6.0).
- **v4-WISHLIST.md** — "Current major" updated to v3.7.0 (was stale at v3.4.0).
- **otel/v4.7.0** tagged for version-line consistency (module unchanged since v3.5.0).

### Verified

- **govulncheck**: 0 vulnerabilities across all 48 modules.
- **All gates green**: build, test, lint, isolation (GOWORK=off), version drift.

## [3.7.0] - 2026-07-07

**Dedup module extraction, SSE production hardening, go-error-family direct adoption, SQLTimerStore.**

### Added

#### Dedup — Bounded Dedup Ring Buffer (`dedup/v4`, first release)

- **`dedup.Ring`** — O(1) fixed-capacity ID deduplication for stream boundaries.
  Extracted from the inline SSE and watermill implementations into a reusable
  module. Used by `projectionhost`, `watermill`, and `transport/http` (SSE).

#### SSE Production Hardening (`transport/http`)

- **Fanout and drop policies** for high-fanout deployments — configurable behavior
  when subscriber count exceeds budget.
- **Backfill REST endpoint** — query missed events by aggregate or timestamp range.
- **Auth middleware** — pluggable authentication for SSE connections.
- **Offline reconnection example** — reference pattern for resilient clients.
- **Byte-budget replay** — stops mid-batch when a configurable byte limit is
  exceeded (prevents memory blowups on large replays).
- **Replay timeout** — caps replay duration; sends an advisory event on timeout
  before live streaming begins.

#### ProjectionHost Graceful Teardown (`projectionhost`)

- **`WorkerDraining` status** — workers transition through Draining before Stopped,
  enabling graceful shutdown that respects in-flight events.

#### SQLTimerStore (`storage`)

- **`SQLTimerStore`** — persistent `scheduling.TimerStore` backed by SQL, enabling
  durable deadline timers that survive restarts.

#### Watermill Batched Replay (`watermill`)

- **CatchUpSubscriber replay** now batches historical events into fixed-size chunks
  instead of loading the entire backlog at once.

#### Pebble GracefulClose (`stack/pebble`, `storage/pebble`)

- **`GracefulClose(ctx)`** — bounds `Close()` with a timeout, preventing hung
  shutdowns on slow flushes.

### Changed

#### Go-Error-Family Direct Adoption

- All modules now import `go-error-family` directly instead of through the `event/`
  package facade. The `event/` package retains type aliases (`event.Family`,
  `event.Error`) for backward compatibility, but error construction and
  classification functions now use `go-error-family` directly.
- **`go-error-family` bumped to v0.6.1.**

#### Turso Database Rebrand

- "LibSQL" terminology replaced with "Turso Database" across the codebase and
  documentation.

### Fixed

- **dedupRing panic** — removed panic from constructor on invalid capacity; returns
  error or falls back to default.
- **Prometheus provider shutdown** — now returns nil on successful shutdown.
- **Tombstone projection** — persists correctly across KV store roundtrip.
- **gRPC test nil-deref** — guard added.
- **Pattern B sentinels** — replaced placeholder sentinels with real versions for
  external consumption.

### Infrastructure

- **47 modules tagged at v3.7.0** (including first-ever `dedup/v4.7.0` and
  version-line-consistency tag for `otel/v4.7.0`).
- Replace directives completed across all modules for GOWORK=off build correctness.
- Go toolchain at 1.26.4.

## [3.6.0] - 2026-07-05

**Error-family taxonomy full sweep, deriver module, flagship example consolidation.**

### Added

#### Deriver — Event→Command Derivation (`deriver/v4`, `example/taskmanager`)

- **`deriver.Deriver`** — reacts to events by deriving new commands. Chainable `Then`,
  `Filter`, `Idempotent`, and `AsHandler` operators for declarative event→command
  pipelines. Implements ADR-0040.
- **Taskmanager example** — auto-assigns new tasks via a `user.created` →
  `task.assign` derivation, demonstrating real-world usage.

#### Flagship Example Consolidation

- **9 examples → 2**: the scattered `deployer-first`, `deployer-first-multidb`,
  `deployer-first-heterogeneous`, `encryption`, `deriver`, `graph-demo`,
  `projectionhost`, `todo`, and `user` examples are consolidated into:
  - **`example/taskmanager`** — the complete reference: event sourcing, projections
    (KV + tombstone), SSE streaming, snapshot strategy, signing, ProjectionHost with
    DLQ, deriver integration.
  - **`example/getting-started`** — minimal getting-started guide.

### Changed

#### Error Family Taxonomy — Full Sweep

Adopted the 5-family error taxonomy (Rejection / Conflict / Transient /
Infrastructure / Corruption via `go-error-family`) across all production modules:

| Module                 | Classification                                                                |
| ---------------------- | ----------------------------------------------------------------------------- |
| `storage`              | `WrapInfrastructure` for event store streams, memory streams, PG bus listener |
| `storage/pebble`       | `WrapInfrastructure` for backend, command read, iteration paths               |
| `storage/relational`   | `WrapInfrastructure` for projection, schema, sink                             |
| `storage` (KV SQL)     | `WrapTransient` for idempotency KV store                                      |
| `middleware`           | `WrapInfrastructure` for dead-letter SQL store                                |
| `catalog/eventcatalog` | `WrapCorruption` for frontmatter marshal                                      |
| `projectionhost`       | `WrapInfrastructure` for dead-letter list                                     |
| `cmd/cqrs-gen`         | `WrapInfrastructure` for scan/walk/parse                                      |
| `stack/sqlite`         | `WrapInfrastructure` for preset errors                                        |
| `stack/postgres`       | `WrapInfrastructure` for preset + `WrapRejection` for bad DSN                 |
| `stack/pebble`         | `WrapInfrastructure` for preset errors                                        |
| `stack/turso`          | `WrapInfrastructure` for preset errors                                        |
| `idempotency`          | `WrapTransient` for KV store                                                  |
| `command`              | Taxonomy for memory bus + typed store                                         |
| `graph`                | Taxonomy for memory driver                                                    |

### Fixed

- **Tombstone projection persistence** — tombstone marks now survive KV store
  roundtrips correctly (`example/taskmanager/projection.go`).
- **Event signing middleware wiring** — signing middleware now correctly wired via
  EventBus type assertion instead of direct `UsePublish`.
- **eventtest module path** — moved to `event/v4/eventtest/` to match the Go module
  path spec for VCS resolution (ADR-0045). Fixes `go mod tidy` warnings.
- **Invalid v0 pseudo-versions** — corrected pseudo-versions for `/v4` module paths
  in cross-module `go.mod` dependencies.
- **go.mod/go.sum stabilization** — convergence tidy across all modules; workspace
  replace directives aligned for consistent local resolution.

## [3.5.0] - 2026-07-01

**CBOR promoted to first-class default, encoding-aware validator, symmetric validation.**

### Added

#### CBOR Adoption Primitives — `event/v4`, `stack/v4`

- **`event.DefaultCodec`** — mutable package-level variable (like `http.DefaultClient`)
  that controls the codec used by `event.New()` when no `WithCodec` option is passed.
  Defaults to `JSONCodec{}` for backwards compatibility. Set to `CBORCodec{}` for
  process-wide CBOR adoption: `event.DefaultCodec = codec.CBORCodec{}`.
- **`stack.WithEventCodec(c codec.Codec) Option`** — one-call adoption for both event
  payloads and read models. Sets `bundle.EventCodec()` and also `bundle.DefaultCodec()`.
  Consumers use `bundle.EventCodec()` in decide functions via `event.WithCodec()`.
- **`Bundle.EventCodec()`** — accessor for the event payload codec. Falls back to
  `event.DefaultCodec` when unset.

#### Codec Utilities — `codec/v4`

- **`AutoDetect(data []byte) Encoding`** — sniffs the serialization format from raw
  bytes by examining structural first-byte patterns. Distinguishes JSON from CBOR.
  Best-effort heuristic for diagnostics and tooling, not a security boundary.
- **`Size(v any) (jsonSize, cborSize int)`** — encodes v with both codecs and returns
  the byte sizes. Useful for evaluating CBOR adoption before committing.
- **`keyasint` example** — `ExampleCBORCodec_keyasint` demonstrating CBOR integer keys
  (CWT claim registry pattern) for 22% size reduction over string keys.

#### gRPC Codec Injection — `transport/grpc/v4`

- **`WithCodec(c codec.Codec) Option`** — shared functional option for
  `RegisterQueryService`, `NewQueryClient` (and future command/event transport).
  Defaults to JSON for backwards compatibility. Both server and client must use the
  same codec.
- **`QueryServer.codec`** — query results are encoded with the configured codec
  instead of hardcoded `json.Marshal`.
- **`QueryClient.codec`** — query results are decoded with the configured codec
  instead of hardcoded `json.Unmarshal`.

#### Encryption Encoding Fix — `encryption/v4`

- **Encoding preservation through middleware** — `AttachEncryption` and
  `decryptEvent` now preserve the original event's `Encoding()` stamp. Previously,
  CBOR events lost their encoding during the encrypt → decrypt cycle, causing
  `DecodePayload` to fail. JSON events were unaffected (the default).
- **`NewCodec` doc comment** — warns that `encryption.NewCodec` is for non-event
  serialization. For event payloads, use `EncryptMiddleware`/`DecryptMiddleware`,
  which preserves the encoding stamp.

#### Encryption Validation Tests — `schema/v4`

- **`TestValidator_EncryptedEncoding_RejectedGracefully`** — encrypted events
  (encoding="encrypted") produce a clean Rejection error, not a panic.
- **`TestValidator_UnknownEncoding_FallsBackToJSON`** — unknown encodings fall
  back to the JSON decoder.
- **`TestValidator_EncryptedEncoding_WithCustomDecoder`** — consumers can register
  a custom decoder for the "encrypted" encoding.

#### Mixed-Stream Decode — `codec/v4`, `event/v4`

- **`codec.ForEncoding(enc Encoding) (Codec, error)`** — resolves the built-in codec
  for a given encoding stamp. Returns `JSONCodec` for JSON, `CBORCodec` for CBOR,
  and an error for unknown encodings. The codec-level counterpart to `AutoDetect`.
- **`event.DecodePayloadAuto[T](evt) (T, error)`** — decodes an event's payload by
  dispatching to the codec matching the event's `Encoding()` stamp via `ForEncoding`.
  This fulfills the mixed-stream promise: JSON and CBOR events in the same store
  decode correctly without the caller knowing or passing the codec. Previously,
  `DecodePayload` rejected events whose encoding didn't match the caller-provided
  codec — making JSON→CBOR migration impossible without manual branching.

#### gRPC Query Tests — `transport/grpc/v4`

- **Query round-trip test coverage** — the query gRPC service had ZERO test coverage.
  Added tests for JSON round-trip, CBOR round-trip (with `WithCodec`), handler error
  propagation, and codec mismatch detection.

#### Encryption Integration Test — `integration/v4/encryption`

- **CBOR event through encrypt→decrypt** — integration test verifying CBOR events
  survive the encrypt→bus→decrypt cycle with encoding stamp preserved, and
  `DecodePayloadAuto` dispatches correctly post-decryption.

#### Documentation

- **`docs/migration/JSON_TO_CBOR.md`** — comprehensive migration guide with
  step-by-step instructions, decision matrix, and encryption guidance.
- **`docs/adr/0044-blind-store-encoding-stamps.md`** — design doc for v4 envelope
  wrapper to add encoding stamps to blind stores.
- **AGENTS.md codec default asymmetry table** — documents which layer defaults to
  which codec and how to override each.
- **`example/deployer-first`** — refactored to use `event.New()` with typed payloads
  (instead of pre-marshaled JSON bytes) and `stack.WithEventCodec(CBORCodec{})`.

#### CBOR as Recommended Default — `codec/v4`

- **CBOR listed first** in README, doc.go, and examples with "Recommended" badge.
  JSON remains fully supported as the interop/debugging codec.
- **`CBORCompactCodec`** — stricter CBOR (RFC 8949 Core Deterministic) with
  unknown-field rejection on decode, enabling schema drift detection.
- **`BufferEncoder` interface** — zero-allocation encoding via `EncodeToBuffer(v, buf)`.
  Implemented by `JSONCodec`, `CBORCodec`, and `CBORCompactCodec`.
- **Streaming CBOR** — `NewCBOREncoder`/`NewCBORDecoder` for batch encoding without
  materializing the full byte slice.
- **`Diagnose(data)`** — converts CBOR bytes to human-readable diagnostic notation
  for debugging.
- **Exported `CBOREncMode()`/`CBORDecMode()`** — shared canonical encoding modes so
  storage backends use one deterministic CBOR configuration.
- **6 new runnable examples** — CBORCompactCodec, toarray, BufferEncoder, streaming,
  Diagnose, CBOREncMode.
- **Realistic benchmarks** — `realisticOrder` struct with nested items. Results:
  CBOR 19% smaller than JSON, CBOR+toarray 43% smaller. Decode: CBOR 66% faster,
  CBOR+toarray 72% faster.
- **Property-based roundtrip tests** (`pgregory.net/rapid`) — 4 tests proving
  JSON, CBOR, CBORCompact all roundtrip correctly, plus CBOR determinism property.

#### Stack-Level Default Codec — `stack/v4`

- **`WithDefaultCodec(c codec.Codec) Option`** — set a bundle-level default codec.
  Defaults to `CBORCodec{}` (changed from JSON).
- **`Bundle.DefaultCodec()`** — returns the configured default codec.
- **`ReadModel()` and `NewMaterialize()`** — use `DefaultCodec()` instead of
  hardcoded `JSONCodec{}` when the caller passes nil codec.

#### Encoding-Aware Validator — `schema/v4`

- **`WithCodec(c codec.Codec) ValidatorOption`** — replaces the old
  `func([]byte, any) error` parameter with a type-safe `codec.Codec` interface.
  The codec's `Encoding()` determines which encoding the decoder handles.
- **`WithDecodeFunc(fn) ValidatorOption`** — backward-compatible deprecated alias
  for the old `WithCodec` raw-function API. Will be removed in v4.
- **`WithDecoder(enc, fn) ValidatorOption`** — register a decode function for a
  specific encoding.
- **Auto-detected CBOR** — the validator now auto-detects event payload encoding
  via `evt.Encoding()` and picks the matching decoder. JSON and CBOR work
  out of the box with no configuration.

### Changed

#### Symmetric Encoding Validation — `event/v4`

- **`validateEncodingMatch` is now symmetric.** Previously, JSON events got a free
  pass — a JSON event decoded with CBORCodec would bypass validation and fail with
  a confusing corruption error. Now ALL encodings are compared equally:
  `evtEnc != codecEnc`. Mismatches in either direction produce a clear
  `event.encoding_mismatch` Rejection error immediately.

### Documentation

- **`codec/README.md`** — full rewrite. CBOR listed first with "Recommended" badge,
  "When to Use" decision table, struct tag guide (toarray/keyasint/omitzero),
  BufferEncoder, streaming, shared CBOR modes, diagnostic notation.
- **`codec/doc.go`** — updated from "Three implementations" to "Four implementations".
  Added "Choosing a Codec" section.
- **`AGENTS.md`** — added toarray, BufferEncoder, streaming, and `WithDefaultCodec`
  code patterns.
- **`SKILL.md`** — cheat sheet changed from `JSONCodec{}` to `CBORCodec{}` with
  "recommended" note.
- **`kv/typed_options.go`** — `WithTypedCodec` doc mentions `stack.Bundle.DefaultCodec`.

### Migration Notes

- **`schema.WithCodec` signature changed** from `func([]byte, any) error` to
  `codec.Codec`. The old function signature is preserved as `schema.WithDecodeFunc`
  (deprecated). Migrate by replacing `WithCodec(json.Unmarshal)` with
  `WithCodec(codec.JSONCodec{})`.

## [3.4.0] - 2026-06-29

**Managed projection host maturity, durable scheduling, scenario-testing DSL, go mod tidy sweep.**

### Added

#### Managed Projection Host — `projectionhost/v4`

- **`Host`** — managed lifecycle for projection workers: per-projection
  goroutines, crash auto-restart with exponential backoff, checkpoint
  persistence, and a poison-message dead-letter queue. The "last loop every
  consumer rewrites", now a library module (framework gap A1).
- **`ReplayDeadLetters`** — re-feeds dead-letter entries to the matching
  projection after a handler fix; purges successful replays. `DeadLetterEntry`
  now carries the original `event.Event` so replay is possible.
- **`WithLogger(*slog.Logger)`** — inject a structured logger for worker
  lifecycle events (crashes, restarts, DLQ captures). Default: `slog.Default()`.
- **`MemoryDeadLetterStore`** — in-memory `DeadLetterStore` for dev/test.

#### Scenario-Testing DSL — `scenario/v4`

- Fluent BDD harness: `Given[Cmd,State](t, apply, initial, events...).When(cmd,
decide).Then(types...)`, plus `ThenError`, `ThenState`, and projection
  `GivenProjection/ThenNoError` (framework gap A5).

#### Scheduling — `scheduling/v4`

- Durable deadline timers: `TimerStore` (`Schedule`/`Due`/`MarkFired`/`Cancel`),
  `MemoryTimerStore`, and `Scheduler` with configurable poll interval and retry.
  Idempotent scheduling (framework gap A6) — "cancel order after 30 min unpaid".

#### Pebble `kv.ConditionalWriter`

- **`KVAdapter.SetIfAbsent`** — atomic compare-and-set on the Pebble KV adapter,
  unlocking `idempotency.KVStore` support on the Pebble backend. Serialized via
  a per-adapter mutex (process-local guarantee, matching `kv.MemStore`).

#### Brutal Self-Review Pass (2026-06-29)

- **`projectionhost.MetricsRecorder`** — zero-dependency metrics interface
  with `WithMetrics()` option. Five lifecycle methods: EventProcessed,
  EventErrored, EventDeadLettered, WorkerRestarted, CheckpointAdvanced.
  Consumers wire Prometheus/OTel/Datadog; host stays backend-agnostic.
- **`projectionhost.DeadLetterStore.Delete`** — entry-scoped removal
  (`Delete(ctx, name, eventID)`); callers can now surgically clear
  successfully-replayed entries instead of purging the whole projection.
- **`projectionhost` jitter backoff** — worker restart backoff now uses full
  jitter (stdlib `math/rand/v2`) to prevent thundering-herd restarts. No new
  dependency.
- **`scheduling` retry backoff** — dispatch retries now use exponential
  backoff with full jitter between attempts, with a new `WithRetryDelay`
  option. Previously retried with zero delay.
- **`testutil.CapturingSlogHandler`** — shared slog test handler, replacing
  two near-identical copies (`capturingSlogHandler` in projectionhost and
  `capturingHandler` in scheduling).
- **`example/deriver`** — runnable demo of the stateless-saga derivation
  pattern (the deriver module previously had zero consumers/examples).
- **ADR-0042** (pure replay design) and **ADR-0043** (DLQ unification options).

### Changed

- **`testing/v4` renamed to `scenario/v4`** — avoids collision with Go's stdlib
  `testing` package in import paths. The package name is now `scenario`
  (`scenario.Given[...]`). Consumers importing `testing/v4` must update to
  `scenario/v4`.
- **`scheduling.WithLogger`** — previously a no-op (discarded the logger); now
  correctly wires the injected `*slog.Logger`.
- **`scenario.DecideFunc` doc** — corrected the false "import cycle" claim;
  the real reason for decoupling is dependency footprint, not a cycle.
- **`projectionhost/example` lint** — cleared 21 shipped golangci-lint warnings
  (sentinel error, named const, unused-param fix).

### Migration Notes

- **`scheduling.Timer` is now generic (`Timer[P any]`)** — `Timer`, `TimerStore`,
  `MemoryTimerStore`, `DispatchFunc`, and `Scheduler` all require a payload type
  parameter. Migrate by adding it at the call site:
  `scheduling.NewMemoryTimerStore()` → `scheduling.NewMemoryTimerStore[YourCmd]()`,
  `scheduling.Timer{...}` → `scheduling.Timer[YourCmd]{...}`.
- **`command.Command.ID()` (v3.1.0 → v3.3.0)** — the `command.Command`
  interface gained a mandatory `ID() id.CommandID` method for idempotency
  support. Consumers upgrading from v3.1.0 must add `ID()` to every command
  type implementing `command.Command`.

## [3.3.0] - 2026-06-28

**Three projection tiers, unified command identity, production dead-letter storage.**

### Added

#### SQL-Backed Dead-Letter Store

- **`middleware.SQLDeadLetterStore`** — persistent dead-letter handler backed by
  SQLite or PostgreSQL. Auto-creates the `dead_letters` table, survives process
  restarts. Implements `DeadLetterHandler` — drop-in replacement for
  `MemoryDeadLetterStore` in `RetryConfig.OnDeadLetter`.

#### Row Column-Name Validation

- **`storage.ProjectionSink`** methods (Upsert/Ensure/Update/DeleteWhere/QueryOne)
  now validate column and table names against `RelationalSchema` before SQL
  execution. Catches typos at the application boundary. New sentinel errors:
  `errSinkUnknownColumn`, `errSinkUnknownTable`.

#### Denormalization Guidance

- **`storage.RelationalStore`** documented decision: single-table queries only.
  For multi-table reads, denormalize FK columns in the projection handler.
  No JOIN API — intentional boundary (the projection tier's promise is "no raw SQL").

### Changed

#### Breaking: Command ID Unification

- **`command.Command` interface** now requires `ID() id.CommandID`. Every command
  gets a stable, auto-minted ID at construction time via `command.New()`.
  Override with the new `command.WithCommandID` option for idempotency-key replay.
- **`command.WithCommandID` (PersistOption)** renamed to
  `command.WithPersistedCommandID` to avoid name collision.
- **Migration:** any type implementing `command.Command` must add `ID()`.
  Embed `command.BasicCommand` to inherit it automatically.

#### Watermill Command Bridge

- **`watermill.CommandToMessage`** now uses `cmd.ID()` instead of minting an
  ephemeral ID per call. Same command instance → same message UUID (stable for
  dedup). Different instances → different UUIDs (auto-minted in `New()`).
- **`watermill.MessageToCommand`** now parses and preserves the command ID
  round-trip (previously discarded).

#### Transport/gRPC

- **`transport/grpc`** now carries `command_id` in envelope metadata. Server
  preserves the client's command ID through dispatch.

#### Zero Lint Findings

- All 46 modules now lint clean. Previous 8 issues resolved:
  stack (contextcheck, errname, wrapcheck, unused), middleware (exhaustruct),
  transport/grpc (gosec G115, containedctx, nolintlint).

### Documentation

- **All research docs stamped** with status markers (RESOLVED/IMPLEMENTED/SUPERSEDED).
  Every doc in `docs/research/` now clearly indicates whether it's live or historical.
- **ROADMAP.md updated** — module count (43→46), transport adapters (NATS/Redis
  superseded by Watermill), three projection tiers marked done.
- **Graph tier scope documented** — MemoryDriver is the v3.x ship target.
- **`go.work` genproto replace** — explanatory comment added.

### Added

#### catalog/v4.2.0

- **`catalog/simple` sub-package** — single-service Builder facade (`New`,
  `Command[T]`, `Query[T]`, `Event[T]`, `Build`, `BuildValid`) with auto-kebab
  service ID via `internal/caseutil.ToKebab`. Streamlines the common case of
  documenting one service.
- **`catalog/docserver` standalone handlers** — `D2Handler` (D2 architecture
  diagram over HTTP), `HealthCheckHandler` (liveness probe verifying the
  catalog has services), `GenerateEventCatalog` (writes EventCatalog MDX files
  at startup). These complement the existing `DocsServer` for lighter use cases.

#### New Module: `projection/`

- **`projection.Projection`** interface and `projection.NewProjection` — extracted
  from `event/` to a dedicated module. The Projection interface is a consumer-side
  abstraction; it belongs with consumers, not with the event producer module.
  Implements proper dependency-direction: `projection → event` (consumer → producer),
  never the reverse.

#### New Module: `graph/`

- **`graph.GraphProjection`** — third projection tier (nodes + edges) for
  traversal-heavy read models. Merges events into graph structures via a
  transactional `GraphSink`. Writes are portable across openCypher backends
  (Neo4j, Memgraph, Apache Age). `MemoryDriver` provides a zero-dep reference
  implementation.

#### New Module: `storage.RelationalProjection`

- **`storage.RelationalProjection`** — multi-table, dialect-portable SQL projection
  with a transactional `ProjectionSink`. Atomic cross-table writes per event.
- **`storage.RelationalStore`** — read-side companion (Count/CountMany/Query).

#### Architecture Enforcement via go-arch-lint

- **`scripts/check-arch.sh`** — two-layer architecture enforcement:
  Layer 1 = cross-module rules via `check-module-layers.sh` (go.mod parsing);
  Layer 2 = intra-module package rules via go-arch-lint (per-module configs).
  Wired into flake.nix as `nix run .#check-arch`.
- **`.go-arch-lint.yml`** (workspace-level) — documents the 7-layer module model.
  Rewritten from stale config that referenced 6 deleted directories.
- **`storage/.go-arch-lint.yml`** — first per-module config, enforces intra-module
  package dependency rules.
- **Per-module configs for `event/`, `command/`, `middleware/`, `kv/`, `catalog/`** —
  extends Layer-2 architecture enforcement to the largest unchecked modules.

### Changed

#### Breaking: `event.Projection` moved to `projection/`

- `event.Projection` → `projection.Projection`
- `event.NewProjection` → `projection.NewProjection`
- **Migration:** change imports from `event/v4` to `projection/v4` for Projection
  types. All other event types (`Event`, `Type`, `Store`, etc.) remain in `event/`.
- **Rationale:** Projections are event CONSUMERS. The Projection interface had zero
  internal consumers in `event/` — it was a layering inversion. Moving it establishes
  correct dependency direction.

#### Relational Store Query Contract

- **`RelationalStore.Query` now accepts `kv.ViewQuery`** — removes the duplicate
  `storage.RelationalQuery` type. The relational read side now shares the same
  filtered/ordered/paginated query contract as `kv.ViewStore` implementations.

### DX Improvements

#### Bundle.RunProjections — One-Call Projection Runner

- **`bundle.RunProjections(ctx, projections...)`** — replays journal + subscribes to
  live + dispatches to all registered projections. Eliminates ~20 lines of
  CatchUpSubscriber + channel consumption + message decoding boilerplate.
- **`stack.Materialize` now implements `projection.Projection`** — added
  `Name()`, `Handle()`, `EventTypes()` methods. Fixes the split brain where
  Materialize returned Watermill's `NoPublishHandlerFunc` but bypassed the
  library's own `Projection` contract. All three projection tiers now satisfy
  the same interface.

### Tests & Infrastructure

#### Graph Contract Test Suite

- **`graph/graphtest/contract.go`** — shared behavioral contract test for
  `GraphDriver` implementations (mirrors `kv/viewstoretest/contract.go`).
  7 tests: MergeNodeCreates, MergeNodeUpdates, MergeEdgeCreatesEndpoints,
  MergeEdgeUpdatesProps, RemoveNodeDeletesIncidentEdges, RemoveEdgeLeavesEndpoints,
  AtomicRollbackOnError. MemoryDriver passes all 7.

#### Architecture Enforcement

- **`scripts/check-arch.sh`** — two-layer arch enforcement (cross-module via
  go.mod parsing + intra-module via go-arch-lint). Wired as `nix run .#check-arch`.
- **`storage/.go-arch-lint.yml`** — first per-module arch-lint config.
- Stack dep budget bumped from 12 to 13 (added `projection/v4` dependency).

#### ADRs

- **ADR-0037**: Projection interface extraction from `event/`
- **ADR-0038**: Graph projection tier design (writes portable, reads native)
- **`docs/projection-tiers.md`**: Decision guide for choosing between tiers

#### Quality

- **`projection/` module: 100% test coverage** (5 tests)
- **`graph/` module: 86.9% coverage** (9 tests + 7 contract tests)

#### Workspace Integration

- **`transport/grpc` is now wired into `go.work`** — resolves the long-standing
  `google.golang.org/genproto` ambiguous-import conflict via a workspace-level
  replace directive. The module builds and tests as a first-class workspace member.
- **BuildFlow pre-commit hook budget increased** from 60s to 300s — eliminates the
  need for `--no-verify` on commits.

#### RunProjections Test Coverage

- **`stack/run_projections_test.go`** — end-to-end test covering journal replay,
  live event handoff, materialized-view updates, and clean shutdown via context
  cancellation.

## [3.1.0] - 2026-06-25

**Feature release — 79 commits since v3.0.0, +69 API exports (1558 → 1627), zero breaking changes.**

### Added

#### SQL-Backed View Stores & Queryable Read Models

- **`storage.SQLViewStore`** — SQL-backed `kv.ViewStore` with column-mapped views. Supports `Query` (WHERE + ORDER BY + LIMIT/OFFSET), `Count`, `BatchSet` (chunked upsert, SQLite 999-param aware), `DeleteAll`, and `Scan`. Tombstone column support for server-side filtering.
- **`storage.ViewMapper[V]`** — declarative column mapping: table name, columns with extractors, `ScanRow`, optional `TombstoneColumn` and `Indexes`.
- **`storage.AutoMapper` / `AutoMapperWithTombstone`** — generates a `ViewMapper` from struct tags (field name → column name).
- **`storage.NewSQLiteViewStore` / `NewSQLViewStore` / `NewViewStoreWithDialect`** — constructors with auto-migration.
- **`kv.ViewStore` interface** — `ViewQuerier`, `ViewCounter`, `ViewBatchSetter`, `ViewResetter`, `TombstoneQuerier` optional interfaces checked at runtime.
- **`kv.ViewQuery` / `Condition` / `Operator`** — typed query DSL (`OpEq`, `OpNeq`, `OpGt`, `OpGte`, `OpLt`, `OpLte`, `OpIn`, `OpLike`).
- **Preset integration** — `sqlite.SQLViewModel[V,K]` and `postgres.SQLViewModel[V,K]` one-call constructors.
- **`storage.WithoutViewAutoMigrate`** / **`storage.SQLiteApplyOptimizations`** — production options.
- **`sqlite.WithForeignKeys()` / `sqlite.WithOptimizations()`** — referential integrity + cache/temp/mmap PRAGMAs.

#### Multi-Database Split

- **Postgres multi-DB split** — `WithEventDB`/`WithQueryDB`/`WithViewDB` options for the Postgres preset, mirroring SQLite and Turso. Routes events+snapshots+checkpoints, commands+queries, and read models to separate databases on the same Postgres server. (ADR-0033)
- **`stack/sqlopt` package** — shared option-assembly logic for SQL-backed presets. Keeps the base `stack` package free of a storage dependency.
- **`stack.WithDatabase` / `Bundle.Database()`** — expose the underlying DB handle for preset-specific constructors.
- **Multi-DB contract test** — `contracttest.RunMultiDBSuite` verifies routing correctness.
- **Multi-DB example** (`example/deployer-first-multidb/`) — runnable end-to-end demo.
- **ADR-0033** — Multi-database split design rationale.
- **ADR-0034** — Session store boundary.

#### Shared Metadata & Lifecycle Helpers

- **`event.CustomData[K]`** — shared generic base for `command.Metadata` and `query.Metadata` (ADR-0031). Carries tracing + custom map with shared `Clone`/`Merge`/`EnsureCustom`.
- **`event.MergeCustomMaps`** — generic zero-allocation merge for custom metadata maps.
- **`stack.MultiCloser` / `stack.FuncCloser`** — shared lifecycle helpers.
- **`Bundle.Debug()`** — prints which capability fields are set for wiring diagnostics.

#### CI & Tooling

- **API stability CI check** — `cmd/api-stability` golden file (1627 exports) verified on every push/PR.
- **Convenience flake apps** — `nix run .#test-grpc`, `.#check-wasm`, `.#check-api-stability`, `.#ci` (aggregate).
- **`nix run .#check-file-size`** — local mirror of the CI file-size gate.
- **Property-based tombstone tests** — 6 `rapid`-based tests (100 iterations each) covering empty stream, last-event-wins, no-mutation, transitions, unmarked, nil.
- **Zero lint findings** — golangci-lint config tuned to 0 findings across all 33 modules (down from 200).
- **12 design documents** (`docs/design/`) — NATS, Redis, secondary indexes, hot-state cache, read-pressure snapshots, compaction, archival, dashboard, distributed runner, blocked items, makezero eval, remaining ideas.

#### Storage & Production Tuning

- **`synchronous=NORMAL` in `SQLiteEnableWAL`** — 3-10x better write throughput without durability loss.
- **Turso WAL default** — Turso preset now enables WAL by default; disable with `WithoutWAL()`.
- **Turso sync contract test** — `TestNewSync_Contract` (skips without `TURSO_SYNC_URL`).
- **Schema migration caveat** documented in `storage/doc.go`.
- **Migration guide** (`docs/MIGRATION_TO_STACK.md`) — replacing hand-wired infrastructure with presets.

### Fixed

- **11 phantom doc references** — corrected stale type names across stack/doc.go, stack/errors.go, bundle.go, options.go, snapshot/doc.go.
- **FEATURES.md stale v2 import paths** — stack modules updated to v3.
- **ROADMAP.md module count** — corrected 38 → 43.
- **ADR-0026 stale WASM claims** — decider/ now compiles to WASM (fixed via `//go:build !js`); removed reference to deleted `wasm/main.go`.
- **9 dead `noinlineerr` references** — removed from `.golangci.yml` exclusion lists.
- **11 stale `//nolint:errcheck` directives** — removed from test files (errcheck excluded for `_test.go`).
- **`stack/go.mod` invalid `eventtest v3.0.0`** — fixed to `v0.0.0` (no major-version suffix).
- **storage/pebble test unchecked errors** — added error checks on constructor calls.

### Changed

- **go-error-family upgraded v0.4.0 → v0.5.1** — across all 12 direct-dep modules. `event.Compose` removed (use stdlib `errors.Join`). Upstream adds `Family.HTTPStatus()`, `Family.RetryPolicy()`, `Error.JSON()`, copy-on-write errors, severity-ordered multi-error classification, lock-free sentinel lookup, injectable `Registry`.
- **API surface** — 1558 → 1627 exports. Golden file regenerated.
- **Coverage documented** — real per-module numbers in AGENTS.md (decider 98.3%, event 91.4%, command 89.4%, workspace total 78.7%). — `WithEventDB`/`WithQueryDB`/`WithViewDB` options for the Postgres preset, mirroring SQLite and Turso. Routes events+snapshots+checkpoints, commands+queries, and read models to separate databases on the same Postgres server. (ADR-0033)
- **Multi-DB contract test** — `contracttest.RunMultiDBSuite` verifies routing correctness for any preset supporting multi-DB. Wired into sqlite and turso test suites; postgres test requires `POSTGRES_TEST_DSN` + `CREATE DATABASE` permission.
- **Migration guide** (`docs/MIGRATION_TO_STACK.md`) — Step-by-step guide showing how to replace 200–400 lines of hand-wired infrastructure with 5–10 lines of stack preset. Covers event store, projection runner (CatchUpSubscriber+Materialize), build-tag switching, and multi-DB split.
- **Turso sync contract test** — `TestNewSync_Contract` runs the full contract suite against a NewSync bundle (skips without `TURSO_SYNC_URL`).
- **ADR-0033** — Multi-database split design rationale.
- **ADR-0034** — Session store boundary (sessions are application-layer, not CQRS infrastructure).
- **Schema migration caveat** documented in `storage/doc.go` — raw constructors do NOT auto-migrate; use a stack preset or call `SQLiteInitSchema`/`PostgresInitSchema` manually.
- **`synchronous=NORMAL` in `SQLiteEnableWAL`** — WAL mode now sets `synchronous=NORMAL` instead of the default FULL, giving 3-10x better write throughput without durability loss (safe with WAL). Affects both SQLite and Turso presets.
- **SQLite `WithOptimizations()`** — applies `cache_size`, `temp_store=MEMORY`, and `mmap_size` PRAGMAs for production throughput. Parity with the existing Turso option.
- **Turso `WithoutWAL()`** — WAL mode is now the default for the Turso preset (was previously off). Disable with `WithoutWAL()`.

## [3.0.0] - 2026-06-22

**Major release — tagged.** All 38 modules migrated to `/v4` import paths. The 11 breaking changes are additive in nature (the new shapes existed in v2). See the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)** for step-by-step instructions.

### Breaking Changes

| #   | Change                                                                                                      | ADR                                                       |
| --- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| 1   | Delete ghost bus code (`event/reactive*.go`, `samber/ro` dep)                                               | [0028](docs/adr/0028-watermill-as-delivery-layer.md)      |
| 2   | Move `memory/` → `storage/memory/`                                                                          | [0029](docs/adr/0029-storage-consolidation.md)            |
| 3   | `event.Version`: `int` → `uint64`                                                                           | —                                                         |
| 4   | Break `command/query.Metadata = event.Metadata` alias (ADR-0031)                                            | [0031](docs/adr/0031-metadata-split.md)                   |
| 5   | Remove `io.Closer` from 9 core interfaces                                                                   | [0010](docs/adr/0010-remove-io-closer-from-interfaces.md) |
| 6   | Delete `readmodel/` (merged into `kv/` as `kv.TypedStore` + `kv.Cache`)                                     | [0032](docs/adr/0032-merge-readmodel-into-kv.md)          |
| 7   | Delete `projection/` (replaced by `bus.SubscribeAll` + `stack.Materialize` + `watermill.CatchUpSubscriber`) | [0030](docs/adr/0030-dissolve-projection.md)              |
| 8   | Move SSE → `transport/http/`; delete healthcheck/metrics_http/pprof                                         | [0025](docs/adr/0025-transport-adapter-strategy.md)       |
| 9   | `query.Handler`: `any` → generic `TypedHandler[Q, R]`                                                       | [0008](docs/adr/0008-typed-handler-signature.md)          |
| 10  | Rename `Decider.Fold` → `Apply`                                                                             | —                                                         |
| 11  | Make `event.Event` a concrete type (`= *ImmutableEvent`)                                                    | —                                                         |

### Added

- **Pebble backup and observability accessors** (`stack/pebble/`) — `pebble.Bundle` wraps `*stack.Bundle` with `Checkpoint(dir)` for point-in-time backups, `Metrics()` for LSM-tree health, `Flush()` for write durability, and `NewSnapshot()` for consistent reads.
- **Bundle.GracefulClose** (`stack/`) — Context-bounded `Close()` for production shutdown. Runs `Close()` in a goroutine; returns `ctx.Err()` if the deadline fires. Lets in-flight handlers drain without hanging forever.
- **SSE Last-Event-ID reconnection** (`transport/http/`) — `WithReconnectJournal(journal, limit)` option on `NewSSEBroker` enables standard SSE reconnection. When a client sends `Last-Event-ID`, the broker replays missed events from the journal before starting live delivery. Uses the same dedup strategy as `watermill.CatchUpSubscriber` (replayIDs set) to prevent duplicate delivery.
- **Streaming event reads** — `StreamingSource`/`StreamingJournal` now implemented on all three stores: `SQLEventStore` (cursor-based via `*sql.Rows`), Pebble `EventStore` (iterator-based with limit + skip), `MemoryStore` (SliceIterator-wrapped). Consumers can type-assert to streaming interfaces uniformly across backends.
- **DistributedRunner** — _Deleted with `projection/` (ADR-0030). The `watermill.CatchUpSubscriber` + `stack.Materialize` pattern replaces it with simpler semantics._
- **cqrs-gen event handler generation** — _Removed: `-type=event` generated `projection.On[T]()` calls, but `projection/` was deleted (ADR-0030). cqrs-gen now supports `command` and `query` only._
- **Postgres LISTEN/NOTIFY event bus** (`storage/`) — `PostgresBus` implements `event.Bus` using `SELECT pg_notify()` with lightweight JSON reference payloads (under 8KB). `NotificationListener` interface abstracts driver-specific LISTEN; the bus calls `Listen(channel)` itself so consumers don't need to pre-arm. Listener re-fetches full events from store with retry for visibility-gap handling. Uses `LoadByEventID` (indexed O(1) lookup) when the store implements `EventByIDLoader`. **Wired into `stack/postgres` preset** via `WithDistributedBus(listener)` option.
- **PgxListener** (`stack/postgres/`) — `PgxListener` implements `storage.NotificationListener` using `pgxpool`. Dedicated single-connection pool for LISTEN; channel-name allow-list defends against SQL injection. `NewPgxListener(pool)` wraps an existing pool; `NewPgxListenerFromDSN(ctx, dsn)` creates an owned single-conn pool.
- **PostgresBus otel spans** — `pg_bus.publish` (SpanKindInternal) and `pg_bus.handle_notification` (SpanKindConsumer) spans for distributed tracing of NOTIFY round-trips.
- **Real-Postgres integration tests** (`stack/postgres/`) — Three `-tags=integration` tests covering the full LISTEN/NOTIFY round-trip, channel validation, and preset wiring. Run in CI's `postgres-integration` job.
- **Documentation site content** — `docs/index.md` landing page with value proposition, quick start, module overview, presets comparison table.
- **PgxListener auto-reconnect** (`stack/postgres/`) — On connection loss, the listener automatically re-acquires a connection and re-issues LISTEN with exponential backoff (default: 10 attempts, 1s→30s). Configurable via `WithReconnect(maxAttempts)`, `WithReconnectBackoff(initial, max)`, `WithoutReconnect()`. A dropped connection no longer silently kills event delivery.
- **PgxListener deadlock regression test** — `TestPgxListener_CloseDoesNotDeadlock` asserts Close() returns within 2s when the receive loop is running, preventing regression of the critical cancelFn fix.
- **Property-based channel-name validation** — `rapid` property tests (3 properties × 100 inputs) covering valid identifiers, digit-first rejection, and no-panic-on-arbitrary-input.

### Changed

- **Module paths** — All 38 modules migrated from `…/v2` to `…/v4` import paths (e.g. `github.com/larsartmann/go-cqrs-lite/event/v4`). Consumers update `go get` targets and import statements. The `example/*` modules remain unversioned.
- **Zero-panic API migration** — All production `panic()` calls converted to error returns. Breaking signature changes:
  - `pebble.NewStore/NewSnapshotStore/NewCheckpointStore/NewKVStore/NewQueryStore/NewCommandStore` now return `(T, error)` — returns `ErrNilDatabase` (classified as `Rejection`) if db is nil.
  - `pebble.NewBackend` now returns `(*Backend, error)`.
  - `multisig.VerifierMap` now returns `(map, error)` — returns `ErrNilSigner` (`Rejection`) if any signer is nil.
  - `Version.Decrement()` and `Version.Sub(n)` now return `(Version, error)` — returns `ErrVersionUnderflow` (`Rejection`) on underflow.
  - `SchemaVersion.Decrement()`, `.Add(n)`, `.Sub(n)` now return `(SchemaVersion, error)` — returns `ErrSchemaVersionUnderflow` (`Rejection`) on underflow.
  - `codec.CBOREncMode()` and `codec.CBORDecMode()` return bare `cbor.EncMode`/`cbor.DecMode` via `sync.OnceValue` (no error — creation cannot fail with hardcoded valid options).
  - `cattest.StringSchema` now returns `(*Schema, error)` instead of panicking on odd-length props.
- **SSE moved to transport/http/** (`transport/http/`) — SSE broker moved from `middleware/` to new `transport/http/` module (ADR-0025). SSE wire format rewritten with proper `SSEEvent` struct, spec-correct multi-line `data:` handling, `SSEEventID` branded type, and 15s heartbeat to prevent proxy timeouts. Healthcheck, metrics_http, and pprof handlers deleted (generic utilities, zero CQRS deps, zero consumers).
- **Ghost streaming interfaces removed** — Consolidated the old `StreamLoader`/`EventStream` types (bool-based `Next()` + `Err()`) into the shipped `EventIterator` interface (standard Go `io.EOF` pattern). Dead code that never compiled against the real interface is gone.
- **WASM compilation** — All 7 core modules (id, codec, dispatcher, event, command, query, decider) now compile to `GOOS=js GOARCH=wasm`. Moved `NewCQRSViews()` behind `//go:build !js` to exclude the OTel SDK's `os/user` dependency.
- **notifyPayload type model** — Replaced 5 stringly-typed fields with branded domain types (`id.EventID`, `event.Type`, `event.AggregateType`, `id.AggregateID`, `event.Version`). Eliminates the manual `String()`→`Parse` roundtrip on the receive side.
- **pgx upgraded v5.7.1 → v5.10.0** — Patches critical memory-safety vulnerability (CVE) and SQL-injection via placeholder confusion.
- **API surface** — 1806 → 1852 exports.

## [2.7.0] - 2026-06-19

The **Bundle composition layer**: consumers stop deciding on infrastructure. A deployer picks a backend via one preset call; the application imports only `readmodel` and `stack` and never touches a storage driver. 8 new modules (~5,500 lines), persistent read models for every preset, a shared contract suite, and a zero-lint release gate.

### Added

- **Bundle composition root** (`stack/v2`) — `Bundle` with ISP-honest fields (EventSink/EventSource/Journal kept separate, not a fat Store), `Option = func(*Bundle)`, pointer-deduplicated `Close()`, and rollback-on-validation. Repository/ReadModel helpers are top-level generic functions (`stack.Repository[State]`, `stack.ReadModel[T,K]`) since Go forbids generic methods.
- **Bundle presets** — `stack/memory`, `stack/sqlite` (modernc, WAL, auto-migrate), `stack/pebble` (single PebbleDB for all stores via disjoint key prefixes), `stack/postgres` (pgx, auto-migrate). Each wires event store+bus, command/query/snapshot/checkpoint stores, and a read-model backend in one call.
- **Typed read-model store** (`readmodel/v2`) — `Store[T any, K fmt.Stringer]` over `kv.Store` with codec + key prefixing; `Backend` is an alias for `kv.Store`, so `kv.MemStore`, `pebble.KVAdapter`, and the new SQL KV store all satisfy it.
- **Read-model cache decorator** (`readmodel/cache/v2`) — Otter-backed `CachedStore[T,K]` (TinyLFU admission) with capacity + TTL, write-through.
- **Typed stores** — `snapshot.TypedSnapshot[State]` + `TypedStore` (closes the `[]byte` hole on snapshot state); `command.TypedCommandStore[P]` (with `AppendBatch`); `query.TypedQueryStore[P]`. Encode/decode happens once at the adapter boundary.
- **Pebble gaps closed** (`pebble/v2`) — `CommandStore`, `QueryStore`, and `ReadModels()` accessor on `Backend`; EventStore.Close() is now a no-op so the Backend owns the DB lifecycle (fixes a double-close).
- **SQL-backed kv.Store** (`storage/v2`) — `SQLKVStore` implements `kv.Store` over a `cqrs_kv` table (Get/Set/Has/Delete/streaming-Iterator/transactional-Batch), exposed via `SQLBackend.KVStore()`. SQLite and Postgres presets now **persist read models across restarts** instead of using `kv.MemStore`. Verified by an E2E reopen test.
- **Shared contract test suite** (`stack/contracttest`) — `RunSuite(t, factory)` runs 5 behavioural checks; 4 presets × 5 = 20 contract assertions.
- **Zero-overhead benchmarks** (`stack/bench/v2`) — proves Bundle field access is a direct struct read (~0.20 ns/op).
- **godoc example** (`stack/memory`) — `ExampleNew` renders the canonical Bundle entry point on pkg.go.dev.

### Changed

- **Dialect interface** (`storage/v2/sql`) — gained `KVSchema()` for the `cqrs_kv` table (BLOB for SQLite, BYTEA for Postgres). The only implementations are the in-package `PostgresDialect`/`SQLiteDialect`; upsert uses `ON CONFLICT(key) DO UPDATE … excluded.value`, identical across dialects.
- **Lint app resilience** (`flake.nix`) — `nix run .#lint` now reports every failing module instead of aborting on the first (it ran under `errexit`).
- **API surface** — 1351 → 1784 exports; golden file regenerated and the checker's module list expanded to 33 consumer-facing modules.
- **Example rewrite** (`example/todo`) — uses the pebble Bundle preset + `readmodel.Store`; dead `storage/` package deleted (7 files).

### Fixed

- **Postgres preset tests ran in CI** — the `postgres-integration` job set `DATABASE_URL` (read by `storage` tests) but not `POSTGRES_TEST_DSN` (read by `stack/postgres` tests), so preset tests were silently skipped despite a running container. Now sets both and runs the preset suite.
- **Zero lint violations** — 39 violations shipped under `--no--verify` in the first pass are now 0 across all 34 modules (readmodel, pebble, snapshot, stack presets cleaned up).
- **Workspace** — `go work sync` applied; dependency budgets reconciled (`DEP_BUDGET[storage]` 11→12 for the new kv dep).

### Infrastructure

- CI matrix, `flake.nix`, `check-module-layers.sh`, and `.golangci.yml` updated for the 8 new modules (otter + pgx added to depguard).

## [2.6.0] - 2026-06-19

27 commits since v2.5.0. Two new modules (schema validator, prometheus exporter), projection replay/live split, replay→live dedup pipeline, OTel correlation enricher, bounded dedup, streaming event reads, exported ID marker types, cqrs-gen struct tags, and leader election interface.

`pebble.DeleteEventsBefore` (added in v2.5.0, the immediately prior release ~24h earlier) is removed: it contradicted event-sourcing immutability and no consumer could depend on it between releases. No other existing API removed or renamed.

### Added

- **Schema registry validator** (`schema/v2`) — `Validator` with `RegisterType[T]()`, `RegisterTypeWithValidator[T]()`, strict/lenient modes, custom codec support. Returns `Rejection` errors on invalid payloads. ADR-0017 accepted
- **Prometheus metrics exporter** (`prometheus/v2`) — New module wrapping OTel Prometheus exporter. `Setup()` creates a `MeterProvider` backed by a Prometheus registry and an HTTP handler for `/metrics`. `WithRegistry()`, `WithHandlerOptions()`, `MustSetup()`
- **Bounded dedup** (`event/v2`, `projection/v2`) — `DistinctByEventIDBounded(cap)` with FIFO ring eviction for bounded memory in 24/7 projections. `DistinctByEventIDBoundedWith(cap, seen)` seeded variant. `WithDedupCapacity(n)` Runner option
- **Streaming event reads** (`event/v2`) — `EventIterator` interface for one-at-a-time event reading without materializing slices. `StreamingSource` and `StreamingJournal` opt-in interfaces. `SliceIterator` adapts pre-loaded slices
- **cqrs-gen struct tag scanning** (`cmd/cqrs-gen`) — Supports `cqrs:"command:CreateUser"` struct tags on `_ struct{}` fields in addition to `//cqrs:command CreateUser` comment markers. Comment markers take precedence
- **LeaderElection interface** (`projection/v2`) — `LeaderElection` interface + `AlwaysLeader` default for distributed projection coordination per ADR-0018. Consumers implement coordination (Redis, etcd, k8s); library provides interface and default
- **Projection replay/live split** (`projection/v2`) — `Runner.RunReplay(ctx)` replays historical events synchronously and returns once the read model is caught up (read-your-writes); `Runner.RunLive(ctx)` then tails live events in the background. `Run` remains as a convenience wrapper calling both. Eliminates `time.Sleep`-based catch-up hacks in consumers. Adds `ErrReplayRequired` when `RunLive` is called before `RunReplay`
- **Replay→live dedup pipeline** (`event/v2`, `projection/v2`) — Closes the duplicate-processing gap at the replay→live boundary. New `event.SubscriberToObservable` adapts callback-based `Subscriber` to `ro.Observable[Event]`; `event.DistinctByEventIDWith(seen)` seeds the dedup set with IDs from journal replay. The Runner's live path now builds `live → DistinctByEventIDWith(replayIDs) → handler`, suppressing overlap-window duplicates
- **OTel correlation enricher** (`middleware/v2`) — `OTelCorrelationEnricher` bridges OTel baggage correlation IDs into event metadata via `event.WithCustom`. Composes with `CommandCausalityEnricher` via `CompositeEnricher`. New `OTelCorrelationIDFromEvent` extractor and `MetadataKeyOTelCorrelationID` constant
- **Exported ID marker types** (`id/v2`) — All 8 phantom marker types are now exported (`AggregateMarker`, `UserMarker`, `CorrelationMarker`, `RequestMarker`, `CausationMarker`, `ClientMarker`, `CommandMarker`, `EventMarker`), enabling downstream `go-branded-id` `BrandNamer` integration and other type-parameterized tooling against the root module's ID types

### Removed

- **Pebble `DeleteEventsBefore`** (`pebble/v2`) — Removed. Events are immutable truth; automatic event deletion contradicts event sourcing principles. Introduced in v2.5.0 (immediately prior release) and removed before any consumer could adopt it. The `Flush()` method remains for durability control

## [2.5.0] - 2026-06-18

70 commits since v2.4.0. Pebble backup/retention/consistent reads, OpenTelemetry baggage correlation + metric views + propagator, load coalescing via singleflight, HKDF multi-tenant key derivation, CBOR streaming, reactive event dedup operators, Watermill middleware wrappers, and turso race fixes. No breaking API changes.

### Added

- **Pebble backup and consistent reads** (`pebble/`) — `PebbleBackend.Checkpoint(dir)` for point-in-time DB snapshots and `NewSnapshot()` for consistent read views via Pebble snapshots
- **OTel baggage correlation IDs** (`otel/`) — `WithCorrelationID(ctx, id)` and `CorrelationIDFromContext(ctx)` propagate correlation IDs across distributed service boundaries via W3C baggage
- **OTel TextMapPropagator** (`otel/`) — `NewTextMapPropagator()` implements W3C trace context + baggage propagation for inject/extract across transports
- **OTel CQRS metric views** (`otel/`) — `NewCQRSViews()` configures customized histogram boundaries (`CQRSHistogramBoundaries`) for CQRS latency ranges; `ServiceResourceAttributes()` for service identification; `CounterAddWithAttributes()` and `AddSpanEvent()` helpers for rate metrics and span events
- **Decider load coalescing via singleflight** (`decider/`) — `Repository[State]` now coalesces concurrent `Load` calls for the same aggregate into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice is safe. Disable via `WithLoadCoalescing[State](false)`
- **HKDF key derivation** (`encryption/`) — `DeriveKey(masterKey, info, length)` derives per-tenant/subscope keys via HKDF-SHA256, enabling multi-tenant encryption without separate master keys
- **SQLite foreign keys helper** (`storage/`) — `SQLiteEnableForeignKeys(ctx, db)` enables `PRAGMA foreign_keys=ON` for opt-in referential integrity
- **Codec BufferEncoder interface** (`codec/`) — `BufferEncoder` extension enables zero-allocation encoding directly into a caller-provided `*bytes.Buffer` via `EncodeToBuffer(payload, buf)`, bypassing intermediate allocations
- **Event stream deduplication operators** (`event/`) — `DistinctByEventID()` suppresses duplicate event IDs; `DistinctByAggregateID()` keeps only the first event per aggregate. Composable via `ro.Pipe1`
- **Watermill middleware wrappers** (`watermill/`) — `CorrelationIDMiddleware()` and `NewRetryMiddleware(config)` for Watermill routers, plus Router integration support
- **CBOR streaming and compact codec docs** (`codec/`) — `CBORCompactCodec` documentation (struct fields as positional array, ~35% smaller payloads); `Diagnose()` for human-readable CBOR debugging
- **Testutil seed control** (`testutil/`) — seed control helper and rapid testing generator patterns for reproducible randomized tests

### Changed

- **Dependency upgrades** — `go-error-family` v0.3.0 → v0.4.0; `go-branded-id` v0.3.0 → v0.3.1 across all consuming modules
- **API surface growth** — 1266 → 1289 exports (29 new public symbols), golden file updated
- **Testutil ghost API removal** (`testutil/`) — removed non-functional `EventSlice` and `SeedFromEnv` exports (dead code that never worked; technically a public surface reduction but no behavioral impact)

### Fixed

- **Turso CheckpointScheduler race** (`turso/indexing/`) — `Stop()` now drains the checkpoint goroutine via a `done` channel before returning, preventing goroutine leaks and races on repeated Start/Stop cycles
- **Turso parallel test flakiness** (`turso/`) — eliminated flaky parallel test failures by isolating state and increasing checkpoint test timing margins
- **Decider singleflight error passthrough** (`decider/`) — singleflight errors now pass through verbatim instead of being wrapped with `fmt.Errorf`, preserving error classification (Rejection/Conflict/etc.) via `errors.Is`
- **OTel NewCQRSViews wildcard** (`otel/`) — corrected view instrument name wildcard matching so all CQRS histograms receive custom boundaries
- **Production dependency budget accuracy** (`scripts/check-module-layers.sh`) — test-only packages (gomega, ginkgo, rapid) now excluded from the production dep count, reflecting true direct dependency budgets

### Infrastructure

- **Watermill Router integration test** — end-to-end test for CorrelationID + Retry middleware through a real Watermill Router

## [2.4.0] - 2026-06-17

15 performance optimizations across 7 modules. No public API changes, no disk format changes, no breaking behavior. Verified with 5-run benchmark averages (allocation deltas are deterministic and reliable; ns/op has ±15% variance), tests + race detector + lint.

### Performance

- **Pebble double serialization eliminated** (`pebble/`) — events serialized once, `batch.Set` called for both event and journal keys. Halves CPU and disk bytes per write
- **Event lazy metadata map initialization** (`event/`) — `NewMetadata()` returns zero-value struct instead of always allocating a map. Eliminates 1 heap allocation per event when no custom metadata is set
- **Projection handler Lookup zero-allocation** (`projection/`) — `lookupSlices()` returns pre-built handler slices directly instead of allocating a combined slice per event. Only benefits `projection.Builder`-created projections
- **Projection Runner event type caching** (`projection/`) — Runner caches `p.EventTypes()` once at `Register()` time, eliminating 10.5M per-event clone allocations (100K events × 100 projections) in the scale benchmark. This is the real fix for the projection allocation hotspot — the original T3/T4 `*builtProjection` type assertion was dead code for `event.NewProjection()` users. Also pre-allocates the candidates slice in `dispatchToProjections`
- **SQL template strings cached per dialect** (`storage/`) — INSERT SQL built once at `SQLEventStore` construction, eliminating `fmt.Sprintf` per call
- **MemoryStore Load double-copy eliminated** (`memory/`) — removed redundant `slices.Clone` wrapper on already-fresh slice from `getEvents()`
- **SSE vestigial goroutine removed** (`middleware/`) — removed useless `go func() { <-ctx.Done() }()` goroutine leak. Consolidated 3× `fmt.Fprintf` into single write
- **Event Merge EnsureCustom hoisted** (`event/`) — `EnsureCustom` called once before the Merge loop instead of per-iteration nil-check
- **Event FilterByTimestamp pre-sized** (`event/`) — result slice initialized with `make([]Event, 0, len(events))` to eliminate nil-slice append growth pattern
- **SQL ScanSlice pre-allocated** (`storage/`) — initial capacity hint of 64 reduces log₂(N) slice growth copies during large Loads
- **CircuitBreaker atomic state machine** (`middleware/`) — replaced `sync.Mutex` + `int` fields with `atomic.Int32`. Happy path (circuit closed) is now lock-free: single `state.Load()` check
- **MemoryBus middleware pre-computation** (`memory/`) — middleware chains pre-computed at `Use()`/`UsePublish()` registration time. `Publish()` reads cached chain under RLock — zero per-publish closure allocation
- **Pebble ReadFrom key-based skip** (`pebble/`) — during cursor skip phase, parse event ID from journal key via `journalKeyEventID()` instead of CBOR-deserializing every skipped event
- **SQL multi-VALUES INSERT batching** (`storage/`) — single `INSERT INTO events ... VALUES (..), (..), (..)` statement replaces N individual INSERTs. SQLite 999-parameter limit handled via automatic chunking (99 events/batch)

### Added

- **Reactive CommandBus and QueryBus** (`command/`, `query/`) — `NewCommandBus`, `NewQueryBus`, `FilterCommandType`, `FilterQueryType`, `HandlerToObserver`, plus replay/behavior variants. Mirrors the existing reactive event API for command and query streams
- **PebbleBackend facade** (`pebble/`) — `Open()` and `NewBackend()` provide a single shared-DB entry point for Pebble-backed EventStore, SnapshotStore, and CheckpointStore, with clear ownership semantics
- **SQLBackend lifecycle facade** (`storage/`) — `SnapshotStore()`, `CheckpointStore()`, and `Close()` methods complete the SQL backend full-stack facade
- **KV module** (`kv/`) — Layer-0 in-memory key-value store abstraction (`MemStore`) with snapshot iteration and atomic batch commit
- **`command.Compose` and `query.Compose`** — re-export `go-error-family.Compose` for classified multi-error composition in command and query modules
- **Integration tests** (`integration/`) — end-to-end tests for pebble-backed projection Runner (replay + live) and decider Repository with Pebble SnapshotStore
- **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`, making pebble the first real consumer of the kv/ abstraction. Supports owned and borrowed DB lifecycle, prefix-bounded iteration, atomic batch commit, and `ErrNotFound`/`ErrClosed` error mapping
- **Built-in pprof endpoints** (`middleware/`) — `ProfilingHandler()` and `RegisterProfiling()` expose Go runtime profiling (heap, goroutine, CPU, allocs, block, mutex) via standard `/debug/pprof/` paths
- **Pebble benchmarks** (`pebble/`) — 4 benchmarks (Save100, SaveLoad100, Save1, LoadEmpty) for performance regression tracking
- **KV contract tests** (`pebble/`) — 10-test contract suite run against both PebbleAdapter and MemStore, proving semantic equivalence
- **Compose tests** (`command/`, `query/`) — 5 tests each for `Compose` error composition (nil, single, multiple, classified, mixed)
- **PostgreSQL CI** (`.github/workflows/ci.yml`) — `postgres-integration` job with PostgreSQL 16 service container wired to storage integration tests

### Fixed

- **Turso error classification** (`storage/sql/query_engine.go`) — `QueryRows` no longer re-wraps classified errors as Infrastructure, preserving Rejection semantics for `LoadNonExistent`
- **Module layer budgets** (`scripts/check-module-layers.sh`) — budgets updated to reflect actual direct dependencies: codec 2, pebble 8, storage 11, turso 10, integration 19
- **Turso lint hygiene** (`turso/indexing/advisor_data.go`) — cleared 3 pre-existing `gochecknoglobals` findings on static advisor data tables

### Infrastructure

- **CI replace-directives check** — `scripts/check-replace-directives.sh` now runs in GitHub Actions to verify every module `replace` directive matches `go.work`
- **`cmd/api-stability` in CI matrix** — per-module-test job now tests the API stability checker in isolation

## [2.3.0] - 2026-06-12

231 commits since v2.2.0. Lint hygiene, coverage improvements, CBOR codec, encryption module, phantom types, and release readiness.

### Added

- **CBOR codec** (`codec/`) — `CBORCodec` with deterministic canonical encoding, sorted map keys, `DecMode` option
- **Pebble CBOR envelope** (`pebble/serialization.go`) — events serialized as CBOR with JSON backward compatibility layer
- **Encryption module** (`encryption/`) — XChaCha20-Poly1305, AES-256-GCM, `Algorithm` enum, `KeyID` phantom type, `KeyResolver` interface, composable `NewCodec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`
- **Command store interfaces** (`command/`) — `CommandSink`, `CommandSource`, `Store` (Sink+Source) for persisted command logs
- **SQL CommandStore** (`storage/`) — `SQLCommandStore` with Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp
- **SQL Backend facade** (`storage/`) — `SQLBackend` returning EventStore, SnapshotStore, CheckpointStore, CommandStore
- **Phantom types** across library modules — `DbPath`, `RemoteURL`, `AuthToken` (turso); `KeyID` (encryption); `Algorithm` (encryption); `DisplayID` (catalog); type-safe domain IDs in examples
- **Event binary blob helpers** (`event/`) — `AttachBlob`, `ExtractBlob`, `HasBlob` for signing/encryption
- **`command.TypedHandler[Q, R]`** with `RegisterTyped[Q, R]` — type-safe command handler
- **`event.DecodePayloads[T]()`** — batch payload deserialization
- **Listing table schema** (`storage/`) — DDL + repository for aggregate status persistence
- **ADR-0008 through ADR-0015** — 8 new architecture decision records (TypedHandler, immutability, OTel re-exports, error taxonomy, CBOR, encryption, saga, config)
- **ADR index** (`docs/adr/README.md`) — complete index of all 15 ADRs with titles, dates, status
- **Comprehensive fuzz testing** — fuzz tests in codec, encryption, signing/multisig, integration
- **Property-based tests** — `pgregory.net/rapid` in command, query, event, decider, id modules
- **go-snaps snapshot tests** — catalog, integration, projection golden test coverage
- **Benchmark infrastructure** — realistic scale benchmarks, fuzz benchmarks, multisig concurrent benchmarks
- **gosec security scanning** in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI
- **17 scale benchmarks** across modules (10K–1M events)
- **`pkg/config/`** — YAML config loader with env-specific overlays
- **`pkg/gracefulshutdown/`** — signal-aware shutdown with timeout and hook support
- **Docker packaging** for `example/user/` (multi-stage Dockerfile + docker-compose.yml)
- **SSE broker** (`middleware/sse.go`) — server-sent events over event bus
- **Health check middleware** (`middleware/healthcheck.go`) — `/health`, `/health/live`, `/health/ready`
- **Metrics HTTP handler** (`middleware/metrics_http.go`) — request count, error rate, avg response time
- **EventCatalog docserver** (`catalog/docserver/`) — embedded SPA with AsyncAPI + Scalar rendering
- **`integration/simulation/`** — event sequence generator + decider stress tests
- **Encryption integration** — end-to-end encrypt→sign→verify→decrypt round-trip tests
- **Test coverage:** storage/sql 37.4%→89.2%, otel 73.0%→97.3%, turso 26.8%→39.0%

### Changed

- **Pebble: migrated event envelope from JSON to CBOR encoding** — deterministic, compact binary format
- **Pebble: sharded mutex pool** (FNV-1a hash, 256 shards) replaces unbounded `sync.Map` — bounded memory, zero allocations
- **storage/sql: extracted generic `LoadWithSpan[T]` + `QueryRows[T]`** — eliminated event/command store load duplication
- **storage/sql: context-aware SQL methods** throughout — `BeginTx`, `ExecContext`, `QueryRowContext` (no more `noctx` lint)
- **storage/sql: `ClosableBase` extracted** — deduplicated store lifecycle boilerplate
- **OTel abstraction** — modules import `otel/` re-exports instead of `go.opentelemetry.io` directly (decider, storage, middleware, projection)
- **Error wrapping** — replaced `fmt.Errorf` wrapping classified errors with `WrapRejection`/`WrapCorruption` across memory, pebble, storage, listing
- **`command/command.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`query/query.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`event/types.go`** — `SchemaVersion.Cmp` now uses `cmp.Compare` (matches `Version.Cmp`)
- **`event/errors.go`** — doc comments on all 30 exported error symbols
- **`event/Clone()`** — deep-copies `eventOptions` pointer to prevent shared mutation
- **`event: Map/ScanState/Tap` reactive wrappers removed** (unused, no consumers)
- **`event: StreamKey` free function removed** (unused)
- **All 120 `//nolint` suppressions** now have documented `// reason` justifications
- **0 lint issues** across all 27 modules — first zero-lint release
- **`golang.org/x/exp`** bumped across all workspace modules
- **`storage/AggregateProjection`** uses `Dialect.Placeholder()` (Postgres-compatible)
- **`listing/AggregateRef` renamed to `AggregateListing`** with JSON tags
- **`catalog: ErrorExporter` deprecated** as type alias to `Exporter[error]`
- **`catalog: asyncapi.Info` and `openapi.Info` consolidated** into shared `DocumentInfo`
- **`snapshot: json tags`** added to `Snapshot` struct
- **Dissolved `core/` module** — all sub-packages are flat peer-level modules (v2.0.0, maintained in v2.3.0)
- **`event.Snapshot*` types moved to `snapshot/` package** — all consumers updated
- **`dispatcher/Lifecycle` field unexported** with method delegation added

### Fixed

- **SSE broker send-on-closed-channel race** — `handleEvent`/`RemoveClient` synchronization
- **SSE broker constructor** — `NewSSEBroker` now returns `(*SSEBroker, error)` instead of nil on error
- **Circuit breaker nil `IsFailure` guard** — defaults to `event.IsRetryable`
- **Circuit breaker error taxonomy** — `ErrCircuitBreakerOpen` uses error taxonomy instead of bare `errors.New`
- **Projection Runner double-wrapping classified errors** in `opError`
- **Projection Runner fresh done channel** per `Run` invocation
- **Projection Runner `Close()`** now waits for `Run` to complete
- **Clone shared opts pointer** — deep-copy `eventOptions` prevents shared mutation
- **Retry middleware** — `ErrRetryCanceled` sentinel actually used on context cancellation
- **Pebble `NewStore(nil, ...)` panics** with clear message instead of nil pointer dereference
- **Pebble `countEvents` uses `iter.Last()`** instead of full scan
- **Pebble `MarshalMetadataJSON` error** — handled instead of discarded
- **Decider `slog.WarnContext` fallback** for snapshot failures (previously OTel-only)
- **Multiple lint issues** — nlreturn, varnameld, noctx, errcheck, unconvert, nolintlint
- **`event.NewMetadata`** now initializes `Custom` map
- **`dispatcher/Lifecycle`** field unexported, added method delegation
- **`event: renamed `WithNewCodec`→`WithCodec`** (kept deprecated alias)
- **Config loader path traversal** — `filepath.Clean` sanitizes paths (gosec G304)
- **Graceful shutdown select guards** on errCh sends to prevent panic

### Performance

- **`catalog.SchemaFromType` cached by `reflect.Type`** — 553ns→8ns, 15→0 allocs
- **`event.New()` lazy-initializes metadata map** — 3→2 allocs per event
- **`event.New()` moves clock/newCodec/deadline to `eventOptions` pointer** — 48B saved per event
- **`event.PayloadReadOnly()` zero-copy** for internal paths (signing, pebble, storage, middleware)
- **`event.DecodePayload` bypasses `Payload()` clone** for zero-copy decoding
- **`listing` caches sorted aggregate index** — 25× faster listing
- **`memory` replaces O(n log n) `collectAllSorted`** with append-only global log
- **`signing.canonicalPayload()` eliminates alloc overhead**

### Security

- **gosec scanning** in CI with SARIF upload
- **Module layer check** enforced in CI
- **Config loader path traversal fix** (G304)
- **Constant-time ciphertext comparison** in encryption module

### Removed

- **`storage/options.go`** — deleted `NewSQLEventStoreWithOptions`, `WithOwnership`, `SQLEventStoreOption` (zero external consumers)
- **`storage/doc.go`** — removed 5 unused re-exports
- **`pebble/config.go`** — deleted entire config abstraction layer (`Backend`, `Config`, `NewConfig`, etc.)
- **`pebble/example_test.go`** — tested only deleted config API
- **`pebble/errors.go`** — removed `ErrPebbleProviderRequired`
- **`turso/errors.go`** — removed `ErrTursoMemorySync` backward-compat alias
- **All `MustParse`/`MustParseType` panic wrappers** removed from command, query, event test code
- **Deprecated backward-compat aliases** from `pebble/` module
- **Dead code and unused APIs** across multiple modules
- **`command/errors.go`** — removed unused `WrapTransient` re-export
- **`event/go.mod`** — removed `query/v2` direct dependency
- **`snapshot/go.mod`** — removed `memory/v2` dependency

## [2.2.0] - 2026-06-08

81 commits since v2.1.0. Operational readiness, testing rigor, and developer experience release.

### Added

- **Health check middleware** (`middleware/`) — `/health`, `/health/live`, `/health/ready` endpoints
- **Metrics HTTP handler** (`middleware/`) — request count, error rate, avg response time
- **SSE broker** (`middleware/`) — server-sent events over event bus with subscription management
- **Config loader** (`pkg/config/`) — YAML config with env-specific overlays
- **Graceful shutdown** (`pkg/gracefulshutdown/`) — signal-aware shutdown with timeout and hook support
- **Docker packaging** (`example/user/`) — multi-stage Dockerfile + docker-compose.yml
- **Production server example** (`example/user/server.go`) — operational endpoints demonstrating health, metrics, graceful shutdown
- **Property-based tests** (`decider/`, `event/`, `id/`) — `pgregory.net/rapid` for deterministic decide, version monotonicity, ULID validity
- **Snapshot tests** (`integration/`) — `go-snaps` for event JSON serialization, catalog exports
- **Simulation framework** (`integration/simulation/`) — event sequence generator + decider stress tests
- **Benchmark baseline** (`benchmark-baseline.txt`) — saved from all benchmarks for regression detection
- **Module READMEs** — 9 modules with usage and API surface documentation
- **Package doc.go** — 7 library modules with usage examples for pkg.go.dev
- **example_test.go** coverage — storage, otel, projection, watermill, schema, signing, snapshot, listing, pebble, turso, codec, dispatcher
- **docserver** (`catalog/docserver/`) — embedded EventCatalog SPA server with AsyncAPI + Scalar rendering

### Changed

- **Standardized flake configuration** — dev shell, test apps, benchmark apps unified
- **Command store split** — `storage/command_store.go` (387L → 3 focused files)
- **Snapshot errors extracted** — `snapshot/errors.go` with all sentinel errors
- **Projection replay refactored** — `loadReplayEvents` extracted (65L → 37L + 28L)
- **Dependencies bumped** — `golang.org/x/exp` across all workspace modules
- **Lint issues resolved** — all catalog, infrastructure, and pre-commit hook failures fixed

### Fixed

- **Catalog ToPascal byte underflow** — unicode boundary bug in case conversion
- **Duplicate package godoc** — removed from non-doc.go files in event, middleware, dispatcher
- **Broken example_test.go** — repaired in projection, schema, signing, watermill

### Security

- **gosec scanning** — Go security scanner integrated in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI

## [2.1.0] - 2026-06-03

62 commits since v2.0.0. Performance-focused release with production bug fixes, new query types, and comprehensive benchmarking.

### Added

- `query.TypedHandler[Q Query, R any]` — typed query parameter + typed result via `RegisterTyped[Q, R]`
- `listing.CacheInvalidationMiddleware(reader)` — auto-invalidates `InMemoryAggregateReader` cache after publish
- `listing.CacheInvalidator` interface — decouples middleware from concrete reader type
- 17 scale benchmarks across event, memory, listing, storage, pebble, turso, watermill, and codec modules
- 6 new benchmark suites with `b.ReportAllocs` for allocation tracking
- `nix run .#bench` app and `benchstat-compare` script for regression detection
- Turso CRUD integration tests for event/snapshot/checkpoint stores
- Realistic scale benchmarks behind `-tags=scale` in integration module
- ADR-0008 for `TypedHandler[Q Query, R any]` dual type parameter signature
- `docs/STORAGE_GUIDE.md` — performance comparison across PostgreSQL/SQLite/Pebble/Turso backends

### Changed

- `MemoryStore` deduplicated event storage — single `globalLog` + `streamIndex` map of indices replaces per-stream event copies (2× memory reduction)
- `event.New()` inlined codec extraction — removed `findCodecOption` helper, fast path for empty opts avoids probe allocation
- `MemoryStore.ReadFrom` uses cursor-based pagination instead of linear scan
- `schema.VersionedStore` load methods deduplicated into shared `loadAndUpcast` helper
- Error wrapping migrated to `event.Wrap*` taxonomy across storage, watermill, command, query, schema, and listing
- Deprecated backward-compat aliases removed from `pebble/` module
- Dead code removed + Go idioms modernized across multiple modules
- `event.Metadata()` documented as returning a defensive copy

### Performance

- `catalog.SchemaFromType` cached by `reflect.Type` — 553ns→8ns, 15→0 allocs
- `event.New()` lazy-initializes metadata map — 3→2 allocs per event
- `event.New()` moves clock/newCodec/deadline to `eventOptions` pointer — 48B saved per event
- `event.Payload()` removes defensive clone — 1 fewer alloc per access
- `event.New()` skips redundant payload copy — 1 fewer alloc
- `event.New()` stamps encoding directly — 1 fewer alloc
- `signing.canonicalPayload()` eliminates alloc overhead
- `listing` caches sorted aggregate index — 25× faster listing
- `memory` replaces O(n log n) `collectAllSorted` with append-only global log

### Fixed

- HealthCheck OOM on large event stores
- `SQLAggregateReader` Postgres compatibility
- `SubscriberAdapter` race condition
- Pebble `Close` not releasing resources
- `Version.Sub` panic on zero value
- `codec.Raw` passthrough encoding
- `GetID` rename consistency
- `ToAny` error propagation
- `HasSignature` false negatives
- `errgroup` error propagation
- `projection.Runner` missing `ErrAlreadyRunning` guard
- `storage` closed state tracking, snapshot SQL filter, `createTable` context
- `subscribeLive` handler guard for nil handlers
- `eventtest.FakeStore` ReadFrom test for sorted ReadAll output

### Removed

- Deprecated backward-compat aliases from `pebble/` module
- Dead code and unused APIs across multiple modules

## [2.0.0] - 2026-06-01

### Added

- `schema/` module — Upcaster, UpcasterRegistry, VersionedSource for schema evolution (extracted from event/)
- `snapshot/` module — Snapshot, SnapshotStore, SnapshotStrategy, helpers, error sentinels (extracted from event/)
- `samber/ro` integration in `event/reactive.go` — EventBus, NewReplayEventBus, NewBehaviorEventBus, FilterEventType/Types, ReplayFilter, HandlerToObserver/WithContext, Map, ScanState, Tap, Observable type alias
- `samber/ro` integration in `command/reactive.go` — CommandBus, FilterCommandType, Observable type alias
- `samber/ro` integration in `query/reactive.go` — QueryBus, FilterQueryType, Observable type alias
- `event/reactive.go` uses context-aware `ro.NewObserverWithContext` API — handler errors terminate the observer via `ErrorWithContext`
- `projection/runner.go` replay uses direct loop filters (`filterByEventTypes`, `filterFromCheckpoint`) instead of ro.Pipe1/ro.Collect overhead — projection no longer depends on `samber/ro`
- `listing/` module added to flake.nix testModules
- `otel/`, `pebble/`, `turso/`, `codec/` modules added to flake.nix testModules

### Changed

- **Dissolved `core/` module** — All 8 sub-packages (event, command, query, decider, id, dispatcher, schema, snapshot) are now flat peer-level modules. Import paths changed from `go-cqrs-lite/core/{pkg}` to `go-cqrs-lite/{pkg}`.
- `event.Snapshot*` types moved to `snapshot/` package — all consumers updated (decider, memory, storage, testhelpers)
- `event.ErrSnapshotNotFound` / `event.ErrSnapshotStoreClosed` moved to `snapshot/store.go`
- `memory/snapshot.go` uses `snappkg` alias to avoid local variable shadowing
- Removed duplicate `EventHandler` type from `event/reactive.go` (identical to `Handler`)
- AGENTS.md fully rewritten with new monorepo structure, dependency graph, key patterns
- Removed self-referencing replace directives (`module => ./`) from 6 go.mod files

### Removed

- `command/reactive.go` — temporarily deleted (restored in this release)
- `event/reactive.go` — restored with context-aware ro API (NewObserverWithContext + ErrorWithContext)
- `core/` directory — all sub-packages promoted to workspace root
- `event.Context() context.Context` — Go anti-pattern removed; use `Event.Deadline()` instead
- `event/context.go` — `deadlineCtx` type deleted (only used by removed `Context()`)

### Fixed

- `flake.nix` now includes all library modules in testModules
- `go.work.sum` stale references cleaned via `go work sync`

### Added

- `event.DecodePayloads[T]()` batch decode helper for processing multiple events at once
- `middleware.WithLogger(*slog.Logger)` option for retry, recovery, and validation middleware
- `storage/tables.go` — 5 table name constants replacing inline SQL strings
- `dispatcher.LifecycleMixin` embedded in `memory/checkpoint` and `memory/outbox`
- Concurrent access tests for MemoryBus, MemoryStore, MemoryOutbox, MemoryCheckpoint, MemorySnapshot
- `CONTEXT.md` — Domain glossary (aggregate, decider, event, fold, projection, saga)
- `docs/adr/` — ADR-0001 (Decider), ADR-0002 (Error taxonomy), ADR-0003 (Multi-module monorepo)
- `docs/ARCHITECTURE_PATTERNS.md` — Time-travel API, state-is-disposable, determinism, versioned events
- `docs/STORAGE_GUIDE.md` — PostgreSQL/SQLite/Pebble/Turso backends, event store operations

### Changed

- `AGENTS.md` trimmed from 384→121 lines (all essential info preserved)
- TODO_LIST.md reconciled: 40+ stale items verified as already done

### Fixed

- `storage/sql_base.go` bare `%w` wrapping → direct sentinel error return
- LSP hints: `sync.WaitGroup.Go` simplification, `fmt.Appendf` replacing `[]byte(fmt.Sprintf(...))`
- `projection/filterEvents` optimized from O(n×k) to O(n+k) via typeSet map

## [1.0.0] - 2026-05-26

### Added

- **saga** — Saga / Process Manager with compensation, retry, and timeout support
- **watermill** — Watermill message bus adapter with metadata-based event serialization
- **stream loading** — Memory-efficient `EventStream` + `StreamLoader` iterator pattern
- **event versioning** — `VersionedStore` with registered `Upcaster`s for transparent legacy event upcasting
- Full CQRS pipeline integration test (Command → Decider → Store → Bus → Projection → Query → Stream)
- Watermill metadata protocol: 15 metadata keys preserving all event fields

### Changed

- Eventcatalog coverage: 85.7% → 92.8%
- Saga coverage: 70.5% → 93.8%
- Watermill coverage: 28.6% → 89.6%
- `go.work` expanded to 13 modules

### Fixed

- Watermill `toEvent` used broken `json.Unmarshal` into `ImmutableEvent` — replaced with metadata reconstruction

## [0.2.0] - 2026-04-05

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters
- **SnapshotStrategy** (`core/event`): Canonical interface and `EveryNEvents(n)` extracted to `core/event/snapshot_strategy.go`
- **Publisher/Subscriber ISP** (`core/event`): Sub-interfaces extracted from `event.Bus` for Interface Segregation
- **Error classification** via `event.RegisterClassification()` in `init()` for aggregate, projection, storage sentinels
- **PublishChanges / SaveSnapshot** (`core/event`): Shared functions eliminating duplication in aggregate/decider repositories
- **Strong ID migration**: 62 bare `string`/`int` violations replaced with named types (`OperationID`, `NodeID`, `ServiceID`, `DomainID`, etc.)
- **Dialect tests** (`storage`): 15 tests for PostgresDialect, SQLiteDialect, `placeholders()`
- **OpenAPI coverage tests** (`catalog/openapi`)
- **Performance benchmarks**: 43 benchmarks across 12 files
- **Design documents**: Outbox transaction API, query handler generics, saga design

### Changed

- **ISP activation**: Repositories accept `Publisher`, projections accept `Subscriber` (backward-compatible)
- Root go.mod module path: `github.com/LarsArtmann/go-cqrs-lite` (consistent casing)
- Zero lint issues across all 8 linted modules (was 50+)
- File splits: all files under 250 lines
- `outboxEvent` fields: `Version`/`SchemaVersion` changed from bare `int` to strong types
- `gomodguard` → `gomodguard_v2`

### Fixed

- All linter issues resolved: exhaustruct, gosec G201, tagliatelle, wrapcheck, noinlineerr, prealloc, goconst, fatcontext
- `FakeSnapshotStore.Save` now records snapshots for verification (was no-op)
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher return errors correctly

## [0.1.0] - 2026-01-01

### Added

- Initial release with core CQRS infrastructure (command, event, query dispatchers)
- Event sourcing with `Store`, `Bus`, `SnapshotStore` interfaces
- In-memory implementations (`memory/` module)
- Branded IDs via `go-branded-id`
- Middleware: logging, retry, recovery, validation
- Test helpers for fakes and mocks
