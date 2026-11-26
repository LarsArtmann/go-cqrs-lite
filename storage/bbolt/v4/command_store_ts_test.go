package bbolt

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Command keys are type/stream/ULID, not received-at: a command received
// before the cutoff but stored after a later one must still be returned.
func TestCommandStore_LoadToTimestamp_UnorderedKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestBackend(t).CommandStore()
	ref := command.NewStreamRef("User", id.NewStreamID())

	first, second := id.NewCommandID(), id.NewCommandID()
	low, high := first, second
	if low.String() > high.String() {
		low, high = high, first
	}

	base := time.Now().Add(-time.Hour).UTC()

	// Stored first in key order, received after the cutoff.
	late, err := command.NewPersistedCommand("user.update", ref, []byte(`{}`),
		command.WithReceivedAt(base.Add(2*time.Hour)), command.WithPersistedCommandID(low))
	if err != nil {
		t.Fatal(err)
	}

	early, err := command.NewPersistedCommand("user.create", ref, []byte(`{}`),
		command.WithReceivedAt(base), command.WithPersistedCommandID(high))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.AppendBatch(ctx, ref, []*command.PersistedCommand{early, late}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	cutoff := base.Add(time.Hour)

	cmds, err := store.LoadToTimestamp(ctx, ref, cutoff)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(cmds) != 1 || cmds[0].ID() != early.ID() {
		t.Fatalf("LoadToTimestamp: got %v, want only the early command %s", cmds, early.ID())
	}

	cmds, err = store.LoadFromTimestamp(ctx, ref, cutoff)
	if err != nil {
		t.Fatalf("LoadFromTimestamp: %v", err)
	}

	if len(cmds) != 1 || cmds[0].ID() != late.ID() {
		t.Fatalf("LoadFromTimestamp: got %v, want only the late command %s", cmds, late.ID())
	}
}
