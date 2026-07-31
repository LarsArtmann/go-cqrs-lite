//nolint:dupl // similar test structure is intentional
package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestB027_DetectsHardcodedStreamType(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func createEvent(aggID string) {
	evt := event.New("user.created", aggID, "User", version, payload)
	_ = evt
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB027Detector(ctx))
	ruletest.AssertRule(t, findings, "B027", 1)
}

func TestB027_DetectsHardcodedInExecute(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(ctx context.Context) {
	repo.Execute(ctx, aggID, "Order", decide)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB027Detector(ctx))
	ruletest.AssertRule(t, findings, "B027", 1)
}

func TestB027_NoFindingForConstantStreamType(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

const streamType = "User"

func createEvent(aggID string) {
	evt := event.New("user.created", aggID, streamType, version, payload)
	_ = evt
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB027Detector(ctx))
	ruletest.AssertRule(t, findings, "B027", 0)
}

func TestB027_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB027Detector(ctx))
	ruletest.AssertRule(t, findings, "B027", 0)
}
