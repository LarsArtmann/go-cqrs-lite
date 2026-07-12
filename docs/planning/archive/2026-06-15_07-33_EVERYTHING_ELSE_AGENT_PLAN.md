# AI Agent Plan: Everything Else — Pebble Completeness + SQL CommandJournal + Tests

> **Status: ✅ COMPLETED** · **Date completed:** 2026-06-15
>
> **For:** AI coding agent (Crush, Claude, GPT, etc.) · **Estimated time:** 4-5 hours
>
> **Goal:** Complete Pebble backend parity, implement SQL CommandJournal, add tests for
> all new interfaces, and update documentation.

## Background

Tiers 1+2 of the command/query full-depth plan are done (commits `b84f2c74` through
`0c0cd5b3`). command/ has Journal + Bus interfaces, query/ has full persistence interfaces,
memory/ has implementations, and event/ has causality tracking.

What remains:

1. **Pebble completeness** — currently only implements `event.Store`. Needs Journal, SnapshotStore, CheckpointStore.
2. **SQL CommandJournal** — `storage.SQLCommandStore` needs ReadAll/ReadFrom methods.
3. **Tests** for MemoryCommandBus and event causality.
4. **Documentation** update.

## Current Pebble State

```bash
# What pebble currently implements:
pebble/store.go      → event.Store (Save, Load, LoadFromVersion, etc.)
pebble/journal.go    → ReadAll, ReadFrom (Journal + SeekableJournal)  ← ALREADY EXISTS
pebble/save.go       → Save implementation
pebble/iteration.go  → internal iteration helpers
pebble/serialization.go → CBOR envelope serialization
pebble/snapshot.go     → SnapshotStore (CBOR, shared DB via cqrs_snapshot: prefix)  ← IMPLEMENTED
pebble/checkpoint.go   → CheckpointStore (CBOR, shared DB via cqrs_checkpoint: prefix)  ← IMPLEMENTED
```

**IMPORTANT:** Check existing code before implementing! Pebble may already have Journal.
Run: `grep -rn 'ReadAll\|ReadFrom\|Journal' pebble/` to verify what exists.

## Step-by-Step Plan

### Step 1: Audit Pebble — what's already implemented? (15min)

**1.1** Run these checks and record results:

```bash
grep -rn 'func.*ReadAll\|func.*ReadFrom' pebble/ --include='*.go' | grep -v _test
grep -rn 'SnapshotStore\|SnapshotSink\|SnapshotSource' pebble/ --include='*.go' | grep -v _test
grep -rn 'CheckpointStore\|CheckpointSink\|CheckpointSource' pebble/ --include='*.go' | grep -v _test
```

**1.2** Read `pebble/go.mod` to check existing dependencies.
**1.3** Read `pebble/store.go` to understand the key schema (how keys are structured).
**1.4** Read `pebble/iteration.go` to understand prefix scan patterns.

### Step 2: Implement Pebble Journal (if missing) (1hr)

If Step 1 shows Journal is already implemented, SKIP this step.

**2.1** Create `pebble/journal.go`:

```go
package pebble

// ReadAll returns all events across all aggregates, ordered by occurred_at.
// Implements event.Journal.
func (s *EventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
    // Use the existing iteration helper to scan the global event prefix
    // Key format: cqrs_event:{aggType}:{aggID}:{version}
    // Global prefix: cqrs_event:
}
```

**2.2** Implement ReadFrom (SeekableJournal):

```go
func (s *EventStore) ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]event.Event, error) {
    // Find the position of afterEventID in the global log
    // Then scan forward from that position, up to limit events
}
```

**2.3** Add compile-time assertions:

```go
var (
    _ event.Journal         = (*EventStore)(nil)
    _ event.SeekableJournal = (*EventStore)(nil)
)
```

**2.4** Test: Create `pebble/journal_test.go` with ReadAll + ReadFrom tests.
**2.5** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**2.6** Commit: `feat(pebble): implement Journal and SeekableJournal`.

### Step 3: Implement Pebble SnapshotStore (1hr)

**3.1** Understand snapshot key schema:

```go
// Snapshot key format: cqrs_snapshot:{aggType}:{aggID}:{version}
// Or:                  cqrs_snapshot:{aggType}:{aggID}  (latest only)
```

**3.2** Create `pebble/snapshot.go`:

```go
package pebble

import (
    "context"
    "github.com/cockroachdb/pebble"
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    "github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

type SnapshotStore struct {
    db    *pebble.DB
    codec codec.Codec  // for serializing snapshot state if needed
    // ...
}

func NewSnapshotStore(db *pebble.DB, opts ...SnapshotOption) *SnapshotStore { ... }

func (s *SnapshotStore) Save(ctx context.Context, snap snapshot.Snapshot) error {
    // Key: cqrs_snapshot:{aggType}:{aggID}
    // Value: CBOR-encoded snapshot
}

func (s *SnapshotStore) Load(ctx context.Context, ref event.AggregateRef) (snapshot.Snapshot, error) {
    // Get latest snapshot by key prefix
}

func (s *SnapshotStore) LoadAtVersion(ctx context.Context, ref event.AggregateRef, version event.Version) (snapshot.Snapshot, error) {
    // Scan for snapshot with version <= requested
}
```

**3.3** Add `snapshot/v2` and `codec/v2` dependencies to `pebble/go.mod`:

```bash
cd pebble
GOWORK=off go get github.com/larsartmann/go-cqrs-lite/snapshot/v4@v2.3.0
echo 'replace github.com/larsartmann/go-cqrs-lite/snapshot/v4 => ../snapshot' >> go.mod
GOWORK=off go mod tidy
```

**3.4** Implement the snapshot key encoding/decoding using the same pattern as event keys.
**3.5** Test: Create `pebble/snapshot_test.go`.
**3.6** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**3.7** Commit: `feat(pebble): implement SnapshotStore`.

### Step 4: Implement Pebble CheckpointStore (30min)

**4.1** Create `pebble/checkpoint.go`:

```go
package pebble

// Checkpoint key format: cqrs_checkpoint:{projectionName}

type CheckpointStore struct {
    db *pebble.DB
}

func NewCheckpointStore(db *pebble.DB) *CheckpointStore { ... }

func (s *CheckpointStore) Save(ctx context.Context, projectionName string, cp event.Checkpoint) error {
    // Key: cqrs_checkpoint:{projectionName}
    // Value: event ID string (the checkpoint position)
}

func (s *CheckpointStore) Load(ctx context.Context, projectionName string) (event.Checkpoint, error) {
    // Get by key, return empty checkpoint if not found
}
```

**4.2** Add compile-time assertions for `event.CheckpointSink`, `event.CheckpointSource`, `event.CheckpointStore`.
**4.3** Test: Create `pebble/checkpoint_test.go`.
**4.4** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**4.5** Commit: `feat(pebble): implement CheckpointStore`.

### Step 5: Implement SQL CommandJournal (1hr)

**5.1** Read `storage/command_store.go` and `storage/command_store_load.go` to understand existing query patterns.

**5.2** Add `ReadAll` and `ReadFrom` methods to `SQLCommandStore`:

Create or edit `storage/command_store_journal.go`:

```go
package storage

import (
    "context"
    "github.com/larsartmann/go-cqrs-lite/command/v4"
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// ReadAll returns all commands across all aggregates, ordered by received_at.
func (s *SQLCommandStore) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
    // SELECT * FROM commands ORDER BY received_at ASC
}

// ReadFrom returns commands after the given command ID, ordered by received_at.
func (s *SQLCommandStore) ReadFrom(ctx context.Context, afterCommandID id.CommandID, limit int) ([]*command.PersistedCommand, error) {
    // Find received_at for afterCommandID, then SELECT WHERE received_at > that
    // ORDER BY received_at ASC LIMIT ?
}
```

**5.3** Add compile-time assertions:

```go
var _ command.CommandJournal = (*SQLCommandStore)(nil)
var _ command.SeekableCommandJournal = (*SQLCommandStore)(nil)
```

**5.4** Test: Add to `storage/command_store_test.go` — test ReadAll + ReadFrom using sqlmock.
**5.5** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**5.6** Commit: `feat(storage): implement CommandJournal on SQLCommandStore`.

### Step 6: Test MemoryCommandBus (30min)

**6.1** Create `memory/command_bus_test.go`:

```go
func TestMemoryCommandBus_PublishSubscribe(t *testing.T) {
    // Create bus, subscribe to type, publish, verify handler called
}

func TestMemoryCommandBus_SubscribeAll(t *testing.T) {
    // Subscribe to all, publish multiple types, verify all received
}

func TestMemoryCommandBus_Middleware(t *testing.T) {
    // Register middleware, verify it wraps handler
}

func TestMemoryCommandBus_Close(t *testing.T) {
    // Close bus, verify Publish/Subscribe return error
}
```

**6.2** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**6.3** Commit: `test(memory): add MemoryCommandBus tests`.

### Step 7: Test event causality (30min)

**7.1** Create `event/causality_test.go`:

```go
func TestWithCommandCausality(t *testing.T) {
    // Set context with command causality, enrich events, verify metadata
}

func TestCommandCausalityEnricher(t *testing.T) {
    // Test enricher returns correct options when causality is set
    // Test enricher returns nil when causality is not set
}
```

**7.2** Verify: `nix run .#build && nix run .#test && nix run .#lint`.
**7.3** Commit: `test(event): add command-event causality tests`.

### Step 8: Update documentation (30min)

**8.1** Update `FEATURES.md`:

- Add CommandJournal/SeekableCommandJournal to command section
- Add CommandBus/Publisher/Subscriber to command section
- Add QuerySink/Source/Store/Journal to query section
- Add MemoryCommandBus, MemoryQueryStore to memory section
- Add event causality to event section

**8.2** Update `AGENTS.md`:

- Update module structure tree
- Add new interfaces to Key Patterns section
- Update module count

**8.3** Update `TODO_LIST.md`:

- Mark all completed items as DONE

**8.4** Update `README.md`:

- Add note about command/query persistence in features table

**8.5** Commit: `docs: update for command/query full depth + pebble completeness`.
**8.6** Push.

## Safety Rules

1. **Build must pass after every step** — `nix run .#build`.
2. **Tests must pass after every step** — `nix run .#test`.
3. **Lint must pass after every step** — `nix run .#lint`.
4. **GOWORK=off builds must work** for any module whose go.mod changes.
5. **Read existing code before implementing** — don't reinvent what's there.
6. **Use existing patterns** — mirror the SQL event store patterns for SQL command store.
7. **One logical change per commit** — each commit leaves the project buildable.
8. **Commit message format:** `feat(<module>): <description>` or `test(<module>): <description>`.
9. **Never modify existing method signatures** — only add new methods.
10. **Never add `replace` directives to published modules without also adding to go.work.**

## Dependency Order

```
Step 1 (audit) → Step 2 (pebble journal, if needed)
                → Step 3 (pebble snapshot)
                → Step 4 (pebble checkpoint)
                → Step 5 (SQL command journal)
                → Step 6 (command bus tests)
                → Step 7 (causality tests)
                → Step 8 (docs)
```

Steps 2-7 are independent and can be done in any order after Step 1.

## Execution Result

All steps completed. Pebble now has full feature parity (Journal, SnapshotStore,
CheckpointStore) with OTel tracing on all operations. SQL CommandJournal implemented
in `storage/command_store_journal.go`. All tests added. Documentation updated.

Note: Pebble SnapshotStore and CheckpointStore were implemented directly on
`*pebble.DB` (not via a `kv.Store` abstraction — see KV_MODULE_AGENT_PLAN.md for
the descoped kv/ module).

## Checklist (print and tick off)

```
[x] Step 1: Pebble audit complete — confirmed Journal already existed
[x] Step 2: Pebble Journal verified existing (commit aae771e4)
[x] Step 3: Pebble SnapshotStore implemented (commit e0e2418e)
[x] Step 4: Pebble CheckpointStore implemented (commit e0e2418e)
[x] Step 5: SQL CommandJournal implemented (commit bf7b3ed8)
[x] Step 6: MemoryCommandBus tested (commit 75533808)
[x] Step 7: Event causality tested (commit 21d28fd2)
[x] Step 8: Documentation updated (commits 69dcaec1, 5f34c138, df2eb93b, a52de496)
[x] Full suite: nix run .#build && nix run .#test && nix run .#lint — ALL PASS
[x] All changes committed
```
