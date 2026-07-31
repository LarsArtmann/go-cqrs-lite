package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
)

func TestA032_DetectsStringID(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

type User struct {
	UserID  string
	OrderID string
	Name    string
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA032Detector(ctx))
	ruletest.AssertRule(t, findings, "A032", 2)
}

func TestA032_NoFindingForBrandedID(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

type UserID = id.Of[id.UserMarker]

type User struct {
	UserID UserID
	Name   string
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA032Detector(ctx))
	ruletest.AssertRule(t, findings, "A032", 0)
}

func TestA032_NoFindingWithoutIDImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	UserID string
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA032Detector(ctx))
	ruletest.AssertRule(t, findings, "A032", 0)
}

func TestA032_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA032Detector(ctx))
	ruletest.AssertRule(t, findings, "A032", 0)
}
