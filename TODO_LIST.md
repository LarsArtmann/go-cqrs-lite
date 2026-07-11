# TODO List

**Updated:** 2026-07-11 (session: remediation execution + self-review)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v4]` = Breaking change, deferred to v4
- `[BLOCKED]` = Blocked on upstream dependency

---

## P0 — Critical (correctness, CI green, trust)

- [ ] **Fix premature codec default flip (CI RED)** — Commit `b3cca247` changed `event.DefaultCodec` from JSON to CBOR (`codec.go:28`), but the v4 flip is marked BLOCKED on user go-ahead. 3 tests fail: `TestDefaultCodec_DefaultIsJSON`, `TestMixedCodecStream`, `TestEventCodec_FallsBackToEventDefaultCodec`. Either revert to JSON or update tests + un-block the v4 task.
- [ ] **Update SKILL.md with new APIs** — `BackfillHandlerWithTransform`, `WithViews`, `VersionedSeekableJournal` not in consumer-facing module table (core file, 34KB). Identified this session, not yet closed.
- [ ] **Investigate auto-commit `0fef413e`** — 99-file commit appeared at 10:15:59 without explicit `git commit`. Determine if a Crush/git hook is active. If so, document in AGENTS.md and consider disabling.

- [x] **Register stdlib error classifications** — `errorfamily.RegisterStdlibDefaults()` called via init() in `storage/sql/classify_init.go`.
- [x] **Register database driver classifiers** — SQLite BUSY/LOCKED→Transient, CONSTRAINT→Conflict; Postgres SQLSTATE classes. Registered via init() in `storage/sql/classify_init.go`.
- [x] **Fix `WithReplayByteBudget(0)` semantics** — Added `SSEReplayBudgetDisabled = -1` sentinel. 0 auto-defaults to 8MB; -1 explicitly disables.
- [x] **Fix `api_surface.txt`** — Removed dead `JSONCodecV2` entry (line 308). Regenerated golden with all new modules tracked.
- [x] **Run `nix fmt` + `nix run .#lint`** — SA1019 deprecated alias warnings cleaned up across ALL modules. All internal code now uses `id.` and `metadata.` directly.

## P1 — High Value (architecture, consumer experience)

- [x] **metadata/ package tests + doc.go** — Comprehensive tests and package documentation.
- [x] **Consumer migration guide for id/ + metadata/ extraction** — Written to `docs/migration/MIGRATION-GUIDE.md`.
- [x] **Deprecated alias verification test** — `event/deprecated_alias_test.go` verifies all 6 deprecated aliases have proper Deprecated: comments.
- [x] **stack/v3 health checks** — `HealthChecker` interface + `Bundle.HealthCheck(ctx)` implemented in `stack/health.go`. Tests in `stack/health_shutdown_test.go`.
- [x] **stack/v3 topological shutdown ordering** — `WithShutdownDependency(before, after)` + Kahn's algorithm topo sort in `stack/shutdown.go`. Tests verify ordering.
- [x] **Update `scripts/check-module-layers.sh`** — Budget violations fixed (deriver=4, stack=14). projectionhost raised 7→9, watermill raised 8→9 (both from feature additions: SQLite DLQ + metadata extraction).
- [x] **CI check: go.work ↔ flake.nix sync** — `scripts/check-workspace-sync.sh` written. 8 missing modules added to flake.nix testModules.
- [x] **CI check: go.work ↔ api-stability tracking sync** — `scripts/check-api-stability-sync.sh` written. 12 missing modules added to api-stability tracking.
- [x] **Adopt `errorfamily.HTTPStatus()` in example/taskmanager** — `writeCQRSError` simplified from 15-line switch to 1-line `errorfamily.HTTPStatus(err)` call.

## P2 — Medium Value (test parity, observability, quality)

- [x] **BDD tests for EventIdempotency middleware** — 3 Ginkgo scenarios added: duplicate dedup, different events pass through, empty key skips dedup.
- [x] **SSE large-payload test (>8MB)** — `TestSSEHandler_ByteBudget_LargePayload` with 100KB×5 events under 250KB budget.
- [x] **ADR-0044: Blind store encoding stamps** — IMPLEMENTED. `codec/envelope.go` with WrapEncode/UnwrapDecode. Wired into all 4 blind stores (kv, snapshot, command, query). Backward-compatible raw decode fallback. ADR status: ACCEPTED.
- [x] **Fix file-size violations** — 3 production files split under 350-line CI limit: signing/cose.go → cose_sign1.go, cmd/doc-check/main.go → exports.go, catalog/eventcatalog/frontmatter_render.go → frontmatter_convert.go.
- [x] **Add SECURITY.md** — Documents vulnerability reporting process.
- [x] **json quality audit: Deterministic(true)** — Added to all Marshal calls in signing, encryption, event, storage, transport, listing, catalog.
- [x] **json quality audit: MatchCaseInsensitiveNames(true)** — Added to all Unmarshal calls across all modules.
- [x] **Projection parallelism** — Already implemented (one goroutine per projection with independent checkpoint, 10ms stagger).

## P3 — Polish & Cleanup

- [x] **Remove `codec/jsonv2_experiment.go`** — Dead code removed.
- [x] **Add `metadata/` to AGENTS.md + SKILL.md** — Module added to all docs.
- [x] **Document dispatcher middleware-at-dispatch-time behavior** — Fixed `dispatcher/doc.go`.
- [x] **Add v4-removal markers** — All 8 deprecated alias sites marked with `// v4-removal:` comments.
- [x] **Add SSEReplayBudgetDisabled to AGENTS.md** — SSE examples updated with byte-budget docs.
- [x] **Add `// Importing this package registers SQL classifiers` doc** — Added to `storage/sql/doc.go`.
- [x] **Write ADRs** — ADR-0047 (json/v2 case-insensitive decode), ADR-0048 (deterministic encoding), ADR-0049 (dispatch-time middleware).
- [x] **Deprecated alias cleanup** — All internal code updated from `event.AggregateRef` → `id.AggregateRef`, etc. ~200 usages across 42 files.
- [ ] **Restore bundle.go architectural comment** — Removed dead `var _ = []any{...}` code but also removed the documentation of Bundle↔CatchUpSubscriber relationship. Add a real doc comment.
- [ ] **Fix histogram test hard-coded values** — `prometheus/exporter_test.go:265` duplicates `otel.CQRSHistogramBoundaries` as a literal. If boundaries change in otel, test passes with stale values.
- [ ] **Run `nix flake check`** — Changed `scripts/check-module-layers.sh` but never re-ran flake check.
- [ ] **Run race detector on `stack/` and `example/taskmanager/`** — Changed `bundle.go`, `http.go`, `setup.go` this session; only ran race on projectionhost + transport/http.

### Experimental / Go-stdlib-blocked

- [BLOCKED] **jsonv2 codec experiment** — Pending Go stdlib stabilization (expected Go 1.27+).
- [BLOCKED] **Arena allocation experiment** — Pending Go arena API stabilization.
- [BLOCKED] **Turso MVCC concurrent-write support** — Blocked on upstream experimental MVCC.

### SQLiteDeadLetterStore production hardening (Gap 3 follow-ups)

- [ ] **DLQ `Purge(ctx, before time.Time)`** — Time-bounded cleanup for production DLQ management (item 28).
- [ ] **DLQ `List(ctx, offset, limit int)`** — Pagination for 100k+ dead letters (item 29).
- [ ] **DLQ `PurgeForProjection(ctx, name)`** — Purge entries for a specific projection during reset (item 33).
- [ ] **DLQ `Count(ctx) (int64, error)`** — Dashboard metrics for DLQ depth (item 30).
- [ ] **DLQ serialization format docs** — Document what event fields are stored and how to migrate the schema (item 32).
- [ ] **DLQ stress test** — 10k entries, verify query performance (item 16).
- [ ] **DLQ concurrent Store test** — Verify SQLite busy_timeout under concurrent writes (item 17).
- [ ] **DLQ corrupt JSON test** — Verify graceful failure on malformed event payload (item 18).

### VersionedSeekableJournal follow-ups (Gap 1)

- [ ] **Property test with rapid** — Random upcaster chains on event streams (item 14).
- [ ] **Upcaster error mid-stream test** — What happens when an upcaster returns an error during projection host replay? (item 23).
- [ ] **Benchmark: upcasting overhead** — `ReadFrom` with 10k events (item 15, P3).

### SSE transform follow-ups (Gap 2)

- [ ] **CBOR→JSON e2e test** — Through all 3 SSE paths (live, replay, backfill) with a real CBOR-encoded event (item 25).

### Projectionhost observability

- [ ] **`LagPerProjection() map[string]time.Duration`** — Per-worker lag for dashboards (item 38).
- [ ] **`WorkerState.Lag` field** — Currently only available via aggregate `LagDuration()` (item 39).
- [ ] **`Reset(ctx, name)` purges DLQ** — Projection reset should optionally clear DLQ entries for that projection (item 46).

### Testing improvements

- [ ] **Race detector on ALL modules** — CI runs `-race` on changed modules; run full suite periodically (item 13).
- [ ] **`scenario.GivenProjection` test** — For VersionedSeekableJournal + projectionhost (item 48, P3).

### Index/Performance

- [ ] **DLQ index optimization audit** — Verify `UNIQUE(projection_name, event_id)` is optimal for List-by-projection (item 31, P3).

### Documentation

- [ ] **Document two DeadLetterEntry types** — ADR-0043 Part B: dispatch-side vs projection poison, intentionally separate (item 45, P3).
- [ ] **README.md docs freshness** — Missing `encryption`, `turso`, `testutil` module sections.

### Rejected (with reasons)

- **Unify VersionedStore + VersionedSeekableJournal** (item 20) — Different interfaces (Store: Load/Save per aggregate, SeekableJournal: ReadFrom position-based). One type can't cleanly implement both. YAGNI.
- **VersionedJournal (ReadAll only)** (items 22, 49) — No consumer needs `ReadAll` with upcasters. SeekableJournal is the projectionhost interface. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** (item 24) — No demand. Consumers configure the transform at construction.
- **`WithPayloadTransform` on SSEHandler** (item 27) — SSEHandler wraps the broker; adding transform there duplicates responsibility (SRP violation).
- **Auto-apply CQRS views by default** (items 35, 50) — Violates "library, not framework" principle. Consumers choose their histogram boundaries.
- **VersionedSeekableJournal implementing event.Store** (item 40) — Different scope (position-based vs aggregate-based reads). YAGNI.
- **Integration test in `integration/` module** (item 19) — Redundant with `projectionhost/versioned_journal_integration_test.go`.

### Performance

- [ ] **Hot-State cache (decider)** — Optional `RepositoryOption[State]` that caches folded aggregate state. Profile before building.
- [ ] **Read-pressure snapshot strategy** — `EveryNEvents` snapshots based on writes; add `ReadPressureStrategy`.

### Transport

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules.
- [ ] **Distributed event bus** — No multi-process backend for event distribution.

### Public Release Readiness

- [ ] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption. **Needs user approval (irreversible).**
- [ ] **Git history scrub for internal docs** — AGENTS.md, docs/planning/\* contain internal strategy. **Needs user approval (irreversible).**
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label experimental.
- [ ] **README polish to "sales page" standard** — Per AGENTS.md rule.

---

## v4 Breaking Changes (deferred)

> **BLOCKED on user go-ahead.** All prep work is done (ADR-0044 envelopes, deprecation markers, alias cleanup, migration guide). Execute when ready to cut v4.

- [v4] **Flip codec defaults** — Events + blind stores → CBOR. Safe now with ADR-0044 envelopes. **NOTE:** `event.DefaultCodec` was prematurely flipped to CBOR in `b3cca247`; see P0 above. Blind stores already self-describing via ADR-0044.
- [v4] **Remove deprecated APIs** — 8 aliases in event/ + schema/ + query.Handler.
- [v4] **Storage/ split execution** — Proposal at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`. Awaits approval.
- [v4] **Event/ god module decomposition** — Explicitly decided: DO NOT SPLIT (27 importers, cohesion is real).
- [v4] **BackfillHandler taking \*SSEBroker** — Cleaner architecture; backfill can access broker's transform (item 47).

---

## Recently Completed

- [x] **ADR-0044 blind store envelopes** — codec.WrapEncode/UnwrapDecode, wired into all 4 blind stores.
- [x] **json Deterministic + MatchCaseInsensitive audit** — All Marshal/Unmarshal calls in library code fixed.
- [x] **Deprecated alias cleanup** — ~200 usages across 42 files updated to id./metadata.
- [x] **CI safety nets** — check-workspace-sync.sh + check-api-stability-sync.sh. 8 modules added to CI, 12 to API tracking.
- [x] **Health checks + shutdown ordering** — Bundle.HealthCheck(ctx) + WithShutdownDependency.
- [x] **Consumer migration guide** — docs/migration/MIGRATION-GUIDE.md.
- [x] **EventIdempotency BDD tests** — 3 Ginkgo scenarios.
- [x] **SSE large-payload test** — Byte budget boundary verification.
- [x] **errorfamily.HTTPStatus() in taskmanager** — Eliminated hand-rolled error→status mapping.
- [x] **ADRs 0047-0049** — json/v2 decode, deterministic encoding, dispatch-time middleware.
- [x] **Dispatcher middleware-at-dispatch-time fix** — Middleware can be added in any order.
- [x] **json/v2 case-insensitive decode fix** — All decode paths use MatchCaseInsensitiveNames(true).
- [x] **kv/ context.Context propagation** — All 11 kv I/O methods accept context.Context.
- [x] **id/ + metadata/ extraction** — AggregateRef → id/, Tracing/CustomData → metadata/.
- [x] **Idempotency merge** — Generic NewIdempotency[M] factory + 3 wrappers.
- [x] **Projectionhost production hardening** — M1-M13 complete.
- [x] **Error taxonomy migration** — All event._ facade calls → errorfamily._ direct imports.
- [x] **Removed ALL `var _ =` hacks** — 4 found and removed: `sse_backfill.go:151` (context.Background), `example/taskmanager/http.go:329` (context.Background, report missed this), `stack/bundle.go:289` (dead-code assertion), `example/taskmanager/setup.go:301` (dead-code event.Version(0)).
- [x] **DiscordSync feedback gaps (Round 2)** — 5 gaps from `2026-07-10_DiscordSync_leverage_review.md`:
  - [x] `schema.VersionedSeekableJournal` — wraps SeekableJournal with upcasters for projection host
  - [x] `transport/http.WithPayloadTransform` + `BackfillHandlerWithTransform` — wire-format transcoding on all 3 SSE paths (live, replay, backfill)
  - [x] `projectionhost.SQLiteDeadLetterStore` — production SQLite-backed DLQ + ADR-0043 documentation
  - [x] `prometheus.WithViews` — custom metric views for Prometheus exporter (compose with `cqrsotel.NewCQRSViews()`)
  - [x] Cross-module integration test: VersionedSeekableJournal + projectionhost.New()
  - [x] API surface golden file updated (2212 exports)
  - [x] `nix fmt` + `nix run .#lint` clean on all changed files

---

_Files read for this update: all 2026-07-0* status/planning/feedback files, ROADMAP.md, v4-WISHLIST.md, BLOCKED-ITEMS.md, all docs/feedback/*, docs/quality/\* freshness report, and code verification of metadata/, codec/, kv/, snapshot/, command/, query/, stack/, signing/, encryption/, event/, transport/http/, middleware/, projectionhost/._
