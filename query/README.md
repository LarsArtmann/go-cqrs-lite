# query — CQRS Query Dispatch

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/query/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/query/v2)

Typed query dispatch with pagination and middleware chains.

```bash
go get github.com/larsartmann/go-cqrs-lite/query/v2
```

## Quick Start

```go
queries := query.NewDispatcher()
queries.Register("user.get", handler)
result, err := queries.Dispatch(ctx, q)

// Type-safe result
user, err := query.DispatchTyped[*GetUserResult](ctx, queries, q)
```

## Key Types

| Type                 | Purpose                                                   |
| -------------------- | --------------------------------------------------------- |
| `Dispatcher`         | Query dispatcher with handler registry + middleware chain |
| `Query`              | Interface: Type()                                         |
| `BasicQuery`         | Embed for interface satisfaction                          |
| `TypedHandler[Q, R]` | Type-safe handler returning (R, error)                    |
| `Pagination`         | Page/PageSize with Offset(), Validate()                   |
| `PaginatedResult[T]` | Data + TotalCount + HasNext()/HasPrev()                   |
