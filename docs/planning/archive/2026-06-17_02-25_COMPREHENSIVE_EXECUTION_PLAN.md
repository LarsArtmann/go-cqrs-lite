# Comprehensive Execution Plan — 2026-06-17 02:25

**Branch:** `consolidate-catalog`
**Source of TODOs:** `TODO_LIST.md` (open + deferred), `ROADMAP.md` (long-term vision + raw ideas), `docs/status/2026-06-17_02-20_COMPREHENSIVE_STATUS_UPDATE.md` (top-25 + self-review findings).
**Granularity:** Every task ≤ 12 min. 99 tasks total.
**Sort order:** Priority → Impact → Effort → Customer value.

---

## Step 1 — Pareto Breakdown

### The 1% that delivers 51% of the result

**Finish the kv→pebble adapter + fix the red race tests.**

- Tasks `T001`–`T017`.
- Why: unblocks branch merge, removes the `kv/` ghost system (zero consumers), restores race-clean CI. Nothing else matters until the branch is mergeable and CI is fully green.

### The 4% that delivers 64% of the result

**Above + the high-impact feature cluster + release prep.**

- Adds `T018`–`T035` (schema registry, distributed checkpointing, Prometheus exporter, structured logging, tracing propagation, PostgreSQL CI) and the release-prep tasks (`T055`–`T068`).
- Why: these are the items consumers actually ask for (observability, validation, SQL-in-CI) and the cleanup that makes the branch merge-safe.

### The 20% that delivers 80% of the result

**Above + medium-impact quality/coverage/docs/experiments scaffolding.**

- Adds `T036`–`T054` and `T067`–`T091`.
- Why: coverage, golden tests, benchmarks, module READMEs, contract tests, and experimental design work compound trust and unblock future sprints without blocking the release.

### The remaining 80%

Deferred breaking changes (`T092`–`T099`, v3/v4) and low-impact raw ideas. Schedule after v2.4.0 ships.

---

## Step 2 — Comprehensive Task Table (all tasks ≤ 12 min)

### Priority legend

- **P0** = Critical / blocking merge or CI. Do first, no exceptions.
- **P1** = High impact, high customer value. Do right after P0.
- **P2** = Medium impact (quality, coverage, docs, release prep).
- **P3** = Experimental / long-term. Design-first, no rush.
- **P4** = Deferred breaking changes (v3/v4 majors).

### Columns

- **Effort** = number of 12-min units (1 ≈ 12 min, 2 ≈ 24 min, …).
- **Impact** = 1–5 (5 = unblocks release / major customer value).
- **Cust** = customer-facing value (H/M/L).

---

### P0 — CRITICAL (blocks merge / red CI / ghost system)

| ID   | Task (≤12 min)                                                                                        | Parent            | Impact | Effort | Cust | Deps      |
| ---- | ----------------------------------------------------------------------------------------------------- | ----------------- | ------ | ------ | ---- | --------- |
| T001 | `pebble/adapter.go`: `Adapter` struct + `Get`/`Has` (copy value, close closer)                        | kv/pebble adapter | 5      | 2      | H    | —         |
| T002 | `pebble/adapter.go`: `NewIterator(prefix)` → `LowerBound`/`UpperBound=prefix+\xff` + `pebbleIterator` | kv/pebble adapter | 5      | 2      | H    | T001      |
| T003 | `pebble/adapter.go`: `Set`/`Delete`/`Batch`/`Close` + `pebbleBatch` wrapper                           | kv/pebble adapter | 5      | 2      | H    | T001      |
| T004 | `pebble/adapter.go`: error mapping (`ErrNotFound`/`ErrClosed`) + `NewKVStore` ctor + options          | kv/pebble adapter | 5      | 1      | H    | T003      |
| T005 | `pebble/adapter_test.go`: CRUD + error-mapping tests                                                  | kv/pebble adapter | 5      | 2      | H    | T004      |
| T006 | `pebble/adapter_test.go`: iteration + prefix-bounds tests                                             | kv/pebble adapter | 5      | 1      | H    | T005      |
| T007 | `pebble/adapter_test.go`: batch atomicity + Close tests                                               | kv/pebble adapter | 5      | 1      | H    | T005      |
| T008 | `pebble/go.mod`: add `kv/v2` require + replace directive                                              | kv/pebble adapter | 5      | 1      | H    | —         |
| T009 | `cd pebble && go mod tidy`                                                                            | kv/pebble adapter | 5      | 1      | H    | T008      |
| T010 | Bump `DEP_BUDGET[pebble]` in `check-module-layers.sh` if needed                                       | kv/pebble adapter | 4      | 1      | M    | T008      |
| T011 | Run `nix run .#test` (pebble-focused + full suite)                                                    | verify adapter    | 5      | 1      | H    | T007,T009 |
| T012 | Run `nix run .#lint` + `nix run .#check-layers`                                                       | verify adapter    | 5      | 1      | H    | T010      |
| T013 | Commit pebble adapter work (detailed message)                                                         | kv/pebble adapter | 5      | 1      | H    | T011,T012 |
| T014 | Push branch to remote                                                                                 | kv/pebble adapter | 4      | 1      | M    | T013      |
| T015 | Investigate `turso/indexing` race root cause (checkpoint scheduler)                                   | race fix          | 5      | 2      | H    | —         |
| T016 | Fix `turso/indexing` checkpoint scheduler race                                                        | race fix          | 5      | 2      | H    | T015      |
| T017 | Re-run `nix run .#test-race`, confirm clean                                                           | race fix          | 5      | 1      | H    | T016      |

### P1 — HIGH IMPACT (features consumers want)

| ID   | Task (≤12 min)                                                      | Parent                     | Impact | Effort | Cust | Deps |
| ---- | ------------------------------------------------------------------- | -------------------------- | ------ | ------ | ---- | ---- |
| T018 | Schema registry: design JSON-Schema validation middleware interface | Schema registry (ADR-0017) | 5      | 2      | H    | —    |
| T019 | Schema registry: implement validation middleware in `middleware/`   | Schema registry            | 5      | 3      | H    | T018 |
| T020 | Schema registry: validation tests + ADR-0017 update                 | Schema registry            | 5      | 2      | H    | T019 |
| T021 | Distributed checkpointing: design coordination interface (ADR-0018) | Distributed checkpointing  | 4      | 2      | M    | —    |
| T022 | Distributed checkpointing: implement multi-instance coordination    | Distributed checkpointing  | 4      | 4      | M    | T021 |
| T023 | Distributed checkpointing: tests                                    | Distributed checkpointing  | 4      | 2      | M    | T022 |
| T024 | Prometheus exporter: design replacing `MetricsRecorder`             | Prometheus metrics         | 4      | 1      | H    | —    |
| T025 | Prometheus exporter: implement in `middleware/`                     | Prometheus metrics         | 4      | 3      | H    | T024 |
| T026 | Prometheus exporter: tests + example handler                        | Prometheus metrics         | 4      | 2      | H    | T025 |
| T027 | Structured logging: design configurable `slog` levels               | Structured logging         | 4      | 1      | H    | —    |
| T028 | Structured logging: implement `slog` middleware (cmd/event/query)   | Structured logging         | 4      | 2      | H    | T027 |
| T029 | Structured logging: tests                                           | Structured logging         | 4      | 1      | H    | T028 |
| T030 | Tracing propagation: design span-context across module boundaries   | Distributed tracing        | 4      | 2      | M    | —    |
| T031 | Tracing propagation: implement context propagation                  | Distributed tracing        | 4      | 3      | M    | T030 |
| T032 | Tracing propagation: tests                                          | Distributed tracing        | 4      | 2      | M    | T031 |
| T033 | PostgreSQL CI: add `postgres` service container to `ci.yml`         | PostgreSQL CI              | 4      | 1      | M    | —    |
| T034 | PostgreSQL CI: wire `pg_integration_test.go` into matrix            | PostgreSQL CI              | 4      | 1      | M    | T033 |
| T035 | PostgreSQL CI: verify green run                                     | PostgreSQL CI              | 4      | 1      | M    | T034 |

### P2 — MEDIUM IMPACT (quality / coverage / docs / release prep)

| ID   | Task (≤12 min)                                                                    | Parent                  | Impact | Effort | Cust | Deps      |
| ---- | --------------------------------------------------------------------------------- | ----------------------- | ------ | ------ | ---- | --------- |
| T036 | Pebble coverage: list uncovered branches in `helpers.go`                          | Pebble coverage 85%+    | 3      | 1      | M    | —         |
| T037 | Pebble coverage: list uncovered branches in `serialization.go`                    | Pebble coverage 85%+    | 3      | 1      | M    | —         |
| T038 | Pebble coverage: add tests to reach 85%+                                          | Pebble coverage 85%+    | 3      | 2      | M    | T036,T037 |
| T039 | Pebble golden test: deterministic CBOR envelope bytes                             | Pebble golden test      | 3      | 2      | M    | —         |
| T040 | MemorySnapshotStore golden test: baseline for pebble comparison                   | MemorySnapshot golden   | 3      | 1      | M    | —         |
| T041 | Reactive bus docs: usage examples in `command/doc.go`                             | Reactive bus docs       | 3      | 1      | M    | —         |
| T042 | Reactive bus docs: usage examples in `query/doc.go`                               | Reactive bus docs       | 3      | 1      | M    | —         |
| T043 | Reactive bus docs: `AGENTS.md` reactive section                                   | Reactive bus docs       | 3      | 1      | M    | —         |
| T044 | Benchmark: pebble vs SQL store (`Save 100 events`)                                | Benchmark pebble vs SQL | 3      | 2      | M    | —         |
| T045 | cqrs-gen v2: design struct-tag scanning                                           | cqrs-gen v2             | 3      | 2      | M    | —         |
| T046 | cqrs-gen v2: implement struct-tag scanning                                        | cqrs-gen v2             | 3      | 3      | M    | T045      |
| T047 | cqrs-gen v2: tests                                                                | cqrs-gen v2             | 3      | 2      | M    | T046      |
| T048 | Built-in pprof endpoints: profiling HTTP handler in `middleware/`                 | pprof endpoints         | 3      | 2      | M    | —         |
| T049 | Built-in pprof endpoints: tests                                                   | pprof endpoints         | 3      | 1      | M    | T048      |
| T050 | Compose tests: `command.Compose` test cases                                       | Compose tests           | 3      | 1      | M    | —         |
| T051 | Compose tests: `query.Compose` test cases                                         | Compose tests           | 3      | 1      | M    | —         |
| T052 | Replace sleep-based integration test with deterministic sync                      | Brittle test fix        | 3      | 2      | M    | —         |
| T053 | `AGENTS.md`: update kv/pebble integration decisions                               | AGENTS.md update        | 3      | 1      | M    | T013      |
| T054 | ADR-0023: pebble-kv adapter decision record                                       | ADR-0023                | 3      | 1      | M    | T013      |
| T055 | Re-run `brutal-self-review` after adapter work                                    | Self-review             | 4      | 2      | M    | T013      |
| T056 | Verify no other ghost systems remain (scan all modules for zero-consumer exports) | Ghost-system sweep      | 4      | 2      | M    | T055      |
| T057 | `CHANGELOG.md`: adapter completion entry                                          | Release prep            | 2      | 1      | L    | T013      |
| T058 | `FEATURES.md`: adapter status update                                              | Release prep            | 2      | 1      | L    | T013      |
| T059 | Module README: `kv/`                                                              | Module READMEs          | 3      | 1      | M    | —         |
| T060 | Module README: `pebble/`                                                          | Module READMEs          | 3      | 1      | M    | T013      |
| T061 | `TODO_LIST.md`: close completed items                                             | Release prep            | 2      | 1      | L    | —         |
| T062 | Squash/rebase branch commits into logical groups                                  | Branch hygiene          | 3      | 2      | M    | T013      |
| T063 | Open PR / prepare merge to `master`                                               | Merge prep              | 4      | 1      | H    | T013,T017 |
| T064 | Tag next release (`v2.4.0`)                                                       | Release                 | 4      | 1      | H    | T063      |
| T065 | Performance baseline update if adapter changes allocations                        | Perf baseline           | 3      | 1      | M    | T013      |
| T066 | Update `docs/planning/` with completed adapter milestone                          | Docs                    | 2      | 1      | L    | T013      |
| T067 | Consumer-driven contract tests: run `kv/` suite against pebble adapter            | Contract tests          | 4      | 2      | H    | T013      |
| T068 | Decide `kv.Store` public-vs-internal scope (answer open question)                 | Scope decision          | 4      | 1      | H    | —         |

### P3 — EXPERIMENTAL / LONG-TERM (design-first)

| ID   | Task (≤12 min)                                                     | Parent             | Impact | Effort | Cust | Deps |
| ---- | ------------------------------------------------------------------ | ------------------ | ------ | ------ | ---- | ---- |
| T069 | gRPC transport adapter: design                                     | gRPC adapter       | 3      | 2      | M    | —    |
| T070 | gRPC transport adapter: implement                                  | gRPC adapter       | 3      | 4      | M    | T069 |
| T071 | gRPC transport adapter: tests                                      | gRPC adapter       | 3      | 2      | M    | T070 |
| T072 | NATS/Redis Stream adapter: design                                  | NATS/Redis adapter | 3      | 2      | M    | —    |
| T073 | NATS/Redis Stream adapter: implement                               | NATS/Redis adapter | 3      | 4      | M    | T072 |
| T074 | NATS/Redis Stream adapter: tests                                   | NATS/Redis adapter | 3      | 2      | M    | T073 |
| T075 | Streaming event reads: design `StreamLoader` interface             | Streaming reads    | 3      | 2      | M    | —    |
| T076 | Streaming event reads: implement without materializing slice       | Streaming reads    | 3      | 3      | M    | T075 |
| T077 | Streaming event reads: tests                                       | Streaming reads    | 3      | 2      | M    | T076 |
| T078 | `jsonv2` codec experiment behind build tag                         | jsonv2 experiment  | 2      | 2      | L    | —    |
| T079 | Arena allocation experiment behind build tag                       | Arena experiment   | 2      | 2      | L    | —    |
| T080 | WASM compilation target for `decider` module                       | WASM target        | 3      | 3      | M    | —    |
| T081 | Documentation site: pick generator (Docusaurus/MkDocs/Hugo)        | Docs site          | 3      | 1      | M    | —    |
| T082 | Documentation site: scaffold + deploy pipeline                     | Docs site          | 3      | 3      | M    | T081 |
| T083 | SIMD-accelerated event serialization: experiment                   | SIMD               | 2      | 3      | L    | —    |
| T084 | Event stream compaction / log truncation: design                   | Compaction         | 2      | 2      | L    | —    |
| T085 | Multi-tenant event store: design schema-per-tenant                 | Multi-tenant       | 2      | 2      | L    | —    |
| T086 | Event archival to S3/GCS/Azure Blob: design                        | Archival           | 2      | 2      | L    | —    |
| T087 | CQRS-lite dashboard web UI: design                                 | Dashboard          | 3      | 2      | M    | —    |
| T088 | Automatic migration generator for schema evolution                 | Migration gen      | 2      | 3      | L    | —    |
| T089 | Property-based integration testing with state-machine verification | PBT integration    | 3      | 3      | M    | —    |
| T090 | Chaos engineering integration (partitions, disk failures)          | Chaos eng          | 2      | 3      | L    | —    |
| T091 | Performance regression dashboard (historical tracking)             | Perf dashboard     | 2      | 3      | L    | —    |

### P4 — DEFERRED BREAKING CHANGES

| ID       | Task (≤12 min)                                                | Parent          | Impact | Effort | Cust  | Deps                                                              |
| -------- | ------------------------------------------------------------- | --------------- | ------ | ------ | ----- | ----------------------------------------------------------------- |
| T092     | v3: Remove `io.Closer` from core interfaces (ADR-0010)        | v3 breaking     | 4      | 3      | H     | v3 branch                                                         |
| ~~T093~~ | ~~v3: Split `event.Store` into Writer/Reader/Deleter~~        | ~~v3 breaking~~ | ~~4~~  | ~~4~~  | ~~H~~ | **REMOVED — Sink/Source split + tombstones already satisfy this** |
| T094     | v3: Add global `TransactionID` branded type                   | v3 breaking     | 3      | 2      | M     | v3 branch                                                         |
| T095     | v3: Make event Core truly immutable                           | v3 breaking     | 3      | 2      | M     | v3 branch                                                         |
| T096     | v3: Move HTTP code out of `middleware/` → `transport/` module | v3 breaking     | 3      | 3      | M     | v3 branch                                                         |
| T097     | v3: Fix `query.Handler` returns `any` → `TypedHandler[T]`     | v3 breaking     | 4      | 2      | H     | v3 branch                                                         |
| T098     | v4: Split `catalog.Message` into Message + MessageMeta        | v4 breaking     | 3      | 3      | M     | v4 branch                                                         |
| T099     | v4: Split `catalog.Service` into Service + ServiceMeta        | v4 breaking     | 3      | 3      | M     | v4 branch                                                         |

---

## Roll-up Summary

| Priority            | Tasks  | Total effort (12-min units) | Est. hours | Theme                        |
| ------------------- | ------ | --------------------------- | ---------- | ---------------------------- |
| **P0** Critical     | 17     | 25                          | ~5.0       | kv/pebble adapter + race fix |
| **P1** High         | 18     | 38                          | ~7.6       | Features consumers want      |
| **P2** Medium       | 33     | 44                          | ~8.8       | Quality + release prep       |
| **P3** Experimental | 23     | 60                          | ~12.0      | Long-term design/impl        |
| **P4** Breaking     | 8      | 22                          | ~4.4       | v3/v4 majors                 |
| **TOTAL**           | **99** | **189**                     | **~37.8**  | —                            |

---

## Step 3 — D2 Execution Graph

```d2
direction: right

p0_critical: P0 Critical (blocks merge) {
  adapter: kv→pebble adapter\n(T001–T014)
  race: Fix turso race\n(T015–T017)
}

p1_high: P1 High Impact {
  schema: Schema registry\n(T018–T020)
  distcp: Distributed checkpointing\n(T021–T023)
  prom: Prometheus exporter\n(T024–T026)
  slog_mw: Structured logging\n(T027–T029)
  tracing: Tracing propagation\n(T030–T032)
  pg_ci: PostgreSQL CI\n(T033–T035)
}

p2_medium: P2 Medium {
  coverage: Pebble coverage\n(T036–T038)
  golden: Golden tests\n(T039–T040)
  docs_buses: Reactive bus docs\n(T041–T043)
  bench: Benchmark\n(T044)
  gen: cqrs-gen v2\n(T045–T047)
  pprof: pprof endpoints\n(T048–T049)
  compose: Compose tests\n(T050–T051)
  flaky: Fix flaky test\n(T052)
  agents: AGENTS.md + ADR-0023\n(T053–T054)
  review: Self-review + ghost sweep\n(T055–T056)
  release_prep: Release prep\n(T057–T061, T065–T066)
  merge: Merge + tag v2.4.0\n(T062–T064)
  contract: Contract tests\n(T067)
  scope: kv scope decision\n(T068)
}

p3_exp: P3 Experimental {
  grpc: gRPC adapter\n(T069–T071)
  nats: NATS/Redis adapter\n(T072–T074)
  stream: Streaming reads\n(T075–T077)
  experiments: jsonv2/arena/SIMD\n(T078–T079, T083)
  wasm: WASM target\n(T080)
  docsite: Docs site\n(T081–T082)
  raw_ideas: Raw ideas\n(T084–T091)
}

p4_breaking: P4 Breaking (v3/v4) {
  v3: v3 changes\n(T092–T097)
  v4: v4 changes\n(T098–T099)
}

adapter -> race -> p1_high -> p2_medium -> p3_exp -> p4_breaking
scope -> adapter: informs ownership
contract -> adapter: validates
review -> merge: gates
```

---

## Recommended Execution Order

1. **Now:** `T001`→`T017` (P0). Branch becomes mergeable, CI green, ghost system gone.
2. **Next sprint:** `T018`→`T035` + `T055`→`T068` (P1 + release prep). Ship v2.4.0.
3. **Following sprints:** `T036`→`T054` (P2 quality). Then P3 design work as capacity allows.
4. **Next major:** `T092`→`T099` (P4) on a v3/v4 branch.

---

_Plan generated 2026-06-17 02:25 CEST. 99 tasks, ~37.8 h total estimated effort._
