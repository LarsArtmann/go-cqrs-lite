# Status Report: ViewStore API Consolidation, Turso Views, gRPC Events, README Refresh

**Date:** 2026-06-24 13:27  
**Session Focus:** Top 7 tasks from the Tier 2 closeout list  
**Status:** All 7 tasks complete. All tests pass, all linters clean.

---

## A) FULLY DONE (7/7 tasks)

### #3 + #12 — Consolidate ViewStore Query API (RED → GREEN)

**What changed:** Eliminated the dual query API (raw SQL `Where`/`Args` vs structured `ViewFilter`/`FilteredQuerier`). Now there is ONE injection-safe query API.

| Before                                          | After                                                                             |
| ----------------------------------------------- | --------------------------------------------------------------------------------- |
| `ViewQuery{Where: "age >= ?", Args: []any{20}}` | `ViewQuery{Conditions: []kv.Condition{{Column: "age", Op: kv.OpGte, Value: 20}}}` |
| `store.QueryFiltered(ctx, filter, query)`       | `store.Query(ctx, query)` — conditions are in the query                           |
| `ViewFilter` type                               | **Removed**                                                                       |
| `FilteredQuerier` interface                     | **Removed**                                                                       |
| `toAnySlice` reflection helper                  | **Removed** — `Condition.Values []any` replaces it for `OpIn`                     |

**Files changed:**

- `kv/view_store.go` — Removed `ViewFilter`, `FilteredQuerier`; merged `Conditions` into `ViewQuery`; added `Condition.Values`
- `storage/view_store_count.go` — Rewrote: `Count` uses `buildWhereClause(q.Conditions, ...)`; removed `QueryFiltered`, `toAnySlice`
- `storage/view_store_query.go` — `Query` uses conditions; `QueryByTombstone` uses conditions; removed `FilteredQuerier` assertion
- 6 test files updated across `storage/` and `stack/sqlite/`
- `SKILL.md`, `AGENTS.md` — Updated all code examples

**Tests:** `go test ./storage/... ./kv/... ./stack/... -race` — all pass.

---

### #6 — Turso View Store Integration Test

**What changed:** `turso.NewViewStore` existed but had ZERO tests. Added 4 tests covering all view store capabilities via embedded LibSQL (no cloud DB needed).

**Files created:**

- `storage/turso/view_store_test.go` — 4 tests: CRUD, QueryConditions (eq/IN/range), Count+DeleteAll, QueryByTombstone

**Bug fixed:** Pre-existing missing `eventtest` dependency in `storage/turso/go.mod` (golden_test.go imported it but it wasn't in go.mod). Added `event/v3/eventtest` require + replace directive.

**Tests:** `cd storage/turso && GOWORK=off go test ./... -race` — all pass (13 tests total).

---

### #7 — Doc-Check CI Gate (Already Existed)

**Finding:** The CI step already exists at `.github/workflows/ci.yml:38-39`. Verified it passes: 445 references valid across 25 packages.

---

### #8 — Fix doc-check Lint Warnings

**What changed:** Fixed `gosec G706` (log injection via taint analysis) on `log.Fatalf` call.

**Result:** `golangci-lint run ./cmd/doc-check/...` → **0 issues**.

---

### #9 — README.md Refresh

**What changed:**

- Added `transport/grpc`, `cmd/doc-check`, `stack/turso`, `example/deployer-first-multidb` to module tables
- Added "SQL-backed read models" bullet point to "What makes it different" section
- Added "SQL-backed read models" row to comparison table
- Updated module count 38 → 42+
- Updated presets list: 4 → 5 (added Turso)
- Updated status line to mention SQL views, gRPC, SSE

---

### #10 — gRPC Event Pub/Sub (ADR-0025 Completion)

**What changed:** ADR-0025 specified events, but only commands+queries were implemented. Added server-streaming event subscription.

**Proto additions (`transport/grpc/proto/cqrs.proto`):**

```protobuf
message EventEnvelope { id, type, aggregate_id, aggregate_type, version, payload, occurred_at_unix_nano, metadata }
message SubscriptionRequest { repeated string event_types }
service EventService { rpc Subscribe(SubscriptionRequest) returns (stream EventEnvelope); }
```

**Go code (regenerated with protoc v34.1 + matching plugins):**

- `event_server.go` — `RegisterEventService(srv, bus)` subscribes to bus, fans out to connected gRPC clients via buffered channels (128-deep, non-blocking send on full). Client-side event-type filtering.
- `event_client.go` — `EventClient.Subscribe(ctx, handler, eventTypes...)` opens streaming RPC, reconstructs `event.Event` from envelope, delivers to handler.
- `event_version.go` — gosec-safe int64↔Version conversion helpers.
- `event_test.go` — 2 tests: round-trip streaming (2 events delivered in order), type filtering (only matching events delivered).

**Pattern:** Mirrors the SSE broker pattern (`transport/http/sse.go`) — subscribe to all events, fan-out via buffered channels, drop on slow consumer.

**Tests:** `cd transport/grpc && GOWORK=off go test ./... -race` — 5 tests pass (3 command + 2 event). 0 lint issues.

---

### #11 — stack/turso SQLViewModel + Integration Test

**What changed:** `stack/turso/` was missing `SQLViewModel[V,K]` (which sqlite and postgres both have). Added it + wired `WithDatabase` into both bundle constructors.

**Files created:**

- `stack/turso/view_models.go` — `SQLViewModel[V,K](bundle, mapper)` constructor (mirrors sqlite/postgres pattern)
- `stack/turso/view_models_integration_test.go` — End-to-end test: Turso bundle → SQLViewModel → Materialize handler → SQL table → View/Query/Count

**Bug fixed:** `stack/turso/preset.go` wasn't calling `stack.WithDatabase(sqlDB)` — both `newLocalBundle` and `newSyncBundle` now pass the DB handle so `SQLViewModel` can access it.

**Tests:** `cd stack/turso && GOWORK=off go test ./... -race` — all pass.

---

## B) PARTIALLY DONE

### Nothing in this session.

---

## C) NOT STARTED (from the original Top 25)

| #   | Task                               | Effort | Why                                                                                           |
| --- | ---------------------------------- | ------ | --------------------------------------------------------------------------------------------- |
| 1   | Add 8 missing modules to CI matrix | 30min  | `event/eventtest`, 5 examples, `prometheus`, `stack/turso` — not in per-module CI test matrix |
| 2   | Fix BuildFlow pre-commit timeout   | 30min  | 60s budget, golangci-lint on 44 modules exceeds it. Every commit needs `--no-verify`          |
| 4   | Add Postgres testcontainer to CI   | 2h     | stack/postgres has 0% local coverage (skips without `POSTGRES_TEST_DSN`)                      |
| 5   | Fix gopls infertypeargs warnings   | 20min  | 9 unnecessary type arguments in `stack/accessors.go`, `example/deployer-first/main.go`        |

---

## D) TOTALLY FUCKED UP

### `nix fmt` corrupts catalog YAML imports

**Problem:** Running `nix fmt` silently replaces `github.com/go-faster/yaml` (the REQUIRED library per AGENTS.md) with `go.yaml.in/yaml/v3` (the BANNED library) across `catalog/asyncapi/serde.go`, `catalog/openapi/serde.go`, `catalog/schema/yaml.go`, and updates `go.mod`/`go.sum` accordingly.

**Impact:** Every `nix fmt` run introduces a banned dependency into the catalog module. This was caught and reverted manually this session, but it WILL recur.

**Root cause:** A `gci` or `goimports` linter step in the nix formatter has a misconfigured import ordering rule that doesn't respect the project's depguard allow-list. It sees `go.yaml.in/yaml/v3` as "more canonical" than `go-faster/yaml` and rewrites the import.

**Fix needed:** Either (a) pin the formatter to not rewrite imports, or (b) add a depguard rule that rejects `go.yaml.in/yaml/v3`.

---

## E) WHAT WE SHOULD IMPROVE

1. **`nix fmt` import corruption** — Fix the formatter config so it stops swapping YAML libraries. This is a recurring foot-gun.

2. **`storage/turso/go.mod` had a broken dependency** — `golden_test.go` imported `eventtest` but the dependency was never added. CI per-module tests should have caught this (turso wasn't in the CI matrix). Adding the 8 missing modules to CI (Task #1 above) would prevent this class of bug.

3. **`stack/turso` was missing `WithDatabase`** — Both sqlite and postgres presets call `stack.WithDatabase(sqlDB)` but turso didn't. This is a contract violation — all SQL presets should expose their DB handle. Consider a shared test that asserts all SQL presets expose `Database()`.

4. **Event envelope reconstruction is lossy** — The `envelopeToEvent` function in `event_client.go` can only reconstruct events with correlation ID metadata. Causation, tombstone, and custom metadata are serialized server-side but not fully deserialized client-side. This is acceptable for v1 but should be documented as a limitation.

5. **transport/grpc still excluded from go.work** — The genproto version conflict (cockroachdb/errors pulls old monolithic genproto; grpc-go needs split genproto) remains. All gRPC tests require `GOWORK=off`. This is a permanent workaround unless cockroachdb/errors upgrades.

6. **gRPC event streaming has no backpressure** — The server drops events when a client's 128-deep buffer is full (non-blocking send). For slow consumers this means data loss. An alternative is to block, but that stalls all clients. Consider per-client overflow tracking + a `Subscribe(opts...)` with configurable drop policy.

---

## F) Top 25 Things to Get Done Next

| Priority | #   | Task                                                                                                               | Effort | Impact                                                  |
| -------- | --- | ------------------------------------------------------------------------------------------------------------------ | ------ | ------------------------------------------------------- |
| 🔴       | 1   | **Fix `nix fmt` YAML import corruption** — stop it from swapping go-faster/yaml → go.yaml.in/yaml/v3               | 30min  | Prevents recurring banned-dep introduction              |
| 🔴       | 2   | **Add 8 missing modules to CI per-module matrix** — `event/eventtest`, 5 examples, `prometheus`, `stack/turso`     | 30min  | Catches missing-dep bugs like the turso eventtest issue |
| 🔴       | 3   | **Fix BuildFlow pre-commit hook timeout** — increase to 300s or exclude transport/grpc                             | 30min  | Every commit currently needs `--no-verify`              |
| 🟡       | 4   | **Add Postgres testcontainer to CI** — stack/postgres has 0% local coverage                                        | 2h     | Real coverage for the distributed preset                |
| 🟡       | 5   | **Fix gopls infertypeargs warnings** — 9 unnecessary type args in `stack/accessors.go`, `example/deployer-first`   | 20min  | Code quality                                            |
| 🟡       | 6   | **Full event metadata round-trip in gRPC** — reconstruct causation, tombstone, custom metadata client-side         | 1h     | Completeness of event pub/sub                           |
| 🟡       | 7   | **gRPC event backpressure policy** — configurable drop vs block, overflow counter                                  | 2h     | Production readiness                                    |
| 🟡       | 8   | **Shared preset contract test** — assert all SQL presets expose `Database()`, have `SQLViewModel`, same option set | 1h     | Prevents contract drift                                 |
| 🟡       | 9   | **ViewStore contract test suite for Turso** — run `viewstoretest.RunSuite` against Turso view store                | 30min  | Parity with storage/ tests                              |
| 🟡       | 10  | **Deprecate `NewViewStoreWithDialect`** — it has zero callers; `NewSQLiteViewStore` covers all use cases           | 15min  | Clean API surface                                       |
| 🟡       | 11  | **Add `WithEventDB` test for stack/turso** — multidb preset is untested                                            | 1h     | Coverage                                                |
| 🟡       | 12  | **gRPC bidirectional streaming for events** — allow client-side filter changes mid-stream                          | 3h     | Advanced subscription patterns                          |
| 🟡       | 13  | **Connection pool tuning docs for Turso** — document `ConfigurePool` behavior under load                           | 30min  | Operations                                              |
| 🟢       | 14  | **Benchmark gRPC event streaming throughput** — events/sec, latency p50/p99                                        | 1h     | Performance baseline                                    |
| 🟢       | 15  | **SSE + gRPC unified event delivery ADR** — document when to use which transport                                   | 30min  | Decision clarity                                        |
| 🟢       | 16  | **ViewStore `Explain` method** — return SQL query plan for debugging slow queries                                  | 2h     | Debuggability                                           |
| 🟢       | 17  | **Auto-generate view table DDL from struct tags** — skip manual `ViewMapper` for simple cases                      | 3h     | DX improvement                                          |
| 🟢       | 18  | **Retry middleware for gRPC clients** — exponential backoff on stream disconnect                                   | 1h     | Resilience                                              |
| 🟢       | 19  | **Add `OpNotIn` operator to ViewFilter conditions**                                                                | 15min  | Feature parity                                          |
| 🟢       | 20  | **Multi-tenant ViewStore** — schema-per-tenant or row-level isolation                                              | 4h     | SaaS use case                                           |
| 🟢       | 21  | **Projection checkpoint integration with gRPC events** — resume from last event ID                                 | 2h     | Reliability                                             |
| 🟢       | 22  | **ViewStore migration tooling** — add/drop columns without data loss                                               | 4h     | Schema evolution                                        |
| 🟢       | 23  | **OTel tracing for gRPC event streaming** — span per event, correlation propagation                                | 1h     | Observability                                           |
| 🟢       | 24  | **Integration test: gRPC events → Materialize projection** — end-to-end remote projection                          | 2h     | E2E coverage                                            |
| 🟢       | 25  | **Document `Condition.Values` vs `Condition.Value`** — make it obvious which to use for which operator             | 15min  | API clarity                                             |

---

## G) Top #1 Question

**Should `transport/grpc` be permanently excluded from `go.work`, or should we invest in fixing the genproto conflict?**

The conflict: `cockroachdb/errors` (used by `event/v3`) pulls the old monolithic `google.golang.org/genproto` package, while `google.golang.org/grpc` requires the split genproto packages (`genproto/googleapis/api`, `genproto/googleapis/rpc`). Having both causes "ambiguous import" errors in workspace mode.

Options:

1. **Permanent exclusion** — Accept `GOWORK=off` for all transport/grpc operations. CI tests it in isolation. Cost: can't import transport/grpc from other workspace modules.
2. **Upgrade cockroachdb/errors** — Check if a newer version dropped the old genproto dependency. Cost: potential breaking changes in error wrapping behavior.
3. **Replace cockroachdb/errors** — Switch to `cosiner/terrors` or plain `fmt.Errorf` + sentinel errors. Cost: large refactor across event/, command/, query/, storage/, etc.

This decision affects whether gRPC can ever be a first-class workspace citizen or remains a permanently-isolated module.

---

## Verification Summary

| Check                | Command                                                   | Result                                   |
| -------------------- | --------------------------------------------------------- | ---------------------------------------- |
| Workspace build      | `go build ./...`                                          | ✅ Pass                                  |
| Storage tests        | `go test ./storage/... ./kv/... -race`                    | ✅ Pass                                  |
| Stack tests          | `go test ./stack/... ./stack/sqlite/... -race`            | ✅ Pass                                  |
| Turso storage tests  | `cd storage/turso && GOWORK=off go test ./... -race`      | ✅ Pass                                  |
| Turso stack tests    | `cd stack/turso && GOWORK=off go test ./... -race`        | ✅ Pass                                  |
| gRPC tests           | `cd transport/grpc && GOWORK=off go test ./... -race`     | ✅ 5/5 pass                              |
| gRPC lint            | `cd transport/grpc && GOWORK=off golangci-lint run ./...` | ✅ 0 issues                              |
| doc-check lint       | `golangci-lint run ./cmd/doc-check/...`                   | ✅ 0 issues                              |
| doc-check validation | `go run ./cmd/doc-check/ SKILL.md AGENTS.md`              | ✅ 445 refs valid                        |
| nix fmt              | `nix fmt`                                                 | ✅ Applied (catalog corruption reverted) |
