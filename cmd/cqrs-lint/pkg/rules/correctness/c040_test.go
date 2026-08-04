package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC040_DeadFoldCase(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
		// handled
	case "user.deleted":
		// dead — nobody emits this
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 1)
}

func TestC040_NoFindingOnExactMatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
	_ = event.New("user.deleted", streamID, "User", UserDeleted{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
	case "user.deleted":
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 0)
}

func TestC040_NoFindingOnNearMiss(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.creted", streamID, "User", UserCreated{}) // typo
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
		// C038 catches this mismatch from the emit side
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 0)
}

func TestC040_NoFindingOnNormalizedMatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("UserProfileUpdated", streamID, "User", UserProfileUpdated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.profile.updated":
		// different naming convention but same event
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 0)
}

func TestC040_NoFindingWithoutEmissions(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
	case "user.deleted":
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 0)
}

func TestC040_NoFindingWithoutFolds(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 0)
}

func TestC040_MultipleDeadCases(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
	case "user.deleted":
	case "user.archived":
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC040Detector(ctx))
	ruletest.AssertRule(t, findings, "C040", 2)
}
