package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

func TestD012_DetectsFmtPrintlnInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handleEvent(ctx context.Context, evt Event) error {
	fmt.Printf("got event: %v\n", evt)
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD012Detector(ctx))
	ruletest.AssertRule(t, findings, "D012", 1)
}

func TestD012_DetectsLogFatalInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handleEvent(ctx context.Context, evt Event) error {
	log.Fatal("unrecoverable")
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD012Detector(ctx))
	ruletest.AssertRule(t, findings, "D012", 1)
}

func TestD012_NoFindingForSlogInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handleEvent(ctx context.Context, evt Event) error {
	logger.Info("got event", "type", evt.Type)
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD012Detector(ctx))
	ruletest.AssertRule(t, findings, "D012", 0)
}

func TestD012_NoFindingForPrintOutsideHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	fmt.Println("starting")
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD012Detector(ctx))
	ruletest.AssertRule(t, findings, "D012", 0)
}
