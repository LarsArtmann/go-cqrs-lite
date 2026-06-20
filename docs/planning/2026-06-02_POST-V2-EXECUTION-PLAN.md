# V2.0.0 Post-Release Execution Plan

**Created:** 2026-06-02 14:00 CEST
**Source:** Session 10 deep audit + TODO_LIST.md (41 open items)
**Total tasks:** 67 (split from 54 logical items into ≤12 min atomic steps)
**Sorted by:** Impact → Effort → Customer Value

---

## Legend

| Column  | Meaning                                                             |
| ------- | ------------------------------------------------------------------- |
| **P**   | Priority: 🔴 CRITICAL / 🟠 HIGH / 🟡 MEDIUM / 🟢 LOW                |
| **I**   | Impact: 1=production-bug, 2=customer-facing, 3=quality, 4=polish    |
| **E**   | Effort: S=≤12min, M=12-30min, L=30-60min, XL=60+min                 |
| **CV**  | Customer Value: ★★★=must-have / ★★=nice-to-have / ★=internal        |
| **Src** | Source: BUG=audit finding, TODO=TODO_LIST, ARCH=architecture review |

---

## Phase 1: Critical Production Bugs (DO NOW)

| #   | Task                                                                           | P   | I   | E   | CV  | Src | File(s)                            |
| --- | ------------------------------------------------------------------------------ | --- | --- | --- | --- | --- | ---------------------------------- |
| 1   | Fix Runner.cancel data race: add sync.Mutex, protect cancel field writes/reads | 🔴  | 1   | S   | ★★★ | BUG | `projection/runner.go:106,230`     |
| 2   | Fix Runner.projections data race: mutex around Register() + snapshot in Run()  | 🔴  | 1   | S   | ★★★ | BUG | `projection/runner.go:85,99`       |
| 3   | Fix HealthCheck OOM: replace ReadAll with checkpoint-only check                | 🔴  | 1   | S   | ★★★ | BUG | `projection/health.go:20-22`       |
| 4   | Fix ReadFrom pagination: cursor-based ordering with id+occurred_at             | 🔴  | 1   | M   | ★★★ | BUG | `storage/event_store_global.go:66` |
| 5   | Add test for Runner concurrent Register+Run                                    | 🔴  | 1   | S   | ★★★ | BUG | `projection/runner_test.go` (new)  |
| 6   | Add test for Runner concurrent Run+Close                                       | 🔴  | 1   | S   | ★★★ | BUG | `projection/runner_test.go` (new)  |

## Phase 2: High-Priority Bugs (DO NEXT)

| #   | Task                                                              | P   | I   | E   | CV  | Src | File(s)                                 |
| --- | ----------------------------------------------------------------- | --- | --- | --- | --- | --- | --------------------------------------- |
| 7   | Fix PublisherAdapter drops context: thread through Publish method | 🟠  | 1   | S   | ★★★ | BUG | `watermill/publisher.go:24`             |
| 8   | Fix SQLAggregateReader `?` placeholders: use Dialect.Placeholder  | 🟠  | 1   | S   | ★★★ | BUG | `storage/sql_aggregate_reader.go:63-86` |
| 9   | Fix SubscriberAdapter.handlers data race: add sync.Mutex          | 🟠  | 1   | S   | ★★★ | BUG | `watermill/subscriber.go:58`            |
| 10  | Add Pebble EventStore Close() method: close DB + cleanup          | 🟠  | 1   | S   | ★★  | BUG | `pebble/store.go`                       |
| 11  | Fix LoadAtVersion snapshot: add version filter to SQL query       | 🟠  | 1   | S   | ★★  | BUG | `storage/snapshot.go:109`               |
| 12  | Fix createTable context.Background: accept ctx param              | 🟠  | 1   | S   | ★★  | BUG | `storage/aggregate_projection.go:100`   |
| 13  | Fix subscribeLive handler leak: unsubscribe on context cancel     | 🟠  | 1   | M   | ★★  | BUG | `projection/runner_live.go:19`          |

## Phase 3: Quality Fixes (Small & Safe)

| #   | Task                                                                    | P   | I   | E   | CV  | Src  | File(s)                               |
| --- | ----------------------------------------------------------------------- | --- | --- | --- | --- | ---- | ------------------------------------- |
| 14  | Fix SQLEventStore no closed state: add closed flag + CheckClosed        | 🟡  | 2   | S   | ★★  | BUG  | `storage/event_store.go:55-61`        |
| 15  | Fix decider opError: replace fmt.Errorf with event.Wrap\* taxonomy      | 🟡  | 3   | S   | ★   | TODO | `decider/load.go:56-64`               |
| 16  | Fix ToAny swallows errors: return error instead of synthetic fallback   | 🟡  | 2   | S   | ★★  | TODO | `catalog/schema/reflect.go:44-57`     |
| 17  | Fix HasSignature swallows errors: propagate corruption errors           | 🟡  | 2   | S   | ★★  | TODO | `signing/event.go:88`                 |
| 18  | Fix Version.Sub negative: add guard matching Decrement behavior         | 🟡  | 2   | S   | ★★  | TODO | `event/types.go:136`                  |
| 19  | Remove dead ErrUnknownBackend sentinel                                  | 🟡  | 3   | S   | ★   | TODO | `pebble/errors.go:12-15`              |
| 20  | Remove dead return nil after exhaustive switch                          | 🟡  | 3   | S   | ★   | TODO | `middleware/circuit_breaker.go:97-98` |
| 21  | Remove dead TombstoneInclude case in listing                            | 🟡  | 3   | S   | ★   | TODO | `listing/in_memory.go:124-147`        |
| 22  | Fix pebble backward-compat aliases: add deprecation comments            | 🟡  | 3   | S   | ★   | TODO | `pebble/config.go:59-69`              |
| 23  | Fix codec/raw.go: add json.RawMessage case to type switch               | 🟡  | 2   | S   | ★★  | TODO | `codec/raw.go:6,13`                   |
| 24  | Fix otel TraceIDLogger: rename or implement actual trace/span injection | 🟡  | 3   | S   | ★   | TODO | `otel/logging.go:16`                  |
| 25  | Fix GetID dishonest fallback: return empty string instead of Name       | 🟡  | 2   | S   | ★★  | TODO | `catalog/types.go:153`                |
| 26  | Remove dead command AggregateRef re-exports (ACCEPTED — close TODO)     | 🟡  | 3   | S   | ★   | TODO | `command/aggregate_ref.go`            |
| 27  | Remove dead event module cycles TODO (ACCEPTED — close TODO)            | 🟡  | 3   | S   | ★   | TODO | `event/` test files                   |
| 28  | Accept pebble vs sql duplicate sentinels (close TODO with note)         | 🟡  | 3   | S   | ★   | TODO | `pebble/errors.go`                    |

## Phase 4: Error Taxonomy Migration

| #   | Task                                                           | P   | I   | E   | CV  | Src  | File(s)                                      |
| --- | -------------------------------------------------------------- | --- | --- | --- | --- | ---- | -------------------------------------------- |
| 29  | Migrate schema/versioned_source.go fmt.Errorf → event.Wrap\*   | 🟡  | 3   | S   | ★   | ARCH | `schema/versioned_source.go`                 |
| 30  | Migrate watermill/protocol.go fmt.Errorf → event.Wrap\*        | 🟡  | 3   | S   | ★   | ARCH | `watermill/protocol.go`                      |
| 31  | Migrate storage/ fmt.Errorf → event.Wrap\* (6 files)           | 🟡  | 3   | M   | ★   | ARCH | `storage/*.go`                               |
| 32  | Migrate codec/ fmt.Errorf → event.Wrap\*                       | 🟡  | 3   | S   | ★   | ARCH | `codec/*.go`                                 |
| 33  | Migrate listing/ fmt.Errorf → event.Wrap\*                     | 🟡  | 3   | S   | ★   | ARCH | `listing/*.go`                               |
| 34  | Migrate projection/health.go fmt.Errorf → event.Wrap\*         | 🟡  | 3   | S   | ★   | ARCH | `projection/health.go`                       |
| 35  | Migrate id/ fmt.Errorf → event.Wrap\*                          | 🟡  | 3   | S   | ★   | ARCH | `id/id.go, aggregate_id.go`                  |
| 36  | Migrate command/dispatcher.go + query/dispatcher.go fmt.Errorf | 🟡  | 3   | S   | ★   | ARCH | `command/dispatcher.go, query/dispatcher.go` |

## Phase 5: Code Decomposition (Long Functions)

| #   | Task                                                                | P   | I   | E   | CV  | Src  | File(s)                                          |
| --- | ------------------------------------------------------------------- | --- | --- | --- | --- | ---- | ------------------------------------------------ |
| 37  | Decompose watermill messageToEvent (81L → 3 functions)              | 🟡  | 3   | M   | ★   | TODO | `watermill/protocol.go:79-160`                   |
| 38  | Decompose projection replay (64L → 2 functions)                     | 🟡  | 3   | S   | ★   | TODO | `projection/runner.go:119-183`                   |
| 39  | Decompose storage ListWithStatus (112L → 3 functions)               | 🟡  | 3   | M   | ★   | TODO | `storage/sql_aggregate_reader.go:47`             |
| 40  | Decompose catalog Export (63L → generic writeEntities[T])           | 🟡  | 3   | M   | ★   | TODO | `catalog/eventcatalog/exporter.go:28-91`         |
| 41  | Extract writeIDListField from cloned addObjectIDsListField          | 🟡  | 3   | S   | ★   | TODO | `catalog/eventcatalog/writer_frontmatter.go:63`  |
| 42  | Move NewTestCreateOrderFlow out of production code to test file     | 🟡  | 3   | S   | ★   | TODO | `catalog/registry_helpers.go:138-152`            |
| 43  | Deduplicate event/batch.go: make NewEvents call New                 | 🟡  | 3   | S   | ★   | TODO | `event/batch.go:40-68`                           |
| 44  | Deduplicate schema 4 load methods: extract loadWithUpcasters helper | 🟡  | 3   | M   | ★   | TODO | `schema/versioned_source.go:33-87`               |
| 45  | Unify signing extract→verify→next into VerifyFunc pattern           | 🟡  | 3   | M   | ★   | TODO | `signing/middleware.go + multisig/middleware.go` |
| 46  | Deduplicate event/reactive.go FilterEventTypes newTypeSet           | 🟡  | 3   | S   | ★   | TODO | `event/reactive.go`                              |
| 47  | Extract memory withRLock/withWLock helpers                          | 🟡  | 3   | S   | ★   | TODO | `memory/store.go, bus.go`                        |

## Phase 6: Library Modernization

| #   | Task                                                                     | P   | I   | E   | CV  | Src  | File(s)                                          |
| --- | ------------------------------------------------------------------------ | --- | --- | --- | --- | ---- | ------------------------------------------------ |
| 48  | Replace dispatchParallel manual semaphore with errgroup.SetLimit         | 🟢  | 3   | S   | ★   | ARCH | `projection/runner_live.go:54-77`                |
| 49  | Migrate filepath.Walk → filepath.WalkDir (cmd/cqrs-gen)                  | 🟢  | 3   | S   | ★   | QUAL | `cmd/cqrs-gen/main.go:118`                       |
| 50  | Migrate filepath.Walk → filepath.WalkDir (scripts/)                      | 🟢  | 3   | S   | ★   | QUAL | `scripts/go-mod-graph-local/main.go:89`          |
| 51  | Migrate event/eventtest store_suite.go []byte(fmt.Sprintf) → fmt.Appendf | 🟢  | 3   | S   | ★   | QUAL | `event/eventtest/store_suite.go:54`              |
| 52  | Modernize middleware middleware_bdd_test.go for range over int           | 🟢  | 3   | S   | ★   | QUAL | `middleware/middleware_bdd_test.go:151`          |
| 53  | Remove unnecessary type arguments (6 LSP hints in middleware/)           | 🟢  | 3   | S   | ★   | QUAL | `middleware/metrics_otel.go, tracing_logging.go` |
| 54  | Migrate cmd/api-stability from deprecated parser.ParseDir → go/packages  | 🟢  | 3   | S   | ★   | QUAL | `cmd/api-stability/main.go:126`                  |

## Phase 7: Test Coverage

| #   | Task                                                       | P   | I   | E   | CV  | Src  | File(s)                     |
| --- | ---------------------------------------------------------- | --- | --- | --- | --- | ---- | --------------------------- |
| 55  | Add listing SQL reader tests                               | 🟡  | 3   | M   | ★   | TODO | `listing/`                  |
| 56  | Add turso EventStore CRUD tests (28.6% → 50%+)             | 🟡  | 2   | M   | ★★  | TODO | `turso/`                    |
| 57  | Add turso error path tests (50% → 70%+)                    | 🟡  | 3   | M   | ★   | TODO | `turso/`                    |
| 58  | Add storage error path tests (72.7% → 80%+)                | 🟡  | 3   | L   | ★   | TODO | `storage/`                  |
| 59  | Increase projection coverage to 95%+                       | 🟡  | 3   | M   | ★   | TODO | `projection/`               |
| 60  | Add BDD tests for Version, SchemaVersion, Pagination types | 🟢  | 3   | M   | ★   | TODO | `event/, snapshot/, query/` |

## Phase 8: Architecture (Post-Stabilization)

| #   | Task                                                  | P   | I   | E   | CV  | Src  | File(s)                                 |
| --- | ----------------------------------------------------- | --- | --- | --- | --- | ---- | --------------------------------------- |
| 61  | Create shared message/ module with TypeName interface | 🟡  | 2   | M   | ★★  | ARCH | `message/` (new)                        |
| 62  | Unify metadata options: shared WithCorrelationID etc. | 🟡  | 2   | M   | ★★  | ARCH | `event/options.go, command/metadata.go` |
| 63  | Generic BrandedInt[T] for Version/SchemaVersion       | 🟢  | 2   | M   | ★   | ARCH | `event/types.go`                        |
| 64  | Query TypedHandler[T] returning (T, error)            | 🟠  | 2   | L   | ★★★ | ARCH | `query/query.go:54`                     |

## Phase 9: CI / DevEx (Background)

| #   | Task                                                | P   | I   | E   | CV  | Src  | File(s)                    |
| --- | --------------------------------------------------- | --- | --- | --- | --- | ---- | -------------------------- |
| 65  | Parallelize CI matrix: one job per module           | 🟢  | 3   | M   | ★   | TODO | `.github/workflows/ci.yml` |
| 66  | Add gofumpt/goimports to pre-commit hook            | 🟢  | 3   | S   | ★   | TODO | pre-commit config          |
| 67  | Enforce 350-line limit on test files via pre-commit | 🟢  | 3   | S   | ★   | TODO | pre-commit config          |

## Phase 10: Documentation & Benchmarks (When Time Permits)

| #   | Task                                                  | P   | I   | E   | CV  | Src  | File(s)                 |
| --- | ----------------------------------------------------- | --- | --- | --- | --- | ---- | ----------------------- |
| 68  | Rewrite example/user/ for full CQRS demo              | 🟢  | 2   | XL  | ★★  | TODO | `example/user/`         |
| 69  | Benchmark storage backends (PG vs SQLite vs Pebble)   | 🟢  | 2   | L   | ★   | TODO | `storage/, pebble/`     |
| 70  | Add fuzz tests for event creation, ID parsing, schema | 🟢  | 3   | M   | ★   | TODO | `event/, id/, catalog/` |
| 71  | Add E2E throughput benchmarks                         | 🟢  | 3   | M   | ★   | TODO | `integration/`          |
| 72  | Performance regression CI                             | 🟢  | 3   | L   | ★   | TODO | `.github/workflows/`    |
| 73  | ROADMAP.md creation                                   | 🟢  | 4   | S   | ★   | TODO | `ROADMAP.md`            |

---

## Execution Order (Phase 1 → Phase 10)

**Phase 1 (6 tasks, ~40 min):** Critical production bugs — data races, OOM, broken pagination
**Phase 2 (7 tasks, ~60 min):** High-priority bugs — context drops, Postgres compat, resource leaks
**Phase 3 (15 tasks, ~60 min):** Quality fixes — dead code, dishonest APIs, accepted items
**Phase 4 (8 tasks, ~50 min):** Error taxonomy migration across 8 packages
**Phase 5 (11 tasks, ~90 min):** Code decomposition — long functions, duplicated patterns
**Phase 6 (7 tasks, ~40 min):** Library modernization — errgroup, WalkDir, LSP hints
**Phase 7 (6 tasks, ~90 min):** Test coverage improvements
**Phase 8 (4 tasks, ~120 min):** Architecture — shared Message interface, generic types
**Phase 9 (3 tasks, ~40 min):** CI/DevEx improvements
**Phase 10 (6 tasks, ~180 min):** Documentation and benchmarks

**Total estimated effort:** ~12.5 hours for all 73 tasks

---

## What We Should Skip (ACCEPTED/CLOSE)

These items from TODO_LIST.md are **accepted as intentional design** and should be closed:

- command re-exports of event types (type aliases are transparent)
- pebble vs sql duplicate sentinels (different domain codes per backend)
- event module cycles (test-only imports, standard Go)

## What's Out of Scope

- `query.TypedHandler[T]` (#64) — breaking API change, defer to v2.1
- Example rewrite (#68) — nice-to-have, not blocking
- Documentation site — future work
- LICENSE change — owner decision
