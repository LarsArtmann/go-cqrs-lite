package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Types for TypeDecoder-based projection test ──

type itemCreatedEvt struct {
	Name string
}

type findByID struct {
	ID string
}

type itemView struct {
	ID   string
	Name string
}

// TestSystem_ProjectionTypeDecoder_MapByKey proves that Map ADT queries
// keyed by stream ID (the entity key) work through system/ when using
// ProjectionTypeDecoder. This was previously IMPOSSIBLE with
// ProjectionDecoder (PayloadDecoder) because it had no access to
// evt.StreamID().
func TestSystem_ProjectionTypeDecoder_MapByKey(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Declare a Map query keyed by stream ID. The fold handler receives
	// EventWithID[itemCreatedEvt] which wraps the payload with the stream ID.
	itemQuery := metaengine.Query[findByID, itemView]("item_views_typed",
		metaengine.OnTyped("item.created",
			projectionadapter.EventWithID[itemCreatedEvt]{},
			func(e projectionadapter.EventWithID[itemCreatedEvt]) (string, itemView) {
				return e.ID, itemView{ID: e.ID, Name: e.Payload.Name}
			}),
	)

	// Build the TypeDecoder — replaces the old PayloadDecoder switch/case.
	dec := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(event.Type("item.created"), itemCreatedEvt{}),
	)

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Item", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "item.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Item",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("item.created",
								cmd.StreamID(), "Item", ver+1,
								itemCreatedEvt{Name: "widget"},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:           []any{itemQuery},
		ProjectionTypeDecoder: dec,
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	streamID := id.NewStreamID()

	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("item.create", streamID)); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	// Wait for projection host to process the event.
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				// Query by the STREAM ID — this is the key proof.
				// With PayloadDecoder, the key would be "" (empty string)
				// because the decoder has no access to evt.StreamID().
				result, err := sys.MetaEngine().Execute(findByID{ID: streamID.String()})
				if err != nil {
					t.Fatalf("store.Execute by stream ID: %v", err)
				}

				view, ok := result.(itemView)
				if !ok {
					t.Fatalf("expected itemView, got %T", result)
				}

				if view.ID != streamID.String() {
					t.Fatalf("view.ID = %q, want stream ID %q", view.ID, streamID.String())
				}

				if view.Name != "widget" {
					t.Fatalf("view.Name = %q, want %q", view.Name, "widget")
				}

				return // success
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("projection host did not process event within timeout")
}

// TestSystem_ProjectionEventDecoder verifies that ProjectionEventDecoder
// (custom function with full event context) also works through system/.
func TestSystem_ProjectionEventDecoder(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	itemQuery := metaengine.Query[findByID, itemView]("item_views_evtdec",
		metaengine.OnTyped("item.created",
			projectionadapter.EventWithID[itemCreatedEvt]{},
			func(e projectionadapter.EventWithID[itemCreatedEvt]) (string, itemView) {
				return e.ID, itemView{ID: e.ID, Name: e.Payload.Name}
			}),
	)

	// Custom EventDecoder — same effect as TypeDecoder but manual.
	customDecoder := func(evt event.Event) (any, error) {
		if evt.Type() != event.Type("item.created") {
			return nil, errors.New("unexpected event type: " + string(evt.Type()))
		}

		return projectionadapter.EventWithID[itemCreatedEvt]{
			ID:      evt.StreamID().String(),
			Payload: itemCreatedEvt{Name: "custom-decoded"},
		}, nil
	}

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Item", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "item.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Item",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("item.created",
								cmd.StreamID(), "Item", ver+1,
								itemCreatedEvt{Name: "ignored-by-decoder"},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:            []any{itemQuery},
		ProjectionEventDecoder: customDecoder,
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	streamID := id.NewStreamID()

	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("item.create", streamID)); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				result, err := sys.MetaEngine().Execute(findByID{ID: streamID.String()})
				if err != nil {
					t.Fatalf("store.Execute: %v", err)
				}

				view, ok := result.(itemView)
				if !ok {
					t.Fatalf("expected itemView, got %T", result)
				}

				// The custom decoder hardcoded "custom-decoded", ignoring the payload.
				if view.Name != "custom-decoded" {
					t.Fatalf(
						"view.Name = %q, want %q (custom decoder output)",
						view.Name,
						"custom-decoded",
					)
				}

				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("projection host did not process event within timeout")
}
