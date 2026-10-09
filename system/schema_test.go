package system

import (
	"context"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

func schemaTestDeployment() DeploymentConfig {
	return DeploymentConfig{
		Engines: map[string]EngineConfig{"primary": {Driver: "memory"}},
		Instances: []InstanceConfig{
			{Role: RoleSourceOfTruth, Engine: "primary"},
		},
	}
}

func TestSchemaDeclaration_AppliesOnEveryReadPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := New(ctx, DomainConfig{
		Schema: []schema.EventSchema{
			schema.Event("user.created", 2,
				schema.RenameField("user.created", 1, "name", "displayName"),
			),
		},
	}, schemaTestDeployment())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sys.Close()

	ref := id.NewStreamRef("User", id.NewStreamID())

	v1, err := event.NewEvent("user.created", ref.ID, "User", 1,
		[]byte(`{"name":"Lars"}`), event.WithSchemaVersion(1))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := sys.EventStore().Save(ctx, ref, []event.Event{v1}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Run("store Load upcasts", func(t *testing.T) {
		t.Parallel()

		loaded, err := sys.EventStore().Load(ctx, ref)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if len(loaded) != 1 {
			t.Fatalf("loaded %d events, want 1", len(loaded))
		}

		if loaded[0].SchemaVersion() != 2 {
			t.Errorf("schema version = %d, want 2", loaded[0].SchemaVersion())
		}

		var fields map[string]any
		if err := json.Unmarshal(loaded[0].Payload(), &fields); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if fields["displayName"] != "Lars" || fields["name"] != nil {
			t.Errorf("rename did not apply: %v", fields)
		}
	})

	t.Run("journal capability survives decoration", func(t *testing.T) {
		t.Parallel()

		journal, ok := sys.EventStore().(event.SeekableJournal)
		if !ok {
			t.Fatal("decorated store lost SeekableJournal")
		}

		all, err := journal.ReadFrom(ctx, id.EventID{}, 10)
		if err != nil {
			t.Fatalf("ReadFrom: %v", err)
		}

		if len(all) == 0 || all[0].SchemaVersion() != 2 {
			t.Errorf("journal read did not upcast: %d events", len(all))
		}
	})
}

func TestSchemaDeclaration_JoinsCoeffectUniverse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := New(ctx, DomainConfig{
		Schema: []schema.EventSchema{
			schema.Event("user.created", 1),
		},
		// A fold consuming a type NEITHER Events NOR Schema declares must
		// trip the dangling gate — Schema extends the universe, it must not
		// bypass the gate.
		Evolutions: []EvolutionSpec{
			OnEvolution(
				Evolve[map[string]any]("schema_gate_view"),
				"user.undeclared", map[string]any{},
			).Done(),
		},
		Projections: []ProjectionDeclaration{
			Lookup[map[string]any]("schema_gate_lookup").Done(),
		},
	}, schemaTestDeployment())
	if err == nil {
		t.Fatal("expected dangling-subscription error, got nil")
	}
}

func TestSchemaDeclaration_RejectsInvalidDeclaration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name   string
		schema []schema.EventSchema
	}{
		{
			name: "duplicate declaration",
			schema: []schema.EventSchema{
				schema.Event("user.created", 1),
				schema.Event("user.created", 2),
			},
		},
		{
			name: "op for a different event inside this declaration",
			schema: []schema.EventSchema{
				schema.Event("user.created", 2,
					schema.RenameField("user.renamed", 1, "a", "b"),
				),
			},
		},
		{
			name: "op source version supersedes current",
			schema: []schema.EventSchema{
				schema.Event("user.created", 1,
					schema.RemoveField("user.created", 1, "legacy"),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(ctx, DomainConfig{Schema: tt.schema}, schemaTestDeployment()); err == nil {
				t.Fatal("New accepted an invalid schema declaration")
			}
		})
	}
}
