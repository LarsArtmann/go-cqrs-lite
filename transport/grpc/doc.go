// Package grpc is DEPRECATED (ADR-0127). Removal at v5. New projects must
// not import this module.
//
// This package provided gRPC transport adapters for CQRS command and query
// dispatch, bridging remote gRPC clients to local [command.Dispatcher] and
// [query.Dispatcher] instances. It had no internal consumers and duplicated
// delivery concerns the library doctrine ("not a framework — no opinionated
// transport") assigns to dedicated libraries:
//
//   - Broker transport: the watermill/ module bridges event.Bus and
//     command.Bus to any Watermill-compatible broker (NATS, Redis, Kafka)
//     via NewEventPublisher / NewCommandPublisher / WithBackend.
//   - Consumers needing plain gRPC should bridge their own dispatcher over
//     grpc-go directly — the transport is a thin adapter, not domain logic.
//
// # Server
//
// Register the CQRS services on a gRPC server:
//
//	import cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
//
//	srv := grpc.NewServer()
//	cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)
//	cqrsgrpc.RegisterQueryService(srv, qDispatcher)
//
// # Client
//
//	Create a remote dispatcher backed by a gRPC connection:
//
//	conn, _ := grpc.NewClient(addr,
//	    grpc.WithTransportCredentials(insecure.NewCredentials()))
//	cmdClient := cqrsgrpc.NewCommandClient(conn)
//	err := cmdClient.Dispatch(ctx, command)
package grpc
