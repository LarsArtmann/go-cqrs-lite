package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type rawCreated struct {
	ID   string
	Name string
}

type rawView struct {
	ID   string
	Name string
}

type rawKey struct {
	ID string
}

// TestSystem_NoProjectionDecoder_CBOREvent proves the zero-config projection
// path end to end: no ProjectionDecoder, no EventDecoder, no TypeDecoder, and
// no Evolutions, so `newProjectionAdapter` is built with a nil decoder and must
// delegate the raw payload to the store. The event is built with `event.New`,
// i.e. the default CBOR codec — the exact payload the adapter could not decode
// before the codec boundary repair.
func TestSystem_NoProjectionDecoder_CBOREvent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := metaengine.Query[rawKey, rawView](
		"raw_views",
		metaengine.OnRecordTyped(
			"raw.created",
			rawCreated{},
			func(_ record.Record, e rawCreated) (string, rawView) {
				return e.ID, rawView{ID: e.ID, Name: e.Name}
			},
		),
	)

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Raw", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](
				sys,
				"raw.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Raw",
						func(_ TaskState, ver event.Version) ([]event.Event, error) {
							// No WithCodec: event.New uses the default (CBOR)
							// codec, so this is the payload the encoded
							// ingress must decode.
							return []event.Event{mustEvent(event.New(
								"raw.created",
								cmd.StreamID(),
								"Raw",
								ver+1,
								rawCreated{ID: "raw-1", Name: "widget"},
							))}, nil
						})
				},
			)
		},
		Projections: []system.ProjectionDeclaration{system.RawQuery(query)},
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
	if err := sys.CommandDispatcher().Dispatch(ctx, newCmd("raw.create", streamID)); err != nil {
		t.Fatalf("dispatch raw.create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, status := range sys.ProjectionHost().Status() {
			if status.Processed < 1 || status.Errors != 0 {
				continue
			}

			result, err := sys.MetaEngine().Execute(rawKey{ID: "raw-1"})
			if err != nil {
				t.Fatalf("store.Execute: %v", err)
			}

			view, ok := result.(rawView)
			if !ok {
				t.Fatalf("expected rawView, got %T", result)
			}

			if view.Name != "widget" {
				t.Fatalf("view.Name = %q, want %q", view.Name, "widget")
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("projection host did not project the CBOR event within timeout")
}
