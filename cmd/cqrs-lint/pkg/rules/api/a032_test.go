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
	findings := runDetector(t, api.NewA032Detector(ctx))
	assertRule(t, findings, "A032", 2)
}

func TestA032_NoFindingForBrandedID(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": "package main\n\nimport \"github.com/larsartmann/go-cqrs-lite/id/v4\"\n\ntype UserID = id.Of[id.UserMarker]\n\ntype User struct {\n\tUserID Name   string",
	})
	findings := runDetector(t, api.NewA032Detector(ctx))
	assertRule(t, findings, "A032", 0)
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
	findings := runDetector(t, api.NewA032Detector(ctx))
	assertRule(t, findings, "A032", 0)
}

func TestA032_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, api.NewA032Detector(ctx))
	assertRule(t, findings, "A032", 0)
}
