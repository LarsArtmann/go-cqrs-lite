# Design Spike: Remaining Raw Ideas

## Multi-Tenant Event Store (Schema-per-Tenant)

**Status:** Raw Idea

Each tenant gets an isolated event store (separate database, schema, or table prefix). A `TenantAwareStore` wraps the factory pattern:

```go
type TenantStoreFactory func(tenantID string) (event.Store, error)
type TenantAwareStore struct {
    factory TenantStoreFactory
    cache   map[string]event.Store
}
```

**Consideration:** Adds operational complexity (N databases to manage). Only for multi-tenant SaaS with strict isolation requirements.

---

## Automatic Migration Generator for Schema Evolution

**Status:** Raw Idea

Extends `cmd/cqrs-gen` to generate upcaster functions from struct diffs:

```bash
cqrs-gen -mode=upcaster -old=user_v1.json -new=user_v2.json
# Generates: user_upcaster_v1_to_v2.go
```

**Consideration:** Requires a canonical schema representation (JSON Schema). The existing `schema.Upcaster` provides the runtime; only the codegen is missing.

---

## Property-Based Integration Testing with State Machine Verification

**Status:** Raw Idea

Model the CQRS system as a state machine: commands are inputs, events are outputs, aggregate state is the model. Use `pgregory.net/rapid` to generate random command sequences and verify that:

1. Fold(events) == model state after each command
2. Events are deterministic for the same (state, command) pair
3. Optimistic concurrency violations are correctly rejected

**Consideration:** The tombstone property tests (already implemented) are a start. Full state-machine verification requires per-aggregate model definitions.

---

## Performance Regression Dashboard

**Status:** Raw Idea

Track benchmark results over time. CI runs `nix run .#bench` and uploads results. A dashboard (simple HTML) shows trends.

**Consideration:** The CI workflow already runs benchmarks and uploads artifacts. A comparison view (using `benchstat`) across commits would be valuable but is a tooling concern, not a library feature.

---

## Evaluation: go-composable-business-types Adoption

**Status:** Evaluation Complete — **Not Recommended**

`go-composable-business-types` (already in the depguard allow list) was evaluated for providing richer domain types (Email, PhoneNumber, etc.).

**Verdict:** Not adopted. The library's job is to provide CQRS/ES primitives, not domain types. Consumers bring their own domain types. Adding opinionated business types would violate the "library, not framework" principle and create coupling. The depguard entry remains because it's available for consumers who want it, but the library itself doesn't import it.
