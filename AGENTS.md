# Project: go-cqrs-lite

A lightweight, zero-dependency CQRS (Command Query Responsibility Segregation) library for Go.

## Quick Reference

| Item | Value |
|------|-------|
| Language | Go 1.21+ |
| Test Command | `go test ./...` |
| Build Command | `go build ./...` |
| Lint Command | `golangci-lint run` |
| Dependencies | google/uuid, cockroachdb/errors |

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

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `command/` | Command handling | `Dispatcher`, `Base`, `Handler` |
| `query/` | Query handling | `Dispatcher`, `Base`, `Query` |
| `event/` | Event sourcing | `Store`, `Bus`, `BaseEvent` |
| `aggregate/` | Aggregate roots | `Aggregate`, `Base` |

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

| Task | Command |
|------|---------|
| Run all tests | `go test ./... -v` |
| Run with coverage | `go test ./... -cover` |
| Run race detector | `go test ./... -race` |
| Format code | `gofumpt -w .` |
| Imports | `goimports -w .` |

## References

- [HOW_TO_GOLANG.md](https://github.com/larsartmann/library-policy) - Coding standards
- CQRS patterns from: ChastityAPI, Cyberdom, Domination, GmbHG
