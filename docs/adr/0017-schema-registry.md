# ADR-0017: Schema Registry for Event Validation

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-14   |
| Status  | Accepted     |
| Decider | Lars Artmann |

## Context

Event payloads evolve over time. Producers add fields, remove fields, and change
types. Without validation, malformed events can corrupt projections and crash
consumers.

The library already has:

- `schema/` module for upcasting (transforming old event versions on load)
- `catalog/schema/` for JSON Schema generation from Go types via reflection
- `codec/` for encoding/decoding (JSON, CBOR, Raw)

What's missing is runtime validation: rejecting events whose payloads don't match
a registered schema before they enter the system.

## Decision

**Add a `schemavalidator` middleware** that validates event payloads against
JSON Schemas registered by event type.

### Design

```go
// Registration: register a JSON Schema for each event type
validator := schemavalidator.New()
validator.Register("user.created", userCreatedSchema)
validator.Register("user.deleted", userDeletedSchema)

// As publish middleware: reject invalid events before they're stored
bus.UsePublish(validator.PublishMiddleware())

// As subscribe middleware: reject invalid events before they reach handlers
bus.Use(validator.SubscribeMiddleware())
```

### Implementation as Middleware (Not Store Wrapper)

**Why middleware, not a Store wrapper?**

1. **Store is format-agnostic** — it stores bytes, not typed payloads. Adding
   schema validation to Store would break the ISP principle.
2. **Middleware is composable** — consumers can add/remove validation without
   changing their store implementation.
3. **Flexible placement** — validate on publish (write path) or subscribe (read
   path) or both.

### Schema Source

Schemas are auto-generated from Go types using the existing `catalog/schema.FromReflect()`
function. Consumers register schemas at startup:

```go
validator.Register("user.created",
    schema.FromReflect(reflect.TypeFor[UserCreated]()))
```

### Error Handling

Invalid events return a `Rejection` error (from go-error-family), allowing
retry middleware to distinguish validation failures from transient errors.

## Consequences

- **+** Runtime validation catches malformed events before they cause damage
- **+** Auto-generated schemas eliminate manual schema maintenance
- **+** Middleware approach preserves Store's format-agnostic design
- **-** Performance overhead: JSON Schema validation on every event (~5-50µs)
- **-** Schema registration is manual (could be auto-wired with codegen)

## Future Extensions

- Schema versioning with compatibility checks (backward/forward compatibility)
- Integration with external schema registries (Confluent, Apicurio)
- Code generation for type-safe event builders from schemas

## References

- [JSON Schema Specification](https://json-schema.org/)
- `catalog/schema/` — existing JSON Schema generation
- `schema/` — existing upcasting for version migration
