// Package sync provides generic synchronization primitives for building
// local-first and distributed applications with event sourcing.
//
// This package offers transport-agnostic building blocks:
//
//   - [VectorClock] for causal ordering across distributed nodes
//   - [Operation] for representing typed sync operations with generic payloads
//   - [ConflictResolver] and [LWWResolver] for pluggable conflict resolution
//
// The types in this package are domain-agnostic and have zero external dependencies
// beyond the Go standard library.
//
// # Quick Start
//
//	import "github.com/larsartmann/go-cqrs-lite/sync"
//
//	vc := sync.NewVectorClock()
//	vc.Increment(sync.NodeID("node-1"))
//	vc.Increment(sync.NodeID("node-2"))
//	vc.Clone()
//	vc.Cmp(otherVC)
//
//	resolver := sync.NewLWWResolver[*MyEntity](func(e *MyEntity) time.Time {
//	    return e.UpdatedAt
//	})
//	winner, err := resolver.Resolve(&sync.Conflict[*MyEntity]{...})
package sync
