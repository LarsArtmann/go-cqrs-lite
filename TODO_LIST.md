# TODO List

**Generated:** 2026-06-11
**Reconciled:** 2026-06-11 — verified against source code
**Files Processed:** 270+

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
- [x] ~~Add panic recovery to OutboxPublisher.run goroutine~~ — DONE (removed during v2 restructuring)
- [x] ~~Fix sync.NewLWWResolver nil panic~~ — DONE (returns `ErrNilTimestampFunc`)
- [x] ~~Fix Pebble store optimistic concurrency check in Save~~ — DONE (Session 106)
- [x] ~~Fix outbox transaction co-participation~~ — DONE (Session 107, outbox via SQLBackend)
- [x] ~~Fix SQLEventStore.Close()~~ — DONE (Session 25)
- [x] ~~Fix SQLSnapshotStore double-marshal~~ — DONE (Session 25)
- [x] ~~Fix retry middleware timer leak~~ — DONE
- [x] ~~Fix decider Execute dual `%w` wrapping~~ — DONE
- [x] ~~Fix loadEvents — propagate snapshot load errors~~ — DONE
- [x] ~~Fix SQLEventStore.Load to return ErrAggregateNotFound~~ — DONE
- [x] ~~Fix collectResults in runner.go~~ — DONE
- [x] ~~Fix MemorySnapshotStore deep copy~~ — DONE
- [x] ~~Fix query handler signature to include `context.Context`~~ — DONE
- [x] ~~Fix FakeStore/MemoryStore key separator mismatch~~ — DONE
- [x] ~~Fix JSON v1/v2 split in storage metadata~~ — DONE
- [x] ~~Fix Pebble LoadToTimestamp full scan~~ — DONE (early termination, ~9.5x faster)
- [x] ~~Fix filterEvents O(n) scan in projection runner~~ — DONE
- [x] ~~Fix pebble unbounded lock map~~ — DONE (sharded [256]sync.Mutex with FNV-1a hash)
- [x] ~~Fix SSE send-on-closed-channel race~~ — DONE (`b652dd32`)
- [x] ~~Fix NewSSEBroker nil return~~ — DONE (`b652dd32`, now returns `(*SSEBroker, error)`)
- [x] ~~Fix circuit breaker error taxonomy~~ — DONE (`9d407f51`)
- [x] ~~Fix circuit breaker nil IsFailure panic~~ — DONE (`9d407f51`)
- [x] ~~Fix catalog/registry.go Build() corruption + nondeterministic maps~~ — DONE
- [x] ~~Fix catalog/docserver XSS~~ — DONE (html/template)
- [x] ~~Fix watermill double-close panic~~ — DONE (sync.Once)
- [x] ~~Fix watermill MustParse\* panic on untrusted input~~ — DONE (Parse\* + error return)
- [x] ~~Fix memory/checkpoint missing CheckClosed~~ — DONE
- [x] ~~Fix signing middleware nil signer/verifier panic~~ — DONE
- [x] ~~Fix cmd/api-stability wrong module paths~~ — DONE
- [x] ~~Fix cmd/cqrs-gen query template~~ — DONE (`Register%sHandler[R any]`)
- [x] ~~Fix example/todo stale references~~ — DONE
- [x] ~~Fix decider snapshot errors silently discarded~~ — DONE (slog.WarnContext)
- [x] ~~Fix ErrRetryCanceled dead sentinel~~ — DONE (properly wraps on context cancellation)
- [x] ~~Fix pebble metadata deserialization swallowed error~~ — DONE (`fe1e5184`)
- [x] ~~Fix Version.Cmp manual comparison~~ — DONE (stdlib `cmp.Compare`)
- [x] ~~Fix pebble nil store panic~~ — DONE (clear panic with message)

### API Surface

- [x] ~~Add IdempotencyKey to Command interface~~ — DONE
- [x] ~~Add ClientID branded type and WithClientID option~~ — DONE
- [x] ~~Add IsReplay getter~~ — DONE (`IsReplay(ctx) bool`)
- [x] ~~Rename WithNewCodec → WithCodec~~ — DONE (`27cf2f2a`)
- [x] ~~Rename ErrNilBus → ErrNilPublisher~~ — DONE (`b3b6801a`)
- [x] ~~Remove StreamKey free function~~ — DONE (`4b183a5c`)
- [x] ~~Remove Map/ScanState/Tap reactive wrappers~~ — DONE (`38f336f5`)
- [x] ~~Consolidate listRefsFromStatus~~ — DONE (`c77a4b05`)
- [x] ~~Fix AggregateProjection hardcoded placeholders~~ — DONE (`cab48302`, uses Dialect.Placeholder)
- [x] ~~Fix README.md broken badges~~ — DONE
- [x] ~~Fix otel.TraceIDLogger → ComponentLogger~~ — DONE (renamed)
- [x] ~~Fix event/reactive.go FilterEventTypes duplication~~ — DONE (uses `newTypeSet`)
- [x] ~~Fix catalog/types.go GetID → Key~~ — DONE
- [x] ~~Fix catalog/writer_frontmatter.go writeIDListField clone~~ — DONE
- [x] ~~Fix Version.Sub negative version panic~~ — DONE (panics on underflow)
- [x] ~~Fix pebble ErrUnknownBackend dead code~~ — DONE (removed)
- [x] ~~Fix listing TombstoneInclude unreachable dead code~~ — DONE (`panic("unreachable")`)
- [x] ~~Fix middleware circuit breaker return nil after exhaustive switch~~ — DONE
- [x] ~~Fix query TypedHandler takes Query not T~~ — DONE (`TypedHandler[Q Query, R any]`)
- [x] ~~Fix event/event_new.go no clone of []byte/json.RawMessage~~ — DONE
- [x] ~~Fix dispatcher CatalogDispatcher race~~ — DONE (mutex on RegisterHandlerMeta)
- [x] ~~Fix decider load.go unclassified fmt.Errorf~~ — DONE (uses `event.WrapInfrastructure`)
- [x] ~~Fix middleware circuit breaker double-wrapped error~~ — DONE
- [x] ~~Fix catalog/ToAny silently swallows marshal errors~~ — DONE (returns `(any, error)`)
- [x] ~~Fix signing/HasSignature swallows corruption errors~~ — DONE
- [x] ~~Fix watermill silently drops malformed ID parse errors~~ — DONE
- [x] ~~Fix event/batch.go vs event_new.go marshal duplication~~ — DONE (`NewEvents` calls `New`)
- [x] ~~Fix schema versioned_source.go 4 near-identical load methods~~ — DONE (extracted `loadAndUpcast`)

### Middleware Consolidation

- [x] ~~middleware/ 3× duplication across command/event/query~~ — DONE (generic `NewX[M]` + 27 thin wrappers)
- [x] ~~Add WithLogger to all middleware constructors~~ — DONE
- [x] ~~Add circuit breaker middleware~~ — DONE (half-open state machine)
- [x] ~~Add OpenTelemetry tracing middleware~~ — DONE
- [x] ~~Add distributed tracing middleware~~ — DONE
- [x] ~~Add SSE broker middleware~~ — DONE
- [x] ~~Add health check middleware~~ — DONE

### Documentation

- [x] ~~Create CONTRIBUTING.md~~ — DONE
- [x] ~~Create CONTEXT.md~~ — DONE
- [x] ~~Create docs/adr/~~ — DONE (15 ADRs)
- [x] ~~Write getting-started README section~~ — DONE
- [x] ~~Write API migration guide~~ — DONE (`docs/MIGRATION.md`)
- [x] ~~Add storage backend guide~~ — DONE
- [x] ~~Add module READMEs~~ — DONE (all 27 modules)
- [x] ~~Archive stale planning docs~~ — DONE
- [x] ~~Write CHANGELOG.md~~ — DONE
- [x] ~~Prune docs/status/~~ — DONE

### CI & Tooling

- [x] ~~Add CI pipeline~~ — DONE (`ci.yml`)
- [x] ~~Add minimum coverage gate to CI (80%)~~ — DONE
- [x] ~~Add -race to CI~~ — DONE
- [x] ~~Add coverage tracking to CI~~ — DONE
- [x] ~~Parallelize CI matrix~~ — DONE (22 modules in parallel)
- [x] ~~Add go.work sync CI check~~ — DONE
- [x] ~~Add gosec security scanning~~ — DONE
- [x] ~~Performance regression CI~~ — DONE (benchmark baseline comparison)
- [x] ~~Add per-module `go vet`~~ — VERIFIED (covered by lint step)
- [x] ~~Add gofumpt/goimports to pre-commit~~ — DONE (flake.nix treefmt)
- [x] ~~Add .goreleaser.yml~~ — DONE
- [x] ~~Standardize integration/go.mod + catalog/go.mod + example/go.mod versions~~ — DONE
- [x] ~~Add GOWORK=off CI verification job~~ — DONE
- [x] ~~Add file-size gate to CI (≤350 lines)~~ — DONE

### Testing

- [x] ~~Add BDD tests for Version, SchemaVersion, Pagination~~ — DONE
- [x] ~~Add fuzz tests for event creation, ID parsing, schema reflection~~ — DONE
- [x] ~~Add E2E throughput benchmarks~~ — DONE (17 scale benchmarks 10K-1M)
- [x] ~~Add tombstone status tests~~ — DONE
- [x] ~~Add listing SQL reader tests~~ — DONE (7 tests)
- [x] ~~Add event/slice.go tests~~ — DONE (8 tests)
- [x] ~~Add event/context.go tests~~ — DONE (6 tests)
- [x] ~~Add dispatcher tests~~ — DONE (catalog + concurrent tests)
- [x] ~~Add watermill tests~~ — DONE (subscriber tests)
- [x] ~~Add rapid PBT on command/ and query/~~ — DONE (9 tests)
- [x] ~~Add catalog adapters coverage~~ — DONE (100.0%)
- [x] ~~Add testhelpers coverage~~ — DONE (94.8%)
- [x] ~~Add projection coverage to 95%+~~ — DONE (95.3%)
- [x] ~~Add decider coverage to 100%~~ — DONE
- [x] ~~Add catalog/openapi coverage~~ — DONE (94.4%)
- [x] ~~Add catalog/docserver coverage~~ — DONE (90.1%)
- [x] ~~Add memory closed-store behavior tests~~ — DONE
- [x] ~~Add eventtest/fake_store tests~~ — DONE (18 test functions, 342 lines)
- [x] ~~Add cqrs-gen tests~~ — DONE (15 test functions)
- [x] ~~Add example/user smoke test~~ — DONE (main_test.go + smoke_test.go)
- [x] ~~Add integration full_flow_test~~ — DONE
- [x] ~~Add integration chaos_test~~ — DONE
- [x] ~~Add signing cross-module integration test~~ — DONE
- [x] ~~Add encryption integration test~~ — DONE
- [x] ~~Add OTel integration test~~ — DONE

### Core Features

- [x] ~~Add publish-side event middleware~~ — DONE (`event.PublishMiddleware` + `Bus.UsePublish`)
- [x] ~~Add Store.LoadToVersion to interface~~ — DONE
- [x] ~~Add SeekableJournal interface~~ — DONE
- [x] ~~Add Store.LoadToTimestamp to interface~~ — DONE
- [x] ~~Add timestamp index to SQL DDL~~ — DONE
- [x] ~~Auto-detect SeekableJournal in projection Runner~~ — DONE
- [x] ~~Implement Store.ReadBackwards~~ — DONE (`event.BackwardsSource`)
- [x] ~~Implement SQL-backed SnapshotStore~~ — DONE
- [x] ~~Implement SQL-backed CheckpointStore~~ — DONE
- [x] ~~Add SQL-backed CommandStore~~ — DONE
- [x] ~~Add Pebble EventStore with CBOR~~ — DONE
- [x] ~~Add Pebble Journal/SeekableJournal~~ — DONE (journal.go with ReadAll/ReadFrom)
- [x] ~~Add Turso connector~~ — DONE
- [x] ~~Add Turso sync (Push/Pull/Checkpoint)~~ — DONE
- [x] ~~Add command.TypedHandler + command.RegisterTyped~~ — DONE
- [x] ~~Add query.TypedHandler + query.RegisterTyped~~ — DONE
- [x] ~~Add event.Event.Clone()~~ — DONE
- [x] ~~Add Clock injection option WithClock~~ — DONE
- [x] ~~Add ContextEnricher wiring to decider~~ — DONE
- [x] ~~Add Upcaster interface + UpcasterRegistry~~ — DONE
- [x] ~~Add UpcasterRegistry cycle detection~~ — DONE
- [x] ~~Add projection parallel processing~~ — DONE (WithParallelism)
- [x] ~~Add projection rebuild/reset API~~ — DONE (Runner.Reset)
- [x] ~~Add dead letter queue to projection runner~~ — DONE (WithDeadLetterHandler)
- [x] ~~Add OpenAPI/Swagger exporter~~ — DONE (`catalog/openapi/`)
- [x] ~~Generate llms.txt alongside EventCatalog output~~ — DONE
- [x] ~~Add event signing/verification~~ — DONE (HMAC + Ed25519 + multisig)
- [x] ~~Add event encryption~~ — DONE (XChaCha20-Poly1305 + AES-256-GCM)
- [x] ~~Add Codec module (JSON, CBOR, Raw)~~ — DONE
- [x] ~~Add listing module~~ — DONE (AggregateReader, ListBuilder, StatusMiddleware)
- [x] ~~Add otel module~~ — DONE (Tracer, Meter, Span helpers)
- [x] ~~Add watermill adapter~~ — DONE (PublisherAdapter, SubscriberAdapter)
- [x] ~~Add config loader~~ — DONE (`pkg/config/`)
- [x] ~~Add graceful shutdown helper~~ — DONE (`pkg/gracefulshutdown/`)

---

## 🟡 MEDIUM Priority

### Code Quality

- [ ] **ADR numbering gap** — ADR-0005 is missing; README.md only lists ADR-0001 through 0003. Need to either add ADR-0005 or renumber 0006+ to fill the gap, and update README.md to list all 15 ADRs
- [ ] **cmd/api-stability zero tests** — API surface checker has no test coverage. The tool guards breaking changes but is itself untested
- [ ] **event.ImmutableEvent.Clone shares opts pointer** — `opts *eventOptions` pointer is shared between original and clone. Currently safe (all fields immutable) but fragile if opts gains mutable fields
- [ ] **query.BasicQuery has no metadata** — Unlike `BasicCommand`, queries carry no correlation/tracing context. Makes distributed tracing through query path inconsistent with command/event
- [ ] **eventtest/ as separate module** — event/go.mod lists 5 test-only deps (command, query, memory, schema, snapshot) that bloat consumer transitive deps. Extracting eventtest to its own go.mod would clean this up
- [ ] **Clean test deps from 12 production go.mod files** — 12 modules have test-only deps in production require blocks. Go doesn't support separate test-only require blocks
- [ ] **Fix nolint:errcheck suppressions in defer .Close() calls** — 31 suppressions, many are lazy. Use explicit error handling or `defer func()` pattern
- [ ] **Verify all `//nolint` comments have justification** — Most suppressions lack explanation. Standardize `//nolint:linter // reason` format
- [ ] **Audit and reduce nolint suppressions** — 123 total across project. 31 `errcheck`, 25 `wrapcheck`, 23 `exhaustruct`
- [ ] **Reduce catalog/ nolint suppressions** — 36 total, worst package. Suggests design issues
- [x] ~~\*\*Fix catalog/registry_build.go deduplication~~ — DONE (generic `sortedCopy[K, V, S]`)
- [x] ~~\*\*Fix catalog/registry_copy.go deduplication~~ — DONE (generic `copyPtr[T]`)

### Documentation

- [ ] **Add godoc examples for decider package** — `Execute`, `Load`, `Repository` patterns have no runnable examples
- [ ] **Add godoc examples for projection package** — `Runner`, `Builder`, `On[T]()` patterns have no examples
- [ ] **Add godoc examples for signing package** — HMAC + Ed25519 setup, middleware configuration
- [ ] **Add godoc examples for schema package** — `Upcaster`, `VersionedStore` usage
- [ ] **Add listing/ package-level example** — `List`, `StatusMiddleware`, `InMemoryAggregateReader`
- [ ] **Add README section linking to docs/benchmarks/** — Consumers should know perf characteristics exist
- [ ] **Document CBOR usage patterns** — codec/README.md exists but lacks consumer-facing CBOR examples
- [x] ~~\*\*Document time-travel API~~ — DONE
- [x] ~~\*\*Document "state is disposable"~~ — DONE
- [x] ~~\*\*Document determinism rule~~ — DONE
- [x] ~~\*\*Document versioned event names convention~~ — DONE
- [x] ~~\*\*Document soft deletes over hard deletes~~ — DONE
- [x] ~~\*\*Document offline-first metadata conventions~~ — DONE

### Performance

- [ ] **Optimize listing/InMemoryAggregateReader** — Cache sorted result, avoid O(n log n) on every `List()` call. Potential 269x improvement (840ms → ~3ms for 10K aggregates)
- [ ] **Arena allocation experiment in event module** — High-throughput event creation could benefit from arena allocation
- [ ] **Zero-allocation event encoding path** — `jsonv2` experiment behind build tag
- [ ] **SIMD-accelerated event serialization** — Go experiment for large payload encoding
- [ ] **Benchmark MemoryStore with concurrent writers** — Stress-test for race conditions under load
- [ ] **Profile allocation patterns** — Compare JSON vs CBOR allocations with `go test -benchmem`

### Features (No Design Yet)

- [ ] **Outbox pattern design doc** — Reliable at-least-once event publishing. Mentioned in ROADMAP but no ADR or prototype
- [ ] **Schema registry design doc** — JSON Schema middleware for event validation. Mentioned in ROADMAP but no design exists
- [ ] **Event schema registry with validation middleware** — Validates event payloads against registered schemas at publish time
- [ ] **Distributed checkpointing for projections** — Multiple projection instances sharing checkpoint state

---

## 🟢 LOW Priority

### Polish

- [ ] **Remove unused `backend` field from Pebble store** — Dead state in production code, no runtime effect
- [ ] **Clean up pebble backward-compat aliases** — Check if any external consumers still use old names
- [ ] **Add `go-snaps` across remaining modules** — signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory (some already have golden tests)
- [ ] **Add ExampleCBORCodec** — Every codec should have a runnable example for pkg.go.dev
- [ ] **Fix CBOR cborEncMode error handling** — Silent `_, _` drop of potential init error (currently safe with fxamacker but sloppy)
- [ ] **Fix CBOR fuzz test** — Uses JSONCodec as intermediary decoder, should use pure CBOR seed corpus
- [ ] **Add CBOR DecMode configuration** — Match encode/decode expectations, enable strict mode
- [ ] **Evaluate CoreDetEncOptions vs CanonicalEncOptions** — Which is right default for signing safety?
- [x] ~~\*\*Fix codec/raw.go json.RawMessage support~~ — DONE

### v2 Breaking Changes (Deferred)

- [v2] **Remove io.Closer from core interfaces** — ADR-0010 accepted but not implemented. Breaking change for event.Store, snapshot.SnapshotStore, command.Store
- [v2] **Add global TransactionID branded type** — Cross-aggregate consistency tracking
- [v2] **Make event Core truly immutable** — Currently opts pointer is shared on Clone
- [v2] **Split event.Store into Writer/Reader/Deleter** — ADR-0010 direction
- [v2] **Move HTTP code out of middleware** — SSE, healthcheck, metrics_http → transport/ module
- [v2] **Fix query.Handler returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` instead of `(any, error)`

---

## ⚪ Unknown / Verified

- [x] ~~Return error from Pebble iterateEvents~~ — DONE
- [x] ~~Implement SQL dialect abstraction~~ — DONE
- [x] ~~Split pebble_event_store.go~~ — DONE
- [x] ~~Split storage/helpers.go~~ — DONE
- [x] ~~Add go-sqlmock tests for LoadToVersion, LoadToTimestamp~~ — DONE
- [x] ~~Storage coverage recovery~~ — DONE (90.2%)
- [x] ~~Parameterize OutboxStatusPending in SQL queries~~ — DONE
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
- [x] ~~Push release tags to remote~~ — DONE (v2.0.0+ tags pushed)
- [x] ~~Create core/pkg/errors/ package~~ — DONE
- [x] ~~Map all existing sentinel errors to error families~~ — DONE
- [x] ~~Register 6 remaining unclassified sentinels~~ — DONE
- [x] ~~Extract error classification to standalone package~~ — VERIFIED (go-error-family)
- [x] ~~Standardize storage error wrapping patterns~~ — DONE
- [x] ~~Replace init() error registration with explicit setup~~ — DONE
- [x] ~~Formally deprecate aggregate package~~ — DONE (deleted)
- [x] ~~Break event↔command cycle~~ — DONE (moved taxonomy test to integration/)
- [x] ~~Break memory↔snapshot cycle~~ — DONE (inline fakeStore in snapshot tests)
- [x] ~~Extract sql.QueryEngine~~ — DONE (`storage/sql/query_engine.go`)
- [x] ~~Remove command error re-exports~~ — DONE (removed unused WrapTransient)
- [x] ~~Fix Lifecycle exported field~~ — DONE (`5218640c`)
- [x] ~~Fix command.Metadata split-brain~~ — DONE (type alias `type Metadata = event.Metadata`)
- [x] ~~Fix VersionedStore embedded Store~~ — DONE (unexported `inner` field)
- [x] ~~Unify ErrDispatcherClosed across packages~~ — ACCEPTED: each module has unique error for independent importability. ADR-0011 accepted as-is
- [x] ~~Fix pebble/errors.go duplicate sentinels~~ — ACCEPTED: each module independently importable
- [x] ~~Fix command re-exports event types~~ — ACCEPTED: intentional boundary for consumer convenience
- [x] ~~Fix dispatcher local checkClosed helpers~~ — ACCEPTED: cross-module sharing would violate isolation
- [x] ~~Fix decider → memory dependency~~ — VERIFIED (test-only, standard Go behavior)
- [x] ~~Fix storage → listing coupling~~ — VERIFIED (correct dependency direction)
- [x] ~~Fix core → memory circular dependency~~ — VERIFIED (test-only, cosmetic)
- [x] ~~Fix middleware 3x duplication~~ — DONE (generic.go with domain-specific adapters)
- [x] ~~Fix event/ module cycles~~ — ACCEPTED (test-only deps)
- [x] ~~Extract shared opError helper~~ — MOOT (different wrapping patterns per module)
- [x] ~~Move example/todo to own repository~~ — BLOCKED (requires manual repo creation)
- [x] ~~Bump testhelpers to v1.2.0~~ — BLOCKED (requires tag push)
- [x] ~~Create go-branded-id v0.2.0~~ — BLOCKED (different repo)
- [x] ~~Design ActaFlow's event sourcing overlay~~ — BLOCKED (different project)
- [x] ~~Extract shared golangci.yml~~ — BLOCKED (different repo)
- [x] ~~Remove cockroachdb/errors from go-localsync~~ — BLOCKED (different repo)
- [x] ~~Change LICENSE~~ — BLOCKED (owner decision)

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
- [ ] Saga module (orchestrated multi-step transactions)
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
- [FUTURE] Set up pkg.go.dev documentation hosting
- [FUTURE] Add catalog diff/breaking-change detection tool
- [FUTURE] Add high-level test utilities (AggregateTester, ProjectionTester, BusTester)
- [FUTURE] Add distributed consensus capability (Raft/CRDT overlay)
- [FUTURE] Add time-series event query language for event store

---

## ❌ STALE / REMOVED / MOOT

Items that existed in previous TODO lists but were found to be incorrect or no longer applicable:

- ~~`MustNew` panic helper~~ — DOES NOT EXIST. Only test-local `mustNewCmd` helper. FEATURES.md was wrong.
- ~~`CatalogDispatcher` embedded in command/query dispatchers~~ — DOES NOT EXIST. No such type in codebase.
- ~~`BatchProjection` optional interface~~ — DOES NOT EXIST. No such type in codebase.
- ~~`Reactive CommandBus/QueryBus`~~ — DOES NOT EXIST. `ro.Subject[Command]` / `ro.Subject[Query]` mentioned in AGENTS.md but never implemented.
- ~~`Outbox pattern`~~ — REMOVED from library scope. Outbox lives in example/ only (SQLBackend + polling). Not a reusable module.
- ~~`Saga module`~~ — DECLINED. Saga pattern demonstrated via example/todo (projection + command dispatch). No dedicated module needed.
- ~~`GraphQL query adapter`~~ — DECLINED. Framework-level concern, not library scope.
- ~~`Extract eventtest as separate module`~~ — ACCEPTED/WONTFIX. Would be breaking change for consumers. Test-dependency bloat is metadata-only.
- ~~`Consolidate testhelpers fake boilerplate via fakeBase struct`~~ — WONTFIX. Each fake has different fields/interfaces; shared base adds complexity.
- ~~`Convert DispatchTyped to method on *query.Dispatcher`~~ — NOT FEASIBLE. Go does not support generic methods on concrete types.
- ~~`Extract shared opError helper`~~ — MOOT. Different wrapping patterns per module.
- ~~`Add AggregateID as ULID type`~~ — MOOT. `DeriveAggregateID` uses SHA-256 for deterministic IDs, not ULID.
- ~~`Test container integration tests for storage (PostgreSQL)`~~ — BLOCKED. Requires Docker/testcontainers setup.

---

_Items verified against source on 2026-06-11. Total open items: 25 medium + 18 low + 33 planned/future = 76 open. Total done/verified: 180+._
