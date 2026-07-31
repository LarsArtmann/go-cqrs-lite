package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC013_EmbeddedTime(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "time"

type TimestampsCreated struct {
	time.Time
	Name string
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC013Detector(ctx))
	ruletest.AssertRule(t, findings, "C013", 1)
}
