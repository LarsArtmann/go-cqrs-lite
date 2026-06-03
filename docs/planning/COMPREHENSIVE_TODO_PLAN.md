# Comprehensive TODO Plan — go-cqrs-lite

> Generated: 2026-06-03  
> All known work items from TODO_LIST.md, FEATURES.md, code quality scan, benchmark bottlenecks, and documentation gaps.  
> Tasks split into max 12-minute units. Sorted by **impact × urgency ÷ effort**.

---

## Priority Tiers

| Tier   | Meaning                                                       |
| ------ | ------------------------------------------------------------- |
| **P0** | Customer-facing breakage, data loss risk, or blocking CI      |
| **P1** | High impact on trust/quality — things a consumer would notice |
| **P2** | Improves developer experience, maintainability, performance   |
| **P3** | Nice-to-have, future-proofing, speculative                    |
| **P4** | Far future or blocked on external action                      |

---

## The Plan

### P0 — Fix Now (Blocking / Breaking)

| #    | Task                                                                                                             | Source             | Est. | Impact                                                        |
| ---- | ---------------------------------------------------------------------------------------------------------------- | ------------------ | ---- | ------------------------------------------------------------- |
| P0.1 | **Fix ADR numbering**: renumber `0007-pebble-scope-event-store-only.md` → `0009`, fill or skip `0005`            | ADR audit          | 2min | Correctness — duplicate numbers confuse history               |
| P0.2 | **Fix CONTRIBUTING.md**: remove all `just` references, replace with `nix run` equivalents                        | Contributing audit | 5min | New contributors can't build without this                     |
| P0.3 | **Fix integration/go.mod**: run `go mod tidy` to move `codec/v2` + `snapshot/v2` from indirect → direct          | Code quality       | 2min | Wrong dependency graph, gopls warnings                        |
|
**P0 subtotal: 3 tasks, ~9min**

---

### P1 — High Impact on Consumer Trust

| #    | Task                                                                                                                             | Source              | Est.  | Impact                                                   |
| ---- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------- | ----- | -------------------------------------------------------- |
| P1.1 | **Add tests for `event/eventtest/fake_store.go`** (274 lines, 0 tests) — cover Save/Load/ReadAll/ReadFrom/AppendBatch edge cases | TODO_LIST open #3   | 12min | Untested mock code in public test helper package         |
| P1.2 | **Add tests for `cmd/api-stability/`** — API surface checker has zero tests                                                      | Code quality        | 10min | Tool that guards API stability is itself untested        |
| P1.3 | **Remove deprecated `otel.TraceIDLogger`** — replace internal callers with `ComponentLogger`, delete function                    | Deprecation audit   | 5min  | Dead API confuses consumers                              |
| P1.4 | **Remove deprecated `query.ErrQueryNotSupported`** — replace with `ErrHandlerNotFound`, delete                                   | Deprecation audit   | 5min  | Dead API confuses consumers                              |
| P1.5 | **Add `cmd/api-stability` golden file for current API surface** — baseline for future API compatibility checks                   | Code quality        | 8min  | API stability tool has no golden file to compare against |
| P1.6 | **Fix Pebble store: implement `Journal` + `SeekableJournal`** — currently only implements `Store`                                | FEATURES partial #2 | 12min | Consumer can't iterate Pebble events cross-aggregate     |
| P1.7 | **Improve `cqrs-gen` CLI test coverage** (currently 70.8%) — test flag parsing, error paths, output formatting                   | FEATURES partial #1 | 12min | Code generator entry point is untested                   |

**P1 subtotal: 7 tasks, ~64min**

---

### P2 — Performance & Quality

| #     | Task                                                                                                                          | Source                   | Est.  | Impact                                                 |
| ----- | ----------------------------------------------------------------------------------------------------------------------------- | ------------------------ | ----- | ------------------------------------------------------ |
| P2.1  | **Optimize `listing/InMemoryAggregateReader`** — cache sorted result, avoid O(n log n) on every `List()` call                 | Benchmark bottleneck     | 12min | 269x improvement potential (840ms → ~3ms for 10K aggs) |
| P2.2  | **Audit and reduce `nolint` suppressions** — review 31 `errcheck`, 25 `wrapcheck`, 23 `exhaustruct` suppressions              | Code quality (123 total) | 12min | Hidden bugs lurk under suppressions                    |
| P2.3  | **Clean up `pebble/config.go` backward-compat aliases** (20 lines) — remove if no external consumers                          | TODO_LIST open #1        | 5min  | Dead code in public API                                |
| P2.4  | **Add `ROADMAP.md`** — long-term direction: outbox pattern, schema registry, distributed consensus                            | Documentation gap        | 10min | No vision document exists                              |
| P2.5  | **Update `CHANGELOG.md`** — add entries for perf optimizations committed this session                                         | Documentation gap        | 8min  | 2 days behind, missing 8 commits                       |
| P2.6  | **Remove unused `backend` field from Pebble store**                                                                           | FEATURES partial #2      | 3min  | Dead state in production code                          |
| P2.7  | **Add `Outbox Pattern` design doc** — ADR or design doc for planned feature                                                   | FEATURES planned #1      | 12min | Documented in CONTEXT.md but no design exists          |
| P2.8  | **Add `Schema Registry` design doc** — ADR for planned validation middleware                                                  | FEATURES planned #2      | 12min | No design decisions recorded                           |
| P2.9  | **Benchmark `MemoryStore` with concurrent writers** — verify no race conditions under load                                    | Quality assurance        | 10min | In-memory store used in tests, never stress-tested     |
| P2.10 | **Review `query.TypedHandler[T]` signature** — takes `Query` not `T`, less type-safe than command equivalent                  | TODO_LIST open #2        | 10min | API inconsistency between command/query modules        |
| P2.11 | **Add integration test for Pebble store `ReadAll`/`ReadFrom`** — after P1.6 implements Journal                                | Dependency on P1.6       | 8min  | New interface needs test coverage                      |
| P2.12 | **Clean up `nolint:errcheck` suppressions in `defer .Close()` calls** — use explicit error handling or `defer func()` pattern | Code quality             | 10min | 31 suppressions, many are lazy                         |

**P2 subtotal: 12 tasks, ~112min**

---

### P3 — Polish & Developer Experience

| #     | Task                                                                                                  | Source         | Est.  | Impact                                             |
| ----- | ----------------------------------------------------------------------------------------------------- | -------------- | ----- | -------------------------------------------------- |
| P3.1  | **Add godoc examples for `decider` package** — `Execute`, `Load`, `Repository` patterns               | Documentation  | 10min | Most complex module, no runnable examples          |
| P3.2  | **Add godoc examples for `projection` package** — `Runner`, `Builder`, `On[T]()` patterns             | Documentation  | 10min | Complex replay+live API, no examples               |
| P3.3  | **Add godoc examples for `signing` package** — HMAC + Ed25519 setup, middleware                       | Documentation  | 8min  | Security-critical, easy to misconfigure            |
| P3.4  | **Add godoc examples for `schema` package** — `Upcaster`, `VersionedStore` usage                      | Documentation  | 8min  | Schema evolution is a hard topic                   |
| P3.5  | **Reduce `catalog/` nolint suppressions** (36 total) — the worst package                              | Code quality   | 12min | Highest suppression count suggests design issues   |
| P3.6  | **Add `listing/` package-level example** — show `List`, `StatusMiddleware`, `InMemoryAggregateReader` | Documentation  | 8min  | Newest module, no usage examples                   |
| P3.7  | **Verify all `//nolint` comments have justification** — add `//nolint:linter // reason` format        | Code quality   | 12min | Most suppressions lack explanation                 |
| P3.8  | **Add README section linking to `docs/benchmarks/`** — consumers should know perf characteristics     | Documentation  | 3min  | Benchmark docs exist but aren't linked from README |
| P3.9  | **Add `go.work` sync CI check** — verify workspace is consistent on every PR                          | CI improvement | 8min  | Workspace drift causes subtle build failures       |
| P3.10 | **Add per-module `go vet` as CI step** — separate from lint, catches different issues                 | CI improvement | 5min  | Defense in depth                                   |

**P3 subtotal: 10 tasks, ~84min**

---

### P4 — Blocked / Future / External

| #     | Task                                                                  | Source  | Blocker                     | Est.          |
| ----- | --------------------------------------------------------------------- | ------- | --------------------------- | ------------- |
| P4.1  | Move `example/todo` to own repository                                 | BLOCKED | Manual repo creation needed | 12min         |
| P4.2  | Add PostgreSQL integration tests with testcontainers                  | BLOCKED | Docker/testcontainers setup | 12min         |
| P4.3  | Change LICENSE from proprietary → MIT/Apache-2.0                      | BLOCKED | Owner decision needed       | 5min          |
| P4.4  | Extract shared `golangci.yml` into `larsartmann/library-policy`       | BLOCKED | Different repo              | 10min         |
| P4.5  | Create documentation site (Docusaurus/MkDocs/Hugo)                    | FUTURE  | Large effort                | 60min+        |
| P4.6  | Add distributed consensus (Raft/CRDT overlay)                         | FUTURE  | Research needed             | Multi-session |
| P4.7  | Add bi-temporal support (`ValidAt`, `LoadToValidTime`)                | FUTURE  | Design decisions needed     | 30min design  |
| P4.8  | Build thin PostgreSQL store adapter (no Watermill)                    | FUTURE  | Consumer demand needed      | 60min+        |
| P4.9  | Build thin NATS bus adapter (no Watermill)                            | FUTURE  | Consumer demand needed      | 60min+        |
| P4.10 | Add catalog diff/breaking-change detection tool                       | FUTURE  | Depends on Schema Registry  | 30min design  |
| P4.11 | Add high-level test utilities (`AggregateTester`, `ProjectionTester`) | FUTURE  | API design needed           | 30min design  |
| P4.12 | Make event Core truly immutable                                       | [v2]    | Breaking API change         | 30min+        |
| P4.13 | Split `event.Store` into Writer/Reader                                | [v2]    | Breaking API change         | 30min+        |
| P4.14 | Add global `TransactionID` branded type                               | [v2]    | Breaking API change         | 15min         |
| P4.15 | Remove `io.Closer` from core interfaces                               | [v2]    | Breaking API change         | 15min         |
| P4.16 | Fix `query.Handler` returns `any` → generic `TypedHandler[T]`         | [v2]    | Breaking API change         | 20min         |
| P4.17 | Implement pull-before-push sync protocol                              | FUTURE  | Research needed             | Multi-session |
| P4.18 | Add HLC (Hybrid Logical Clock) implementation                         | FUTURE  | Research needed             | 30min design  |
| P4.19 | Add network simulator for testing                                     | FUTURE  | Research needed             | 30min design  |
| P4.20 | Add multi-client test harness                                         | FUTURE  | Research needed             | 30min design  |

**P4 subtotal: 20 tasks, blocked/future**

---

## Summary

| Tier                         | Tasks  | Time              | Status                                 |
| ---------------------------- | ------ | ----------------- | -------------------------------------- |
| **P0** Fix Now               | 3      | 9min              | Ready to execute                       |
| **P1** Consumer Trust        | 7      | 64min             | Ready to execute                       |
| **P2** Performance & Quality | 12     | 112min            | Ready to execute                       |
| **P3** Polish                | 10     | 84min             | Ready to execute                       |
| **P4** Blocked/Future        | 20     | Blocked           | Waiting on external action or v2 cycle |
| **Total**                    | **52** | **~269min ready** |                                        |

---

## Execution Order (Ready Tasks)

Recommended execution sequence — P0 first, then P1 → P2 → P3:

```
P0.1 → P0.2 → P0.3 → 
P1.1 → P1.2 → P1.3 → P1.4 → P1.5 → P1.6 → P1.7 →
P2.1 → P2.2 → P2.3 → P2.4 → P2.5 → P2.6 → P2.7 → P2.8 → P2.9 → P2.10 → P2.11 → P2.12 →
P3.1 → P3.2 → P3.3 → P3.4 → P3.5 → P3.6 → P3.7 → P3.8 → P3.9 → P3.10
```

### Fastest Impact (Top 5 if time is limited)

1. **P2.1** — Listing cache optimization (269x speedup, 12min)
2. **P1.1** — FakeStore tests (untested public code, 12min)
3. **P0.2** — Fix CONTRIBUTING.md (unblocks contributors, 5min)
4. **P1.6** — Pebble Journal implementation (feature completeness, 12min)
5. **P1.2** — api-stability tests (untested API guard tool, 10min)
