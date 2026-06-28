package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsgraph "github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

type graphUserCreated struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type graphUserFollowed struct {
	FolloweeID string `json:"followeeId"`
	FollowedID string `json:"followedId"`
}

func TestBundle_RunProjections_GraphProjection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store := memory.NewMemoryStore()
	defer store.Close()

	bus := cqrswatermill.NewEventBus()
	defer bus.Close()

	checkpointStore := memory.NewMemoryCheckpointStore()
	defer checkpointStore.Close()

	bundle, err := stack.New(
		stack.WithEventStore(store),
		stack.WithBus(bus),
		stack.WithCheckpointStore(checkpointStore),
	)
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}

	driver := cqrsgraph.NewMemoryDriver()

	graphProj, err := cqrsgraph.NewGraphProjection(
		"social-graph",
		driver,
		func(_ context.Context, evt cqrsevent.Event, sink cqrsgraph.GraphSink) error {
			switch evt.Type() {
			case "user.created":
				var p graphUserCreated
				if err := json.Unmarshal(evt.Payload(), &p); err != nil {
					return err
				}

				return sink.MergeNode(
					cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.ID},
					map[string]any{"name": p.Name},
				)

			case "user.followed":
				var p graphUserFollowed
				if err := json.Unmarshal(evt.Payload(), &p); err != nil {
					return err
				}

				return sink.MergeEdge(cqrsgraph.EdgeRef{
					Type: "FOLLOWS",
					From: cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.FolloweeID},
					To:   cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.FollowedID},
				}, nil)
			}

			return nil
		},
		[]cqrsevent.Type{"user.created", "user.followed"},
	)
	if err != nil {
		t.Fatalf("create graph projection: %v", err)
	}

	// Phase 1: save historical events before RunProjections starts.
	aggID, _ := id.ParseAggregateID("social")
	ref := cqrsevent.NewAggregateRef("Social", aggID)

	createEvents := make([]cqrsevent.Event, 0, 3)

	for _, u := range []graphUserCreated{
		{ID: "alice", Name: "Alice"},
		{ID: "bob", Name: "Bob"},
	} {
		payload, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("marshal user: %v", err)
		}

		evt, _ := cqrsevent.NewEvent("user.created", aggID, "Social", cqrsevent.Version(1), payload)
		createEvents = append(createEvents, evt)
	}

	followPayload, err := json.Marshal(graphUserFollowed{FolloweeID: "alice", FollowedID: "bob"})
	if err != nil {
		t.Fatalf("marshal follow: %v", err)
	}
	followEvt, _ := cqrsevent.NewEvent("user.followed", aggID, "Social", cqrsevent.Version(1), followPayload)
	createEvents = append(createEvents, followEvt)

	_ = store.Save(ctx, ref, createEvents, cqrsevent.Version(0))

	// Phase 2: start RunProjections and wait for replay.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runErr := make(chan error, 1)

	go func() {
		runErr <- bundle.RunProjections(runCtx, graphProj)
	}()

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		users := driver.Query(cqrsgraph.Pattern{Label: "User"})
		if len(users) == 2 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	users := driver.Query(cqrsgraph.Pattern{Label: "User"})
	if len(users) != 2 {
		t.Fatalf("expected 2 users in graph, got %d", len(users))
	}

	// Verify the follow edge exists.
	neighbors, edges := driver.Neighbors(
		cqrsgraph.NodeRef{Label: "User", KeyProp: "id", KeyValue: "alice"},
	)
	if len(edges) == 0 {
		t.Fatalf("expected at least 1 edge from alice, got 0 (neighbors: %d)", len(neighbors))
	}

	// Phase 3: shutdown.
	cancel()

	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
		t.Fatal("RunProjections did not stop after context cancellation")
	}
}
