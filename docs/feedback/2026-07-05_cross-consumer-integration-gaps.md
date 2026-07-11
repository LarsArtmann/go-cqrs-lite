# Cross-Consumer Integration Feedback: stack/v4, schema/v4, and the Lifecycle Gap

**From:** Deep integration review of DiscordSync ↔ SwettySwipperWeb
**Date:** 2026-07-05
**Perspective:** Architectural audit of two real production consumers
**Severity:** Two structural gaps (stack lifecycle, schema evolution) + one framing error (PascalCase)

> This document differs from the per-consumer feedback files: it captures
> **structural gaps surfaced only by cross-referencing two consumers against
> each other.** Neither consumer could surface these alone — the gaps emerge
> from how the SDK's module boundaries interact in real multi-service
> architectures.

---

## Context: How These Two Services Integrate

```
DiscordSync (Go, capture-only CQRS)              SwettySwipperWeb (Go, decider CQRS)
├── event/v4                                     ├── event/v4
├── storage/v4 (SQLite + Turso)                  ├── storage/v4 (SQLite)
├── projectionhost/v4                            ├── decider/v4
├── projection/v4                                ├── command/v4
├── catalog/v4 (events only)                     ├── query/v4
├── watermill/v4                                 ├── catalog/v4 (events only)
├── middleware/v4                                ├── middleware/v4
├── codec/v4 (CBOR default)                      ├── idempotency/v4
├── otel/v4 + prometheus/v4                      └── storage/memory/v4 (tests)
├── scenario/v4 + testutil/v4
├── id/v4
└── samber/do/v2 (NOT go-cqrs-lite) ← container

    REST API (GET /api/messages, /api/guilds, /api/channels)
    SSE stream (GET /api/events/stream)
    ──────────────────────────────────────────→  SwettySwipperWeb consumes both
```

The integration review found **3 critical bugs** in the REST API contract, all
caused by gaps in go-cqrs-lite's module design — not by consumer misuse.

---

## Gap 1: `stack/v4` Lacks Lifecycle Primitives That Real Services Need

### The Problem

`stack.Bundle` provides:

- `Bundle.Close()` — pointer-deduplicated resource shutdown
- `Bundle.GracefulClose(ctx)` — drain-then-close
- `Bundle.Debug()` / `Bundle.DebugStructured()` — capability presence

It does NOT provide:

- **Health check aggregation** (no `HealthChecker` interface)
- **Topological shutdown ordering** (closers shut down in registration order, not dependency order)
- **Lazy provisioning** (no factory/DAG — capabilities are pre-built at Bundle construction)

Real services need all three. DiscordSync proves this: it rejected `stack/v4`
entirely and built its own container on `samber/do/v2` (`container.go`, 183
lines) that provides exactly these capabilities.

### The Evidence

DiscordSync's `container.go` declares per-service dependency edges:

```go
do.Provide(i, func(inj do.Injector) (*projectionRuntime, error) {
    _ = do.MustInvoke[*db.DB](inj)           // depends on database
    _ = do.MustInvoke[*storage.EventCapture](inj) // depends on event capture
    return b.projRuntime, nil
})
```

This gives `injector.ShutdownWithContext(ctx)` a topological DAG — the database
shuts down last because everything depends on it. Plus `HealthcheckerWithContext`
aggregates health from every service into `/readyz`:

```go
// /readyz returns 503 with per-service detail when any service is unhealthy:
{"status":"unhealthy","services":{"db":"ping failed","bot":"discord disconnected"}}
```

`Bundle` has no equivalent. `DebugStructured()` reports which capabilities are
_wired_ (presence), not whether they're _healthy_ (liveness). A wired database
that can't ping is "healthy" according to `Bundle`.

### Why samber/do Was Removed From go-cqrs-lite

The CHANGELOG records `samber/do/v2` being removed (commit `f5d36a2`) with the
note _"lighter footprint."_ Multiple status docs list it as _"evaluated and
rejected"_ or _"premature now."_

This was correct for the **library** — go-cqrs-lite should not depend on a DI
library. But the removal left a vacuum: the library has a composition root
(`stack/v4`) that is strictly less capable than what real consumers build
themselves. Every consumer that needs health checks or topological shutdown
must bypass `stack/v4` and build their own container.

### What Should Happen

**Option A — Add lifecycle interfaces to Bundle (minimal, no deps):**

```go
// In stack/bundle.go — no new dependencies, just two interfaces

type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// BundleHealthCheck aggregates all registered health checkers.
func (b *Bundle) HealthCheck(ctx context.Context) map[string]error { ... }

// RegisterHealthChecker associates a name with a health checker.
func (b *Bundle) RegisterHealthChecker(name string, hc HealthChecker) { ... }
```

Options that hand the Bundle a store/backend register its health checker
automatically. Consumers call `bundle.HealthCheck(ctx)` for `/readyz`.

**Option B — Add shutdown ordering via dependency declaration:**

```go
// WithDependency declares that this capability depends on others.
// Shutdown runs in reverse dependency order (dependents first).
func WithDependency(deps ...string) Option { ... }
```

This is simpler than a full DI container but gives topological shutdown
without samber/do.

**Option C — Document that consumers SHOULD use samber/do alongside Bundle:**

If the library doesn't want lifecycle complexity, explicitly say so: "Bundle
handles resource close and drain. For health checks, topological shutdown, and
lazy provisioning, wrap Bundle in your own container (we use samber/do)."

The current state — where `stack/v4` looks like a complete composition root
but silently lacks the lifecycle primitives real services need — is the worst
of both worlds. Consumers either don't know they need more (and discover it in
production) or bypass Bundle entirely (and lose the dedup/drain logic).

### Severity: HIGH

This is why DiscordSync's ADR rejected `stack/v4`. The stated reason ("shared
DB architecture") is a secondary concern. The primary reason is that `Bundle`
provides strictly less lifecycle than `samber/do`, which DiscordSync already
had.

---

## Gap 2: `schema/v4` Is Critically Undervalued by Consumers

### The Problem

DiscordSync's ADR excludes `schema/v4` with this justification:

> _"Go typed structs ARE the schema validation."_

This is **objectively wrong** — it conflates two different concerns:

| Concern                              | What Go structs handle             | What `schema/v4` handles        |
| ------------------------------------ | ---------------------------------- | ------------------------------- |
| Compile-time type checking           | ✅                                 | —                               |
| Current payload shape                | ✅                                 | —                               |
| **Old events after payload changes** | ❌ **BREAKS**                      | ✅ Upcasters transform on load  |
| **Schema version tracking**          | ❌ All events stamped `v1` forever | ✅ `event.WithSchemaVersion(2)` |
| **Breaking field renames**           | ❌ Corruption floods DLQ           | ✅ Transparent upcast chain     |

### The Evidence

DiscordSync has `const EventVersion event.Version = 1` (`types.go:33`). Every
event for all 13 types is created with this single version. There are zero
upcasters registered. `DecodePayloadAuto[P]` in the projection builder
(`builder.go:60`) assumes the current struct shape matches every stored event.

This has worked so far because all payload changes have been **additive** — new
optional fields that decode to zero values. But the first **breaking** change
(field rename, type change, field removal, structural restructure) will:

1. Break `DecodePayloadAuto` for every old event of that type
2. Flood the projection DLQ with Corruption errors
3. Require either an event store wipe + full re-backfill, or a custom
   migration script against the raw payload bytes

### Why This Matters for go-cqrs-lite

The `schema/v4` module is purpose-built for exactly this problem, but
consumers don't understand its value. The SKILL.md and module docs describe
it as "schema validation" (implying compile-time type safety), when its
actual purpose is **runtime schema evolution without data loss**.

### What Should Happen

1. **Rename/reframe the module in docs:** "Schema Evolution" not "Schema
   Validation." The `Validator` type is secondary; the `Upcaster` +
   `VersionedStore` are the primary value.

2. **Add a "when you need this" decision guide to the SKILL.md:**

   > Use `schema/v4` when you have event payloads that may change shape over
   > time. If you ever rename a field, remove a field, change a type, or
   > restructure a payload, old events in your store will fail to decode.
   > `schema.NewUpcaster` + `schema.NewVersionedStore` transform old events
   > to the current shape on load — without modifying stored data or writing
   > migration scripts.
   >
   > If all your events use the current struct shape (new project, or you've
   > never changed a payload), you don't need this yet. But wire it in
   > **before** your first breaking change — after is too late.

3. **Show the failure mode:** Add a recipe showing what happens WITHOUT
   upcasters when a field is renamed — the DLQ floods, projections break.
   Then show the same scenario WITH an upcaster — transparent fix.

4. **Make `event.Version` bumpable in the SDK:** Currently `event.New()` takes
   a version parameter, but there's no SDK-level guidance on when to bump it.
   Add to conventions: "Bump `event.Version` when a payload struct gains a
   field that old events should default, or when any field is
   renamed/removed/restructured."

### Severity: HIGH (ticking bomb)

DiscordSync has months of captured Discord events at `v1`. The first payload
change will be a production incident unless upcasters are wired in
prophylactically. SwettySwipperWeb is at lower risk (memory store, replayable)
but will face the same issue as it grows.

---

## Gap 3: JSON Serialization Convention Is Undocumented

### The Problem

Go's `encoding/json` serializes struct fields using their Go names (PascalCase)
by default. This is **correct** — Go uses PascalCase for exported identifiers,
and JSON is just a transport encoding of Go types.

But there's no SDK-level guidance on this. Consumers independently invent
conventions:

- DiscordSync's `db.*` structs: **no json tags** → PascalCase (correct default)
- DiscordSync's `events.*Payload` structs: **snake_case json tags** → snake_case
- SwettySwipperWeb's `source.*` client structs: **snake_case json tags** → snake_case
- SwettySwipperWeb's `handler/response.go` DTOs: **camelCase json tags** → camelCase

The result: DiscordSync's REST API returns `{"ID":"...", "ChannelID":"..."}`
but SwettySwipperWeb's client expects `{"id":"...", "channel_id":"..."}`.
Every field deserializes to zero. **This is the root cause of 3 critical
integration bugs.**

### Why PascalCase Is the Right Default

| Language   | Native JSON field convention | Why                                               |
| ---------- | ---------------------------- | ------------------------------------------------- |
| Go         | **PascalCase**               | `encoding/json` uses Go field names — zero config |
| Python     | snake_case                   | PEP 8 naming convention                           |
| Ruby       | snake_case                   | Ruby naming convention                            |
| JavaScript | camelCase                    | JS naming convention                              |
| Rust       | snake_case                   | serde default                                     |

Go's default is **PascalCase** because exported identifiers are PascalCase.
Adding `json:"snake_case"` tags to every field is:

1. Extra work on every struct
2. A naming convention from a different language
3. A source of bugs when tags are forgotten (some fields get snake_case, others get PascalCase)

The right contract for Go services is: **let Go's default serialization do its
job.** If a consumer needs a different convention, the consumer adapts — not
the producer.

### What Should Happen

1. **Add a JSON serialization convention to the SKILL.md:**

   > **Convention: Use Go's default JSON serialization (PascalCase).**
   >
   > Go's `encoding/json` uses struct field names as JSON keys. This is the
   > zero-config default and is correct for Go services. Do not add
   > `json:"snake_case"` tags unless you have a specific reason (e.g.,
   > matching an external API contract).
   >
   > For cross-service contracts:
   >
   > - **Producer owns the format.** The service that creates the type
   >   defines its serialization.
   > - **Consumer adapts.** If a consumer needs different field names, it
   >   defines its own struct with the right tags and decodes into that.
   > - Never add json tags to one struct to match another service's convention
   >   — that creates silent drift.

2. **Add the `sql.NullInt64` gotcha to conventions:**

   > **Never return `sql.Null*` types in JSON API responses.** They serialize
   > as `{"Int64": 1920, "Valid": true}` instead of `1920`. Use `*int64` or
   > `*int` for nullable numeric fields in structs that will be JSON-serialized.

3. **Consider a `jsonutil` helper** for consumers that need to convert between
   conventions (e.g., a struct-to-struct mapper that reads PascalCase and
   writes snake_case for external API compatibility).

### Severity: MEDIUM

This didn't cause a production outage because the integration isn't live yet.
But it will cause silent data loss (all fields zero-valued) in any future
cross-service contract where the producer uses default serialization and the
consumer expects a different convention.

---

## Gap 4: `catalog/v4` Documents Events But Not REST Endpoints

### The Problem

Both DiscordSync and SwettySwipperWeb use `catalog/v4` for **event
documentation only**. Neither documents their REST API endpoints.

DiscordSync's catalog (`internal/catalog/catalog.go`) registers 11 event types
with typed payloads via `cataloghtmx.Event[T]()`. It also registers domain
metadata (7 ubiquitous language terms, 1 channel, 1 data store).

It registers **zero** REST endpoints. The `/api/docs` OpenAPI spec documents
event payloads but not the 15+ REST endpoints that consumers actually call.

### The Evidence

SwettySwipperWeb's API client (`source/api_client.go`) defines its own struct
types for DiscordSync's REST responses:

```go
type Guild struct {
    ID      DiscordGuildID `json:"id"`
    Name    string         `json:"name"`
    IconURL string         `json:"icon_url"`
}
```

These types exist **only because** there's no shared contract. If DiscordSync
registered its REST responses in the catalog, SwettySwipperWeb could generate
(or at least validate against) the canonical types.

### What Should Happen

Add REST endpoint support to `catalog/v4`:

```go
// Proposed API (conceptual):
builder.AddEndpoint(catalog.Endpoint{
    Method:   "GET",
    Path:     "/api/guilds",
    Response: catalog.SchemaFromType[db.Guild](),
    Summary:  "List all tracked guilds",
})
```

This would generate proper OpenAPI path definitions alongside the event-driven
AsyncAPI definitions. Consumers could generate client types from the spec,
eliminating the manual struct duplication that caused the integration bugs.

### Severity: MEDIUM

Not blocking, but the catalog's "events-only" scope means REST consumers
must reverse-engineer the API shape from source code. The schema contract
document (`SCHEMA_CONTRACT.md`) is a manual artifact that drifts from
reality — it already documents the wrong integration model (shared DB instead
of HTTP API).

---

## What Already Works (Acknowledged)

These are not affected by the gaps above — they work correctly and should be
preserved:

| Module                            | Assessment                                                                                                                             |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `event/v4`                        | Superb. ISP split (Sink/Source/Journal/SeekableJournal), 5-family error taxonomy, immutable events. Both consumers use it as designed. |
| `decider/v4`                      | Clean Decider + Repository pattern. SwettySwipper uses all of it. DiscordSync correctly excludes it (capture-only system).             |
| `projectionhost/v4`               | Production-grade. Replaced DiscordSync's 297-line custom Runner. Per-projection checkpoints, crash recovery, SQLite DLQ.               |
| `codec/v4`                        | Exemplary. CBOR default (19% smaller, 66% faster), `DecodePayloadAuto` handles mixed JSON+CBOR streams transparently.                  |
| `middleware/v4`                   | Complete and symmetric. Recovery/Logging/Retry/CircuitBreaker/Metrics for all three message types.                                     |
| `id/v4`                           | Branded types prevent ID mixing at compile time. Deterministic IDs for dedup.                                                          |
| `watermill/v4`                    | Solid in-process event bus with middleware support.                                                                                    |
| `storage/v4` + `storage/turso/v4` | SQLite + Turso backends work well. Shared-DB recipe now documented.                                                                    |
| `testutil/v4` + `scenario/v4`     | Property-based test generators and BDD DSL are excellent for projection testing.                                                       |

---

## Prioritized Action Items

| #   | Gap                                              | Severity | Effort | Action                                                                                                                              |
| --- | ------------------------------------------------ | -------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `stack/v4` lacks health checks                   | HIGH     | Medium | Add `HealthChecker` interface + `Bundle.HealthCheck(ctx)` method. ~4h.                                                              |
| 2   | `stack/v4` lacks topological shutdown            | HIGH     | Medium | Add `WithDependency()` option for shutdown ordering, OR document that consumers should wrap Bundle in samber/do. ~4h or ~1h (docs). |
| 3   | `schema/v4` undervalued by consumers             | HIGH     | Low    | Reframe as "Schema Evolution" in docs. Add decision guide + failure-mode recipe. ~2h.                                               |
| 4   | JSON serialization convention undocumented       | MEDIUM   | Low    | Add convention to SKILL.md: "Use Go default (PascalCase)." Add `sql.Null*` gotcha. ~1h.                                             |
| 5   | `catalog/v4` doesn't document REST endpoints     | MEDIUM   | High   | Add endpoint registration API to catalog builder. ~8h.                                                                              |
| 6   | `schema/v4` — no SDK guidance on version bumping | MEDIUM   | Low    | Add to conventions: "Bump `event.Version` when payload struct changes shape." ~30min.                                               |

---

## The Deeper Pattern

Three of these four gaps share a root cause: **go-cqrs-lite optimizes for the
event-sourcing core (events, commands, projections) but underinvests in the
service lifecycle layer that wraps it.**

Real services need:

1. Health checks (Gap 1) → not in `stack/v4`
2. Schema evolution (Gap 2) → exists but consumers don't understand it
3. API contracts (Gap 4) → `catalog/v4` covers events, not REST
4. Serialization conventions (Gap 3) → undocumented, left to consumer guesswork

The event-sourcing core is excellent. The "last mile" between the core and a
running production service is where consumers are on their own.

---

_This feedback is based on a full integration review of two production
consumers. The critique is structural, not subjective — every claim is backed
by code in DiscordSync or SwettySwipperWeb. The goal is to close the gaps so
the next consumer doesn't rediscover them._
