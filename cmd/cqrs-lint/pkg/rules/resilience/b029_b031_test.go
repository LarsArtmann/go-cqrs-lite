package resilience_test

import (
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
