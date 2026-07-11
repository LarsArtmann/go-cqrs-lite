// Package grpc provides gRPC transport adapters for CQRS command and query
// dispatch. It bridges remote gRPC clients to local [command.Dispatcher] and
// [query.Dispatcher] instances.
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
