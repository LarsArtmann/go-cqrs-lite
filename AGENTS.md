# Project: go-cqrs-lite

A lightweight, zero-dependency CQRS (Command Query Responsibility Segregation) library for Go.

## Quick Reference

| Item          | Value                           |
| ------------- | ------------------------------- |
| Language      | Go 1.21+                        |
| Test Command  | `go test ./...`                 |
| Build Command | `go build ./...`                |
| Lint Command  | `golangci-lint run`             |
| Dependencies  | google/uuid, cockroachdb/errors |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     APPLICATION LAYER                        │
│   HTTP Handlers ──► Command/Query Dispatchers                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      CQRS-LITE CORE                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Command    │  │    Query     │  │    Event     │       │
│  │  Dispatcher  │  │  Dispatcher  │  │     Bus      │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

## Package Overview

| Package                 | Purpose                                        | Key Types                                  |
| ----------------------- | ---------------------------------------------- | ------------------------------------------ |
| `command/`              | Command handling                               | `Dispatcher`, `Base`, `Handler`            |
| `query/`                | Query handling                                 | `Dispatcher`, `Base`, `Query`              |
| `event/`                | Event sourcing                                 | `Store`, `Bus`, `BaseEvent`                |
| `aggregate/`            | Aggregate roots                                | `Aggregate`, `Base`                        |
| `catalog/`              | Auto-documentation registry, schema reflection | `Registry`, `Catalog`, `SchemaFromType[T]` |
| `catalog/asyncapi/`     | AsyncAPI 3.0 YAML exporter                     | `Exporter`, `Document`                     |
| `catalog/eventcatalog/` | EventCatalog MDX generator                     | `Exporter`                                 |
| `catalog/yaml/`         | Zero-dependency YAML marshaler                 | `Marshal`                                  |

## Design Principles

1. **Zero external dependencies** - Only stdlib + `google/uuid` + `cockroachdb/errors`
2. **Composition over inheritance** - Per Go best practices
3. **Interface-first design** - All core types are interfaces
4. **Context-aware** - All operations accept `context.Context`
5. **Errors as values** - No panics, explicit error returns
6. **File size limits** - Max 250 lines per file

## Code Conventions

- Use `fmt.Errorf` for error messages with context
- Use `errors.New` (cockroachdb/errors) for sentinel errors
- Wrap errors with context using `errors.Wrapf` or `errors.Wrap`
- Context as first parameter in all public functions
- Max 30 lines per function
- No `any` types

## Error Handling Pattern

```go
// Sentinel errors (in errors.go)
var ErrNotFound = errors.New("not found")

// Contextual errors (in functions)
if id == "" {
    return fmt.Errorf("id is required for operation %q", operation)
}

// Error wrapping
if err != nil {
    return errors.Wrapf(err, "failed to process %s", name)
}
```

## Key Patterns

### Command Handler

```go
func (h *Handler) Handle(ctx context.Context, cmd *CreateUser) error {
    if cmd.Email == "" {
        return fmt.Errorf("email is required for user creation")
    }
    // ... implementation
}
```

### Event Creation

```go
event, err := event.NewEvent(
    "user.created",
    aggregateID,
    "User",
    1,
    payload,
    event.WithCorrelationID(correlationID),
)
```

## Test Patterns

- Table-driven tests preferred
- Use `t.Parallel()` for independent tests
- Test error messages contain context
- 100% coverage for core packages

## Common Tasks

| Task              | Command                |
| ----------------- | ---------------------- |
| Run all tests     | `go test ./... -v`     |
| Run with coverage | `go test ./... -cover` |
| Run race detector | `go test ./... -race`  |
| Format code       | `gofumpt -w .`         |
| Imports           | `goimports -w .`       |

## BuildFlow Commands

| Task           | Command                                          |
| -------------- | ------------------------------------------------ |
| Full build     | `buildflow --semantic --fix --dupl-threshold 50` |
| Branching flow | `branching-flow all .`                           |

**Note:** The `--dupl-threshold 50` flag is required due to intentional code duplication in dispatcher Use()/Close() methods (different typed middleware).

## Catalog System Architecture

The `catalog` package provides automatic documentation generation from Go CQRS types to AsyncAPI 3.0 and EventCatalog formats.

### Three-Layer Design

```
┌──────────────────────────────────────────────────────┐
│                   catalog (core)                      │
│  types.go — Message, Service, Domain, Channel, Schema │
│  schema.go — SchemaFromType[T]() via reflect          │
│  registry.go — Thread-safe Registry, Build() → Catalog│
└──────────────────────┬───────────────────────────────┘
                       │ Catalog (immutable IR)
           ┌───────────┴───────────┐
           ▼                       ▼
┌─────────────────────┐  ┌─────────────────────────┐
│ catalog/asyncapi/   │  │ catalog/eventcatalog/   │
│ AsyncAPI 3.0 YAML   │  │ MDX files on disk       │
│ Document.MarshalYAML│  │ services/{id}/index.mdx │
│ Document.MarshalJSON│  │ schemas/schema.json     │
└─────────────────────┘  └─────────────────────────┘
```

### Key Design Decisions

1. **Custom YAML marshaler** (`catalog/yaml/yaml.go`) — Zero new dependencies. Uses `reflect` to handle structs (via `marshalFields` with `[]structField`), maps, slices, primitives. Structs must use `marshalFields` not `marshalMap` to avoid `reflect.Value` being treated as a terminal type.

2. **Reflection-based schema generation** — `SchemaFromType[T any]() *Schema` uses `reflect.TypeOf` to inspect struct fields. Reads `json` (name + omitempty), `doc`/`description` (description), and `format` (format) struct tags. Returns `*Schema` with no error.

3. **Type alias for MarshalJSON** — AsyncAPI `Document.MarshalJSON()` uses `type alias Document` to break infinite recursion when calling `json.MarshalIndent`.

4. **Registry pattern** — Thread-safe with `sync.RWMutex`. `AddService` merges messages into existing services. `Build()` produces an immutable `*Catalog`.

5. **AsyncAPI mapping** — Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive`. Channel addresses via `toDotAddress` (CamelCase → dot.separated).

6. **EventCatalog structure** — MDX files with YAML frontmatter (`---` delimited). `schema.json` only created when schema is non-nil. `eventcatalog.config.js` generated at root.

### CRITICAL: GOWORK=off

All `go` commands must use `GOWORK=off` prefix because `/Users/larsartmann/go.work` exists but does not include go-cqrs-lite.

```
GOWORK=off go test ./... -count=1
GOWORK=off go build ./...
```

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Coding standards
- CQRS patterns from: ChastityAPI, Cyberdom, Domination, GmbHG
