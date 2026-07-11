# AI Agent Plan: kv/ Interface Module + Pebble Refactor

> **Status: ⏭ DESCOPED** · **Decision date:** 2026-06-15
>
> This plan was **not executed**. Pebble SnapshotStore and CheckpointStore were
> implemented directly on `*pebble.DB` (commit `e0e2418e`) without a `kv.Store`
> abstraction layer. The kv/ module was descoped because:
>
> 1. The abstraction added complexity without immediate consumer demand for
>    alternative KV backends (BadgerDB, bbolt).
> 2. Pebble's native API already provides iteration, batch writes, and byte-slice
>    keys — the `kv.Store` interface would be a thin wrapper with no added value.
> 3. The "Everything Else" plan (EVERYTHING_ELSE_AGENT_PLAN.md) implemented
>    SnapshotStore and CheckpointStore directly, making this plan's prerequisite
>    unnecessary.
>
> This plan remains as reference for future KV backend interchangeability work
> if/when a second KV backend (BadgerDB, bbolt) is added.
>
> **For:** AI coding agent (Crush, Claude, GPT, etc.) · **Estimated time:** 3-4 hours
>
> **Prerequisite for:** Pebble SnapshotStore, Pebble CheckpointStore, future badger/bbolt backends.
>
> **Goal:** Extract the `kv/` interface module from `kv-store-abstraction-research.md`, then
> refactor Pebble to use it — enabling any KV backend (Pebble, Badger, bbolt) to implement
> event.Store, SnapshotStore, CheckpointStore via a thin adapter.

## Background

The research (`docs/research/kv-store-abstraction-research.md`) surveyed every Go KV meta-API.
None fit — all lack iteration, batch writes, or byte-slice keys. The recommendation: define
our own minimal `kv/` module with 14 methods across 5 interfaces.

**This plan must execute BEFORE the "Everything Else" plan's Pebble SnapshotStore/CheckpointStore
steps.** Those become trivial adapters on top of `kv.Store` instead of reimplementing KV logic.

Without `kv/`: each KV backend (pebble, future badger, future bbolt) reimplements event store
logic from scratch — duplicated prefix scans, batch logic, key encoding.

With `kv/`: event store logic lives once, depending only on `kv.Store`. Backends contribute a
~80-line adapter. This is the "data store aware but independent" principle from the architecture docs.

## The Interface (Already Designed — Don't Redesign)

From `kv-store-abstraction-research.md` §5. This is the DECIDED interface:

```go
package kv

// Store is the core key-value store interface.
// Keys are byte slices with lexicographic ordering.
type Store interface {
    Reader
    Writer
    io.Closer
}

// Reader provides read access to the store.
type Reader interface {
    Get(key []byte) ([]byte, error)
    Has(key []byte) (bool, error)
    NewIterator(prefix []byte) (Iterator, error)
}

// Writer provides write access to the store.
type Writer interface {
    Set(key, value []byte) error
    Delete(key []byte) error
    Batch() (Batch, error)
}

// Iterator yields key-value pairs in lexicographic order.
type Iterator interface {
    Next() bool
    Key() []byte
    Value() []byte
    Error() error
    Close() error
}

// Batch collects write operations for atomic commit.
type Batch interface {
    Set(key, value []byte) error
    Delete(key []byte) error
    Commit() error
    Close() error
}
```

**Total: 14 methods across 5 interfaces. Zero production dependencies.**

## Step-by-Step Plan

### Step 1: Create kv/ module (45min)

**1.1** Create the module structure:

```
kv/
├── go.mod           ← module github.com/larsartmann/go-cqrs-lite/kv/v4, go 1.26.3, zero deps
├── doc.go           ← package doc with usage examples
├── kv.go            ← Store, Reader, Writer, Iterator, Batch interfaces
├── errors.go        ← ErrNotFound, ErrClosed sentinel errors
├── mem.go           ← in-memory implementation for testing
├── mem_test.go      ← tests for in-memory impl
└── kv_test.go       ← interface contract tests
```

**1.2** `kv/kv.go` — Define the 5 interfaces exactly as above.

**1.3** `kv/errors.go`:

```go
package kv

import "errors"

var (
    ErrNotFound = errors.New("kv: key not found")
    ErrClosed   = errors.New("kv: store is closed")
)
```

**1.4** `kv/mem.go` — In-memory implementation using `sort.Slice` or a B-tree:

```go
// memStore implements kv.Store using an in-memory sorted map.
// Keys are sorted lexicographically for correct iteration order.
type memStore struct {
    mu    sync.RWMutex
    data  map[string][]byte
    keys  [][]byte  // sorted key list (or use a btree)
    closed bool
}
```

**1.5** `kv/doc.go` — Package documentation with usage example:

```go
// Package kv defines a minimal key-value store abstraction for
// embedded KV backends (Pebble, BadgerDB, bbolt).
//
// # Why This Exists
//
// No existing Go KV meta-API supports iteration, atomic batch writes,
// and byte-slice keys with lexicographic ordering — all three are
// required for event store implementations.
//
// # Quick Start
//
//	store := kv.NewMemStore()
//	defer store.Close()
//	store.Set([]byte("key"), []byte("value"))
//	val, _ := store.Get([]byte("key"))
package kv
```

**1.6** Add `./kv` to `go.work`.

**1.7** Add `kv` to `flake.nix` `testModules` list.

**1.8** Verify: `cd kv && GOWORK=off go build ./... && GOWORK=off go test ./...`.

**1.9** Verify: `nix run .#build && nix run .#test && nix run .#lint`.

**1.10** Commit: `feat(kv): add minimal KV store abstraction module`.

### Step 2: Write kv/ contract tests (30min)

**2.1** `kv/kv_test.go` — Test the in-memory implementation:

```go
func TestMemStore_GetSet(t *testing.T)     // basic CRUD
func TestMemStore_Delete(t *testing.T)      // delete + not-found
func TestMemStore_Has(t *testing.T)         // existence check
func TestMemStore_Iterator(t *testing.T)    // prefix scan, ordering
func TestMemStore_Batch(t *testing.T)       // atomic batch commit
func TestMemStore_Close(t *testing.T)       // operations after close return ErrClosed
```

**2.2** These tests also serve as a contract — any future adapter (pebble, badger, bbolt)
should pass the same test suite. Consider using a test suite function:

```go
func TestStoreContract(t *testing.T, store kv.Store) { ... }
```

**2.3** Verify and commit: `test(kv): add contract tests for Store interface`.

### Step 3: Pebble adapter — implement kv.Store (45min)

**3.1** Read existing pebble code:

```bash
# Understand key encoding
grep -n 'cqrs_event:\|cqrs_snapshot:\|cqrs_checkpoint:' pebble/*.go | grep -v _test
# Understand iteration patterns
cat pebble/iteration.go
# Understand batch usage
grep -n 'Batch\|batch' pebble/save.go
```

**3.2** Create `pebble/kv_adapter.go`:

```go
package pebble

import (
    "github.com/cockroachdb/pebble"
    "github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// pebbleStore adapts *pebble.DB to the kv.Store interface.
type pebbleStore struct {
    db *pebble.DB
}

// NewKVStore wraps a Pebble DB as a kv.Store.
func NewKVStore(db *pebble.DB) kv.Store {
    return &pebbleStore{db: db}
}

func (s *pebbleStore) Get(key []byte) ([]byte, error) {
    val, closer, err := s.db.Get(key)
    if err == pebble.ErrNotFound {
        return nil, kv.ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    defer closer.Close()
    return slices.Clone(val), nil
}

func (s *pebbleStore) Has(key []byte) (bool, error) {
    _, closer, err := s.db.Get(key)
    if err == pebble.ErrNotFound {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    closer.Close()
    return true, nil
}

func (s *pebbleStore) Set(key, value []byte) error {
    return s.db.Set(key, value, pebble.Sync)
}

func (s *pebbleStore) Delete(key []byte) error {
    return s.db.Delete(key, pebble.Sync)
}

func (s *pebbleStore) NewIterator(prefix []byte) (kv.Iterator, error) {
    iter, err := s.db.NewIter(&pebble.IterOptions{
        LowerBound: prefix,
        UpperBound: keyPrefixEnd(prefix), // next lexicographic byte after prefix
    })
    if err != nil {
        return nil, err
    }
    iter.SeekGE(prefix)
    return &pebbleIterator{iter: iter}, nil
}

func (s *pebbleStore) Batch() (kv.Batch, error) {
    return &pebbleBatch{batch: s.db.NewBatch()}, nil
}

func (s *pebbleStore) Close() error {
    return s.db.Close()
}
```

**3.3** Implement `pebbleIterator` and `pebbleBatch` types.

**3.4** Add `kv/v2` dependency to `pebble/go.mod`:

```bash
cd pebble
GOWORK=off go get github.com/larsartmann/go-cqrs-lite/kv/v4@latest
echo 'replace github.com/larsartmann/go-cqrs-lite/kv/v4 => ../kv' >> go.mod
GOWORK=off go mod tidy
```

**3.5** Add test: `pebble/kv_adapter_test.go` — run the kv.Store contract tests against the Pebble adapter.

**3.6** Verify: `nix run .#build && nix run .#test && nix run .#lint`.

**3.7** Commit: `feat(pebble): add kv.Store adapter for Pebble backend`.

### Step 4: Refactor Pebble event store to use kv.Store (1hr)

**This is the highest-risk step.** The existing pebble event store works. Only do this if confident.

**4.1** Read all pebble store files to understand what calls `*pebble.DB` directly:

```bash
grep -n '\.db\.' pebble/store.go pebble/save.go pebble/load.go pebble/journal.go
```

**4.2** Change the `EventStore` struct to hold `kv.Store` instead of `*pebble.DB`:

```go
// Before:
type EventStore struct {
    db     *pebble.DB
    // ...
}

// After:
type EventStore struct {
    kv     kv.Store
    // ...
}
```

**4.3** Update `NewStore` to accept `kv.Store`:

```go
// Before:
func NewStore(db *pebble.DB, logger *slog.Logger) *EventStore

// After:
func NewStore(store kv.Store, logger *slog.Logger) *EventStore
```

**4.4** Keep a convenience constructor that wraps Pebble:

```go
func NewPebbleStore(db *pebble.DB, logger *slog.Logger) *EventStore {
    return NewStore(NewKVStore(db), logger)
}
```

**4.5** Update all methods that use `s.db.Get/Set/Delete/NewIter` to use `s.kv.Get/Set/Delete/NewIterator`.

**4.6** Update all tests to use either `NewKVStore(pebbleDB)` or `kv.NewMemStore()`.

**4.7** Verify: `nix run .#build && nix run .#test && nix run .#lint`.

**4.8** Commit: `refactor(pebble): depend on kv.Store instead of raw pebble.DB`.

### Step 5: Add SnapshotStore and CheckpointStore to Pebble (1hr)

**Now trivial** — they just encode/decode values and use `kv.Store` for persistence.

**5.1** `pebble/snapshot.go`:

```go
func NewSnapshotStore(store kv.Store) *SnapshotStore { ... }
// Save: encode snapshot to bytes, kv.Set(cqrs_snapshot:{type}:{id}, bytes)
// Load: kv.Get(cqrs_snapshot:{type}:{id}), decode
```

**5.2** `pebble/checkpoint.go`:

```go
func NewCheckpointStore(store kv.Store) *CheckpointStore { ... }
// Save: kv.Set(cqrs_checkpoint:{name}, eventID)
// Load: kv.Get(cqrs_checkpoint:{name})
```

**5.3** Tests for both.

**5.4** Verify and commit.

## Safety Rules

1. **The kv/ interface is ALREADY DESIGNED** — don't redesign it. Copy from the research doc.
2. **Build must pass after every step.**
3. **Tests must pass after every step.**
4. **Lint must pass after every step.**
5. **Step 4 (Pebble refactor) is optional** — if it gets hairy, keep the adapter (Step 3) and add SnapshotStore/CheckpointStore using kv.Store directly without refactoring the existing event store.
6. **Never break existing consumers** — `NewStore` signature change in Step 4 is breaking. Either provide backward-compatible constructor or accept it (this is still pre-v3).
7. **Use `slices.Clone` on all Get returns** — Pebble's `closer.Close()` invalidates the buffer.

## Dependency Order

```
Step 1 (kv/ module) → Step 2 (contract tests)
                    → Step 3 (Pebble adapter)
                    → Step 4 (Pebble refactor)  ← OPTIONAL, highest risk
                    → Step 5 (Snapshot + Checkpoint) ← TRIVIAL with kv.Store
```

**If Step 4 feels risky, skip it.** Step 5 can use `kv.Store` directly (via the adapter from
Step 3) without refactoring the existing event store.

## Checklist

```
[~] Step 1: kv/ module created (5 interfaces, errors, mem impl)          ← DESCOPED
[~] Step 2: Contract tests pass                                           ← DESCOPED
[~] Step 3: Pebble adapter implements kv.Store                            ← DESCOPED
[~] Step 3: Pebble adapter passes contract tests                          ← DESCOPED
[~] Step 4: (optional) Pebble event store refactored to use kv.Store      ← DESCOPED
[x] Step 5: Pebble SnapshotStore implemented                              ← DONE (directly on *pebble.DB, commit e0e2418e)
[x] Step 5: Pebble CheckpointStore implemented                            ← DONE (directly on *pebble.DB, commit e0e2418e)
[x] Full suite: nix run .#build && nix run .#test && nix run .#lint — ALL PASS
[x] All changes committed
```
