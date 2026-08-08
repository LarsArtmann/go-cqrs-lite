package resilience_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/resilience"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
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

	findings := ruletest.RunDetector(t, resilience.NewB031Detector(ctx))
	ruletest.AssertRule(t, findings, "B031", 0)
}
