package integration_test

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type integCounterInput struct{}

type integItemCreated struct {
	Status string
}

type integItemCompleted struct{}

func integCounterQuery() metaengine.QueryDecl[integCounterInput, map[string]int64] {
	return metaengine.Query[integCounterInput, map[string]int64](
		"integ_task_counts",
		metaengine.On(integItemCreated{}, func(e integItemCreated) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
		metaengine.On(integItemCompleted{}, func(e integItemCompleted) metaengine.Delta {
			return metaengine.Delta{"active": -1, "completed": +1}
		}),
	)
}

func integPayloadDecoder(eventType string, payload []byte) (any, error) {
	switch eventType {
	case "IntegItemCreated":
		var p integItemCreated
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "IntegItemCompleted":
		return integItemCompleted{}, nil
	default:
		return nil, nil
	}
}

func TestMetaEngine_CounterPipeline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, integCounterQuery())
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	adapter := projectionadapter.New("integ-counter", store, integPayloadDecoder)

	// Create events: 2 pending, 1 active, then complete 1 active → {pending:2, active:0, completed:1}
	events := []struct {
		eventType string
		payload   any
	}{
		{"IntegItemCreated", integItemCreated{Status: "pending"}},
		{"IntegItemCreated", integItemCreated{Status: "pending"}},
		{"IntegItemCreated", integItemCreated{Status: "active"}},
		{"IntegItemCompleted", integItemCompleted{}},
	}

	for _, e := range events {
		evt, err := event.New(
			event.Type(e.eventType),
			id.NewStreamID(),
			"TestStream",
			event.Version(1),
			e.payload,
		)
		if err != nil {
			t.Fatalf("event.New %s: %v", e.eventType, err)
		}

		if err := adapter.Handle(ctx, evt); err != nil {
			t.Fatalf("adapter.Handle %s: %v", e.eventType, err)
		}
	}

	counts, err := metaengine.ExecuteTyped[integCounterInput, map[string]int64](
		ctx, store, integCounterInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if counts["pending"] != 2 {
		t.Errorf("pending = %d, want 2", counts["pending"])
	}

	if counts["active"] != 0 {
		t.Errorf("active = %d, want 0", counts["active"])
	}

	if counts["completed"] != 1 {
		t.Errorf("completed = %d, want 1", counts["completed"])
	}
}

func TestMetaEngine_MapPipeline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type mapKey string

	type itemEvt struct {
		ID    mapKey
		Title string
	}

	type itemResult struct {
		ID    mapKey
		Title string
	}

	queryDecl := metaengine.Query[mapKey, itemResult](
		"integ_items",
		metaengine.On(itemEvt{}, func(e itemEvt) (mapKey, itemResult) {
			return e.ID, itemResult{ID: e.ID, Title: e.Title}
		}),
	)

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, queryDecl)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	decoder := func(eventType string, payload []byte) (any, error) {
		var p itemEvt
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	}

	adapter := projectionadapter.New("integ-map", store, decoder)

	items := []itemEvt{
		{ID: "a", Title: "Alpha"},
		{ID: "b", Title: "Beta"},
		{ID: "c", Title: "Gamma"},
	}

	for _, item := range items {
		evt, err := event.New(
			event.Type("ItemEvt"),
			id.NewStreamID(),
			"TestStream",
			event.Version(1),
			item,
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}

		if err := adapter.Handle(ctx, evt); err != nil {
			t.Fatalf("adapter.Handle: %v", err)
		}
	}

	for _, expected := range items {
		result, err := metaengine.ExecuteTyped[mapKey, itemResult](ctx, store, expected.ID)
		if err != nil {
			t.Fatalf("ExecuteTyped %s: %v", expected.ID, err)
		}

		if result.Title != expected.Title {
			t.Errorf("item %s: title = %q, want %q", expected.ID, result.Title, expected.Title)
		}
	}
}
