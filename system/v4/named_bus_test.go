package system_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// recordingPublisher counts publishes per target.
type recordingPublisher struct {
	name  string
	count int
}

func (p *recordingPublisher) Publish(_ context.Context, _ ...event.Event) error {
	p.count++

	return nil
}

// TestMultiBus_PublisherByName verifies bind-by-name lookup: named entries
// resolve, positionally added entries do not.
func TestMultiBus_PublisherByName(t *testing.T) {
	t.Parallel()

	local := &recordingPublisher{name: "local"}
	external := &recordingPublisher{name: "external"}

	multi := system.NewMultiBus(local)
	multi.AddNamedPublisher("nats", external)

	if pubs := multi.Publishers(); len(pubs) != 2 {
		t.Fatalf("expected 2 publishers (local + nats), got %d", len(pubs))
	}

	if names := multi.Names(); len(names) != 1 || names[0] != "nats" {
		t.Fatalf("Names() = %v, want [nats]", names)
	}

	got, ok := multi.PublisherByName("nats")
	if !ok {
		t.Fatal("PublisherByName(nats): expected found")
	}

	if got != event.Publisher(external) {
		t.Fatalf("PublisherByName(nats) = %p, want the external publisher", got)
	}

	// Positionally added entries have no name binding.
	if _, ok := multi.PublisherByName("local"); ok {
		t.Fatal("PublisherByName(local): positional entries must not resolve by name")
	}

	if _, ok := multi.PublisherByName("ghost"); ok {
		t.Fatal("PublisherByName(ghost): expected not found")
	}
}

// TestSystem_PublisherFor verifies the by-name binding against the YAML
// Publish targets: each target name resolves to its own fan-out bus, unknown
// names and single-bus deployments report false.
func TestSystem_PublisherFor(t *testing.T) {
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

	defer sys.Close()

	bus1, ok := sys.PublisherFor("bus1")
	if !ok {
		t.Fatal("PublisherFor(bus1): expected found")
	}

	bus2, ok := sys.PublisherFor("bus2")
	if !ok {
		t.Fatal("PublisherFor(bus2): expected found")
	}

	if bus1 == bus2 {
		t.Fatal("PublisherFor(bus1) and PublisherFor(bus2) must be distinct buses")
	}

	if _, ok := sys.PublisherFor("ghost"); ok {
		t.Fatal("PublisherFor(ghost): expected not found")
	}

	if _, ok := sys.PublisherFor("local"); ok {
		t.Fatal("PublisherFor(local): the local bus has no name binding")
	}
}

// TestSystem_PublisherFor_SingleBus verifies the negative case: without a
// multi-bus deployment there is no by-name binding.
func TestSystem_PublisherFor_SingleBus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if _, ok := sys.PublisherFor("anything"); ok {
		t.Fatal("PublisherFor on a single-bus deployment must report false")
	}
}

// TestSystem_PublisherFor_SingleNamedTarget pins the single-target fan-out
// fix (M09/F29): a source-of-truth with exactly ONE publish target gets the
// same MultiBus fan-out as a multi-target deployment — the named bus is
// reachable via PublisherFor, and the local bus stays entry 0 so existing
// subscribers are unaffected. Before the fix, len(Publish)==1 silently fell
// back to the local bus and the operator's named target never existed.
func TestSystem_PublisherFor_SingleNamedTarget(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Buses:   map[string]system.BusConfig{"orders": {Driver: "gochannel"}},
		Instances: []system.InstanceConfig{{
			Role:    system.RoleSourceOfTruth,
			Engine:  "primary",
			Publish: []string{"orders"},
		}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	orders, ok := sys.PublisherFor("orders")
	if !ok {
		t.Fatal("PublisherFor(orders): single publish target must resolve to a named fan-out bus")
	}

	if orders == sys.Publisher() {
		t.Fatal("the named fan-out bus must be distinct from the publisher itself")
	}

	multi, ok := sys.Publisher().(*system.MultiBus)
	if !ok {
		t.Fatalf(
			"Publisher must be a MultiBus for any declared publish target, got %T",
			sys.Publisher(),
		)
	}

	if len(multi.Publishers()) != 2 {
		t.Fatalf(
			"MultiBus must carry local + named bus, got %d publishers",
			len(multi.Publishers()),
		)
	}
}

// TestSystem_New_UnknownPublishTargetFails pins M09/F30: a publish target
// that is not declared under buses fails construction with
// ErrUnknownPublishTarget instead of silently creating a fan-out to a bus the
// operator never configured.
func TestSystem_New_UnknownPublishTargetFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{
			Role:    system.RoleSourceOfTruth,
			Engine:  "primary",
			Publish: []string{"ghost"},
		}},
	})
	if err == nil {
		defer sys.Close()
		t.Fatal("system.New must reject a publish target that is not declared under buses")
	}

	if !errors.Is(err, system.ErrUnknownPublishTarget) {
		t.Fatalf("error must wrap ErrUnknownPublishTarget, got: %v", err)
	}

	if !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("error must name the unknown target, got: %v", err)
	}
}
