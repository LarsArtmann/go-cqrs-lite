package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC020_DetectsPanicInInlineHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(bus *Bus) {
	bus.SubscribeAll(func(evt Event) {
		if evt.ID() == "" {
			panic("empty id")
		}
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC020Detector(ctx))
	assertRule(t, findings, "C020", 1)
}

func TestC020_NoFindingForCleanHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(bus *Bus) {
	bus.SubscribeAll(func(evt Event) {
		if evt.ID() == "" {
			return
		}
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC020Detector(ctx))
	assertRule(t, findings, "C020", 0)
}
