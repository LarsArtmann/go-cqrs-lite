# TODO List

**Generated:** 2026-06-12
**Reconciled:** 2026-06-12 — verified against source code, all docs/status/, docs/planning/, ROADMAP.md, FEATURES.md, and CHANGELOG.md
**Version:** v2.3.0 (commit `4e221fbc`)
**Test Status:** 38/40 packages pass. 2 pre-existing golden test failures: `codec/TestGolden_JSONCodec_Encode` (compact vs indented), `middleware/TestGolden_HealthCheckResponse` (golden drift)

## Legend

- `[x]` = DONE (verified in source)
- `[ ]` = NOT DONE (code checked, no implementation)
- `[v2]` = Breaking change, deferred to v2
- `[v3]` = Breaking change, deferred to v3
- `[BLOCKED]` = Requires external action (push tags, different repo, etc.)
- `[FUTURE]` = Speculative/far-future, not actionable now
- `~~strikethrough~~` = Item was misidentified or no longer relevant

---

## 🔴 HIGH Priority

### Safety & Correctness

- [x] ~~Add panic recovery to HandleParallel goroutine~~ — DONE (`projection/runner.go`)
- [x] ~~Fix Pebble store optimistic concurrency check in Save~~ — DONE
- [x] ~~Fix SQLEventStore.Close()~~ — DONE
- [x] ~~Fix SQLSnapshotStore double-marshal~~ — DONE
- [x] ~~Fix retry middleware timer leak~~ — DONE
- [x] ~~Fix decider Execute dual `%w` wrapping~~ — DONE
- [x] ~~Fix loadEvents — propagate snapshot load errors~~ — DONE
- [x] ~~Fix SQLEventStore.Load to return ErrAggregateNotFound~~ — DONE
- [x] ~~Fix MemorySnapshotStore deep copy~~ — DONE
- [x] ~~Fix query handler signature to include `context.Context`~~ — DONE
- [x] ~~Fix FakeStore/MemoryStore key separator mismatch~~ — DONE
- [x] ~~Fix JSON v1/v2 split in storage metadata~~ — DONE
- [x] ~~Fix Pebble LoadToTimestamp full scan~~ — DONE (early termination)
- [x] ~~Fix filterEvents O(n) scan in projection runner~~ — DONE
- [x] ~~Fix pebble unbounded lock map~~ — DONE (sharded [256]sync.Mutex)
- [x] ~~Fix SSE send-on-closed-channel race~~ — DONE
- [x] ~~Fix circuit breaker error taxonomy~~ — DONE
- [x] ~~Fix circuit breaker nil IsFailure panic~~ — DONE
- [x] ~~Fix catalog/registry.go Build() corruption~~ — DONE
- [x] ~~Fix catalog/docserver XSS~~ — DONE
- [x] ~~Fix watermill double-close panic~~ — DONE
- [x] ~~Fix watermill MustParse\* panic on untrusted input~~ — DONE
- [x] ~~Fix memory/checkpoint missing CheckClosed~~ — DONE
- [x] ~~Fix signing middleware nil signer/verifier panic~~ — DONE
- [x] ~~Fix cmd/api-stability wrong module paths~~ — DONE
- [x] ~~Fix cmd/cqrs-gen query template~~ — DONE
- [x] ~~Fix example/todo stale references~~ — DONE
- [x] ~~Fix decider snapshot errors silently discarded~~ — DONE
- [x] ~~Fix ErrRetryCanceled dead sentinel~~ — DONE
- [x] ~~Fix pebble metadata deserialization swallowed error~~ — DONE
- [x] ~~Fix Version.Cmp manual comparison~~ — DONE
- [x] ~~Fix pebble nil store panic~~ — DONE

### API Surface

- [x] ~~Add IdempotencyKey to Command interface~~ — DONE
- [x] ~~Add ClientID branded type and WithClientID option~~ — DONE
- [x] ~~Add IsReplay getter~~ — DONE
- [x] ~~Remove StreamKey free function~~ — DONE
- [x] ~~Fix AggregateProjection hardcoded placeholders~~ — DONE
- [x] ~~Fix query TypedHandler takes Query not T~~ — DONE
- [x] ~~Fix event/event_new.go no clone of []byte/json.RawMessage~~ — DONE
- [x] ~~Fix dispatcher CatalogDispatcher race~~ — DONE
- [x] ~~Fix decider load.go unclassified fmt.Errorf~~ — DONE
- [x] ~~Fix middleware circuit breaker double-wrapped error~~ — DONE
- [x] ~~Fix catalog/ToAny silently swallows marshal errors~~ — DONE
- [x] ~~Fix signing/HasSignature swallows corruption errors~~ — DONE
- [x] ~~Fix watermill silently drops malformed ID parse errors~~ — DONE
- [x] ~~Fix event/batch.go vs event_new.go marshal duplication~~ — DONE
- [x] ~~Fix schema versioned_source.go 4 near-identical load methods~~ — DONE

### Middleware Consolidation

- [x] ~~middleware/ 3× duplication across command/event/query~~ — DONE (generic `NewX[M]` + 27 thin wrappers)
- [x] ~~Add WithLogger to all middleware constructors~~ — DONE
- [x] ~~Add circuit breaker middleware~~ — DONE
- [x] ~~Add OpenTelemetry tracing middleware~~ — DONE
- [x] ~~Add distributed tracing middleware~~ — DONE
- [x] ~~Add SSE broker middleware~~ — DONE
- [x] ~~Add health check middleware~~ — DONE

### CI & Tooling

- [x] ~~Add CI pipeline~~ — DONE (`ci.yml`)
- [x] ~~Add minimum coverage gate to CI (80%)~~ — DONE
- [x] ~~Add -race to CI~~ — DONE
- [x] ~~Add coverage tracking to CI~~ — DONE
- [x] ~~Parallelize CI matrix~~ — DONE
- [x] ~~Add gosec security scanning~~ — DONE
- [x] ~~Performance regression CI~~ — DONE
- [x] ~~Add per-module `go vet`~~ — VERIFIED
- [x] ~~Add gofumpt/goimports to pre-commit~~ — DONE
- [x] ~~Add .goreleaser.yml~~ — DONE
- [x] ~~Standardize module versions~~ — DONE
- [x] ~~Add GOWORK=off CI verification~~ — DONE
- [x] ~~Add file-size gate (≤350 lines)~~ — DONE

---

## 🟡 MEDIUM Priority

### Code Quality

- [ ] **Fix ADR numbering gap** — ADR-0005 is missing; README lists ADR-0001 through ADR-0015 but 0005 is a gap. Decide: add placeholder or renumber
- [ ] **cmd/api-stability zero tests** — API surface checker has no test coverage. The tool guards breaking changes but is itself untested
- [ ] **query.BasicQuery has no metadata** — Unlike `BasicCommand`, queries carry no correlation/tracing context. Makes distributed tracing through query path inconsistent with command/event
- [ ] **eventtest/ as separate module** — event/go.mod lists 5 test-only deps (command, query, memory, schema, snapshot) that bloat consumer transitive deps. Extracting eventtest to its own go.mod would clean this up. ACCEPTED/WONTFIX — breaking change for consumers
- [ ] **Clean test deps from 12 production go.mod files** — 12 modules have test-only deps in production require blocks. Go doesn't support separate test-only require blocks
- [ ] **Fix nolint:errcheck suppressions in defer .Close() calls** — 31 suppressions, many are lazy. Use explicit error handling or `defer func()` pattern
- [ ] **Verify all `//nolint` comments have justification** — Most suppressions now have reasons (verified in event/command/query). Audit remaining modules: middleware, storage, catalog, encryption
- [ ] **Reduce catalog/ nolint suppressions** — 36 total, worst package

### Documentation

- [ ] **Add godoc examples for listing package** — `List`, `StatusMiddleware`, `InMemoryAggregateReader`
- [ ] **Add README section linking to docs/benchmarks/** — Consumers should know perf characteristics exist
- [ ] **Document CBOR usage patterns** — codec/README.md exists but lacks consumer-facing CBOR examples
- [x] ~~Document time-travel API~~ — DONE
- [x] ~~Document soft deletes over hard deletes~~ — DONE
- [x] ~~Document offline-first metadata conventions~~ — DONE

### Performance

- [ ] **Arena allocation experiment in event module** — High-throughput event creation could benefit from arena allocation
- [ ] **Zero-allocation event encoding path** — `jsonv2` experiment behind build tag
- [ ] **SIMD-accelerated event serialization** — Go experiment for large payload encoding
- [ ] **Benchmark MemoryStore with concurrent writers** — Stress-test for race conditions under load
- [ ] **Profile allocation patterns** — Compare JSON vs CBOR allocations with `go test -benchmem`

### Features (No Design Yet)

- [ ] **Outbox pattern design doc** — Reliable at-least-once event publishing. Mentioned in ROADMAP but no ADR or prototype
- [ ] **Schema registry design doc** — JSON Schema middleware for event validation. Mentioned in ROADMAP but no design exists
- [ ] **Distributed checkpointing for projections** — Multiple projection instances sharing checkpoint state

### Coverage Gaps

- [ ] **Add turso integration tests** — sync.go (OpenSync, Push, Pull, Checkpoint, Stats) only tested for rejection paths; needs real Turso server or better mocking. Coverage ~75% (up from 28.6%)
- [ ] **Add storage/sql direct tests for query_engine** — coverage_test.go exists but query_engine edge cases are light

### Encryption Module

- [ ] **Add `StaticKeyResolver` helper (map-based)** — 80% use case for key rotation
- [ ] **Add versioned ciphertext format (prefix byte for algorithm)** — Future-proof algorithm changes
- [ ] **Add `example/encryption/` project** — Standalone example demonstrating full encryption lifecycle
- [ ] **Add storage wrapper: `storage.NewEncryptedEventStore`** — Convenience for encrypted stores
- [ ] **Field-level encryption (`encryption/fieldlevel/`)** — Per-field encryption for selective payload protection

### Turso Indexing (Deferred from Sprint)

- [ ] **Comparison report generator** — CLI tool comparing indexed vs unindexed performance
- [ ] **Hooks API (`turso.WithIndexingHooks`)** — Pre/post index creation callbacks
- [ ] **Schema evolution / migration integration** — Integrate with `schema/` module
- [ ] **Health check integration** — Integrate with `listing/health` module
- [ ] **Postgres/Compact-specific indexing guidance** — Platform-specific optimization tips

---

## 🟢 LOW Priority

### Polish

- [ ] **Add `go-snaps` across remaining modules** — signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory (some already have golden tests)
- [ ] **Add CBOR fuzz test for pure CBOR→CBOR** — codec_fuzz_test.go exists but should test pure CBOR fidelity without JSON intermediary
- [ ] **Add CBOR DecMode configuration** — Match encode/decode expectations, enable strict mode
- [ ] **Evaluate CoreDetEncOptions vs CanonicalEncOptions** — Which is right default for signing safety?
- [x] ~~Fix codec/raw.go json.RawMessage support~~ — DONE
- [x] ~~Godoc examples for decider~~ — DONE (example_test.go)
- [x] ~~Godoc examples for projection~~ — DONE (example_test.go)
- [x] ~~Godoc examples for signing~~ — DONE (6 examples across signing/ and signing/multisig/)
- [x] ~~Godoc examples for schema~~ — DONE (example_test.go)
- [x] ~~Remove unused `backend` field from Pebble store~~ — DONE (Backend type removed)
- [x] ~~Fix CBOR cborEncMode error handling~~ — DONE (panics on init failure, not silently dropped)
- [x] ~~Fix listing/InMemoryAggregateReader caching~~ — DONE (has cached field, rebuildCache, InvalidateCache)
- [x] ~~Add tests for command.Type.IsZero/ParseType~~ — DONE
- [x] ~~Add tests for query.Type.IsZero/ParseType~~ — DONE
- [x] ~~Add CBORCodec to codec module~~ — DONE
- [x] ~~Add Pebble CBOR envelope~~ — DONE
- [x] ~~Add XChaCha20-Poly1305 to encryption~~ — DONE
- [x] ~~Fix encryption/Ciphertext.Equal() constant-time~~ — DONE (uses subtle.ConstantTimeCompare)
- [x] ~~Fix ADR README.md only listing 0001–0003~~ — DONE (now lists all 15 ADRs)

### Pre-existing Test Failures

- [ ] **Fix `codec/TestGolden_JSONCodec_Encode`** — Golden file expects indented JSON but codec outputs compact JSON. Update golden file with `-update`
- [ ] **Fix `middleware/TestGolden_HealthCheckResponse`** — Golden file drift. Update with `-update`

### Superb Types (Phantom Types)

- [ ] **Add `String()` + `IsZero()` to all 17 catalog phantom types** — Bare `type X string` definitions lack methods
- [ ] **Add `Int()` method to `example/todo/domain.Priority`** — `type Priority int` but requires explicit casts
- [ ] **Bool→Enum conversions** — 7 locations (catalog/schema, catalog/asyncapi, dispatcher, event/replay, storage/sql, pebble, projection)
- [ ] **Split `catalog.Message` into Message+MessageMeta** — 17 fields → structured embedding
- [ ] **Split `catalog.Service` into Service+ServiceMeta** — 16 fields → structured embedding

### Process

- [ ] **Add benchmark regression detection in CI** — Baselines exist but no automated gate
- [ ] **Add `go vulncheck` to CI** — Security hardening
- [ ] **Docker build CI step (linux/amd64 + linux/arm64)** — Multi-arch Docker builds not in CI
- [ ] **Playwright E2E tests for example/user/** — Health + command→event→query flow

### v2 Breaking Changes (Deferred)

- [v2] **Remove io.Closer from core interfaces** — ADR-0010 accepted but not implemented. Breaking change for event.Store, snapshot.SnapshotStore, command.Store
- [v2] **Add global TransactionID branded type** — Cross-aggregate consistency tracking
- [v2] **Make event Core truly immutable** — Currently opts pointer is shallow-copied on Clone (payload/metadata are deep-copied)
- [v2] **Split event.Store into Writer/Reader/Deleter** — ADR-0010 direction
- [v2] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module
- [v2] **Fix query.Handler returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` instead of `(any, error)`

---

## ⚪ Verified / Accepted

- [x] ~~Return error from Pebble iterateEvents~~ — DONE
- [x] ~~Implement SQL dialect abstraction~~ — DONE
- [x] ~~Split pebble_event_store.go~~ — DONE
- [x] ~~Split storage/helpers.go~~ — DONE
- [x] ~~Add go-sqlmock tests for LoadToVersion, LoadToTimestamp~~ — DONE
- [x] ~~Storage coverage recovery~~ — DONE (90.2%)
- [x] ~~Parameterize OutboxStatusPending~~ — DONE
- [x] ~~NewEvent: accept event.Version instead of raw int~~ — DONE
- [x] ~~Add publish-side event middleware~~ — DONE
- [x] ~~Log OutboxPublisher poll errors~~ — DONE
- [x] ~~Add ProcessedAt to CheckpointStore~~ — DONE
- [x] ~~Add PublishedAt to OutboxEntry~~ — DONE
- [x] ~~Consolidate Root.LoadEvents vs Core.LoadFromHistory~~ — MOOT (aggregate deleted)
- [x] ~~Unify aggregate/decider repository persistence logic~~ — DONE
- [x] ~~Implement Repository.LoadAtVersion / LoadAtTime~~ — DONE
- [x] ~~Delete deprecated Catalogable/CatalogMeta/CatalogCore~~ — VERIFIED (gone)
- [x] ~~Consolidate CatalogMeta~~ — DONE
- [x] ~~Add SchemaVersion strong type~~ — DONE
- [x] ~~Add OutboxStatus enum~~ — DONE
- [x] ~~Push release tags to remote~~ — DONE (v2.0.0–v2.3.0 pushed)
- [x] ~~Create core/pkg/errors/ package~~ — DONE
- [x] ~~Map all existing sentinel errors to error families~~ — DONE
- [x] ~~Register 6 remaining unclassified sentinels~~ — DONE
- [x] ~~Extract error classification to standalone package~~ — VERIFIED (go-error-family)
- [x] ~~Standardize storage error wrapping patterns~~ — DONE
- [x] ~~Replace init() error registration with explicit setup~~ — DONE
- [x] ~~Formally deprecate aggregate package~~ — DONE (deleted)
- [x] ~~Break event↔command cycle~~ — DONE
- [x] ~~Break memory↔snapshot cycle~~ — DONE
- [x] ~~Extract sql.QueryEngine~~ — DONE
- [x] ~~Remove command error re-exports~~ — DONE
- [x] ~~Fix Lifecycle exported field~~ — DONE
- [x] ~~Fix command.Metadata split-brain~~ — DONE (type alias)
- [x] ~~Fix VersionedStore embedded Store~~ — DONE
- [x] ~~Unify ErrDispatcherClosed across packages~~ — ACCEPTED: each module has unique error
- [x] ~~Fix pebble/errors.go duplicate sentinels~~ — ACCEPTED: each module independently importable
- [x] ~~Fix command re-exports event types~~ — ACCEPTED: intentional boundary for consumer convenience
- [x] ~~Fix dispatcher local checkClosed helpers~~ — ACCEPTED: cross-module sharing would violate isolation
- [x] ~~Fix decider → memory dependency~~ — VERIFIED (test-only)
- [x] ~~Fix storage → listing coupling~~ — VERIFIED (correct dependency direction)
- [x] ~~Fix middleware 3x duplication~~ — DONE
- [x] ~~Fix event/ module cycles~~ — ACCEPTED (test-only deps)
- [x] ~~Extract shared opError helper~~ — MOOT (different wrapping patterns per module)
- [x] ~~Extract eventtest as separate module~~ — ACCEPTED/WONTFIX — breaking change for consumers
- [x] ~~Consolidate testhelpers fake boilerplate~~ — WONTFIX — each fake has different fields/interfaces

---

## 📐 PLANNED / FUTURE (No Code Yet)

### From ROADMAP.md

- [ ] Docker build CI step (linux amd64 + arm64)
- [ ] Playwright E2E tests for example/user/
- [ ] `jsonv2` codec experiment behind build tag
- [ ] Arena allocation for high-throughput event creation
- [ ] SIMD-accelerated event serialization
- [ ] Streaming event reads without materializing full slice
- [ ] Outbox pattern implementation (reliable at-least-once publishing)
- [ ] Saga module (orchestrated multi-step transactions) — DECLINED: demonstrated via example/todo
- [ ] Event schema registry with validation middleware
- [ ] Distributed checkpointing for projections
- [ ] cqrs-gen v2 with struct tag scanning
- [ ] WebAssembly compilation target for decider module
- [ ] gRPC transport adapter
- [ ] NATS / Redis Stream adapter
- [ ] Built-in pprof endpoints
- [ ] Custom metrics exporter (Prometheus format)
- [ ] Structured logging middleware with configurable levels
- [ ] Distributed tracing span propagation across module boundaries
- [ ] Event stream compaction / log truncation strategies
- [ ] Multi-tenant event store (schema-per-tenant)
- [ ] Event archival to S3 / GCS / Azure Blob
- [ ] CQRS-lite dashboard (web UI)
- [ ] Automatic migration generator for schema evolution
- [ ] Property-based integration testing with state machine verification
- [ ] Chaos engineering integration
- [ ] Performance regression dashboard

### From docs/planning/

- [FUTURE] Add bi-temporal support: ValidAt, WithValidAt, LoadToValidTime
- [FUTURE] Add HLC (Hybrid Logical Clock) implementation
- [FUTURE] Implement pull-before-push sync protocol
- [FUTURE] Implement rebase mechanism
- [FUTURE] Build network simulator for testing
- [FUTURE] Build multi-client test harness
- [FUTURE] Build thin PostgreSQL store adapter (no Watermill)
- [FUTURE] Build thin NATS bus adapter (no Watermill)
- [FUTURE] Add hybrid service example
- [FUTURE] Add multi-engine storage support via sqlc
- [FUTURE] Add schema migration tool
- [FUTURE] Create documentation site (Docusaurus/MkDocs/Hugo)
- [FUTURE] Add catalog diff/breaking-change detection tool
- [FUTURE] Add high-level test utilities (AggregateTester, ProjectionTester, BusTester)
- [FUTURE] Add distributed consensus capability (Raft/CRDT overlay)
- [FUTURE] Add time-series event query language for event store

---

## ❌ STALE / REMOVED / MOOT

- ~~`MustNew` panic helper~~ — DOES NOT EXIST. Only test-local `mustNewCmd` helper
- ~~`CatalogDispatcher` embedded in command/query dispatchers~~ — DOES NOT EXIST
- ~~`BatchProjection` optional interface~~ — DOES NOT EXIST
- ~~`Reactive CommandBus/QueryBus`~~ — DOES NOT EXIST as separate types. `ro.Subject[Command]` mentioned in AGENTS.md but never implemented
- ~~`Outbox pattern`~~ — REMOVED from library scope. Lives in example/ only
- ~~`Saga module`~~ — DECLINED. Saga pattern demonstrated via example/todo
- ~~`GraphQL query adapter`~~ — DECLINED. Framework-level concern
- ~~`event.ImmutableEvent.Clone shares opts pointer~~ — SHALLOW copy only. Payload and metadata are properly deep-copied. opts is shallow-copied (all fields are immutable). Not a real issue.
- ~~Fix CBOR cborEncMode error handling~~ — ALREADY FIXED. Panics on init failure.
- ~~Remove unused backend field from Pebble~~ — ALREADY DONE. Backend type removed.
- ~~Fix ADR README.md only listing 0001–0003~~ — ALREADY FIXED. Lists all 15 ADRs.
- ~~listing/InMemoryAggregateReader uncached~~ — ALREADY FIXED. Has cache with invalidation.
- ~~Tests for command.Type.IsZero/ParseType~~ — ALREADY DONE.
- ~~Tests for query.Type.IsZero/ParseType~~ — ALREADY DONE.
- ~~encryption/Ciphertext.Equal() not constant-time~~ — ALREADY FIXED. Uses subtle.ConstantTimeCompare.
- ~~Move example/todo to own repository~~ — BLOCKED (requires manual repo creation)
- ~~Create go-branded-id v0.2.0~~ — BLOCKED (different repo)
- ~~Design ActaFlow's event sourcing overlay~~ — BLOCKED (different project)
- ~~Extract shared golangci.yml~~ — BLOCKED (different repo)
- ~~Remove cockroachdb/errors from go-localsync~~ — BLOCKED (different repo)
- ~~Change LICENSE~~ — BLOCKED (owner decision)

---

_Items verified against source on 2026-06-12. Total done/verified: 190+. Total open actionable: ~50 (medium + low). Total planned/future: ~45._
