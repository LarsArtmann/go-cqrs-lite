package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestA027_DetectsRepeatedWithCodec(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

func emit() {
	_ = event.New("a", "id", "T", 1, p, event.WithCodec(c.JSONCodec{}))
	_ = event.New("b", "id", "T", 2, p, event.WithCodec(c.JSONCodec{}))
	_ = event.New("c", "id", "T", 3, p, event.WithCodec(c.JSONCodec{}))
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA027Detector(ctx))
	ruletest.AssertRule(t, findings, "A027", 1)
}

func TestA027_NoFindingForFewCalls(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

func emit() {
	_ = event.New("a", "id", "T", 1, p, event.WithCodec(c.JSONCodec{}))
	_ = event.New("b", "id", "T", 2, p)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA027Detector(ctx))
	ruletest.AssertRule(t, findings, "A027", 0)
}
