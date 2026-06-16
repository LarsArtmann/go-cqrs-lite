# branching-flow phantom — Full Triage Report

**Date:** 2026-06-16
**Tool:** `branching-flow phantom . --format markdown`
**Violations Found:** 264 (118 critical · 83 high · 13 medium · 50 low)
**Actionable Code Changes:** 0
**Verdict:** All 264 findings are correctly triaged as **non-actionable**. The codebase already captured the high-value phantom types in prior sprints. Every remaining finding falls into an architecturally-justified rejection category verified against source code.

---

## Methodology

1. Ran `branching-flow phantom .` — generated 264 findings.
2. Read the **entire** markdown report (all 4 severity sections).
3. Aggregated findings by file, primitive type (string/int/bool/uint), and kind (fn param / struct field).
4. Read the **actual source** for every borderline cluster to verify the architectural justification.
5. Categorized each finding into one of 10 verdict categories (below).
6. Cross-referenced against prior analysis (`docs/status/2026-06-11_03-57_SUPERB-TYPES-SPRINT-FINAL-STATUS.md`, `docs/quality/2026-06-16_BRANCHING_FLOW_REVIEW.md`) to confirm conclusions still hold.

---

## Verdict Categories

| # | Category | Count | Verdict | Why |
|---|----------|-------|---------|-----|
| A | Spec-mirroring serialization structs | ~35 | **Reject** | Mirror external JSON/YAML specs (AsyncAPI 2.x, OpenAPI 3.x, JSON Schema). Field types must be `string`/`bool` to serialize correctly via `encoding/json`. |
| B | Builder/option boundary functions | ~65 | **Reject** | Documented pattern: public API accepts `string`, converts to phantom type internally. Catalog already defines 17 domain phantom types in `types_phantom.go`. |
| C | SQL/storage plumbing | ~35 | **Reject** | Table names, WHERE fragments, SQL templates, column lists flow into `database/sql`. Must be plain `string`. `limit int` is part of cross-module interface signatures. |
| D | OTel API constraint | 8 | **Reject** | Params feed directly into `go.opentelemetry.io/otel/attribute` which takes `string`. Values originate from phantom-typed upstream types (`event.Type`, etc.). |
| E | Watermill API constraint | 4 | **Reject** | Watermill's `Publisher.Publish(topic string)` / `Subscriber.Subscribe(topic string)` dictate `string`. |
| F | Turso SQL analysis internals | ~28 | **Reject** | `query`/`table`/`sqlDDL` flow into `EXPLAIN QUERY PLAN` SQL statements. Internal-only. |
| G | Example/demo code | ~24 | **Reject** | `example/` demonstrates the library, not itself. Not the product. |
| H | Public API — breaking change | ~12 | **Reject** | Changing field types on released v2.3.0 structs (e.g. `query.Pagination.Page uint`) would break consumers. Cannot change without major version bump. |
| J | Test helpers | 10 | **Reject** | `cattest` builders mirror the catalog builder pattern. `limit`/`update`/`expected` are standard test API params. |
| K | Internal single-concept params | ~36 | **Reject** | Unexported functions with a single string/int param. Zero mixing risk → phantom types add noise. |

---

## Detailed Analysis Per Category

### A. Spec-Mirroring Serialization Structs (~35 findings)

**Files:** `catalog/asyncapi/types.go` (15), `catalog/openapi/types.go` (10), `catalog/schema/types.go` (3), `catalog/d2/exporter.go` (5 fields), `catalog/types.go` (1)

**Verified against source:** `catalog/asyncapi/types.go` — every field carries `json:"..."` tags matching the AsyncAPI 2.x spec exactly:

```go
type Document struct {
    AsyncAPI           string `json:"asyncapi"                     yaml:"asyncapi"`
    DefaultContentType string `json:"defaultContentType,omitempty" yaml:"defaultContentType,omitempty"`
    // ...
}
type Server struct {
    Host            string `json:"host"                      yaml:"host"`
    Protocol        string `json:"protocol"                  yaml:"protocol"`
    ProtocolVersion string `json:"protocolVersion,omitempty" yaml:"protocolVersion,omitempty"`
}
```

Similarly, `catalog/openapi/types.go` mirrors OpenAPI 3.x spec fields (`OpenAPI`, `OperationID`, `In`, `Ref`). These structs are populated by reflection-driven builders and serialized to JSON/YAML matching the spec. Phantom types would require unwrapping at every serialization point with zero bug-prevention value.

The `bool` fields here (`Deprecated`, `Required`, `Nullable`) are spec-mandated boolean JSON values — converting to an enum would break spec-compliant serialization.

### B. Builder/Option Boundary Functions (~65 findings)

**Files:** `catalog/build.go`, `catalog/channel_config.go`, `catalog/message_config.go`, `catalog/service_config.go`, `catalog/registry.go`, `catalog/asyncapi/builder.go`, `catalog/asyncapi/exporter.go`, `catalog/openapi/exporter.go`, `catalog/docserver/*`, `catalog/eventcatalog/*`, `catalog/internal/cattest/builders.go` (14), `catalog/d2/services.go`, `catalog/types_helpers.go`, `catalog/types_resources.go`

The catalog module defines **17 domain phantom types** in `types_phantom.go` (`Name`, `Version`, `Summary`, `Title`, `Description`, `Address`, `Protocol`, `Host`, `Email`, `URL`, `ContentType`, `DeliveryGuarantee`, `Method`, `Icon`, `Color`, `Language`, `Role`). The builder/option functions deliberately accept `string` at the public boundary and convert internally:

```go
func WithSummary(summary string) MessageOption { /* converts to catalog.Summary internally */ }
func NewRegistry(title string) *Registry       { /* converts to catalog.Title internally */ }
```

This is the **documented and deliberate pattern** — it provides ergonomic public API while the internal model uses phantom types. The `cattest/builders.go` helpers mirror this same pattern for tests. Changing the public signature to require phantom types would be a **breaking API change** that forces consumers to import catalog types for simple string values.

The `catalog/types_resources.go` structs (`DataStore`, `FlowEdge`) and `catalog/types_helpers.go` structs (`Badge`, `ChannelParam`) serialize to eventcatalog YAML frontmatter — plain strings are correct for the serialization boundary.

### C. SQL/Storage Plumbing (~35 findings)

**Files:** `storage/sql/query_engine.go` (9), `storage/sql/helpers.go` (6), `storage/sql/base.go` (4), `storage/sql/reconstruction.go` (2), `storage/sql_aggregate_reader.go` (2), `storage/event_store*.go` (5), `storage/command_store_journal.go` (2), `storage/query_store_load.go` (2), `storage/aggregate_projection.go` (1), `storage/sqlite_helpers.go` (1)

**Verified against source:** `storage/sql/query_engine.go`:

```go
type QueryConfig[T any] struct {
    Columns    string  // SQL column list: "id, type, payload, ..."
    Table      string  // SQL table name
    DomainNoun string  // for error messages
}
type LoadParams struct {
    Where      string  // SQL WHERE clause fragment
    ErrMsg     string  // error message template
    CountAttr  string  // SQL COUNT expression
}
```

These are SQL query construction primitives. `Table`, `Columns`, `Where` are interpolated into SQL strings and passed to `db.QueryContext`. They **must** be plain `string` — phantom types would require unwrapping at every `database/sql` boundary with no type-safety benefit (there's only one "SQL identifier" concept, not multiple distinguishable ones).

The `limit int` params are part of the `ReadFrom` / `ReadQueriesFrom` interface signatures shared across `memory/`, `storage/`, `pebble/`, `encryption/`, and `event/eventtest/`. Changing one implementation's signature would create inconsistency; changing the interface would be breaking.

The `ownDB bool` (storage/sql/base.go ×4) is a lifecycle ownership flag — already reviewed as "borderline-common" in `docs/quality/2026-06-16_BRANCHING_FLOW_REVIEW.md` (finding H3).

### D. OTel API Constraint (8 findings)

**Files:** `otel/attributes.go` (4), `otel/meter.go` (1), `otel/spans.go` (1), `otel/tracer.go` (1), `otel/types.go` (1)

These functions wrap `go.opentelemetry.io/otel/attribute`:

```go
func CommandAttrs(commandType string) []cqrsotel.KeyValue {
    return []cqrsotel.KeyValue{attribute.String("command.type", commandType), ...}
}
```

The `commandType`/`eventType`/`aggregateType` values **originate from phantom-typed upstream types** (`event.Type`, `event.AggregateType`) but are passed as `string` here because `attribute.String()` requires `string`. The OTel API is the constraint. Adding phantom types would require unwrapping immediately.

### E. Watermill API Constraint (4 findings)

**Files:** `watermill/protocol.go` (2), `watermill/publisher.go` (1), `watermill/subscriber.go` (1)

Watermill's `Publisher.Publish(topic string, ...)` and `Subscriber.Subscribe(ctx, topic string)` dictate `string`. The adapter cannot impose phantom types without breaking the Watermill interface contract.

### F. Turso SQL Analysis Internals (~28 findings)

**Files:** `turso/indexing/advisor.go` (12), `turso/indexing/index.go` (5), `turso/indexing/stats.go` (4), `turso/indexing/policy.go` (3), `turso/indexing/optimizations.go` (2), `turso/indexing/auto.go` (1), `turso/indexing/checkpoint.go` (1)

The turso indexing module runs `EXPLAIN QUERY PLAN` and analyzes SQLite statistics. `query` is a SQL statement passed to `db.Query`, `table` is used in `WHERE tbl_name = ?`, `sqlDDL` is parsed `CREATE INDEX` DDL. These flow directly into SQL — plain `string` is correct.

The `int` fields (`ID`, `Parent`, `RowEst`, `SizeBytes`, `pages`) are SQLite `sqlite_stat` table columns scanned from query results. Their types are dictated by the SQLite PRAGMA output shape.

The `bool` fields (`Unique`, `Partial`, `autoAnalyze`, `stopped`) are simple operational toggles with no mixing risk.

### G. Example/Demo Code (~24 findings)

**Files:** `example/todo/*` (15), `example/user/*` (7), `example/encryption/main.go` (1)

Example apps demonstrate how to **use** the library. They are not the product. Adding phantom types to `example/todo/domain/todo.go`'s `Title string` or `example/user/events.go`'s `Email string` would obscure the usage demo with type ceremony that distracts from the library patterns being shown. Per AGENTS.md: "example/ is a usage demo, not a deployment."

### H. Public API — Breaking Change (~12 findings)

**Files:** `query/pagination.go` (4), `middleware/circuit_breaker.go` (2), `middleware/middleware.go` (1), `middleware/healthcheck.go` (5)

**Verified against source:** `query/pagination.go`:

```go
type Pagination struct {
    Page     uint  // ← changing to type Page uint breaks consumers
    PageSize uint
}
```

This is a **released v2.3.0 public type**. Changing `Page uint` to `Page Page` (defined type) breaks every consumer that uses `p.Page` as a `uint` in arithmetic, comparisons, or function calls. The code **already applies type-level thinking**: it uses `uint` (not `int`) specifically to make negative values unrepresentable.

`middleware/circuit_breaker.go`'s `FailureThreshold int` / `SuccessThreshold int` could theoretically be swapped, but they're public config fields — changing their types is breaking. The struct-literal usage `CircuitBreakerConfig{FailureThreshold: 5, SuccessThreshold: 3}` already provides field-name disambiguation.

`middleware/healthcheck.go` fields (`Version`, `ComponentType`, `ObservedUnit`, `Time`) are part of the public `HealthCheckResponse` / `Check` types serialized as JSON health status — they mirror common health-check response conventions.

### J. Test Helpers (10 findings)

**Files:** `catalog/internal/cattest/catalog.go` (3), `catalog/internal/cattest/assertions.go` (2), `event/eventtest/store_suite.go` (3), `event/eventtest/fake_store.go` (1), `event/eventtest/golden.go` (1)

The `cattest` builders mirror the catalog builder pattern (accept `string`, convert internally). `AssertGolden(goldenPath, mismatchMsg string, update bool)` — `goldenPath` is a file path, `mismatchMsg` is a test failure message. These are test-internal APIs with no bug risk. The `update bool` is the standard golden-test update flag.

### K. Internal Single-Concept Parameters (~36 findings)

**Files:** `memory/store.go`, `memory/command_store.go`, `memory/store_load.go`, `memory/query_store.go`, `pebble/store.go`, `pebble/journal.go`, `pebble/helpers.go`, `middleware/recovery.go`, `middleware/metrics.go`, `middleware/metrics_otel.go`, `middleware/logging.go`, `middleware/tracing_logging.go`, `middleware/generic.go`, `middleware/retry.go`, `event/reconstruct.go`, `event/base64.go`, `event/metadata_json.go`, `event/tombstone.go`, `event/types.go`, `event/replay.go`, `encryption/envelope.go`, `encryption/store.go`, `projection/health.go`, `id/aggregate_id.go`, `dispatcher/lifecycle.go`, `listing/in_memory.go`, `kv/mem.go`

Every finding here is an **unexported function** or **internal struct field** with a single string/int/bool parameter where there is no possibility of mixing two semantically-different values of the same type:

- `logWithContext(ctx, msgType string, ...)` — one string param, one concept
- `recordMetrics(label string, ...)` — one string param
- `commitAndLog(logMsg string)` — one string param
- `streamKey` — one internal concept (map key)
- `journalPrefix` — one internal concept (KV key prefix)

Phantom types prevent bugs when **two values of the same primitive type could be swapped**. Single-concept params have zero swap risk. Adding phantom types here creates ceremony without safety.

The `bool` fields (`closed` in `dispatcher/lifecycle.go`, `kv/mem.go`; `replay` in `event/replay.go`) are simple lifecycle/processing-mode flags. Note: `event/replay.go`'s `replay bool` corresponds to `ProcessingMode` which **already has a proper enum** (`ModeLive`/`ModeReplay`) in the context propagation layer — the bool is just the internal storage.

---

## Delta From Prior Analysis (233 → 264)

The prior analysis (`docs/status/2026-06-11_03-57_SUPERB-TYPES-SPRINT-FINAL-STATUS.md`) recorded **233 violations**. The current run shows **264** (+31). The delta is attributable to **new code added since v2.3.0**, not regression:

| New area | Findings | Category |
|----------|----------|----------|
| `catalog/docserver/` (Scalar/AsyncAPI HTML server) | 11 | B (builder/config) |
| `catalog/types_resources.go` (DataStore, FlowEdge) | 6 | B (catalog domain → frontmatter) |
| `catalog/eventcatalog/writer_frontmatter.go`, `writer_llms.go` | 5 | B (builder) |
| `catalog/types_helpers.go` (Badge, ChannelParam) | 2 | B (serialization) |
| `kv/mem.go` (new KV module, uncommitted) | 1 | K (internal `closed` bool) |

All new findings fall into the same rejection categories. **No new actionable phantom types were introduced.**

---

## Conclusion

The codebase has already captured all high-value phantom types:
- **Domain identifiers:** `id.Of[T]` branded IDs, `event.Type`, `event.AggregateType`, `event.Version`, `event.SchemaVersion`
- **Catalog domain model:** 17 phantom types in `types_phantom.go`
- **Cross-cutting:** `middleware.SSEClientID`, `turso.DbPath`/`RemoteURL`/`AuthToken`, `signing.Actor`, `encryption.KeyID`/`Algorithm`

The remaining 264 findings are architecturally justified rejections across 10 categories. **Zero code changes are warranted.** Forcing phantom types into spec-mirroring structs, SQL plumbing, external API constraints, or released public APIs would degrade the library — adding ceremony without preventing real bugs, or breaking consumers.

**Recommendation:** Re-run this triage only when (a) new modules are added, or (b) a major version bump (v3) removes the public-API-breaking constraint. Until then, the 264 count is the expected steady-state.
