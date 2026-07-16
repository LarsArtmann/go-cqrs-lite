package integration_test

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type snapUserState struct {
	Name  string
	Email string
}

func applySnapUser(state snapUserState, evt event.Event) (snapUserState, error) {
	if evt.Type() == "UserCreated" {
		var payload struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
			return state, err
		}

		return snapUserState{Name: payload.Name, Email: payload.Email}, nil
	}

	return state, nil
}

func snapDecider() decider.Decider[snapUserState] {
	return decider.Decider[snapUserState]{
		Initial: snapUserState{},
		Apply:   applySnapUser,
	}
}

func matchSnapshot(t *testing.T, name string, data []byte) {
	t.Helper()

	golden := filepath.Join("testdata", "snapshots", name+".snap")
	dir := filepath.Dir(golden)

	if os.Getenv("UPDATE_SNAPS") == "true" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("create snapshot dir: %v", err)
		}

		if err := os.WriteFile(golden, data, 0o600); err != nil {
			t.Fatalf("write snapshot: %v", err)
		}

		t.Logf("updated snapshot: %s", golden)

		return
	}

	expected, err := os.ReadFile(golden)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("snapshot not found: %s (run with UPDATE_SNAPS=true to create)", golden)
		}

		t.Fatalf("read snapshot: %v", err)
	}

	if !bytes.Equal(data, expected) {
		diff := fmt.Sprintf("snapshot mismatch: %s\n--- expected ---\n%s\n--- actual ---\n%s",
			golden, string(expected), string(data))

		t.Fatal(diff)
	}
}

func TestSnapshot_EventSerialization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	defer store.Close()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	d := snapDecider()
	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	aggID := id.NewAggregateID()

	err = repo.Execute(
		ctx,
		aggID,
		"User",
		func(_ snapUserState, currentVersion event.Version) ([]event.Event, error) {
			payload, _ := json.Marshal(map[string]string{
				"name":  "Alice",
				"email": "alice@example.com",
			})
			evt, err := event.NewEvent(
				"UserCreated",
				aggID,
				"User",
				currentVersion.Add(1),
				payload,
			)
			if err != nil {
				return nil, err
			}

			return []event.Event{evt}, nil
		},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	events, err := store.Load(ctx, id.NewAggregateRef("User", aggID))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	serialized := make([]map[string]any, 0, len(events))
	for _, evt := range events {
		serialized = append(serialized, map[string]any{
			"type":          string(evt.Type()),
			"aggregateType": string(evt.AggregateType()),
			"version":       int(evt.Version()),
		})
	}

	data, err := json.Marshal(serialized, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	matchSnapshot(t, "event_serialization", data)
}
