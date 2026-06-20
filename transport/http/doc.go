// Package http provides HTTP transport adapters for CQRS event streams.
//
// SSEBroker bridges an event.Bus to Server-Sent Events HTTP clients,
// enabling real-time event delivery to browsers and other HTTP consumers.
//
// This module implements ADR-0025's transport/http/ boundary.
// Future transports (gRPC, NATS, Redis) will live as sibling modules under transport/.
package http
