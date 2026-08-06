# ADR-0062: Metaengine Dependency Boundary — Subpackage for event/projection Integration

| Field      | Value                                                 |
| ---------- | ----------------------------------------------------- |
| Status     | **Amended** (see addendum below)                      |
| Date       | 2026-07-25 (original), 2026-08-06 (amendment)         |
| Deciders   | Lars Artmann                                          |
| Related    | ADR-0061 (SQLite engine), ADR-0046 (seven-tier model) |
| Supersedes | —                                                     |

## Context

The `metaengine/v4` module was designed as a **zero-dependency** core: it depends
only on the standard library and test frameworks (ginkgo/gomega). This is a
deliberate architectural choice — the metaengine is the lowest tier in the
module hierarchy, and keeping it dep-free means:

1. Consumers can adopt the metaengine planner without pulling in event sourcing.
2. The cost model and ADT abstractions stay pure — no accidental coupling to
   event lifecycle or projection infrastructure.
3. The module can be extracted to a standalone repo without untangling imports.

When the projection adapter (M10) was implemented, it required `event/v4` (for
`event.Event`, `event.Type`) and `projection/v4` (for the `projection.Projection`
interface). Initially these deps were added directly to `metaengine/go.mod`,
breaking the zero-dep boundary and causing `go mod tidy` failures under
`GOWORK=off` (the published `event/v4.1.0` tag references intermediate sibling
versions that were never tagged).

## Decision

**Move the projection adapter to a separate subpackage:
`metaengine/projectionadapter/`.**

This follows the same pattern as:

- `idempotency/kvstore/` — KV-backed idempotency pulls `kv/v4`
- `idempotency/sqlstore/` — SQL-backed idempotency pulls `modernc.org/sqlite`

The core `metaengine/v4` module remains zero-dep. The adapter lives in its own
Go module (`metaengine/projectionadapter/v4`) that depends on `event/v4`,
`projection/v4`, `projectionhost/v4`, and the parent `metaengine/v4`.

To enable the adapter to derive event types without accessing unexported fields,
`Store.EventTypes() []string` was added to the core module's public API.

## Alternatives Considered

### A. Accept the dependency in core metaengine

**Rejected.** This violates the zero-dep design principle and drags the entire
event/ transitive dependency tree (codec, id, metadata, schema, snapshot) into
every metaengine consumer. It also breaks `GOWORK=off` builds for metaengine
itself, because `event/v4.1.0`'s published `go.mod` references intermediate
sibling tags that don't exist on the remote.

### B. Define a local projection interface in metaengine

**Rejected.** This would create a parallel type that consumers must adapt to the
real `projection.Projection`. It adds friction without value — the adapter
subpackage achieves the same isolation with zero consumer overhead.

### C. Tag all missing intermediate versions

**Considered but out of scope.** The missing tags (`codec/v4.0.4`, `id/v4.0.3`,
etc.) are a pre-existing release hygiene problem affecting multiple modules.
Fixing it would require finding the exact commit for each module at each version
and tagging them. The subpackage approach with workspace-local `replace`
directives sidesteps this entirely until metaengine gets its first stable tag.

## Consequences

- **Positive:** Core metaengine stays zero-dep. The adapter is opt-in — consumers
  who don't need projection host integration don't pull `event/` or `projection/`.
- **Positive:** Clear precedent for future metaengine integrations (e.g., a
  Watermill bridge would be `metaengine/watermillbridge/`).
- **Negative:** The `projectionadapter` module requires workspace-local `replace`
  directives until `metaengine/v4` gets its first tag. This is documented in the
  module's `go.mod` and is consistent with how untagged workspace modules work.
- **Neutral:** `Store.EventTypes() []string` is now part of the public API. This
  is a generally useful accessor that any integration adapter can use.

---

## Addendum 2026-08-06: Zero-Dependency Boundary Is Wrong

**The zero-dependency principle for metaengine core is superseded.**

### What Was Wrong

The original decision assumed the metaengine is a **generic storage planner**
that should not understand event sourcing. This was a fundamental
misunderstanding of the metaengine's purpose. The metaengine is — and should
always have been — **the Event Sourcing projection planner**: the system that
takes Commands, Events, and Queries as input and automatically decides what
materialized views to build, where to store them, and whether to materialize or
replay.

The zero-dependency boundary:
1. **Kneecapped the planner** — it sees events as `any` blobs, unable to reason
   about event types, causality, tombstones, or command-to-event relationships.
2. **Blocked graph unification** — ADR-0077 kept GraphBackend and graph/
   separate solely because importing graph/ would violate the zero-dep boundary.
3. **Required an adapter layer with translation loss** — projectionadapter/
   exists only to bridge the artificial gap between the planner and event types.
4. **Did not match reality** — pgengine already connects to a server (Postgres),
   proving the "embedded-library model" was never a real constraint.

### New Principle: No Artificial Boundary

Modules are split by **deployment concern** (CGo isolation, heavy external
dependencies like database drivers), not by an arbitrary purity rule. The
metaengine core depends on the shared `Record` type (ADR-0111) and can depend on
any module whose types it needs to reason about during planning.

**The new dependency rule:** a module's dependencies are justified if they make
the planner better at its job. Modules are split when:
- An external dependency requires CGo (DuckDB)
- A dependency adds significant binary weight (Pebble, Badger)
- A dependency requires a running server (Postgres, Dgraph)
- Isolation improves consumer choice (consumers who don't need Dgraph don't
  import the dgraphengine module)

### What Changes

- metaengine core gains a dependency on the `Record` type (ADR-0111)
- SQLite engine moves to `metaengine/sqliteengine/` (ADR-0115) — the only
  reason it was in core was `database/sql`, which is stdlib but conceptually
  belongs with the engine implementations
- `metaengine/projectionadapter/` is simplified or absorbed — the core no
  longer needs an adapter to understand events
- New engine modules (badgerengine, dgraphengine) follow the same deployment
  isolation pattern as pebbleengine, pgengine, etc.

### What Stays the Same

- The `projectionadapter/` subpackage pattern for heavy external deps
  (`event/`, `projection/`, `projectionhost/`) remains valid for modules that
  genuinely need isolation
- The cost-based planner, ADT classification, and Engine interface are unchanged
- Cross-engine parity testing via `adttest/` is unchanged
