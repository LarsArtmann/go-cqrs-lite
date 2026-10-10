# ADR-0157: Engine Fleet Operation — Compile-Time Import Selection, Runtime cqrs.yaml

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner (graph-native adoption-closure plan V2, Wave 4)
- Related: [ADR-0123](0123-v5-unification-single-composition-root-universal-engines-auto-projection.md) §3 (driver registry),
  [ADR-0124](0124-operator-driven-layout-planning.md) (operator-driven layout),
  [ADR-0125](0125-developer-priority-is-layout-only.md) (developer priority is layout-only),
  [ADR-0156](0156-graph-edge-labels-at-v5.md) (edge labels),
  [`example/graph-native/composition.go`](../../example/graph-native/composition.go) (the pattern, executable)

## Context

"Where data lives is up to operators at DEPLOYMENT time" is the north-star promise. Today that
promise is delivered by two mechanisms that were never written down as one story, so operators
keep asking the same two questions: *which engines can I choose from?* and *where do I choose?*

1. **Compile time — the app author fixes the candidate set.** Engines are separate modules that
   self-register via `init()` → `metaengine.RegisterDriver` ([ADR-0123](0123-v5-unification-single-composition-root-universal-engines-auto-projection.md) §3).
   A blank import (`_ "…/metaengine/sqliteengine/v4"`) is the only wiring; what is not imported
   is not in the binary. `system/` never imports engines (ADR-0123: no dependency inversion).
2. **Runtime — the operator routes among the compiled-in set.** `system.LoadConfig` reads
   `cqrs.yaml` (plus `CQRS_*` env overrides): `engines.<name>.{driver,dsn,pragmas,priority}`,
   `buses`, `instances` (which engine serves `source-of-truth` vs `projections`), and global /
   per-engine / per-query `priority`. A driver name in YAML that was never compiled in fails at
   boot with an unknown-driver error — the two levels compose loudly, never silently.

The open question was whether to ALSO ship a convenience module ("allengines") that
blank-imports every engine, so apps could route to anything at runtime without choosing imports.

## Decision

### D1 — No "allengines" convenience module. Import selection IS the compile-time story.

Measured on 2026-10-10 (go1.27.1, linux/amd64, blank-import probe modules, proxy-resolved
tagged versions):

| Probe | Binary size | Unique modules | `go mod graph` edges |
| --- | --- | --- | --- |
| sqlite engine only | 12,747,630 B (~12.2 MiB) | 64 | 168 |
| all 10 pure-Go engines | 68,185,216 B (~65.0 MiB) | 221 | 871 |

Blank-importing the full pure-Go fleet is a **5.3× binary tax and 3.5× module-count tax** on
every consumer. A convenience module would force that tax onto all importers to save one import
block. Rejected. The canonical pattern (executable in `example/graph-native/composition.go`) is
one `engines.go`-style import block naming the drivers the deployment may use — typically two:
one durable source-of-truth engine plus one specialized projections engine.

### D2 — The candidate set (membership table).

| Driver | Module | Pure Go | Graph ADT | Notes |
| --- | --- | --- | --- | --- |
| `memory` | core `metaengine` | yes | yes | default; no persistence |
| `sqlite` | `sqliteengine` | yes (modernc) | yes | recursive CTE, iterative fallback |
| `turso` | `tursoengine` | yes | yes | wraps the sqlite engine |
| `postgres` | `pgengine` | yes (pgx) | yes | `WITH RECURSIVE` |
| `mysql` | `mysqlengine` | yes | yes | 8.0+ CTE, probed fallback |
| `badger` | `badgerengine` | yes | yes | prefix-scan BFS |
| `bbolt` | `bboltengine` | yes | **no** | no `ADTGraph` methods |
| `pebble` | `pebbleengine` | yes | **no** | no `ADTGraph` methods |
| `iroh` | `irohengine` | yes (go-sse) | yes | replicated, passthrough/BFS |
| `dgraph` | `dgraphengine` | yes (dgo gRPC) | yes | native `n(depth:)` traversal |
| `duckdb` | `duckdbengine` | **no — CGo** | yes | `//go:build cgo` on registration |
| `bigtable` | `bigtableengine` | yes but heavy | — | 3 direct GCP deps, ~50 indirect; excluded from any convenience-set discussion |

"Pure Go" = builds without a C toolchain. Graph capability is interface-detected
(`GraphAddEdge`/`GraphNeighbors`/`GraphRemoveEdge` on the engine), not Profile-declared — a
driver without the ADT simply fails graph queries with an unsupported error at runtime.

> **Correction (verified from source 2026-10-10, capability-gate session):**
> two refinements to the snapshot above. (1) `iroh` is NOT a registry driver —
> `irohengine` registers no `RegisterDriver` name; it is constructed
> programmatically (`irohengine.Replicated(local, ...)`), as is `graphadapter`
> (`graph-memory`; never a DeploymentConfig driver — 11 registry drivers
> total). (2) "Graph ADT yes" is per-feature: duckdb implements
> `GraphAddEdge` + traversal but neither `GraphRemoveEdge` nor undirected
> traversal. The per-engine truth now lives GENERATED from source and gated in
> [docs/engine-capabilities.md](../engine-capabilities.md)
> (`nix run .#check-engine-capabilities`); this table stays as the decision
> snapshot.

### D3 — Optional app-side build-tag variants are the app's concern, not the library's.

Fleet binaries that want one source tree with per-environment engine sets (e.g. a lean
`CGO_ENABLED=0` scratch image vs a duckdb-enabled analysis image) gate their own import block
behind app-defined `//go:build` tags in their own repo. The library adds no tags, keeps no
engine matrix in `flake.nix` beyond what CI already tests, and promises only: every engine
module compiles standalone, and registration is side-effect-free (`init()` only).

## Consequences

**Positive:** the operator story is two sentences — *authors import candidates, operators pick
in YAML* — with measured evidence that import-minimalism is a real lever (5.3× binary spread).
Binary size stays a first-class deployment artifact; CGo isolation (duckdb) remains the app's
explicit choice.

**Negative:** adding a new engine to a deployment requires an app re-import + rebuild before
the YAML can name it. That is intentional (deployment surface = compiled surface) but must be
stated in operator docs, which this ADR now is.

**Non-goals:** no federation across engines (ADR-0146), no mesh policy enforcement (ADR-0147),
no dynamic plugin loading.
