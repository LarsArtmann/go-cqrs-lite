package resilience_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/resilience"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestB029_BusWithoutRetry(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := newBus()
	bus.Publish(evt)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 1)
}

func TestB029_BusWithRetry(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := newBus()
	bus.Use(middleware.Retry())
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 0)
}

func TestB029_NoFindingForNonCQRSBus(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	errorBus := newErrorBus()
	errorBus.Notify(evt)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 0)
}

func TestB029_TypeAwareSkipCustomBus(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"main.go": `package main

type CustomBus struct{}

func (c *CustomBus) Use(mw any)     {}
func (c *CustomBus) Publish(evt any) {}

func main() {
	bus := &CustomBus{}
	bus.Use(nil)
}
`,
	})
	defer cleanup()
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 0)
}

// TestB029_NoFindingForReadOnlySubscriber pins the bus-kind discrimination:
// a bus variable whose only calls are Subscribe/SubscribeAll is an
// in-process journal tail the project READS — retry middleware is category
// confusion for it (CV feedback, 2026-10-03).
func TestB029_NoFindingForReadOnlySubscriber(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := engine.Bus()
	bus.SubscribeAll(handler)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 0)
}

// TestB029_NoFindingForEngineJournalTailFeed pins the FEED side of the
// journal-tail vs dispatch-bus distinction: a variable assigned from an
// engine Bus() accessor IS the in-process notification tail — publishing
// onto it feeds the fan-out (best-effort, drop-counted, journal-replayed)
// rather than calling downstream services — so retry middleware is still
// category confusion (CV feedback, 2026-10-03).
func TestB029_NoFindingForEngineJournalTailFeed(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := engine.Bus()
	bus.Publish(evt)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 0)
}

// TestB029_StillFiresWhenPublishing: a bus the PROJECT constructed (not one
// obtained from the engine) stays a dispatch pipeline when it publishes —
// the journal-tail skips must not over-suppress constructed buses.
func TestB029_StillFiresWhenPublishing(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := newBus()
	bus.SubscribeAll(handler)
	bus.Publish(evt)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 1)
}

// TestB030_NoFindingForReadOnlySubscriber is the B030 twin of the bus-kind
// discrimination: a Subscribe-only bus variable is a journal tail, not a
// dispatch path to downstream services.
func TestB030_NoFindingForReadOnlySubscriber(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := engine.Bus()
	bus.Subscribe(topic, handler)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB030Detector(ctx))
	ruletest.AssertRule(t, findings, "B030", 0)
}

// TestB030_NoFindingForEngineJournalTailFeed is the B030 twin of the
// feed-side distinction: publishing onto an engine Bus() accessor result
// feeds the in-process fan-out — circuit breaker middleware isolates no
// downstream service there (CV feedback, 2026-10-03).
func TestB030_NoFindingForEngineJournalTailFeed(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := engine.Bus()
	bus.Publish(evt)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB030Detector(ctx))
	ruletest.AssertRule(t, findings, "B030", 0)
}

func TestB030_BusWithoutCircuitBreaker(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	dispatcher := newDisp()
	dispatcher.Handle(cmd)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB030Detector(ctx))
	ruletest.AssertRule(t, findings, "B030", 1)
}

func TestB030_BusWithCircuitBreaker(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	dispatcher := newDisp()
	dispatcher.Use(middleware.CircuitBreaker(cfg))
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB030Detector(ctx))
	ruletest.AssertRule(t, findings, "B030", 0)
}

func TestB031_ProjectionHostWithoutDLQ(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	host, _ := projectionhost.New(journal, cpStore)
	host.Register(proj)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB031Detector(ctx))
	ruletest.AssertRule(t, findings, "B031", 1)
}

func TestB031_ProjectionHostWithDLQ(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	dlq, _ := projectionhost.NewSQLiteDeadLetterStore(ctx, db)
	host, _ := projectionhost.New(journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 3))
	host.Register(proj)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB031Detector(ctx))
	ruletest.AssertRule(t, findings, "B031", 0)
}

func TestB031_JournalTailFedHostStillNeedsDLQ(t *testing.T) {
	t.Parallel()

	// Symmetry pin for the B029/B030 journal-tail accessor signal: a
	// projection host fed from the engine's Bus() journal tail still needs
	// a dead-letter store — journal replay recovers LOST deliveries, but a
	// poison event replays the same fold failure forever; only a DLQ
	// isolates it. B031 keys on projectionhost.New construction and never
	// consults bus variables, so no accessor shape can suppress it.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	bus := engine.Bus()
	host, _ := projectionhost.New(bus, cpStore)
	host.Register(proj)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, resilience.NewB031Detector(ctx))
	ruletest.AssertRule(t, findings, "B031", 1)
}

// buildBusFixtureContext loads the committed busfixture module so Bus()
// accessor call sites carry genuine static result types — the typed-path
// harness BuildContextFromSource cannot provide (empty types.Info).
func buildBusFixtureContext(t *testing.T) *analyzer.AnalysisContext {
	t.Helper()

	t.Setenv("GOWORK", "off")

	fixture, err := filepath.Abs("../../../testdata/busfixture")
	if err != nil {
		t.Fatalf("abs fixture path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture, "go.mod")); err != nil {
		t.Fatalf("bus fixture missing (expected committed testdata): %v", err)
	}

	ctx, err := analyzer.BuildContext(fixture)
	if err != nil {
		t.Fatalf("BuildContext(busfixture): %v", err)
	}
	if len(ctx.LoadErrors) > 0 {
		t.Fatalf("bus fixture loaded with errors: %v", ctx.LoadErrors)
	}

	return ctx
}

// TestB029_AccessorResultTypeDiscriminates pins the typed tightening of the
// Bus() accessor signal (row 49): in ONE typed fixture, an accessor whose
// result type is the engine's event.Bus keeps the journal-tail skip, while
// a same-named accessor returning a downstream transport type
// (rabbitConn.Bus()) draws the retry-middleware advice again.
func TestB029_AccessorResultTypeDiscriminates(t *testing.T) {
	// Not parallel: buildBusFixtureContext sets GOWORK via t.Setenv.

	ctx := buildBusFixtureContext(t)

	findings := ruletest.RunDetector(t, resilience.NewB029Detector(ctx))
	ruletest.AssertRule(t, findings, "B029", 1)

	if name := findings[0].Message; !strings.Contains(name, "rabbitBus") {
		t.Errorf("B029 must anchor at rabbitBus (the transport-typed accessor), got: %s", name)
	}
}

// TestB030_AccessorResultTypeDiscriminates is the B030 twin: the
// transport-typed accessor loses the journal-tail skip and draws the
// circuit-breaker advice; the event.Bus-typed accessor stays silent.
func TestB030_AccessorResultTypeDiscriminates(t *testing.T) {
	// Not parallel: buildBusFixtureContext sets GOWORK via t.Setenv.

	ctx := buildBusFixtureContext(t)

	findings := ruletest.RunDetector(t, resilience.NewB030Detector(ctx))
	ruletest.AssertRule(t, findings, "B030", 1)

	if name := findings[0].Message; !strings.Contains(name, "rabbitBus") {
		t.Errorf("B030 must anchor at rabbitBus (the transport-typed accessor), got: %s", name)
	}
}
