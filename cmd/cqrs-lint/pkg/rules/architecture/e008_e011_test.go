package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
)

// --- E008: Stack preset bypass ---

func TestE008_DetectsStackBypass(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/stack"
	"github.com/larsartmann/go-cqrs-lite/decider"
)

func setup() {
	_ = decider.NewRepository(nil, nil, decider.Decider[State]{})
	_ = stack.Bundle{}
}

type State struct{}
`,
	})
	findings := runDetector(t, architecture.NewE008Detector(ctx))
	assertRule(t, findings, "E008", 1)
}

func TestE008_NoFindingWithoutStackImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider"

func setup() {
	_ = decider.NewRepository(nil, nil, decider.Decider[State]{})
}

type State struct{}
`,
	})
	findings := runDetector(t, architecture.NewE008Detector(ctx))
	assertRule(t, findings, "E008", 0)
}

func TestE008_NoFindingWithoutDirectNewRepository(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack"

func setup() {
	_ = stack.Bundle{}
}
`,
	})
	findings := runDetector(t, architecture.NewE008Detector(ctx))
	assertRule(t, findings, "E008", 0)
}

// --- E009: No HTTP integration ---

func TestE009_DetectsNoHTTP(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func setup() {
	_ = command.BasicCommand{}
	_ = query.PaginatedResult[any]{}
}
`,
	})
	findings := runDetector(t, architecture.NewE009Detector(ctx))
	assertRule(t, findings, "E009", 1)
}

func TestE009_NoFindingWithHTTPTransport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
	"github.com/larsartmann/go-cqrs-lite/transport/http"
)

func setup() {
	_ = command.BasicCommand{}
	_ = query.PaginatedResult[any]{}
	_ = http.SSEBroker{}
}
`,
	})
	findings := runDetector(t, architecture.NewE009Detector(ctx))
	assertRule(t, findings, "E009", 0)
}

func TestE009_NoFindingWithOnlyCommand(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/command"

func setup() {
	_ = command.BasicCommand{}
}
`,
	})
	findings := runDetector(t, architecture.NewE009Detector(ctx))
	assertRule(t, findings, "E009", 0)
}

// --- E010: Event capture without validation ---

func TestE010_DetectsDirectStoreSave(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func capture(store Store) {
	store.Save(nil)
}

type Store interface{ Save(any) error }
`,
	})
	findings := runDetector(t, architecture.NewE010Detector(ctx))
	assertRule(t, findings, "E010", 1)
}

func TestE010_NoFindingWhenUsingDecider(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func handle(store Store, decider Decider) {
	store.Save(nil)
	decider.Execute(nil, nil)
}

type Store interface{ Save(any) error }
type Decider interface{ Execute(any, any) error }
`,
	})
	findings := runDetector(t, architecture.NewE010Detector(ctx))
	assertRule(t, findings, "E010", 0)
}

func TestE010_NoFindingWithoutStoreSave(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup() {
	println("hello")
}
`,
	})
	findings := runDetector(t, architecture.NewE010Detector(ctx))
	assertRule(t, findings, "E010", 0)
}

// --- E011: Excessive adapter layers ---

func TestE011_DetectsThreeAdapters(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"adapters.go": `package main

type EventSourcingAdapter struct{}
type BusAdapter struct{}
type CommandAdapter struct{}
`,
	})
	findings := runDetector(t, architecture.NewE011Detector(ctx))
	assertRule(t, findings, "E011", 1)
}

func TestE011_NoFindingWithTwoAdapters(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"adapters.go": `package main

type EventSourcingAdapter struct{}
type BusAdapter struct{}
`,
	})
	findings := runDetector(t, architecture.NewE011Detector(ctx))
	assertRule(t, findings, "E011", 0)
}

func TestE011_NoFindingOnEmptyProject(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE011Detector(ctx))
	assertRule(t, findings, "E011", 0)
}
