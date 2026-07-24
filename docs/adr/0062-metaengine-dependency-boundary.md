# ADR-0062: Metaengine Dependency Boundary — Subpackage for event/projection Integration

| Field        | Value                                                            |
| ------------ | ---------------------------------------------------------------- |
| Status       | Accepted                                                         |
| Date         | 2026-07-25                                                       |
| Deciders     | Lars Artmann                                                     |
| Related      | ADR-0061 (SQLite engine), ADR-0046 (seven-tier model)            |
| Supersedes   | —                                                                |

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
