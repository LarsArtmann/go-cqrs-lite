# stack/bbolt

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4)

> **Deprecated:** removed in v5 (ADR-0123). `system.New` is the single
> composition root — pass the equivalent engine and codec options there.

One-call [stack.Bundle](../) preset backed by [bbolt](../../storage/bbolt/) (B+tree, pure Go).

```bash
go get github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4
```

## Quick Start

```go
import bolt "github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"

bundle, err := bolt.New("myapp.db")
defer bundle.Close()

// bundle has: EventStore, SnapshotStore, CheckpointStore, ReadModels,
//             CommandStore, QueryStore, EventBus (watermill GoChannel)
```

## Options

### WithDurability

```go
bundle, _ := bolt.New("myapp.db",
    bolt.WithDurability(stack.DurabilityRelaxed))
// Relaxed: NoSync=true + NoFreelistSync=true (faster, data loss on crash)
// Normal/Strict: default sync-on-commit (bbolt always fsyncs)
```

### WithLogger

```go
bundle, _ := bolt.New("myapp.db",
    bolt.WithLogger(myLogger))
```

## Capabilities

| Capability  | Value |
| ----------- | ----- |
| Persistent  | yes   |
| Embedded    | yes   |
| Distributed | no    |
| OLAP        | no    |
| CGoRequired | no    |
| SyncEnabled | n/a   |

## When to Choose bbolt

- **Predictable write latency** matters more than peak write throughput
- **Pure Go** is required (no CGo)
- **Point-read-heavy** workloads (B+tree excels at random reads)
- **Benchmarking** B+tree vs LSM (Pebble) tradeoffs

## When NOT to Choose bbolt

- **High write concurrency** is critical (single-writer model)
- **Large datasets** with heavy scans (LSM compaction is more efficient)
- You need **concurrent writers** (use Pebble or SQLite instead)
