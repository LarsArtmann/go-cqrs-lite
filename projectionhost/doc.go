// Package projectionhost provides a managed lifecycle for projection workers.
//
// A Host reads events from a [event.SeekableJournal], applies them to registered
// [projection.Projection] handlers, tracks per-projection checkpoints via
// [event.CheckpointStore], and handles failures with automatic restart and
// exponential backoff. Poison messages that exceed a configurable retry
// threshold are captured to a [DeadLetterStore] and the checkpoint advances,
// preventing a single bad event from blocking the entire stream.
//
// This is the "last loop every consumer rewrites" — per-projection goroutines,
// crash auto-restart, health/liveness exposure, and graceful drain on shutdown —
// wrapped in a single embeddable component that stays a library, not a framework.
//
// # Quick Start
//
//	store := storage.NewMemoryStore()
//	journal := store // implements event.SeekableJournal
//	cpStore := storage.NewMemoryCheckpointStore()
//
//	host, _ := projectionhost.New(journal, cpStore)
//	host.Register(&MyProjection{})
//	host.Register(&AnotherProjection{})
//
//	go host.Start(ctx)
//	// ... process events ...
//	host.Stop() // graceful drain
package projectionhost
