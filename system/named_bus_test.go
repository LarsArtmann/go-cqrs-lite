package system_test

import (
	"context"
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
