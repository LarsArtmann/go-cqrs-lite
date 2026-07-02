# Error Family Taxonomy Adoption — Status Report

**Date:** 2026-07-02 04:40 UTC  
**Branch:** master  
**Base:** a79edf16 (v3.5.0 release)  
**Working tree:** CLEAN  
**Quality gate:** ALL GREEN (build + test + api-stability)

---

## Executive Summary

Systematic adoption of the `go-error-family` 5-family taxonomy (Rejection / Conflict / Transient / Infrastructure / Corruption) across 14 modules. The codebase went from **1 site** using `errorfamily.Wrapf` to **~100 error sites** properly classified.

### Before vs After (branching-flow metrics)

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Semantic context error rows | 59 | 21 | **-64%** |
| Sites with `Error Family: false` | ~55 | ~10 | **-82%** |
| Bare `return err` (no wrapping) | ~40 | ~10 | **-75%** |
| `fmt.Errorf` without family | ~20 | ~3 | **-85%** |

The remaining 21 rows are:
- **9 propagating pre-classified errors** (graph validate() → Rejection, relational rowColumns → Rejection, ctx.Err() → context cancellation)
- **3 storage/view** sites (not in original scope — added in a later session)
- **1 watermill/protocol.go** (not in original scope)

---

## Modules Completed (17 commits)

| Module | Files | Sites | Families Used |
|--------|-------|-------|---------------|
| idempotency | kv_store.go | 3 | Transient |
| storage/pebble | journal.go, command_read.go, query_read.go, iteration.go, backend.go | ~18 | Infrastructure, Corruption |
| projectionhost | host.go | 1 | Infrastructure |
| catalog | frontmatter_render.go | 1 | Corruption |
| storage (core) | pg_bus_listen.go, event_store_stream.go, memory/stream.go | 6 | Infrastructure |
| cmd/cqrs-gen | main.go | 3 | Infrastructure |
| stack/pebble | preset.go | 2 | Infrastructure |
| stack/postgres | pg_listener.go, preset.go | 12 | Rejection (DSN), Infrastructure |
| stack/sqlite | preset.go | 11 | Infrastructure |
| stack/turso | preset.go | 12 | Rejection (multi-DB), Infrastructure |
| middleware | deadletter_sql.go | 7 | Infrastructure, Transient, Corruption |
| storage (kv_sql) | kv_sql.go | 13 | Transient, Infrastructure, Corruption |
| storage/relational | store.go, sink.go, schema.go, projection.go | ~25 | Rejection, Transient, Corruption |

**Graph module** (`graph/memory.go`, `schema_sink.go`): Already fully classified via `event.NewRejection` sentinels in `errors.go`. Bare returns propagate pre-classified errors — no wrapping needed.

---

## Classification Strategy

| Family | When | HTTP Status | Retryable |
|--------|------|-------------|-----------|
| **Rejection** | Bad input, caller misuse, schema validation, not found | 400 | No |
| **Conflict** | Version mismatch, duplicate key, optimistic lock failure | 409 | No |
| **Transient** | Network, timeout, connection pool, SQL execution failure | 503 | Yes |
| **Corruption** | Deserialization failure, data integrity violation | 500 | No |
| **Infrastructure** | DB down, config error, resource unavailable, lifecycle | 503 | No |

### Facade Pattern

- Modules importing `event/v3` → `event.Wrap*` / `event.New*` (re-exports from `event/errors.go`)
- Leaf modules (`catalog`, `cmd/cqrs-gen`) → `errorfamily.Wrap*` / `errorfamily.New*` directly
- Pre-classified sentinels propagate through bare `return err` — no double-wrapping

---

## What's Left (Not Started)

1. **storage/view/** (3 sites in auto.go, count.go, crud.go) — `fmt.Errorf` → `event.Wrap*`
2. **watermill/protocol.go** (1 site) — bare err return
3. These were not in the original branching-flow report's high-priority set

---

## Commits (17 total, all pushed)

```
cdd98f60 Classify relational storage layer errors with error family taxonomy
eddad2be Classify KV SQL store errors with error family taxonomy
9a36922f Classify deadletter SQL store errors with error family taxonomy
45cca373 Classify stack/turso preset errors with error family taxonomy
c31c7e2c Classify stack/sqlite preset errors as Infrastructure
8859739f Classify stack/postgres preset errors as Infrastructure
68f84ed4 Classify pgx_listener errors: Rejection for bad DSN, Infrastructure for pool
2d2e05c6 Classify stack/pebble preset errors as Infrastructure
99e69de7 Classify cqrs-gen scan/walk/parse errors as Infrastructure
d39c42e1 Classify storage layer errors with event.WrapInfrastructure
3e80beaf Classify catalog frontmatter marshal error as Corruption
ec9f9333 Classify projectionhost dead-letter list error as Infrastructure
77506c31 chore: fix trailing whitespace in status report
f4daac66 chore: remove stale nolint directive from golangci-lint auto-fix
4f3640fb chore: update .gitignore via pre-commit hook
fe64c544 Classify pebble storage layer errors with event.WrapInfrastructure
81f6c930 Classify idempotency KV store errors as Transient
```
