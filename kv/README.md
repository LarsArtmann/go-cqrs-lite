# kv — Backend-Agnostic Key-Value Store Interface

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/kv/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/kv/v2)

Minimal interface for embedded key-value stores with ordered iteration and atomic batch writes. No existing Go KV meta-API (gokv, valkeyrie) provides all three operations an event store needs: iteration, batch, and byte-slice keys.

```bash
go get github.com/larsartmann/go-cqrs-lite/kv/v2
```

## Interfaces

| Interface    | Methods                                    | Purpose                                  |
| ------------ | ------------------------------------------ | ---------------------------------------- |
| `Store`      | `Reader` + `Writer` + `Close`              | Full read-write access                   |
| `Reader`     | `Get`, `Has`, `NewIterator`                | Read-only access                         |
| `Writer`     | `Set`, `Delete`, `Batch`                   | Write access                             |
| `Iterator`   | `Next`, `Key`, `Value`, `Error`, `Close`   | Ordered key-value iteration (snapshot)   |
| `Batch`      | `Set`, `Delete`, `Commit`, `Close`         | Atomic multi-key writes                  |

## Usage

```go
s := kv.NewMemStore()
defer s.Close()

// Single operations
s.Set([]byte("user:1"), []byte("alice"))
val, _ := s.Get([]byte("user:1"))

// Atomic batch
batch, _ := s.Batch()
batch.Set([]byte("a"), []byte("1"))
batch.Delete([]byte("old"))
batch.Commit()

// Prefix iteration (lexicographic order)
iter, _ := s.NewIterator([]byte("user:"))
defer iter.Close()
for iter.Next() {
    fmt.Printf("%s = %s\n", iter.Key(), iter.Value())
}
```

## MemStore

The in-memory implementation is safe for concurrent use and returns point-in-time snapshots from `NewIterator`. All public accessors (`Get`, `Set`) defensively clone byte slices.
