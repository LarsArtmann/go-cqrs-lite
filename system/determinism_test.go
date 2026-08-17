package system_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_UnknownProjectionEngineFails is the regression test for silent
// projection loss: a projections instance referencing an engine name that is
// not defined in Engines must fail New() with ErrUnknownEngine instead of
// silently skipping the projections.
func TestSystem_UnknownProjectionEngineFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engines: []string{"proj-engine-typo"}},
		},
	}

	_, err := system.New(ctx, system.DomainConfig{}, deployment)
	if !errors.Is(err, system.ErrUnknownEngine) {
		t.Fatalf("expected ErrUnknownEngine for typo'd projection engine, got %v", err)
	}
}

// TestSystem_EngineNamesDeterministic pins sorted engine creation: EngineNames
// must be sorted regardless of Go map iteration seed.
func TestSystem_EngineNamesDeterministic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"zeta":    {Driver: "memory"},
			"alpha":   {Driver: "memory"},
			"midway":  {Driver: "memory"},
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	want := []string{"alpha", "midway", "primary", "zeta"}

	for range 10 {
		sys, err := system.New(ctx, system.DomainConfig{}, deployment)
		if err != nil {
			t.Fatalf("system.New: %v", err)
		}

		got := sys.EngineNames()
		if len(got) != len(want) {
			t.Fatalf("EngineNames: want %d engines, got %d (%v)", len(want), len(got), got)
		}

		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("EngineNames not deterministic: want %v, got %v", want, got)
			}
		}

		sys.Close()
	}
}

// TestSystem_UnknownBusDriverAnywhereFails is the regression test for
// first-iteration validation: an unknown bus driver must fail New() even when
// another bus entry (which may be iterated first) is valid.
func TestSystem_UnknownBusDriverAnywhereFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Buses: map[string]system.BusConfig{
			"aaa-valid":   {Driver: "gochannel"},
			"zzz-unknown": {Driver: "kafka"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	_, err := system.New(ctx, system.DomainConfig{}, deployment)
	if !errors.Is(err, system.ErrUnknownBusDriver) {
		t.Fatalf("expected ErrUnknownBusDriver for unknown driver in bus map, got %v", err)
	}
}

// TestSystem_FanOutBusesClosedOnClose is the regression test for the fan-out
// bus leak: Close() must close the per-publish-target buses created by
// buildPublisher, not just the local event bus.
func TestSystem_FanOutBusesClosedOnClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Buses: map[string]system.BusConfig{
			"bus1": {Driver: "gochannel"},
			"bus2": {Driver: "gochannel"},
		},
		Instances: []system.InstanceConfig{{
			Role:    system.RoleSourceOfTruth,
			Engine:  "primary",
			Publish: []string{"bus1", "bus2"},
		}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	multi, ok := sys.Publisher().(*system.MultiBus)
	if !ok {
		t.Fatalf("expected publisher to be *MultiBus, got %T", sys.Publisher())
	}

	pubs := multi.Publishers()
	if len(pubs) != 3 {
		t.Fatalf("expected 3 publishers, got %d", len(pubs))
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Publishing on a fan-out bus after Close must fail — proving the system
	// closed it instead of leaking it.
	ref := id.NewStreamRef("FanOutClose", id.NewStreamID())
	evt := eventtest.NewEvent(t, "fanout.closed", ref.ID, ref.Type, 1, nil)

	pub := pubs[1]

	if err := pub.Publish(ctx, evt); err == nil {
		t.Fatal("fan-out bus still publishable after system Close() — bus was leaked")
	}
}
