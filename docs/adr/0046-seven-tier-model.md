# ADR-0046: Seven-Tier Dependency Model

> Originally titled "Four-Tier Dependency Model." Renamed because the model
> actually describes seven tiers (0–6), not four. The "four" referred only to
> the conceptual grouping (primitives → core → infrastructure → composition),
> but the numbered tier table has always listed seven.

**Status:** **Amended** (see addendum below)
**Date:** 2026-07-09 (original), 2026-08-06 (amendment)

## Context

The project historically used a 7-layer system to describe module dependencies. A detailed
analysis revealed this system was fake:

1. `kv/` claims Layer 0 but depends on `codec/` — not a true leaf
2. `event/` claims Layer 1 but depends on Tier 2–4 modules via test deps that leak into go.mod
3. 44 of 68 modules depend on `codec/` — the true hub was invisible in the old system
4. `command/` and `query/` each pull `event/` into their go.mod as `// indirect`
   via `storage/memory/` (test-only dep). Production code has zero `event/`
   imports — the `metadata/` extraction (ADR-0031) broke the real compile
   dependency. The indirect leak remains: `storage/memory/` transitively
   requires `event/`, `snapshot/`, and `query/` for its own purposes.

The 7-layer system provided false confidence that dependencies were well-stratified when
they were not.

## Decision

Replace the fake 7-layer system with an honest **seven-tier model** (0–6):

| Tier | Name               | Rule                                 | Modules |
| ---- | ------------------ | ------------------------------------ | ------- |
| 0    | Primitives         | No internal deps (or same-tier only) | 8       |
| 1    | Core Domain        | Depends on Tier 0                    | 5       |
| 2    | Domain Utilities   | Depends on Tier 0–1                  | 5       |
| 3    | Aggregation        | Depends on Tier 0–2                  | 5       |
| 4    | Infrastructure     | Depends on Tier 0–3                  | 23      |
| 5    | Composition        | Depends on Tier 0–4                  | 9       |
| 6    | Tooling & Examples | Depends on all                       | 13      |

**Total: 68 modules** across 69 `go.mod` files (68 modules + 1 root workspace
placeholder).

### Tier Assignment: Structural + Conceptual

Tier assignment uses two inputs:

1. **Structural** (sets the minimum tier): a module cannot be lower than its
   dependencies allow. `snapshot/` depends on `event/` (Tier 1), so it is at
   least Tier 2.
2. **Conceptual** (can raise the tier): a module's role can place it higher
   than its dependency floor. `otel/` has zero internal deps but is Tier 4
   (infrastructure). `catalog/` has zero deps but is Tier 6 (tooling).
   `idempotency/` has zero deps but is Tier 2 (domain utility).

Same-tier dependencies are allowed (e.g. `kv/` → `codec/`, both Tier 0;
`metaengine/` → `dedup/`, both Tier 0) as long as there is no cycle.

### Tier Diagram

```mermaid
%%{init: {"theme": "base", "themeVariables": {"fontSize": "13px"}}}%%
flowchart TB

  %% ── Tier 6 ──
  subgraph T6["Tier 6 — Tooling & Examples (13)"]
    direction LR
    catalog["catalog/"]
    integration["integration/"]
    benchkit["benchkit/"]
    stackbench["stack/bench/"]
    cqrsgen["cmd/cqrs-gen/"]
    cqrslint["cmd/cqrs-lint/"]
    cqrsbench["cmd/cqrs-bench/"]
    apistability["cmd/api-stability/"]
    doccheck["cmd/doc-check/"]
    exTaskmgr["example/taskmanager/"]
    exGetting["example/getting-started/"]
    exReadme["example/readme-quickstart/"]
    eventtest["event/v4/eventtest/"]
  end

  %% ── Tier 5 ──
  subgraph T5["Tier 5 — Composition (9)"]
    direction LR
    stack["stack/"]
    stackMem["stack/memory/"]
    stackSqlite["stack/sqlite/"]
    stackDuck["stack/duckdb/"]
    stackPebble["stack/pebble/"]
    stackPg["stack/postgres/"]
    stackMysql["stack/mysql/"]
    stackTurso["stack/turso/"]
    system["system/"]
  end

  %% ── Tier 4 ──
  subgraph T4["Tier 4 — Infrastructure (23)"]
    direction LR
    subgraph T4storage["Storage Backends"]
      stMem["storage/memory/"]
      storage["storage/"]
      stPebble["storage/pebble/"]
      stTurso["storage/turso/"]
    end
    subgraph T4sec["Security"]
      signing["signing/"]
      encryption["encryption/"]
    end
    subgraph T4obs["Observability"]
      otel["otel/ ⚡0 deps"]
      prometheus["prometheus/"]
    end
    subgraph T4cross["Cross-Cutting"]
      middleware["middleware/"]
      testutil["testutil/"]
    end
    subgraph T4trans["Transport"]
      trHttp["transport/http/"]
      trGrpc["transport/grpc/"]
    end
    subgraph T4msg["Messaging"]
      watermill["watermill/"]
    end
    subgraph T4me["Metaengine Infra"]
      meAdapter["metaengine/projectionadapter/"]
      mePebble["metaengine/pebbleengine/"]
      meDuck["metaengine/duckdbengine/"]
      mePg["metaengine/pgengine/"]
      meIroh["metaengine/irohengine/"]
      meLoop["metaengine/irohengine/loopback/"]
      meQuic["metaengine/irohengine/quic/"]
    end
    subgraph T4sub["Sub-Stores"]
      idemSql["idempotency/sqlstore/"]
      idemKv["idempotency/kvstore/"]
      schedSql["scheduling/sqlstore/"]
    end
  end

  %% ── Tier 3 ──
  subgraph T3["Tier 3 — Aggregation (5)"]
    direction LR
    decider["decider/"]
    graph["graph/"]
    scenario["scenario/"]
    projectionhost["projectionhost/"]
    listing["listing/"]
  end

  %% ── Tier 2 ──
  subgraph T2["Tier 2 — Domain Utilities (5)"]
    direction LR
    schema["schema/"]
    snapshot["snapshot/"]
    projection["projection/"]
    idempotency["idempotency/ ⚡0 deps"]
    deriver["deriver/"]
  end

  %% ── Tier 1 ──
  subgraph T1["Tier 1 — Core Domain (5)"]
    direction LR
    event["event/"]
    command["command/"]
    query["query/"]
    scheduling["scheduling/"]
    metadata["metadata/"]
  end

  %% ── Tier 0 ──
  subgraph T0["Tier 0 — Primitives (8)"]
    direction LR
    id["id/"]
    codec["codec/ ❗44/68 depend on this"]
    kv["kv/"]
    dedup["dedup/"]
    dispatcher["dispatcher/"]
    retry["retry/"]
    flightrec["flightrecorder/"]
    metaengine["metaengine/"]
  end

  %% ── Same-tier deps (Tier 0) ──
  kv --> codec
  metaengine --> dedup

  %% ── Representative cross-tier deps ──
  %% Tier 1 → Tier 0
  event --> codec
  event --> id
  event --> metadata
  command --> dispatcher
  command --> metadata
  query --> dispatcher
  scheduling --> metadata

  %% Tier 2 → Tier 1
  schema --> event
  snapshot --> event
  projection --> event
  deriver --> command

  %% Tier 3 → Tier 2/1
  decider --> event
  decider --> snapshot
  graph --> projection
  projectionhost --> projection
  projectionhost --> schema

  %% Tier 4 → Tier 3/2/1
  storage --> event
  storage --> kv
  signing --> event
  encryption --> event
  middleware --> command
  middleware --> event
  trHttp --> event
  watermill --> event
  meAdapter --> metaengine
  meAdapter --> projection
  otel -.-> |"conceptual: 0 deps"| T0

  %% Tier 5 → Tier 4
  stack --> storage
  stack --> metaengine
  system --> projectionhost
  system --> metaengine

  %% ── CQRS separation callout ──
  command -.-> |"✗ NO compile dep"| event
  query -.-> |"✗ NO compile dep"| event

  %% ── Styling ──
  classDef zeroDep fill:#2d5a3d,color:#fff,stroke:#4a9d6a
  classDef hub fill:#8B4513,color:#fff,stroke:#d4823c
  classDef noDep fill:#1a3a5c,color:#fff,stroke:#4a8dc4

  class codec hub
  class otel,idempotency,catalog noDep
```

> **Key insights:**
>
> - **`codec/` is the true hub** — 44 of 68 modules depend on it (more than `id/`)
> - **CQRS separation is clean** — `command/` and `query/` have zero `event/`
>   production imports (dotted arrows = no dependency). Shared types live in
>   `metadata/` (ADR-0031).
> - **⚡ = zero-dep modules** tiered by conceptual role, not dependency structure:
>   `otel/` (Tier 4), `idempotency/` (Tier 2), `catalog/` (Tier 6)
> - **Same-tier deps are allowed** — `kv/` → `codec/`, `metaengine/` → `dedup/`

See [`FOUR-TIER-MODEL.md`](../architecture-understanding/FOUR-TIER-MODEL.md) for
the complete module-to-tier mapping with every module listed.

## Enforcement

Three complementary mechanisms enforce the tier model at different granularities:

### 1. Cross-module layer DAG + dependency budgets (`check-module-layers.sh`)

`nix run .#check-layers` runs `scripts/check-module-layers.sh`, which parses
every `go.mod` directly and enforces:

- **Layer ordering**: each module may only depend on modules at its layer or
  lower. The script uses 8 layers (0–7) that map to the 7 conceptual tiers:
  Tier 4 is split into 4a (leaf infra: signing, encryption, otel, memory) and
  4b (composite infra: storage, middleware, transport). This finer granularity
  catches violations the merged tier would miss.
- **Dependency budgets**: each module has a max direct production dep count.
  Test-only deps (gomega, ginkgo, rapid, and auto-detected test-only imports)
  are excluded.
- **Coverage check**: every `go.mod` in the repo must have both a `LAYER` and
  `DEP_BUDGET` entry. Missing entries fail the check — prevents drift when new
  modules are added.

**Why not go-arch-lint for cross-module?** go-arch-lint operates on Go import
paths within a single module. In a `go.work` monorepo with `/v4` import suffixes,
it cannot resolve cross-module dependencies — it treats them as vendor packages.
The bash script parses `go.mod` directly, which is authoritative.

### 2. Intra-module package rules (`go-arch-lint`)

`nix run .#check-arch` runs `scripts/check-arch.sh`, which:

1. Runs `check-module-layers.sh` (above)
2. Runs `go-arch-lint check` per-module for modules with `.go-arch-lint.yml`

Per-module configs exist for: `event/`, `command/`, `kv/`, `storage/`,
`middleware/`, `catalog/`. These enforce intra-module package dependencies
(e.g., `storage/sql/` may not import `storage/eventstore/`).

### 3. External import allow-list (`depguard`)

`.golangci.yml` contains a depguard configuration with a global allow-list of
~90 external packages. `nix run .#lint` (golangci-lint) enforces this. New
external dependencies must be added to the list or lint fails.

## Notable Tier-0 Exceptions

- **`metaengine/` is Tier 0 by design** (ADR-0062). The core planner has zero
  internal deps (stdlib + `database/sql` + `dedup/` only). The bridge to the
  CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).
  Conceptually it aggregates events into projections, but tiering is
  dependency-based.
- **`idempotency/` is Tier 2 conceptually** despite zero internal deps. It
  re-exports `github.com/larsartmann/go-idempotency` — the types (Store,
  MemoryStore, ErrDuplicate) are domain utilities, not primitives.
- **`catalog/` is Tier 6 conceptually** despite zero internal deps. It is a
  documentation generator — tooling, not a library primitive.

## Alternatives Considered

### Keep the old 7-layer system

Rejected: The old system was inaccurate. It claimed clean stratification that
did not exist. Modules were assigned layers by aspiration, not by actual
dependency structure.

### Merge tiers 1–3 into a single "Domain" tier

Rejected: The distinction between Core Domain (event/command/query), Domain
Utilities (schema/snapshot/projection), and Aggregation (decider/projectionhost)
is meaningful. Each tier has different stability guarantees and change rates.
Collapsing them would hide the real dependency distances.

### Pure structural tiering (deps only, no conceptual role)

Rejected: Would place `catalog/` and `otel/` in Tier 0 alongside `id/` and
`codec/`, which is misleading. A documentation generator is not a primitive,
even if it has zero deps. Conceptual role must be able to raise the tier.

```

---

## Addendum 2026-08-06: Metaengine Reclassified

**metaengine/ moves from Tier 0 to Tier 3 (Aggregation).**

### What Changed

The original tier model placed metaengine in Tier 0 because it had zero internal
deps. This was structurally correct but conceptually wrong, and the structural
correctness is now superseded by ADR-0062's addendum (the zero-dep boundary is
removed — metaengine depends on the `Record` type from ADR-0111).

The metaengine is conceptually an **aggregation** layer: it takes Records
(events + commands) and aggregates them into query-optimized projections. This
is the same conceptual tier as `decider/` (aggregates events into state),
`projectionhost/` (aggregates events into read models), and `graph/` (aggregates
events into graph data).

### New Tier Assignment

| Module | Old Tier | New Tier | Reason |
|--------|----------|----------|--------|
| `metaengine/` | 0 | 3 | Depends on Record type (ADR-0111). Conceptually aggregates records into projections. Same tier as decider/, projectionhost/, graph/. |

The engine submodules (`pebbleengine/`, `duckdbengine/`, `pgengine/`,
`irohengine/`, future `badgerengine/`, `dgraphengine/`) remain Tier 4
(Infrastructure) — they provide storage backends, not domain logic.

`metaengine/adttest/` remains Tier 4 (test infrastructure, consumed by engine
modules).

### Impact on Tier Counts

| Tier | Old count | New count | Change |
|------|-----------|-----------|--------|
| 0    | 8         | 7         | -1 (metaengine leaves) |
| 3    | 5         | 6         | +1 (metaengine joins) |
