// Package cqrsproto holds the generated gRPC wire types and services for
// transport/grpc (package cqrs.v1), produced by protoc from cqrs.proto.
//
// Payloads are JSON-encoded bytes matching the watermill protocol. It defines
// unary CommandService.Dispatch and QueryService.Ask, and a server-streaming
// EventService.Subscribe. Application code uses transport/grpc, not this
// package directly.
package cqrsproto
