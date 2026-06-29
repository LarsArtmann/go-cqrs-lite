package main

import (
	"context"
	"encoding/json"
	"testing"

	cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/deriver/v3"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestIdempotent_DeterministicCommandIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mkDeriver := func(cmdType string) deriver.Deriver {
		return deriver.Deriver(
			func(_ context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
				cmd, err := cqrscommand.New(cqrscommand.Type(cmdType), evt.AggregateID())
				if err != nil {
					return nil, err
				}

				return []cqrscommand.Command{cmd}, nil
			},
		)
	}

	composed := mkDeriver("email.send_welcome").
		Then(mkDeriver("crm.upsert_user")).
		Filter("user.signed_up").
		Idempotent()

	payload, _ := json.Marshal(userSignedUp{Email: "alice@example.com"})
	aggID := id.NewAggregateID()

	evt, err := cqrsevent.NewEvent("user.signed_up", aggID, "User", cqrsevent.Version(1), payload)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	first, err := composed(ctx, evt)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := composed(ctx, evt)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected 2 commands each, got %d and %d", len(first), len(second))
	}

	for i := range first {
		if first[i].ID() != second[i].ID() {
			t.Errorf("command[%d]: first ID %v != second ID %v (not deterministic)",
				i, first[i].ID(), second[i].ID())
		}

		if !id.IsDerivedCommandID(first[i].ID()) {
			t.Errorf("command[%d]: ID should be derived (zero timestamp), got %v", i, first[i].ID())
		}
	}
}

func TestIdempotent_SourceEventMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	d := deriver.Deriver(func(_ context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, err := cqrscommand.New("test.cmd", evt.AggregateID())
		if err != nil {
			return nil, err
		}

		return []cqrscommand.Command{cmd}, nil
	}).
		Idempotent()

	aggID := id.NewAggregateID()
	evt, _ := cqrsevent.NewEvent("test.event", aggID, "Test", cqrsevent.Version(1), []byte("{}"))

	cmds, err := d(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bc, ok := cmds[0].(*cqrscommand.BasicCommand)
	if !ok {
		t.Fatal("expected *BasicCommand")
	}

	sourceID, ok := bc.Metadata().Custom[deriver.SourceEventIDKey]
	if !ok {
		t.Fatal("expected source_event_id in command metadata")
	}

	if sourceID != evt.ID().String() {
		t.Errorf("source_event_id = %q, want %q", sourceID, evt.ID().String())
	}
}
