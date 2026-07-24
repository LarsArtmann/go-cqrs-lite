# transport/grpc — gRPC Transport for Command and Query Dispatch

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/transport/grpc/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/transport/grpc/v4)

Remote command and query dispatch over gRPC. Transparently bridges local `command.Dispatcher` and `query.Dispatcher` to remote clients. The bidirectional alternative to SSE.

```bash
go get github.com/larsartmann/go-cqrs-lite/transport/grpc/v4
```

## Quick Start

### Server

```go
import (
    "google.golang.org/grpc"
    cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
)

srv := grpc.NewServer()
cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)
cqrsgrpc.RegisterQueryService(srv, qDispatcher)
```

### Client

```go
conn, _ := grpc.NewClient("host:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()))

cmdClient := cqrsgrpc.NewCommandClient(conn)
err := cmdClient.Dispatch(ctx, command)

qryClient := cqrsgrpc.NewQueryClient(conn)
result, err := qryClient.Query(ctx, query)
```

### Custom Codec (CBOR for smaller payloads)

```go
// Only query entry points accept a codec option:
srv := grpc.NewServer()
cqrsgrpc.RegisterQueryService(srv, qDispatcher,
    cqrsgrpc.WithCodec(codec.CBORCodec{}))

conn, _ := grpc.NewClient(addr, ...)
qryClient := cqrsgrpc.NewQueryClient(conn,
    cqrsgrpc.WithCodec(codec.CBORCodec{}))
```

## API

### Server Registration

| Symbol                              | Description                                     |
| ----------------------------------- | ----------------------------------------------- |
| `RegisterCommandService(srv, disp)` | Registers a gRPC command service on the server. |
| `RegisterQueryService(srv, disp)`   | Registers a gRPC query service on the server.   |

### Client Construction

| Symbol                   | Description                                         |
| ------------------------ | --------------------------------------------------- |
| `NewCommandClient(conn)` | Returns a client that dispatches commands remotely. |
| `NewQueryClient(conn)`   | Returns a client that dispatches queries remotely.  |

### Options

| Symbol         | Description                                                                |
| -------------- | -------------------------------------------------------------------------- |
| `Option`       | `func(*config)` — query server/client only.                                |
| `WithCodec(c)` | Sets the wire codec (default: `codec.JSONCodec{}`). Both sides must match. |

## Design

- **Transparent remote dispatch**: The client implements the same `Dispatch` interface, so callers see no difference between local and remote dispatch.
- **`Option` type**: Only the two query entry points (`RegisterQueryService`, `NewQueryClient`) accept an `Option` for codec configuration. Command entry points do not — command payloads travel as metadata strings.
- **No format negotiation**: Server and client must use the same codec. Mismatch causes unmarshal failure. Default is JSON for backward compatibility.
- **Error mapping**: CQRS error families (Rejection, Conflict, Transient, Infrastructure, Corruption) are mapped to appropriate gRPC status codes.

## Related Modules

- [**command**](../../command/README.md) — `command.Dispatcher` is the dispatch target
- [**query**](../../query/README.md) — `query.Dispatcher` is the dispatch target
- [**codec**](../../codec/README.md) — JSON and CBOR codecs for wire encoding
- [**transport/http**](../http/README.md) — SSE for unidirectional event streaming
