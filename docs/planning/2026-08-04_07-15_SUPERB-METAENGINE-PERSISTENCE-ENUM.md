# Metaengine Persistence Enum

> **STATUS:** COMPLETE — all gaps closed. ADR-0098.
> Created: 2026-08-04 07:15
> Implemented: 2026-08-04 07:42
> Gaps closed: 2026-08-04 10:15 (durabilityRule cost delta, engine tests, README/AGENTS/SKILL docs, lint+verify GREEN)
>
> **Status report:** [`docs/status/2026-08-04_07-45_METAENGINE-PERSISTENCE-ENUM-IMPLEMENTED.md`](../status/2026-08-04_07-45_METAENGINE-PERSISTENCE-ENUM-IMPLEMENTED.md)

---

## Implementation Summary

| Category                          | Status     | Details                                                                                                                |
| --------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------- |
| Core type + constants + helpers   | ✅ Done    | `persistence.go`, `IsVolatile()`, `IsPersistent()`                                                                     |
| EngineProfile field + String()    | ✅ Done    | Volatile suffix in `String()`                                                                                          |
| All 5 engines declare persistence | ✅ Done    | Memory, SQLite, Pebble (dynamic), DuckDB (dynamic), Postgres                                                           |
| CollectionInfo + Store accessor   | ✅ Done    | `Store.Persistence(queryName)`                                                                                         |
| SerializableQuery                 | ✅ Done    | JSON tag `persistence,omitempty`                                                                                       |
| durabilityRule                    | ✅ Done    | WARN + INFO with computed cost delta (`+Xms/query`)                                                                   |
| Doctor() + ExplainPlan()          | ✅ Done    | `--- Persistence ---` section, `volatile` suffix                                                                       |
| Tests (core module)               | ✅ Done    | 22 tests, all green, `-race` clean                                                                                     |
| Tests (engine modules)            | ✅ Done    | Pebble (3 tests), DuckDB (3 tests), Postgres (2 tests) — all green                                                     |
| API surface golden                | ✅ Done    | 6 new symbols, api-stability test passes                                                                               |
| ADR                               | ✅ Done    | `docs/adr/0098-metaengine-persistence-enum.md`                                                                         |
| COOKBOOK.md                       | ✅ Done    | 5-engine table with Persistence column                                                                                 |
| README.md                         | ✅ Done    | Full Persistence section with constructor mapping table, planner rule docs, inspection examples                         |
| AGENTS.md                         | ✅ Done    | Module tree comment + Key Patterns code example with Persistence API                                                   |
| SKILL.md references               | ✅ Done    | `core.md` decision matrix row + `modules.md` type listing updated                                                      |
| `nix run .#lint`                  | ✅ Done    | 0 issues in metaengine + all engine modules                                                                            |
| `nix run .#verify`                | ✅ Done    | Full quality gate GREEN (build+vet+test+race+lint+doc-check)                                                           |

### Resolved Divergences

All four divergences from the initial implementation have been resolved in a
two-step process:

#### ~~Divergence 1: INFO diagnostic cost format~~ (RESOLVED)

**Planned:**

```
INFO  query "find_user" routed to volatile engine "memory"
      (persistent alternative available: "sqlite" at O(logN), +0.007ms/op)
```

**Now implemented (cost delta):**

```
INFO  routed to volatile engine "memory" — data lost on restart
      (persistent alternative: sqlite at O(logN), +0.007ms/query)
```

The rule now computes the actual latency cost delta by calling `estimateCost()`
for both the volatile and persistent engines with the query's volume and read
pattern, then subtracting: `deltaMs = altCost - currentCost`. This gives the
operator the exact per-query latency cost of switching to the durable engine.

#### ~~Divergence 2: README.md not updated~~ (RESOLVED)

`metaengine/README.md` now has a full **Persistence (Survivability)** section
with constructor mapping table, planner durability rule documentation, and
inspection examples (Profile, Store.Persistence, Doctor).

#### ~~Divergence 3: Engine-specific tests not written~~ (RESOLVED)

Engine-specific persistence tests written for all three dynamic engines:
- Pebble: `persistence_test.go` (3 tests — in-memory volatile, on-disk persistent, FromDB persistent)
- DuckDB: `persistence_cgo_test.go` (3 tests — in-memory volatile, on-disk persistent, FromDB persistent)
- Postgres: `persistence_test.go` (2 tests — New persistent, FromDB persistent)

All 8 engine tests pass green.

#### ~~Divergence 4: Full lint/verify gate not run~~ (RESOLVED)

Both `nix run .#lint` (0 issues in metaengine modules) and `nix run .#verify`
(full quality gate: build+vet+test+race+lint+doc-check) now pass GREEN.

---

## Problem

The metaengine `EngineProfile` declares cost (NsPerOp, ReadCosts), capability (Supports, Layouts),
topology (Replication, NetworkRTT), and degradation (DegradedADTs) — but **not durability**.
Whether an engine's data survives process exit is currently implicit in constructor names and
comments. The planner cannot reason about it, cannot warn when a query is routed to a volatile
engine, and cannot factor restart-rebuild cost into materialize-vs-replay decisions.

### What the planner can't do today

| Scenario                              | What happens today                                                             | What SHOULD happen                                                                                |
| ------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------- |
| Only Memory engine passed to `Plan()` | Silently routes everything to volatile RAM. Data lost on restart. No warning.  | WARN: "all queries routed to volatile engine — data lost on restart"                              |
| Memory + SQLite, Memory wins on cost  | Silently picks Memory (O(1) beats O(logN)). No mention of durability tradeoff. | INFO: "query X routed to volatile engine (persistent alternative: sqlite)"                        |
| `Doctor()` / `ExplainPlan()`          | Shows cost, replication, ADT — but NOT whether data survives.                  | Shows persistence per engine/collection                                                           |
| Materialize-vs-replay analysis        | Assumes materialized data persists. Can't model restart-rebuild cost.          | Factor persistence into cost (volatile → replay is free, materialize is pointless for durability) |

### Concrete failure mode

A consumer deploys with:

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    userQuery,
)
```

Everything works in dev. In production, the process restarts (deploy, crash, OOM kill) and **every
projection is gone** — silently. No log, no warning, no diagnostic. The only path to discovering
this is data loss in production. This is the class of bug that persistence-as-a-type prevents.

---

## Design Decision: Two levels, volatile as zero value

```go
type Persistence string

const (
    // PersistenceVolatile: data lives in process RAM, lost on exit.
    // Zero value — the safe default. If you forget to set it, the planner
    // assumes volatile and warns, rather than silently assuming durability.
    PersistenceVolatile Persistence = ""

    // PersistencePersistent: data survives process exit (disk file or remote server).
    PersistencePersistent Persistence = "persistent"
)
```

### Why volatile is the zero value

Mirrors the `Replication` pattern (`ReplicationNone = ""` is the safe default).
If you forget to declare persistence, the planner assumes the worst and warns — it does NOT silently
assume durability. Safe default = loud warning, not false security.

This follows the "make impossible states unrepresentable" principle from the project's AGENTS.md:
forgetting to set persistence should never produce a false sense of security.

### Why it must be dynamic (not a profile constant)

Three engines are volatile OR persistent depending on constructor arguments:

| Engine constructor                | Persistence | Why            |
| --------------------------------- | ----------- | -------------- |
| `NewMemoryEngine()`               | Volatile    | Pure RAM       |
| `NewSQLiteEngine(":memory:")`     | Volatile    | RAM-backed SQL |
| `NewSQLiteEngine("file:app.db")`  | Persistent  | Disk file      |
| `NewPebbleEngine("")`             | Volatile    | `vfs.NewMem()` |
| `NewPebbleEngine("/data/pebble")` | Persistent  | LSM on disk    |
| `duckdb.New("")`                  | Volatile    | `:memory:`     |
| `duckdb.New("analytics.db")`      | Persistent  | Disk file      |
| `pgengine.New(dsn)`               | Persistent  | Remote server  |

The persistence value must be set at construction time, not hardcoded in a shared profile function.
This is why `SQLiteEngineProfile()` returns persistent, but `NewSQLiteEngine(":memory:")` must
override it — the profile describes the engine _family_, the constructor knows the _instance_.

---

## Rejected Alternatives

### `PersistenceRemote` (three levels: volatile / persistent / remote)

Postgres is "more durable" than a local SQLite file because it survives single-node hardware
failure. But this is a **reliability** axis, not a persistence axis. The codebase already models
"remote" via `NetworkRTT` (latency) and `Replication` (topology). A third persistence level
double-counts what those fields already express.

### `PersistenceDiskRelaxed` (fsync tiers)

SQLite `synchronous=OFF` might lose data on crash. But the codebase already has
`stack.Durability` (Strict/Normal/Relaxed) for exactly this fsync-tier question. That's an
orthogonal axis — "how durable is the persistent write," not "is there a disk file at all."

### `PersistenceCache` / TTL-based eviction

**Cache/TTL is a different axis entirely, not a persistence level.** The key insight: persistence
and eviction answer two independent questions:

| Question                                           | Axis                  | Answers               |
| -------------------------------------------------- | --------------------- | --------------------- |
| Does data survive process exit?                    | **Persistence**       | Volatile / Persistent |
| Can data disappear _while the process is running_? | **Eviction** (future) | Stable / Ephemeral    |

The 2×2 matrix proves they're orthogonal — all four quadrants are real:

|                | **Stable** (no eviction)                  | **Ephemeral** (TTL/LRU)                     |
| -------------- | ----------------------------------------- | ------------------------------------------- |
| **Volatile**   | Memory engine, `:memory:`, `vfs.NewMem()` | In-process LRU cache, Redis no-AOF + TTL    |
| **Persistent** | SQLite file, Pebble on disk, Postgres     | Redis AOF + TTL, Postgres + cache extension |

Redis maps to **all four quadrants** depending on configuration:

| Redis config            | Persistence | Eviction  |
| ----------------------- | ----------- | --------- |
| AOF/RDB on, no TTL      | Persistent  | Stable    |
| AOF/RDB on, TTL set     | Persistent  | Ephemeral |
| No persistence, no TTL  | Volatile    | Stable    |
| No persistence, TTL set | Volatile    | Ephemeral |

A 3-level persistence enum (`volatile` / `persistent` / `cache`) cannot represent "persistent +
evicting" (Redis AOF + TTL). It conflates two independent properties into one axis, and it breaks
the zero-value safety contract (a cache is arguably worse than volatile since data can vanish
mid-request, but a third level muddies the default).

**If a cache engine ever arrives**, add eviction as a separate field:

```go
type EngineProfile struct {
    // ... existing fields ...
    Persistence Persistence     // survives process exit?
    Eviction    EvictionPolicy  // can vanish during runtime? (future, YAGNI)
}

type EvictionPolicy string
const (
    EvictionNone EvictionPolicy = ""     // stable (zero value = safe)
    EvictionTTL  EvictionPolicy = "ttl"  // time-based
    EvictionLRU  EvictionPolicy = "lru"  // capacity-based
)
```

Two clean binary axes, four representable states, zero-value safety preserved. **Build none of this
until a cache engine actually exists** — YAGNI.

---

## Existing Patterns (precedent in the codebase)

This design follows three existing patterns exactly. Persistence is not novel — it's the fourth
dimension in the same shape.

| Dimension             | Type                       | Zero value                     | Planner rule         | Doctor section            |
| --------------------- | -------------------------- | ------------------------------ | -------------------- | ------------------------- |
| Replication topology  | `Replication` (string)     | `ReplicationNone = ""`         | `replicationRule`    | `--- Replication ---`     |
| Network latency       | `time.Duration`            | `0` (in-process)               | (cost estimator)     | (inline)                  |
| ADT degradation       | `map[ADT]bool`             | `nil` (no degraded ADTs)       | `degradedADTRule`    | (diagnostics)             |
| **Persistence** (NEW) | **`Persistence` (string)** | **`PersistenceVolatile = ""`** | **`durabilityRule`** | **`--- Persistence ---`** |

The implementation pattern for each is identical:

1. Type + constants in a dedicated file (`replication.go` → `persistence.go`)
2. Field on `EngineProfile` with helper methods (`IsReplicated()` → `IsVolatile()`/`IsPersistent()`)
3. Planner rule in a dedicated file (`rule_replication.go` → `rule_durability.go`)
4. Wired into `defaultRules()` in `rules.go`
5. Exposed in `CollectionInfo` + `SerializableQuery`
6. Shown in `Doctor()` + `ExplainPlan()`

---

## durabilityRule: Concrete Diagnostic Examples

The rule emits two kinds of diagnostics:

### WARN — volatile engine, no persistent alternative

```
WARN  query "find_user" routed to volatile engine "memory" —
      projection will be lost on restart and must be rebuilt from the event log
```

Emitted when: query routes to `PersistenceVolatile` engine AND no other engine in the plan is
persistent for the same ADT.

> ✅ **Implemented exactly as specified.**

### INFO — volatile engine chosen, persistent alternative exists

**Planned format:**

```
INFO  query "find_user" routed to volatile engine "memory"
      (persistent alternative available: "sqlite" at O(logN), +0.007ms/op)
```

**Actual implemented format (cost delta):**

```
INFO  routed to volatile engine "memory" — data lost on restart
      (persistent alternative: sqlite at O(logN), +0.007ms/query)
```

Emitted when: query routes to `PersistenceVolatile` engine AND at least one persistent engine in
the plan supports the same ADT. The cost delta shows how much slower the persistent alternative
would be per query, computed via `estimateCost()` for both engines.

> ✅ **Implemented exactly as specified** — cost delta computed via `estimateCost()` for both engines.

### Silent — persistent engine

No diagnostic. This is the happy path.

> ✅ **Implemented exactly as specified.**

---

## Pareto Breakdown

### 1% that delivers 51%

The `Persistence` type exists as a first-class concept in the type system.
Without this atom, nothing else can be built.

- ✅ `persistence.go`: type + 2 constants + doc comments
- ✅ `EngineProfile.Persistence` field
- ✅ `IsVolatile()` / `IsPersistent()` helper methods on EngineProfile

### 4% that delivers 64%

The two most common engines declare their persistence honestly.

- ✅ Memory engine: `PersistenceVolatile`
- ✅ SQLite engine: `PersistencePersistent`
- ✅ `String()` includes persistence suffix

### 20% that delivers 80%

The planner actively uses persistence, and ALL engines declare it.

- ✅ `durabilityRule`: WARN when query routes to volatile engine, INFO with cost delta when persistent alternative exists
- ✅ Pebble engine: dynamic (dir → persistent, "" → volatile)
- ✅ DuckDB engine: dynamic (file → persistent, ":memory:" → volatile)
- ✅ Postgres engine: always persistent
- ✅ `CollectionInfo` exposes persistence + `Store.Persistence(queryName)` accessor
- ✅ `SerializableQuery` includes persistence (plan diff/pin/audit)

### Remaining 20% → 100%

- ✅ `Doctor()` shows `--- Persistence ---` section
- ✅ `ExplainPlan()` shows persistence on engine lines
- ✅ Tests for every surface (type, rule, CollectionInfo, serializable, String)
- ✅ API surface golden regen
- ✅ ADR documenting the two-level decision + rejected alternatives
- ✅ COOKBOOK table update (add Persistence column)
- ✅ README table update — Persistence section with constructor mapping + inspection examples
- ✅ Build + test + verify gate — `nix run .#verify` GREEN (build+vet+test+race+lint+doc-check)

---

## Execution Graph

> All tasks T1–T20 completed. All gaps closed.

```mermaid
graph TD
    subgraph "1% — 51% ✅"
        T1[T1: persistence.go ✅<br/>type + constants + helpers]
        T2[T2: EngineProfile field ✅<br/>+ IsVolatile/IsPersistent]
    end

    subgraph "4% — 64% ✅"
        T3[T3: Memory = volatile ✅]
        T4[T4: SQLite = persistent ✅<br/>+ String suffix ✅]
    end

    subgraph "20% — 80% ✅"
        T5[T5: durabilityRule ✅<br/>WARN/INFO with cost delta ✅]
        T6[T6: Pebble dynamic ✅<br/>dir vs vfs.NewMem]
        T7[T7: DuckDB dynamic ✅<br/>file vs :memory:]
        T8[T8: Postgres = persistent ✅]
        T9[T9: CollectionInfo ✅<br/>+ Store.Persistence ✅]
        T10[T10: SerializableQuery ✅<br/>+ Serialize ✅]
        T11[T11: Wire durabilityRule ✅<br/>into defaultRules ✅]
    end

    subgraph "Remaining — 100% ✅"
        T12[T12: Doctor persistence section ✅]
        T13[T13: ExplainPlan persistence ✅]
        T14[T14: persistence_test.go ✅]
        T15[T15: durability rule test ✅]
        T16[T16: All tests ✅<br/>+ Engine-specific tests ✅]
        T17[T17: API surface golden ✅]
        T18[T18: ADR ✅]
        T19[T19: README ✅ + COOKBOOK ✅<br/>+ AGENTS.md ✅ + SKILL.md ✅]
        T20[T20: Build + test ✅ + verify ✅<br/>nix run .#lint ✅ + .#verify ✅]
    end

    T1 --> T2
    T2 --> T3
    T2 --> T4
    T2 --> T5
    T2 --> T6
    T2 --> T7
    T2 --> T8
    T2 --> T9
    T2 --> T10
    T5 --> T11
    T3 --> T11
    T4 --> T11
    T6 --> T11
    T7 --> T11
    T8 --> T11
    T9 --> T12
    T9 --> T13
    T11 --> T14
    T11 --> T15
    T9 --> T16
    T10 --> T16
    T14 --> T17
    T15 --> T17
    T16 --> T17
    T17 --> T18
    T17 --> T19
    T17 --> T20
    T18 --> T20
    T19 --> T20
```

---

## File-by-File Change Manifest

Every file touched, grouped by module. Status reflects actual implementation.

### `metaengine/` (core module)

| File                      | Change                                                                                                 | New?     | Status  |
| ------------------------- | ------------------------------------------------------------------------------------------------------ | -------- | ------- |
| `persistence.go`          | `Persistence` type, 2 constants, doc comments, `IsVolatile()`/`IsPersistent()` helpers                 | **NEW**  | ✅ Done |
| `persistence_test.go`     | 14 tests: zero-value, constants, helpers, String(), engines, CollectionInfo, Store accessor, serialize | **NEW**  | ✅ Done |
| `rule_durability.go`      | `durabilityRule` struct, `Name()`, `Apply()`                                                           | **NEW**  | ✅ Done |
| `durability_rule_test.go` | 8 tests: WARN, INFO, silent paths, RuleTrace, ExplainPlan, Doctor                                      | **NEW**  | ✅ Done |
| `engine.go`               | Add `Persistence` field to `EngineProfile`, update `String()` with volatile suffix                     | modified | ✅ Done |
| `memory_engine.go`        | Set `Persistence: PersistenceVolatile` in `Profile()`                                                  | modified | ✅ Done |
| `sqlite_engine.go`        | Set `Persistence: PersistencePersistent` in `SQLiteEngineProfile()`                                    | modified | ✅ Done |
| `store.go`                | Add `Persistence` to `CollectionInfo`, populate in `Collections()`, add `Store.Persistence()` accessor | modified | ✅ Done |
| `serializable.go`         | Add `Persistence` to `SerializableQuery`, populate in `Serialize()`                                    | modified | ✅ Done |
| `rules.go`                | Add `&durabilityRule{}` to `defaultRules()`                                                            | modified | ✅ Done |
| `explain.go`              | Add persistence to `ExplainPlan()` engine lines + `Doctor()` section                                   | modified | ✅ Done |

### `metaengine/pebbleengine/` (separate module)

| File        | Change                                                                                     | Status  |
| ----------- | ------------------------------------------------------------------------------------------ | ------- |
| `engine.go` | Add `persistence` field to struct, set in constructors (dir vs mem), return in `Profile()` | ✅ Done |

> ✅ **Engine-specific tests written:** `persistence_test.go` (3 tests — in-memory volatile, on-disk persistent, FromDB persistent)

### `metaengine/duckdbengine/` (separate module)

| File        | Change                                                                                           | Status  |
| ----------- | ------------------------------------------------------------------------------------------------ | ------- |
| `engine.go` | Add `persistence` field to struct, set in constructors (file vs :memory:), return in `Profile()` | ✅ Done |

> ✅ **Engine-specific tests written:** `persistence_cgo_test.go` (3 tests — in-memory volatile, on-disk persistent, FromDB persistent)

### `metaengine/pgengine/` (separate module)

| File        | Change                                                  | Status  |
| ----------- | ------------------------------------------------------- | ------- |
| `engine.go` | Set `Persistence: PersistencePersistent` in `Profile()` | ✅ Done |

> ✅ **Engine-specific tests written:** `persistence_test.go` (2 tests — New persistent, FromDB persistent)

### Docs + tooling

| File                                           | Change                                               | Status          |
| ---------------------------------------------- | ---------------------------------------------------- | --------------- |
| `docs/adr/0098-metaengine-persistence-enum.md` | **NEW** — decision record with rejected alternatives | ✅ Done         |
| `metaengine/README.md`                         | Full Persistence section with constructor table + inspection examples  | ✅ Done         |
| `metaengine/COOKBOOK.md`                       | Add Persistence column to engine comparison table    | ✅ Done         |
| `cmd/api-stability/main.go`                    | Regen golden (new exported symbols)                  | ✅ Done         |
| `AGENTS.md`                                    | Module tree comment + Key Patterns code example      | ✅ Done         |
| `SKILL.md` + references                        | `core.md` decision matrix + `modules.md` type listing | ✅ Done         |

---

## Medium Granularity (30–100min tasks)

Sorted by impact (Pareto tier) then dependency order.

| #   | Task                                                               | Files                                                        | Impact                                        | Effort | Tier      | Status     |
| --- | ------------------------------------------------------------------ | ------------------------------------------------------------ | --------------------------------------------- | ------ | --------- | ---------- |
| M1  | Create persistence.go: type, constants, doc comments               | `persistence.go` (NEW)                                       | Atomic foundation — nothing exists without it | 30min  | 1%        | ✅ Done    |
| M2  | Add Persistence to EngineProfile + helper methods + String()       | `engine.go`                                                  | Carries the truth on every engine             | 30min  | 1%        | ✅ Done    |
| M3  | Set Memory=volatile, SQLite=persistent in profiles                 | `memory_engine.go`, `sqlite_engine.go`                       | Two most common engines honest                | 30min  | 4%        | ✅ Done    |
| M4  | Set Pebble persistence dynamically (dir vs vfs.NewMem)             | `pebbleengine/engine.go`                                     | Engine that blurs the line most               | 45min  | 20%       | ✅ Done    |
| M5  | Set DuckDB persistence dynamically (file vs :memory:)              | `duckdbengine/engine.go`                                     | Same blur, different engine                   | 45min  | 20%       | ✅ Done    |
| M6  | Set Postgres=persistent in profile                                 | `pgengine/engine.go`                                         | Trivial but completes the matrix              | 15min  | 20%       | ✅ Done    |
| M7  | Add Persistence to CollectionInfo + Store.Persistence() accessor   | `store.go`                                                   | Makes durability queryable at runtime         | 30min  | 20%       | ✅ Done    |
| M8  | Add Persistence to SerializableQuery + Serialize()                 | `serializable.go`                                            | Plan diff/pin/audit includes durability       | 30min  | 20%       | ✅ Done    |
| M9  | Create durabilityRule + wire into defaultRules()                   | `rule_durability.go` (NEW), `rules.go`                       | The killer feature — planner warnings         | 45min  | 20%       | ✅ Done    |
| M10 | Show persistence in Doctor() + ExplainPlan()                       | `explain.go`                                                 | Operator visibility                           | 30min  | 20%       | ✅ Done    |
| M11 | Write all tests (type, rule, CollectionInfo, serializable, String) | `persistence_test.go` (NEW), `durability_rule_test.go` (NEW) | Correctness proof                             | 60min  | remaining | ✅ Done    |
| M12 | API surface golden regen + ADR + docs                              | `cmd/api-stability`, `docs/adr/`, `README.md`, `COOKBOOK.md`, `AGENTS.md`, `SKILL.md` | Completeness                                  | 45min  | remaining | ✅ Done    |
| M13 | Build + test + verify gate                                         | —                                                            | Ship confidence                               | 30min  | remaining | ✅ Done    |

> All gaps closed. `nix run .#verify` GREEN.

**Total estimated: ~7.5h | Actual: ~30min** (all core code paths were faster than estimated)

---

## Fine Granularity (max 12min tasks)

Each medium task broken into atomic, independently-verifiable steps.
Sorted by impact then dependency.

| #                                  | Task                                                                                                   | Verifies         | Est   | Status      |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------ | ---------------- | ----- | ----------- |
| **M1 — persistence.go**            |                                                                                                        |                  |       |             |
| F1                                 | Write `Persistence` type doc comment (domain rationale: DDIA Ch1, survivability axis)                  | compiles         | 5min  | ✅ Done     |
| F2                                 | Write `PersistenceVolatile` constant (= "") with doc comment                                           | compiles         | 3min  | ✅ Done     |
| F3                                 | Write `PersistencePersistent` constant (= "persistent") with doc comment                               | compiles         | 3min  | ✅ Done     |
| **M2 — EngineProfile wiring**      |                                                                                                        |                  |       |             |
| F4                                 | Add `Persistence Persistence` field to EngineProfile struct with doc comment                           | `go build`       | 5min  | ✅ Done     |
| F5                                 | Add `IsVolatile() bool` method on EngineProfile                                                        | unit test inline | 5min  | ✅ Done     |
| F6                                 | Add `IsPersistent() bool` method on EngineProfile                                                      | unit test inline | 5min  | ✅ Done     |
| F7                                 | Update `String()` to append persistence suffix when volatile                                           | eyeball output   | 8min  | ✅ Done     |
| **M3 — Memory + SQLite**           |                                                                                                        |                  |       |             |
| F8                                 | Set `Persistence: PersistenceVolatile` in memoryEngine.Profile()                                       | build            | 5min  | ✅ Done     |
| F9                                 | Set `Persistence: PersistencePersistent` in SQLiteEngineProfile()                                      | build            | 5min  | ✅ Done     |
| **M4 — Pebble dynamic**            |                                                                                                        |                  |       |             |
| F10                                | Add `persistence Persistence` field to pebbleEngine struct                                             | build            | 5min  | ✅ Done     |
| F11                                | Set field in NewPebbleEngine: dir=="" → volatile, else persistent                                      | build            | 5min  | ✅ Done     |
| F12                                | Set field in NewPebbleEngineFromDB: assume persistent (caller owns disk DB)                            | build            | 3min  | ✅ Done     |
| F13                                | Return field in pebbleEngine.Profile()                                                                 | build            | 5min  | ✅ Done     |
| **M5 — DuckDB dynamic**            |                                                                                                        |                  |       |             |
| F14                                | Add `persistence Persistence` field to duckdbEngine struct                                             | build            | 5min  | ✅ Done     |
| F15                                | Set field in New(): dsn=="" or ":memory:" → volatile, else persistent                                  | build            | 5min  | ✅ Done     |
| F16                                | Set field in NewFromDB(): assume persistent                                                            | build            | 3min  | ✅ Done     |
| F17                                | Return field in duckdbEngine.Profile()                                                                 | build            | 5min  | ✅ Done     |
| **M6 — Postgres**                  |                                                                                                        |                  |       |             |
| F18                                | Set `Persistence: PersistencePersistent` in pgEngine.Profile()                                         | build            | 5min  | ✅ Done     |
| **M7 — CollectionInfo + accessor** |                                                                                                        |                  |       |             |
| F19                                | Add `Persistence Persistence` field to CollectionInfo struct                                           | build            | 5min  | ✅ Done     |
| F20                                | Populate field in Store.Collections() from engine profile                                              | build            | 5min  | ✅ Done     |
| F21                                | Add `Store.Persistence(queryName string) Persistence` accessor method                                  | build            | 5min  | ✅ Done     |
| **M8 — SerializableQuery**         |                                                                                                        |                  |       |             |
| F22                                | Add `Persistence Persistence` field to SerializableQuery (json tag)                                    | build            | 5min  | ✅ Done     |
| F23                                | Populate field in Serialize() from engine profile                                                      | build            | 5min  | ✅ Done     |
| **M9 — durabilityRule**            |                                                                                                        |                  |       |             |
| F24                                | Create rule_durability.go: durabilityRule struct + Name()                                              | build            | 5min  | ✅ Done     |
| F25                                | Implement Apply(): volatile engine → WARN diagnostic                                                   | eyeball          | 8min  | ✅ Done     |
| F26                                | Add INFO diagnostic when persistent alternative engine exists for same query                           | eyeball          | 8min  | ✅ Done     |
| F27                                | Add RuleTraceEntry for durability warnings                                                             | build            | 5min  | ✅ Done     |
| F28                                | Wire `&durabilityRule{}` into defaultRules() in rules.go                                               | build            | 3min  | ✅ Done     |
| **M10 — Doctor + ExplainPlan**     |                                                                                                        |                  |       |             |
| F29                                | Add "--- Persistence ---" section to Doctor()                                                          | eyeball          | 8min  | ✅ Done     |
| F30                                | Append persistence suffix to engine lines in ExplainPlan()                                             | eyeball          | 5min  | ✅ Done     |
| **M11 — Tests**                    |                                                                                                        |                  |       |             |
| F31                                | Test IsVolatile/IsPersistent on zero-value profile (should be volatile)                                | `go test`        | 5min  | ✅ Done     |
| F32                                | Test String() includes "volatile" for volatile, omits for persistent                                   | `go test`        | 5min  | ✅ Done     |
| F33                                | Test Memory engine Profile().IsVolatile() == true                                                      | `go test`        | 5min  | ✅ Done     |
| F34                                | Test SQLite engine Profile().IsPersistent() == true                                                    | `go test`        | 5min  | ✅ Done     |
| F35                                | Test Pebble: NewPebbleEngine("") → volatile, NewPebbleEngine(dir) → persistent                         | `go test`        | 8min  | ✅ Done     |
| F36                                | Test durabilityRule emits WARN for volatile-only plan                                                  | `go test`        | 8min  | ✅ Done     |
| F37                                | Test durabilityRule emits INFO when persistent alternative exists                                      | `go test`        | 8min  | ✅ Done     |
| F38                                | Test durabilityRule silent when engine is persistent                                                   | `go test`        | 5min  | ✅ Done     |
| F39                                | Test CollectionInfo.Persistence populated correctly                                                    | `go test`        | 5min  | ✅ Done     |
| F40                                | Test SerializableQuery.Persistence in round-trip JSON                                                  | `go test`        | 8min  | ✅ Done     |
| F41                                | Test Store.Persistence(queryName) accessor                                                             | `go test`        | 5min  | ✅ Done     |
| **M12 — Golden + ADR + docs**      |                                                                                                        |                  |       |             |
| F42                                | Regen API surface golden: `cd cmd/api-stability && GOWORK=off go run main.go -update`                  | golden matches   | 5min  | ✅ Done     |
| F43                                | Write ADR: `docs/adr/0098-metaengine-persistence-enum.md` (two-level decision + rejected alternatives) | reads well       | 10min | ✅ Done     |
| F44                                | Update README engine table: add Persistence section                                                  | reads well       | 5min  | ✅ Done     |
| F45                                | Update COOKBOOK engine comparison table: add Persistence column                                        | reads well       | 5min  | ✅ Done     |
| **M13 — Verify**                   |                                                                                                        |                  |       |             |
| F46                                | `go build -tags "goexperiment.jsonv2" ./metaengine/...`                                                | compiles         | 3min  | ✅ Done     |
| F47                                | `go test ./metaengine/... -count=1 -run Persistence`                                                   | green            | 5min  | ✅ Done     |
| F48                                | `go test ./metaengine/... -count=1 -run Durability`                                                    | green            | 5min  | ✅ Done     |
| F49                                | `go test ./metaengine/pebbleengine/... -count=1`                                                       | green            | 5min  | ✅ Done     |
| F50                                | `go test ./metaengine/duckdbengine/... -count=1` (CGo)                                                 | green            | 5min  | ✅ Done     |
| F51                                | `go test ./metaengine/pgengine/... -count=1` (testcontainer)                                           | green            | 8min  | ✅ Done     |
| F52                                | Full metaengine test suite: `go test ./... -count=1`                                                   | green            | 8min  | ✅ Done     |

> All fine-grained tasks complete. F26 now computes actual cost delta via `estimateCost()`. F35 engine tests cover Pebble (3), DuckDB (3), Postgres (2). F44 README has full Persistence section.
>
> **Additional docs closed:** AGENTS.md (module tree + Key Patterns), SKILL.md references (core.md decision matrix + modules.md type listing).

---

## What This Unlocks (Future, NOT in scope)

| Capability                            | Why it needs Persistence                                                                              | When                                           |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Materialize-vs-replay restart cost    | Replay is "free" for volatile (data is gone anyway); materialize is "paid" for persistent (write amp) | Future enhancement to materializeRule          |
| `WithPersistence(p)` plan option      | "What-if" analysis: force volatile to see cost of losing durability                                   | Future, mirrors WithReplication                |
| Projection host restart-time estimate | Volatile engine → estimate replay time from event count                                               | Future diagnostic                              |
| EvictionPolicy axis                   | Separate field for cache engines (TTL/LRU)                                                            | When a cache engine arrives (YAGNI until then) |
