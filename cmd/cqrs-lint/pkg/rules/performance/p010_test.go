package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP010_DetectsLargeAggregateWithoutSnapshot(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type CartState struct {
	Items    []CartItem
	Total    int
	Discount float64
}

type CartItem struct {
	SKU   string
	Qty   int
	Price float64
}
`,
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository[CartState](store, bus, decider.Decider[CartState]{})
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 1)
}

func TestP010_DetectsMapStateWithoutSnapshot(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type GameState struct {
	Players map[string]Player
	Score   int
}

type Player struct {
	Name string
}
`,
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewTypedRepository[GameState, JoinCmd](store, bus, d)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 1)
}

func TestP010_NoFindingWhenSnapshotStrategySet(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type CartState struct {
	Items    []CartItem
	Total    int
}
`,
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository[CartState](store, bus, decider.Decider[CartState]{},
		decider.WithSnapshotStore[CartState](snapStore),
		decider.WithSnapshotStrategy[CartState](snapshot.EveryNEvents(100)))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 0)
}

func TestP010_NoFindingWhenStateCacheSet(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type CartState struct {
	Items    []CartItem
	Total    int
}
`,
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository[CartState](store, bus, decider.Decider[CartState]{},
		decider.WithStateCache[CartState](cache))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 0)
}

func TestP010_NoFindingForScalarState(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type CounterState struct {
	Count int
	Label string
}
`,
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository[CounterState](store, bus, decider.Decider[CounterState]{})
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 0)
}

func TestP010_NoFindingForNonDeciderNewRepository(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

type CartState struct {
	Items []CartItem
}
`,
		"setup.go": `package main

func setup() {
	repo := other.NewRepository[CartState](store)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP010Detector(ctx))
	ruletest.AssertRule(t, findings, "P010", 0)
}
