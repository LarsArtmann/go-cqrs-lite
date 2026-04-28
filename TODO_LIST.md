# TODO List

**Generated:** 2026-04-11
**Files Processed:** 125

## 🔴 HIGH Priority

- [ ] Fix Go toolchain cache corruption blocking test execution (source: 2026-04-02_21-22)
- [ ] Breaking: `Root.ID()` to return `id.AggregateID` instead of `string` (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Breaking: `Event.AggregateID()` to return `id.AggregateID` instead of `string` (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Breaking: Make `Command.AggregateID()` optional or remove from interface (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Fuzz tests for critical parsing functions (source: 2026-03-15_16-02)

## 🟡 MEDIUM Priority

- [ ] Fix `query.Handler` to accept `context.Context` parameter and convert type alias to type definition (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Fix `catalog/registry.go` Build() shared backing array corruption and non-deterministic map iteration (source: 2026-04-02_23-02)
- [x] Fix `Register()` error semantics to use `ErrDispatcherClosed` ✅ 2026-04
- [ ] Fix `time.Time` schema generation to use `{type:"string", format:"date-time"}` (source: 2026-04-12_00-57)
- [ ] Fix paralleltest warnings in `pkg/id/id_test.go` (source: 2026-03-30_20-34)
- [ ] Fix `nilnil` warning in `query_test.go` (source: 2026-03-30_20-34)
- [ ] Improve aggregate test coverage to 80%+ (source: 2026-03-15_16-02)
- [ ] Fix `.golangci.yaml` depguard configuration (source: 2026-04-08_19-45)
- [ ] Run `buildflow --semantic --fix` (source: 2026-04-08_19-45, 2026-04-11_20-54, ROADMAP.md)
- [ ] Update `.golangci.yml` (source: 2026-04-11_20-54, ROADMAP.md)
- [ ] Fix `varnamelen` warnings in production code (source: 2026-04-05_20-23)
- [ ] Fix stale LSP diagnostics (source: 2026-04-11_20-54)
- [ ] Fix `MemoryBus.Subscribe` nil handler check + error wrapping consistency (source: 2026-04-02_23-02)
- [ ] Fix asyncapi component message key collision when command/event share same ID (source: 2026-04-02_23-02)
- [x] Fix `MemoryStore.LoadFromVersion` to copy slice instead of returning sub-slice ✅ 2026-04
- [ ] Add error path tests for aggregate repo to improve coverage from 75% to 85%+ (source: 2026-04-08_19-45)
- [ ] Run `golangci-lint run` and `dupl` to find and fix code duplication (source: 2026-04-02_21-22)
- [ ] Improve `pkg/id` and `xtypes` test coverage to 80% (source: 2026-03-30_20-34)
- [ ] Re-run full benchmark suite and update results (source: 2026-04-11_20-54)
- [ ] Refactor test files to use helpers: `catalog/eventcatalog/exporter_test.go`, `catalog/asyncapi/exporter_test.go`, `internal/dispatcher/dispatcher_test.go`, `event/memory_bus_test.go`, `event/memory_store_test.go`, `aggregate/aggregate_test.go` (source: 2026-04-05_20-23)
- [ ] Update `example/user/` to use xtypes and middleware (source: 2026-04-08_19-45) — repository now uses aggregate.EventSourcedRepository
- [ ] Review and update CHANGELOG.md (source: 2026-04-11_20-54)
- [ ] Fix `TypedCommand` to implement `command.Command` interface (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Fix `Payload()`/`Metadata()` immutability by returning copies (source: 2026-04-02_23-02)
- [x] Create/update CONTRIBUTING.md ✅ 2026-04
- [ ] Create/update CODE_OF_CONDUCT.md (source: 2026-04-08_19-45, ROADMAP.md)
- [ ] Update README.md with test helper usage examples (source: 2026-04-05_20-23)
- [ ] Update AGENTS.md with auto-catalog pattern (source: 2026-04-10_12-14)
- [ ] Improve `catalog/yaml` coverage from 79.8% to 90%+ (source: 2026-04-02_22-58)
- [ ] Improve `catalog/eventcatalog` coverage from 86.4% to 90%+ (source: 2026-04-02_22-58)
- [ ] Refactor remaining test files: `catalog/registry_test.go`, `catalog/benchmark_test.go`, `catalog/integration_test.go`, `catalog/schema_test.go`, `event/event_test.go`, `command/command_test.go`, `query/query_test.go` (source: DEDUPLICATION_PLAN)

## 🟢 LOW Priority

- [ ] Document testing approach in AGENTS.md (source: 2026-04-08_19-45, 2026-04-11_20-54, ROADMAP.md)
- [ ] Consider Nix Flakes migration (source: 2026-04-11_20-54)

## ⚪ Unknown Priority

- [x] Add `aggregate.Repository` interface ✅ 2026-04
- [x] Remove dead exports: `query.Result[T]`, `event.StreamOptions`, `BatchSize`, `Streamer` ✅ 2026-04
- [x] Remove unused sentinels: `ErrCommandValidation`, `ErrQueryValidation`, `ErrInvalidEventType` ✅ 2026-04
- [ ] Commit `RecordEvent` rename (source: 2026-04-02_22-58)
- [x] Remove duplicated `Lifecycle` methods — MemoryBus/MemorySnapshotStore now use LifecycleMixin ✅ 2026-04
- [ ] Standardize errors: replace `fmt.Errorf` with `cockroachdb/errors` in `event/event.go`, `event/types.go`, `xtypes/event.go` (source: 2026-04-02_22-58)
- [x] Wire `example/user/aggregate.go` to use catalog-aware event constructors ✅ 2026-04
- [ ] Add `AggregateType` field to example/user event catalog metadata (source: 2026-04-12_00-57)
- [ ] Filter embedded `*Core`/`*CatalogCore` from schema output (source: 2026-04-10_12-14)
- [ ] Add tests for `EventRetry()` and `EventValidation()` middleware functions (source: 2026-04-08_19-45) — EventValidation tests added, EventRetry still needs tests
- [ ] Add tests for `QueryValidation()` middleware function (source: 2026-04-08_19-45)
- [ ] Split `TestTypedAggregate` into smaller functions (source: 2026-03-30_20-34)
- [ ] Add `t.Parallel()` to all subtests (source: 2026-03-30_20-34)
- [ ] Add Ginkgo + Gomega dependencies and create BDD test files: `command/command_bdd_test.go`, `query/query_bdd_test.go`, `event/event_bdd_test.go`, `integration_bdd_test.go` (source: BDD_TESTS_REVIEW)
- [x] Create `.github/workflows/test.yml` and `.github/workflows/lint.yml` CI workflows ✅ 2026-04
- [ ] Add pre-commit hook with `go test ./...` validation (source: 2026-04-08_19-45)
- [ ] Create justfile with `build`, `test`, `lint`, `fd` targets (source: 2026-04-02_21-22)
- [x] Create `example/user/` directory structure ✅ 2026-04
- [x] Create `example/main.go` working example ✅ 2026-04
- [ ] Add validation to `command.New()`, `query.New()`, `NewEvent()` constructors (source: 2026-04-02_23-02)
- [ ] Standardize errors globally: replace `fmt.Errorf` with `cockroachdb/errors` (source: 2026-04-02_23-02)
- [ ] Deduplicate validation in `EventBuilder.Build()` (source: 2026-04-02_22-58)
- [ ] Add `Dispatcher.CatalogEntries()`, `EventBus.CatalogEntries()`, and `QueryDispatcher.CatalogEntries()` for auto-discovery (source: 2026-04-10_12-14)
- [ ] Add `enum` and `default` struct tag support to Schema/Property (source: 2026-04-12_00-57)
- [ ] Add `Examples` field to AsyncAPI Message type and wire from catalog (source: 2026-04-12_00-57)
- [ ] Export `SchemaFromType` via generic adapter method (source: 2026-04-10_12-14)
- [x] Add `AddCommandFromType[T]()` generic helpers ✅ 2026-04
- [ ] Make asyncapi/eventcatalog exporter fields unexported (source: 2026-04-02_23-02)
- [ ] Implement catalog deduplication Phase 2+ (source: 2026-04-11_20-54)
- [ ] Unify addCommand/addEvent/addQuery into `addMessage(kind)` in `catalog/asyncapi/exporter.go` (source: DEDUPLICATION_PLAN)
- [ ] Unify MDX frontmatter writing in `catalog/eventcatalog/exporter.go` (source: DEDUPLICATION_PLAN)
- [ ] Deduplicate `catalog/yaml/yaml.go` by extracting `marshalValue()` (source: DEDUPLICATION_PLAN)
- [ ] Add `context.Context` to MemoryStore operations (source: 2026-04-02_23-02)
- [ ] Add PostgreSQL event store implementation (source: 2026-04-11_20-54)
- [ ] Add NATS/JetStream event bus (source: 2026-04-11_20-54)
- [x] Add `middleware/logging.go` and `middleware/recovery.go` implementations ✅ 2026-04
- [x] Create pre-built retry middleware with backoff ✅ 2026-04
- [ ] Add distributed tracing middleware (source: 2026-04-11_20-54)
- [ ] Implement circuit breaker middleware (source: 2026-04-11_20-54)
- [ ] Add tests for `Parse*` ID variants (ParseCausationID, ParseCorrelationID, etc.) (source: 2026-04-08_19-45)
- [ ] Add tests for nil-metadata branches and `WithMetadata` in event options (source: 2026-04-02_23-02)
- [ ] Add `LoadFromHistory` error path tests for aggregate and xtypes (source: 2026-04-02_23-02)
- [ ] Add full CQRS roundtrip integration test (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Add benchmarks for ID generation, dispatcher throughput, event store operations (source: 2026-04-02_22-58)
- [ ] Add integration tests for catalog adapters (source: 2026-04-11_20-54)
- [ ] Create performance benchmarks for catalog system (source: 2026-04-11_20-54)
- [ ] Table-drive `pkg/id/id_test.go` and `event/types_test.go` (source: 2026-04-05_20-23, DEDUPLICATION_PLAN)
- [ ] Extract `newTestEvent()` in `xtypes/xtypes_test.go` (source: 2026-04-05_20-23)
- [ ] Add `assertYAML()` helper to `catalog/yaml/yaml_test.go` (source: 2026-04-05_20-23)
- [ ] Extract `ensureService()` helper in `catalog/registry.go` (source: 2026-04-05_20-23)
- [ ] Unify Ptr-unwrapping in `catalog/schema.go` (source: 2026-04-05_20-23)
- [ ] Extract validation helper in `pkg/id/id.go` (source: 2026-04-05_20-23, DEDUPLICATION_PLAN)
- [ ] Unify Apply pattern in `example/user/aggregate.go` (source: 2026-04-05_20-23, DEDUPLICATION_PLAN)
- [ ] Extract common middleware test helpers to reduce command/query duplication (source: 2026-04-05_20-23)
- [ ] Create more comprehensive example (e-commerce domain) (source: 2026-04-11_20-54)
- [ ] Add metrics endpoint example (source: 2026-04-11_20-54)
- [x] Create `flake.nix` with build/test/lint apps ✅ 2026-04
- [ ] Archive old status reports (keep 3 most recent) (source: 2026-04-11_20-54)
- [x] Remove unused `Dispatcher.Dispatch` handler parameter ✅ 2026-04
- [ ] Remove redundant state in `TypedAggregate` (source: 2026-04-02_23-02, 2026-04-02_22-58)
- [ ] Snapshot support for aggregates (source: 2026-03-15_16-02, 2026-03-30_20-34)
- [ ] SQL/database event store implementation (source: 2026-04-08_19-45)
- [ ] Projection/read-model support (source: 2026-04-08_19-45)
- [ ] Saga/process manager support (source: 2026-04-08_19-45)
- [ ] Event upcasting/schema evolution (source: 2026-04-08_19-45)
- [ ] Dead letter queue for failed events (source: 2026-04-08_19-45)
- [ ] Health check endpoints (source: 2026-04-08_19-45)
- [ ] gRPC and HTTP transport adapters (source: 2026-03-30_20-34, 2026-04-11_20-54)
- [ ] Add `middleware/tracing.go` OpenTelemetry-compatible tracing middleware (source: 2026-04-02_22-58)
- [x] Add `middleware/metrics.go` metrics collection middleware ✅ 2026-04
- [ ] Add typed command dispatcher helper (source: 2026-04-02_22-58)
- [ ] Context propagation through middleware chain (source: 2026-04-08_19-45)
- [ ] OpenTelemetry integration (source: 2026-04-08_19-45, 2026-04-11_20-54)
- [ ] CLI tool for catalog generation (source: 2026-04-11_20-54)
- [ ] Web UI for browsing EventCatalog (source: 2026-04-11_20-54)
- [x] Generate `llms.txt` alongside EventCatalog output ✅ 2026-04
- [ ] YAML frontmatter improvements (versioned paths, owners) (source: 2026-04-10_12-14)
- [ ] Create architecture documentation (source: 2026-04-08_19-45, 2026-04-11_20-54, ROADMAP.md)
- [ ] Add GoDoc package examples for core packages (source: 2026-04-08_19-45, 2026-04-11_20-54, ROADMAP.md)
- [ ] Add version constants to all packages (source: 2026-04-02_22-58)
- [ ] Add semantic versioning tags and create release workflow (source: 2026-04-02_22-58)
- [ ] Add coverage tracking (codecov/coveralls) (source: 2026-04-08_19-45, ROADMAP.md)
- [ ] Add error assertion tests (source: 2026-04-08_19-45, ROADMAP.md)
- [ ] Benchmark tests for hot paths (source: 2026-03-15_16-02)
- [ ] Add Go report card badge to README (source: 2026-04-02_22-58)
- [ ] Add missing `t.Parallel()` to existing tests (source: BDD_TESTS_REVIEW)
- [ ] Add art-dupl to CI pipeline with threshold enforcement (source: 2026-04-05_20-23)
