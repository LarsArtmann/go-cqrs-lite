package commandtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// StoreSuite combines [command.Store] (sink + source) with
// [command.SeekableCommandJournal] (global journal reads). Every backend's
// command store implementation satisfies this interface.
type StoreSuite interface {
	command.Store
	command.SeekableCommandJournal
}

// StoreFactory creates a fresh store for each subtest. Each call must produce
// an independent store with its own backing data.
type StoreFactory func(t *testing.T) StoreSuite

// MustCreateCommand creates a [command.PersistedCommand] for testing. It
// fails the test on error.
func MustCreateCommand(
	t *testing.T,
	cmdType string,
	ref command.StreamRef,
) *command.PersistedCommand {
	t.Helper()

	cmd, err := command.NewPersistedCommand(
		command.Type(cmdType),
		ref,
		[]byte(`{"action":"test"}`),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	return cmd
}

// RunStoreSuite runs the full command store conformance suite. Each subtest
// gets a fresh store via the factory.
func RunStoreSuite(t *testing.T, factory StoreFactory) {
	t.Helper()

	t.Run("SaveAndLoad", func(t *testing.T) {
		t.Parallel()
		testSaveAndLoad(t, factory(t))
	})
	t.Run("DuplicateDetection", func(t *testing.T) {
		t.Parallel()
		testDuplicateDetection(t, factory(t))
	})
	t.Run("AppendBatch", func(t *testing.T) {
		t.Parallel()
		testAppendBatch(t, factory(t))
	})
	t.Run("ReadAll", func(t *testing.T) {
		t.Parallel()
		testReadAll(t, factory(t))
	})
	t.Run("ReadFrom", func(t *testing.T) {
		t.Parallel()
		testReadFrom(t, factory(t))
	})
	t.Run("LoadFromTimestamp", func(t *testing.T) {
		t.Parallel()
		testLoadFromTimestamp(t, factory(t))
	})
}

// newTestStream creates a fresh stream ref plus a user.create command bound
// to it — the shared setup for the Save/Load conformance subtests.
func newTestStream(t *testing.T, store StoreSuite) (*command.PersistedCommand, command.StreamRef, context.Context) {
	t.Helper()

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := command.NewStreamRef("User", streamID)
	cmd := MustCreateCommand(t, "user.create", ref)

	return cmd, ref, ctx
}

func testSaveAndLoad(t *testing.T, store StoreSuite) {
	t.Helper()

	cmd, ref, ctx := newTestStream(t, store)

	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	if loaded[0].ID() != cmd.ID() {
		t.Errorf("ID mismatch: got %s, want %s", loaded[0].ID(), cmd.ID())
	}

	if loaded[0].Type() != cmd.Type() {
		t.Errorf("Type mismatch: got %s, want %s", loaded[0].Type(), cmd.Type())
	}
}

func testDuplicateDetection(t *testing.T, store StoreSuite) {
	t.Helper()

	cmd, ref, ctx := newTestStream(t, store)

	if err := store.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := store.Save(ctx, ref, cmd)
	if !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatalf("expected ErrDuplicateCommand, got %v", err)
	}
}

func testAppendBatch(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	streamID := id.NewStreamID()
	ref := command.NewStreamRef("Order", streamID)

	cmds := []*command.PersistedCommand{
		MustCreateCommand(t, "order.create", ref),
		MustCreateCommand(t, "order.confirm", ref),
		MustCreateCommand(t, "order.ship", ref),
	}

	if err := store.AppendBatch(ctx, ref, cmds); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(loaded))
	}
}

func testReadAll(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	ref1 := command.NewStreamRef("User", id.NewStreamID())
	ref2 := command.NewStreamRef("Order", id.NewStreamID())

	cmd1 := MustCreateCommand(t, "user.create", ref1)
	if err := store.Save(ctx, ref1, cmd1); err != nil {
		t.Fatalf("Save cmd1: %v", err)
	}

	time.Sleep(2 * time.Millisecond) // ensure ULID time ordering

	cmd2 := MustCreateCommand(t, "order.create", ref2)
	if err := store.Save(ctx, ref2, cmd2); err != nil {
		t.Fatalf("Save cmd2: %v", err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 commands in journal, got %d", len(all))
	}
}

func testReadFrom(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	ref := command.NewStreamRef("User", id.NewStreamID())

	cmdIDs := make([]id.CommandID, 0, 5)

	for range 5 {
		cmd := MustCreateCommand(t, "user.action", ref)
		cmdIDs = append(cmdIDs, cmd.ID())

		if err := store.Save(ctx, ref, cmd); err != nil {
			t.Fatalf("Save %d: %v", len(cmdIDs), err)
		}

		time.Sleep(2 * time.Millisecond) // ensure ULID ordering
	}

	// Read from beginning
	all, err := store.ReadFrom(ctx, id.CommandID{}, 0)
	if err != nil {
		t.Fatalf("ReadFrom zero ID: %v", err)
	}

	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}

	// Read after 3rd command with limit 2
	page, err := store.ReadFrom(ctx, cmdIDs[2], 2)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(page) != 2 {
		t.Fatalf("expected 2 in page, got %d", len(page))
	}

	if page[0].ID() != cmdIDs[3] {
		t.Errorf("expected first in page to be cmd 4, got %s", page[0].ID())
	}
}

func testLoadFromTimestamp(t *testing.T, store StoreSuite) {
	t.Helper()

	ctx := context.Background()

	ref := command.NewStreamRef("User", id.NewStreamID())

	before := time.Now()

	cmd1 := MustCreateCommand(t, "user.create", ref)
	if err := store.Save(ctx, ref, cmd1); err != nil {
		t.Fatalf("Save cmd1: %v", err)
	}

	midpoint := time.Now()

	time.Sleep(2 * time.Millisecond)

	cmd2 := MustCreateCommand(t, "user.update", ref)
	if err := store.Save(ctx, ref, cmd2); err != nil {
		t.Fatalf("Save cmd2: %v", err)
	}

	// Load all after midpoint → should get only cmd2
	filtered, err := store.LoadFromTimestamp(ctx, ref, midpoint)
	if err != nil {
		t.Fatalf("LoadFromTimestamp: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 command after midpoint, got %d", len(filtered))
	}

	// Load all after before → should get both
	all, err := store.LoadFromTimestamp(ctx, ref, before)
	if err != nil {
		t.Fatalf("LoadFromTimestamp all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 commands after before, got %d", len(all))
	}
}
