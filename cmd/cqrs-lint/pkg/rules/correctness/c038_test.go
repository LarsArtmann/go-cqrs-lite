package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC038_EventTypeTypo(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.creted", streamID, "User", UserCreated{}) // typo: "creted"
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
		// handle
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 1)
}

func TestC038_NoTypoExactMatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
		// handle
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 0)
}

func TestC038_NoFindingWhenTooFar(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("completely.different", streamID, "User", UserCreated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.created":
		// handle
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 0)
}

func TestC038_NoFindingWithoutFolds(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 0)
}

func TestC038_NormalizationMultiSeparator(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("UserProfileUpdated", streamID, "User", UserProfileUpdated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "user.profile.updated":
		// handle
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 1)
}

func TestC038_NormalizationCatchesCaseConventionMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func decideCreate() {
	_ = event.New("user.created", streamID, "User", UserCreated{})
}

func foldUser(state State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "UserCreated":
		// PascalCase vs dot.notation — same event, different strings
		// The fold will NOT match "user.created" at runtime
	}
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC038Detector(ctx))
	ruletest.AssertRule(t, findings, "C038", 1)
}
