# Migration Guide — catalog v4.0.0 → v4.0.1

**v4.0.1 is fully backward-compatible.** All new features are opt-in additions —
existing code that doesn't use them will compile and behave identically.

## What's new

v4.0.1 adds **REST / HTTP operation support** to the catalog. You can now
attach explicit HTTP method + path metadata to any message, declare typed
response schemas, and define security schemes. The OpenAPI, AsyncAPI, and D2
exporters all consume this metadata.

## Do I need to change anything?

**No.** If your catalog only registers events/commands/queries without
operations, the exporters will continue to derive default paths exactly as
before. The new features activate only when you call `MsgOperation` or
`Response[T]`.

## Opting in: adding REST operations

### Before (v4.0.0) — auto-derived paths

```go
builder := catalog.NewBuilder("My API", "1.0.0")
builder.AddService("user-svc", "User Service", "1.0.0", "Users",
    catalog.Command[CreateUserCmd]("user.create"),
)
```

OpenAPI path: `POST /api/user-svc/user-create` (auto-derived)

### After (v4.0.1) — explicit REST operation

```go
builder := catalog.NewBuilder("My API", "1.0.0")
builder.AddService("user-svc", "User Service", "1.0.0", "Users",
    catalog.Command[CreateUserCmd]("user.create",
        catalog.MsgOperation("POST", "/api/users", "201", "400"),
        catalog.Response[UserDTO]("201", "User created"),
    ),
)
```

OpenAPI path: `POST /api/users` (explicit)
Response: typed 201 with `UserDTO` schema.

## New packages

| Package     | Purpose                                                                        |
| ----------- | ------------------------------------------------------------------------------ |
| `httptyped` | Generic typed request/response envelopes for framework-agnostic HTTP contracts |
| `huma`      | Adapter for converting Huma router operations into catalog messages            |

## New validation rules

`catalog.Validate()` now checks:

1. **Duplicate operations** — two messages with the same `(method, path)` produce a violation.
2. **Method without path** — an operation with `Method` set but `Path` empty is flagged.
3. **2xx without schema** — a `ResponseSpec` with a 2xx status code but no `Schema` is flagged.

If your existing catalog triggers any of these, they are real issues worth fixing
before generating documentation.

## CLI tool

```bash
go run ./cmd/go-cqrs-lite-catalog -format openapi -o ./docs
go run ./cmd/go-cqrs-lite-catalog -format asyncapi -o ./docs
go run ./cmd/go-cqrs-lite-catalog -format d2 -o ./docs
go run ./cmd/go-cqrs-lite-catalog -format llms -o ./docs
```
