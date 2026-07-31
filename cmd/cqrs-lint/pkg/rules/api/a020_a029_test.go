package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// --- A020: Custom event.Bus reimplementation ---

func TestA020_DetectsCustomBus(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CustomBus struct{}

func (b *CustomBus) Subscribe(topic string, handler func(evt event.Event)) {}
func (b *CustomBus) SubscribeAll(handler func(evt event.Event)) {}
func (b *CustomBus) Use(mw interface{}) {}
func (b *CustomBus) UsePublish(mw interface{}) error { return nil }
func (b *CustomBus) Close() error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA020Detector(ctx))
	ruletest.AssertRule(t, findings, "A020", 1)
}

func TestA020_NoFindingForPartialBus(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type PartialBus struct{}

func (b *PartialBus) Subscribe(topic string, handler func(evt event.Event)) {}
func (b *PartialBus) Use(mw interface{}) {}
func (b *PartialBus) Close() error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA020Detector(ctx))
	ruletest.AssertRule(t, findings, "A020", 0)
}

func TestA020_NoFindingWithoutUsePublish(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type NoPublishBus struct{}

func (b *NoPublishBus) Subscribe(topic string, handler func(evt event.Event)) {}
func (b *NoPublishBus) SubscribeAll(handler func(evt event.Event)) {}
func (b *NoPublishBus) Use(mw interface{}) {}
func (b *NoPublishBus) Publish(evt event.Event) {}
func (b *NoPublishBus) Close() error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA020Detector(ctx))
	ruletest.AssertRule(t, findings, "A020", 0)
}

func TestA020_NoFindingWithoutCQRSImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

type CustomBus struct{}

func (b *CustomBus) Subscribe(topic string, handler func()) {}
func (b *CustomBus) SubscribeAll(handler func()) {}
func (b *CustomBus) Use(mw interface{}) {}
func (b *CustomBus) UsePublish(mw interface{}) error { return nil }
func (b *CustomBus) Close() error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA020Detector(ctx))
	ruletest.AssertRule(t, findings, "A020", 0)
}

// --- A021: Custom event.Store reimplementation ---

func TestA021_DetectsCustomStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CustomStore struct{}

func (s *CustomStore) Save(ctx interface{}, ref string, events []event.Event) error { return nil }
func (s *CustomStore) Load(ctx interface{}, ref string) ([]event.Event, error) { return nil, nil }
func (s *CustomStore) LoadFromVersion(ctx interface{}, ref string, ver event.Version) ([]event.Event, error) {
	return nil, nil
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA021Detector(ctx))
	ruletest.AssertRule(t, findings, "A021", 1)
}

func TestA021_NoFindingForPartialStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

type CustomStore struct{}

func (s *CustomStore) Save(ctx interface{}, ref string) error { return nil }
func (s *CustomStore) Load(ctx interface{}, ref string) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA021Detector(ctx))
	ruletest.AssertRule(t, findings, "A021", 0)
}

// --- A022: Raw otel.Tracer/Meter ---

func TestA022_DetectsRawOtelTracer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"go.opentelemetry.io/otel"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func setup() {
	tracer := otel.Tracer("my-app")
	_ = tracer
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA022Detector(ctx))
	ruletest.AssertRule(t, findings, "A022", 1)
}

func TestA022_DetectsRawOtelMeter(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"go.opentelemetry.io/otel"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func setup() {
	meter := otel.Meter("my-app")
	_ = meter
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA022Detector(ctx))
	ruletest.AssertRule(t, findings, "A022", 1)
}

func TestA022_NoFindingForCqrsotel(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func setup() {
	tracer := cqrsotel.NewTracer("my-app")
	_ = tracer
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA022Detector(ctx))
	ruletest.AssertRule(t, findings, "A022", 0)
}

func TestA022_NoFindingWithoutCQRSImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import "go.opentelemetry.io/otel"

func setup() {
	tracer := otel.Tracer("my-app")
	_ = tracer
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA022Detector(ctx))
	ruletest.AssertRule(t, findings, "A022", 0)
}

// --- A023: Custom snapshot store ---

func TestA023_DetectsCustomSnapshotStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"snapshot.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type MemorySnapshotStore struct{}

func (s *MemorySnapshotStore) Save(ctx interface{}, ref string, snap interface{}) error { return nil }
func (s *MemorySnapshotStore) Load(ctx interface{}, ref string) (interface{}, error) {
	return nil, nil
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA023Detector(ctx))
	ruletest.AssertRule(t, findings, "A023", 1)
}

func TestA023_NoFindingForNonSnapshotNamedStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CustomStore struct{}

func (s *CustomStore) Save(ctx interface{}, ref string) error { return nil }
func (s *CustomStore) Load(ctx interface{}, ref string) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA023Detector(ctx))
	ruletest.AssertRule(t, findings, "A023", 0)
}

// --- A024: Decorative event sourcing ---

func TestA024_DetectsDecorativeEventSourcing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
)

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) { return s, nil }

var d = decider.Decider[State]{Initial: State{}, Fold: fold}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA024Detector(ctx))
	ruletest.AssertRule(t, findings, "A024", 1)
}

func TestA024_NoFindingWhenWired(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
)

func setup() {
	repo := decider.NewRepository(store, bus, d)
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA024Detector(ctx))
	ruletest.AssertRule(t, findings, "A024", 0)
}

func TestA024_NoFindingWithoutEventImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider/v4"

type State struct{ Count int }
var d = decider.Decider[State]{Initial: State{}}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA024Detector(ctx))
	ruletest.AssertRule(t, findings, "A024", 0)
}

// --- A025: Command/query only, no events ---

func TestA025_DetectsCommandQueryWithoutEvents(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func setup() {
	d := command.NewDispatcher()
	_ = d
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA025Detector(ctx))
	ruletest.AssertRule(t, findings, "A025", 1)
}

func TestA025_NoFindingWhenEventSourcingPresent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func setup() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA025Detector(ctx))
	ruletest.AssertRule(t, findings, "A025", 0)
}

// --- A026: Event bus only, no CQRS pipeline ---

func TestA026_DetectsEventBusOnly(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func setup() {
	bus := watermill.NewEventBus(pub, "events")
	_ = bus
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA026Detector(ctx))
	ruletest.AssertRule(t, findings, "A026", 1)
}

func TestA026_NoFindingWhenCommandPresent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

func setup() {
	_ = command.NewDispatcher()
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA026Detector(ctx))
	ruletest.AssertRule(t, findings, "A026", 0)
}

// --- A029: UsePublish stub returning nil ---

func TestA029_DetectsUsePublishStub(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CustomBus struct{}

func (b *CustomBus) UsePublish(mw interface{}) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA029Detector(ctx))
	ruletest.AssertRule(t, findings, "A029", 1)
}

func TestA029_NoFindingForImplementedUsePublish(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CustomBus struct {
	middlewares []interface{}
}

func (b *CustomBus) UsePublish(mw interface{}) error {
	b.middlewares = append(b.middlewares, mw)
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA029Detector(ctx))
	ruletest.AssertRule(t, findings, "A029", 0)
}

func TestA029_NoFindingWithoutCQRSImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

type CustomBus struct{}

func (b *CustomBus) UsePublish(mw interface{}) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA029Detector(ctx))
	ruletest.AssertRule(t, findings, "A029", 0)
}
