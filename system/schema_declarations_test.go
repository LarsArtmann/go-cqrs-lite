package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type userCreatedV2 struct {
	DisplayName string `json:"displayName"`
}

type lookupByID struct {
	ID string
}

type userView struct {
	ID          string
	DisplayName string
}

// TestSchemaDeclarations_BuilderDerivesDecoderAndChain proves the ONE-list
// promise: Schemas().Event[T](...) produces BOTH the upcasting declaration
// and the typed projection decoder from the same list, and a v1 stored event
// reaches the typed fold as its current-shape payload (rename applied, typed
// decode, stream-ID key intact).
func TestSchemaDeclarations_BuilderDerivesDecoderAndChain(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	decls, err := system.Schemas().
		Event[userCreatedV2]("user.created", 2,
		schema.RenameField("user.created", 1, "name", "displayName"),
	).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(decls.Declarations()) != 1 {
		t.Fatalf("declarations = %d, want 1", len(decls.Declarations()))
	}

	if decls.TypeDecoder() == nil {
		t.Fatal("derived TypeDecoder is nil")
	}

	userQuery := metaengine.Query[lookupByID, userView]("schema_builder_users",
		metaengine.OnRecordTyped(
			"user.created",
			projectionadapter.EventWithID[userCreatedV2]{},
			func(_ record.Record, e projectionadapter.EventWithID[userCreatedV2]) (string, userView) {
				return e.ID, userView{ID: e.ID, DisplayName: e.Payload.DisplayName}
			},
		),
	)

	sys, err := system.New(ctx, system.DomainConfig{
		Schema:                decls.Declarations(),
		ProjectionTypeDecoder: decls.TypeDecoder(),
		Projections:           []system.ProjectionDeclaration{system.RawQuery(userQuery)},
	}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("User", streamID)

	v1, err := event.NewEvent("user.created", streamID, "User", 1,
		[]byte(`{"name":"Lars"}`), event.WithSchemaVersion(1))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := sys.EventStore().Save(ctx, ref, []event.Event{v1}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				result, err := sys.MetaEngine().Execute(lookupByID{ID: streamID.String()})
				if err != nil {
					t.Fatalf("Execute: %v", err)
				}

				view, ok := result.(userView)
				if !ok {
					t.Fatalf("expected userView, got %T", result)
				}

				if view.DisplayName != "Lars" {
					t.Fatalf("DisplayName = %q, want %q — v1 rename must reach the typed fold",
						view.DisplayName, "Lars")
				}

				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("projection host did not process the v1 event within timeout")
}

// TestSchemaDeclarations_BuildRejectsDuplicates pins the fail-fast rule: a
// duplicate event type is rejected at Build, before composition.
func TestSchemaDeclarations_BuildRejectsDuplicates(t *testing.T) {
	t.Parallel()

	builder := system.Schemas().
		Event[userCreatedV2]("user.created", 2).
		Event[userCreatedV2]("user.created", 3)

	if _, err := builder.Build(); err == nil {
		t.Fatal("Build accepted a duplicate event type")
	}
}
