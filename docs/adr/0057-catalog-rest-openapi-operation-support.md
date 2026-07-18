# ADR-0057: Catalog REST/OpenAPI Operation Support

## Status

Accepted — 2026-07-18

## Context

The catalog module (catalog/v4) originally documented only asynchronous event
flows — commands, events, and queries mapped to EventCatalog MDX pages,
AsyncAPI specs, and D2 diagrams. It had no concept of HTTP REST endpoints.

Consumers like DiscordSync expose both:
- **CQRS events** (Discord Gateway → event store → projections)
- **REST API endpoints** (`GET /api/messages`, `POST /api/backfill/restart`, ...)

The OpenAPI exporter auto-derived paths (`/api/<service>/<message>`) that didn't
match the real HTTP routes, making the generated OpenAPI spec useless for REST
discovery. There was no way to declare:
- The actual HTTP method (GET, POST, PUT, DELETE, PATCH)
- The actual URL path (`/api/users/{id}`)
- Typed response schemas per status code
- Security scheme requirements

## Decision

Add an optional `Operation` field to `Message` that carries explicit HTTP
metadata, consumed by all exporters:

```go
catalog.Command[CreateUserCmd]("user.create",
    catalog.MsgOperation("POST", "/api/users", "201", "400"),
    catalog.Response[UserDTO]("201", "User created"),
    catalog.MsgSecurity("bearerAuth"),
)
```

### Design choices

1. **Opt-in, not default** — Messages without `Operation` keep the auto-derived
   path behavior. Existing catalogs are unaffected.

2. **Operation on Message, not on Service** — Each message independently
   declares its HTTP contract. A service can mix REST and event-only messages.

3. **ResponseSpec is a separate slice** — Not nested inside Operation. This
   allows responses to be validated and exported independently of whether the
   message has an explicit operation (the validation bug fix in v4.0.1).

4. **Generic Response[T]** — Schema derivation uses Go generics + reflection,
   matching the existing Command[T]/Query[T] pattern. No schema strings.

5. **Exporters consume uniformly** — OpenAPI maps to `paths`, AsyncAPI tags
   with `http:GET`/`http:POST`, D2 renders `[POST /api/users]` labels.

6. **Security at catalog + message level** — `SecurityScheme` on the catalog,
   `MsgSecurity` on individual messages. Matches OpenAPI 3.0's model.

### Shortcut helpers

- `DELETE[T]`, `PUT[T]`, `PATCH[T]` — pre-tagged command constructors
- `WithOperation[T]` — composite option combining operation + typed response
- `httptyped` package — framework-agnostic typed request/response envelopes
- `huma` package — adapter for Huma router operation metadata

## Consequences

**Positive:**
- REST endpoints appear in generated documentation with their real paths
- Typed responses give consumers accurate API contracts
- Security requirements surface in the OpenAPI spec
- Catalog.Validate() catches duplicate routes and missing schemas at startup

**Negative:**
- Catalog registration is more verbose for REST-heavy services
- Operation adds a new axis of variation on Message (kind × direction × operation)
- The `httptyped` and `huma` packages are optional dependencies that consumers
  must discover (mitigated by godoc links in the README)

## Alternatives considered

- **Separate RESTCatalog type** — Rejected: would duplicate the entire builder
  API and prevent mixing REST + event messages on one service.
- **OpenAPI annotations on structs** — Rejected: Go struct tags can't express
  HTTP method, path, and multiple response schemas cleanly.
- **Code generation from HTTP handler registration** — Rejected: couples the
  catalog to a specific router framework. The `huma` adapter covers this
  use case for Huma users without making it mandatory.
