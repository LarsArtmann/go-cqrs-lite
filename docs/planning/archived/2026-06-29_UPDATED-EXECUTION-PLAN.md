# Updated Execution Plan — go-cqrs-lite

> **Updated:** 2026-06-29
> **Head:** `3d3319fb` (v3.4.0 changelog)
> **Modules:** 54 go.mod files (53 in go.work + root)
> **Previous plan:** `docs/planning/2026-06-28_20-37_GRAPH-TIER-TYPEDB-HARDENING.md`

---

## What Changed Since the Last Plan

| Item                           | Old Plan Status               | Current Status                                                                                                       |
| ------------------------------ | ----------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| transport/http dep budget      | P0 blocker (budget=2, need=3) | **BLOCKER AGAIN** — concurrent session added more deps, now needs 4                                                  |
| Deriver module                 | T37-T39, not started          | **SHIPPED** (`deriver/`, 8 tests) + example (`example/deriver/`)                                                     |
| `PropertyType.Required`        | T03, lying field              | **REMOVED**                                                                                                          |
| `graph.Schema.Indexes`         | T10, not started              | **SHIPPED** (4 tests)                                                                                                |
| ReadableDriver contracts       | T11-T13, not started          | **SHIPPED** (4 contract tests)                                                                                       |
| Property-based tests           | T42, not started              | **SHIPPED** (400 rapid iterations)                                                                                   |
| Graph integration test         | T17, not started              | **SHIPPED**                                                                                                          |
| PG test scaffold               | T14-T16, not started          | **SHIPPED** (build-tagged)                                                                                           |
| Denorm examples                | T43, not started              | **SHIPPED** (2 tests)                                                                                                |
| projectionhost                 | Mentioned in C7               | **MASSIVELY EXPANDED** by concurrent session — MetricsRecorder, jitter, DLQ Delete, stress test, SQL checkpoint test |
| scheduling                     | Mentioned in C10              | **GENERIC** `Timer[P any]` + exponential backoff + jitter (concurrent session)                                       |
| scenario DSL                   | Not in plan                   | **SHIPPED** by concurrent session                                                                                    |
| idempotency module             | Not in plan                   | **EXISTS** (shipped by prior session)                                                                                |
| example/deriver                | T37 sub-task                  | **SHIPPED** by concurrent session                                                                                    |
| example/projectionhost         | Not in plan                   | **EXISTS** (shipped by concurrent session)                                                                           |
| BuildFlow hook scope detection | T18                           | **SHIPPED** (local .git/hooks only — not committed to repo)                                                          |
| Research doc status stamps     | T19-T20                       | **SHIPPED** (10 docs stamped)                                                                                        |
| Durability profiles design     | T45                           | **SHIPPED** (design doc)                                                                                             |
| Concurrent corruption doc      | T55                           | **SHIPPED**                                                                                                          |

---

## Current Blockers

### BL-1: check-layers FAILING — transport/http dep budget

```
BUDGET: transport/http has 4 production deps (budget: 3, total: 7, test: 3)
```

A concurrent session added `go.opentelemetry.io/otel` + `go.opentelemetry.io/otel/sdk` as direct deps. The budget was bumped to 3 in my session but now needs 4.

**Fix:** `sed -i 's/DEP_BUDGET\[transport\/http\]=3/DEP_BUDGET[transport\/http]=4/' scripts/check-module-layers.sh`

### BL-2: `deriver/` missing from check-module-layers.sh

The deriver module is in go.work but has no LAYER or DEP_BUDGET entry. check-arch may or may not flag this depending on how it discovers modules.

**Fix:** Add `LAYER[deriver]=3` and `DEP_BUDGET[deriver]=2` to the script.

---

## Updated Task Table

Sorted by priority → impact → effort. Status reflects the **actual repo state** as of `3d3319fb`.

### P0 — CI Blockers (must fix before any merge)

| ID      | Task                                                                 | Impact   | Effort | Status      |
| ------- | -------------------------------------------------------------------- | -------- | ------ | ----------- |
| **U01** | Bump `DEP_BUDGET[transport/http]` from 3→4 in check-module-layers.sh | Critical | 5min   | 🔴 BLOCKING |
| **U02** | Add `deriver/` to check-module-layers.sh (LAYER + DEP_BUDGET)        | High     | 5min   | 🔴 MISSING  |

### P1 — Critical Correctness / High Leverage

| ID      | Task                                                                                   | Impact   | Effort | Status         |
| ------- | -------------------------------------------------------------------------------------- | -------- | ------ | -------------- |
| **U03** | Wire Deriver idempotency — deterministic command IDs from source event causation chain | Critical | 30min  | ⬜ Not started |
| **U04** | Commit BuildFlow hook config to repo (scope detection is local only)                   | High     | 10min  | ⬜ Local only  |
| **U05** | Add deriver example to AGENTS.md module list + test command                            | Medium   | 10min  | ⬜ Stale docs  |

### P2 — High-Value Improvements

| ID      | Task                                                                          | Impact | Effort | Status                                          |
| ------- | ----------------------------------------------------------------------------- | ------ | ------ | ----------------------------------------------- |
| **U06** | **God-package split: storage/** (39 files → 8 sub-packages with type aliases) | High   | 4h     | ⬜ Deferred — recommended for v3.x with aliases |
| **U07** | PG integration tests in CI (GitHub Actions service container)                 | High   | 1h     | ⬜ Scaffold exists, CI job missing              |
| **U08** | FTS5 full-text search for RelationalStore (DiscordSync SearchMessages)        | Medium | 2h     | ⬜ Not started                                  |
| **U09** | Versioned schema migrations (goose/atlas-style)                               | Medium | 2h     | ⬜ Not started                                  |
| **U10** | Durability profiles implementation (Sync/BatchedSync/Async)                   | Medium | 1.5h   | ⬜ Design done                                  |
| **U11** | Outbox DLQ + reference-based outbox                                           | Medium | 2h     | ⬜ ADR-0042/0043 discuss DLQ direction          |

### P3 — Quality and Completeness

| ID      | Task                                                                         | Impact   | Effort | Status                                                                      |
| ------- | ---------------------------------------------------------------------------- | -------- | ------ | --------------------------------------------------------------------------- |
| **U12** | **projection.Runner** — standalone journal-replay pipeline (not Bundle-tied) | Low      | 30min  | ⬜ YAGNI until consumer asks — `bundle.RunProjections` covers common case   |
| **U13** | Neo4j/Cypher GraphDriver (`graph/neo4j/`)                                    | High     | 3-4h   | ⬜ Consumer-pulled; Schema.Indexes ready                                    |
| **U14** | NATS JetStream transport adapter (ADR-0025)                                  | Medium   | 3h     | ⬜ Needs external broker dep                                                |
| **U15** | DiscordSync migration to RelationalProjection                                | Critical | 2-3h   | ⬜ Blocked — separate repo                                                  |
| **U16** | Documentation site (Docusaurus/MkDocs)                                       | Low      | 4h+    | ⬜ 54 modules need browsable docs                                           |
| **U17** | `event/` god-package split (30 files)                                        | Low      | 3h     | ⬜ **NOT RECOMMENDED** — 197 importers, high blast radius, cohesion is real |
| **U18** | Commit remaining uncommitted files from concurrent sessions                  | Medium   | Varies | ⬜ Working tree now clean — concurrent session committed everything         |

### P4 — Long-Term / Design-Only

| ID      | Task                                            | Impact | Effort | Status                                    |
| ------- | ----------------------------------------------- | ------ | ------ | ----------------------------------------- |
| **U19** | Hot-state cache for decider (profile first)     | Low    | Large  | ⬜ Snapshot+page-cache may suffice        |
| **U20** | Read-pressure snapshot strategy                 | Low    | Medium | ⬜ Subsumed by hot-state cache            |
| **U21** | Event redaction middleware                      | Low    | Medium | ⬜ Design reviewed                        |
| **U22** | Bi-temporal model (`ValidAt`)                   | Low    | Large  | ⬜ Niche — finance/HR/healthcare          |
| **U23** | Scheduler module expansion (beyond scheduling/) | Low    | Large  | ⬜ scheduling/ shipped Timer[P] generics  |
| **U24** | Graph read API on real driver backends          | Low    | Large  | ⬜ Cypher abstraction rejected (ADR-0038) |
| **U25** | Event-history visualization tools               | Low    | Large  | ⬜ Research doc OPEN                      |

---

## Decisions Locked

| Decision                                       | Rationale                                                                                        |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| **`storage/` split: YES in v3.x with aliases** | 39 files, 8 clear clusters, only 15 importers, aliases keep backward compat                      |
| **`event/` split: NO**                         | 197 importers, `Event` is the most imported type, cohesion is real, 350-line file limit suffices |
| **`projection.Runner`: YAGNI**                 | `bundle.RunProjections` covers the common case. Extract when a consumer asks. 30min if needed.   |
| **Deriver API: functional/composable**         | Locked by ADR-0040. Not a declarative rule registry.                                             |
| **Graph Schema: minimal constraint grammar**   | Locked by ADR-0039. Structural validation only, no business rules.                               |
