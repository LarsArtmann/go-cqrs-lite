package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestB021_DetectsFoldWithoutStrictApply(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	next := s
	switch evt.Type() {
	case "incremented":
		next.Count++
	default:
		return s, nil
	}
	return next, nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB021Detector(ctx))
	ruletest.AssertRule(t, findings, "B021", 1)
}

func TestB021_NoFindingForFoldWithErrorDefault(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	next := s
	switch evt.Type() {
	case "incremented":
		next.Count++
	default:
		return s, errUnknownEvent
	}
	return next, nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB021Detector(ctx))
	ruletest.AssertRule(t, findings, "B021", 0)
}
