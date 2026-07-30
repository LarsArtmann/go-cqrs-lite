package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
)

func TestB028_DetectsManualGoroutineDispatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, evt Event) error {
	go func() {
		disp.Dispatch(ctx, cmd)
	}()
	return nil
}
`,
	})
	findings := runDetector(t, boilerplate.NewB028Detector(ctx))
	assertRule(t, findings, "B028", 1)
}

func TestB028_NoFindingForSynchronousDispatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, evt Event) error {
	disp.Dispatch(ctx, cmd)
	return nil
}
`,
	})
	findings := runDetector(t, boilerplate.NewB028Detector(ctx))
	assertRule(t, findings, "B028", 0)
}

func TestB028_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB028Detector(ctx))
	assertRule(t, findings, "B028", 0)
}
