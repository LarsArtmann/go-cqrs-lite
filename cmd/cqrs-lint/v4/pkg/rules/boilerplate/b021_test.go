package boilerplate_test

import (
	"go/token"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
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

// TestB021_MethodFoldSuppressedByStrictApply pins the method-fold half of the
// suppression: StrictApplyFolds keys on the bare fold name, while a method
// fold's FuncName is "(Recv).Fold" — without the last-segment fallback the
// rule would keep firing for such folds after decider.StrictApply adoption.
// The registry is injected directly because the fold scanner currently skips
// receiver functions (methods never enter Registry.Folds).
func TestB021_MethodFoldSuppressedByStrictApply(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

type State struct{ Count int }
`,
	})
	ctx.Registry.Folds = append(ctx.Registry.Folds, analyzer.FoldInfo{
		FuncName:   "(State).Fold",
		File:       "fold.go",
		Pos:        token.Position{Filename: "fold.go", Line: 4, Column: 1},
		HasDefault: true,
		DefaultNil: true,
	})
	ctx.Registry.StrictApplyFolds["Fold"] = true

	findings := ruletest.RunDetector(t, boilerplate.NewB021Detector(ctx))
	ruletest.AssertRule(t, findings, "B021", 0)
}

// TestB021_MethodFoldStillFiresWithoutStrictApply: the same injected method
// fold with no decider.StrictApply record must still fire (the suppression
// must not over-suppress).
func TestB021_MethodFoldStillFiresWithoutStrictApply(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

type State struct{ Count int }
`,
	})
	ctx.Registry.Folds = append(ctx.Registry.Folds, analyzer.FoldInfo{
		FuncName:   "(State).Fold",
		File:       "fold.go",
		Pos:        token.Position{Filename: "fold.go", Line: 4, Column: 1},
		HasDefault: true,
		DefaultNil: true,
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
