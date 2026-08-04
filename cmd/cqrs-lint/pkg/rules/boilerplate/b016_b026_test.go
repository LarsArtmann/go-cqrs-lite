package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestB016_DetectsManualCheckpointReplay(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

func setupReplay() {
	db.Exec("CREATE TABLE checkpoint (id TEXT, position INTEGER)")
	events, _ := journal.ReadFrom(ctx, lastPos)
	for evt := range events {
		process(evt)
	}
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB016Detector(ctx))
	ruletest.AssertRule(t, findings, "B016", 1)
}

func TestB016_NoFindingWithoutJournalLoop(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

func setupDB() {
	db.Exec("CREATE TABLE checkpoint (id TEXT, position INTEGER)")
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB016Detector(ctx))
	ruletest.AssertRule(t, findings, "B016", 0)
}

func TestB017_DetectsManualRebuild(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func Rebuild(ctx context.Context) error {
	events, _ := store.ReadAll(ctx)
	for evt := range events {
		applyEvent(evt)
	}
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB017Detector(ctx))
	ruletest.AssertRule(t, findings, "B017", 1)
}

func TestB017_NoFindingForRehydrateWithCheckpoint(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func Rehydrate(ctx context.Context) error {
	cp, _ := cpStore.Load(ctx, "my-proj")
	events, _ := journal.ReadFrom(ctx, cp.Position)
	for evt := range events {
		applyEvent(evt)
	}
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB017Detector(ctx))
	ruletest.AssertRule(t, findings, "B017", 0)
}

func TestB018_DetectsRepeatedSubscribe(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"projections.go": `package main

func setup(bus *eventBus) {
	bus.Subscribe("user.created", handleUserCreated)
	bus.Subscribe("user.updated", handleUserUpdated)
	bus.Subscribe("user.deleted", handleUserDeleted)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB018Detector(ctx))
	ruletest.AssertRule(t, findings, "B018", 1)
}

func TestB018_NoFindingForTwoSubscribes(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"projections.go": `package main

func setup(bus *eventBus) {
	bus.Subscribe("user.created", handleUserCreated)
	bus.Subscribe("user.updated", handleUserUpdated)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB018Detector(ctx))
	ruletest.AssertRule(t, findings, "B018", 0)
}

func TestB019_DetectsLoadInSubscribeAll(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cqrs.go": `package main

func setup(bus *eventBus) {
	bus.SubscribeAll(func(evt event.Event) error {
		state, _ := repo.Load(ctx, streamID)
		_ = state
		return nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB019Detector(ctx))
	ruletest.AssertRule(t, findings, "B019", 1)
}

func TestB019_NoFindingForDirectProjection(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cqrs.go": `package main

func setup(bus *eventBus) {
	bus.SubscribeAll(func(evt event.Event) error {
		view := projectFromEvent(evt)
		_ = view
		return nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB019Detector(ctx))
	ruletest.AssertRule(t, findings, "B019", 0)
}

func TestB020_DetectsManualUpcasting(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"item_adapter.go": `package main

import "encoding/json"

func decodeItem(data []byte) (*Item, error) {
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if old, ok := raw["oldName"]; ok {
		raw["newName"] = old
	}
	b, _ := json.Marshal(raw)
	var item Item
	json.Unmarshal(b, &item)
	return &item, nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB020Detector(ctx))
	ruletest.AssertRule(t, findings, "B020", 1)
}

func TestB020_NoFindingInsideSchemaUpcaster(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"upcaster.go": `package main

import "encoding/json"

func makeUpcaster() {
	schema.NewUpcaster("ItemCreated", 1, func(evt event.Event) (*event.ImmutableEvent, error) {
		var raw map[string]any
		json.Unmarshal(evt.Payload(), &raw)
		if old, ok := raw["oldName"]; ok {
			raw["newName"] = old
		}
		return evt, nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB020Detector(ctx))
	ruletest.AssertRule(t, findings, "B020", 0)
}

func TestB022_DetectsCustomEnricher(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d, correlation.ContextEnricher())
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB022Detector(ctx))
	ruletest.AssertRule(t, findings, "B022", 1)
}

func TestB022_NoFindingForCommandCausalityEnricher(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d, decider.CommandCausalityEnricher())
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB022Detector(ctx))
	ruletest.AssertRule(t, findings, "B022", 0)
}

func TestB022_NoFindingForWithEnricherWrappingCanonical(t *testing.T) {
	t.Parallel()

	// This is the exact pattern consumers use: WithEnricher(event.CommandCausalityEnricher).
	// B022 must NOT fire — it is the canonical enricher, not a custom one.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d,
		decider.WithEnricher(event.CommandCausalityEnricher))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB022Detector(ctx))
	ruletest.AssertRule(t, findings, "B022", 0)
}

func TestB022_DetectsCustomEnricherInWithEnricher(t *testing.T) {
	t.Parallel()

	// WithEnricher wrapping a truly custom enricher function should still fire.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d,
		decider.WithEnricher(customCorrelationEnricher))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB022Detector(ctx))
	ruletest.AssertRule(t, findings, "B022", 1)
}

func TestB025_DetectsMissingStateCache(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB025Detector(ctx))
	ruletest.AssertRule(t, findings, "B025", 1)
}

func TestB025_NoFindingWithStateCache(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	cache := decider.NewStateCache[State](256)
	repo, _ := decider.NewRepository(store, bus, d, decider.WithStateCache(cache))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB025Detector(ctx))
	ruletest.AssertRule(t, findings, "B025", 0)
}

// TestB025_NoFindingWithStateCacheViaHelper verifies the detector traces
// through an option-builder helper spread into NewRepository. This is the
// common pattern in libraries with reusable wiring (cqrs-htmx): the helper
// constructs WithStateCache, but the call site only shows a variadic spread.
func TestB025_NoFindingWithStateCacheViaHelper(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func repositoryOptions(cfg Config) []decider.RepositoryOption[State] {
	opts := snapshotOptions(cfg)
	return append(opts, decider.WithStateCache[State](decider.NewStateCache[State](0)))
}

func setup() {
	repo, _ := decider.NewRepository(store, bus, d, repositoryOptions(cfg)...)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB025Detector(ctx))
	ruletest.AssertRule(t, findings, "B025", 0)
}

// TestB025_NoFindingWithStateCacheViaGenericHelper covers the generic
// instantiation form: repositoryOptions[State](cfg)... where the helper is a
// generic function. This is the exact pattern reported by the cqrs-htmx
// consumer (helper builds options across multiple State types).
func TestB025_NoFindingWithStateCacheViaGenericHelper(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func repositoryOptions[State any](cfg Config) []decider.RepositoryOption[State] {
	return []decider.RepositoryOption[State]{
		decider.WithStateCache[State](decider.NewStateCache[State](0)),
	}
}

func setup() {
	repo, _ := decider.NewRepository(store, bus, d, repositoryOptions[State](cfg)...)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB025Detector(ctx))
	ruletest.AssertRule(t, findings, "B025", 0)
}

// TestB025_FiresWhenHelperLacksStateCache ensures the detector still fires
// when a helper IS used but does NOT wire WithStateCache — the trace must not
// suppress genuine missing-cache findings.
func TestB025_FiresWhenHelperLacksStateCache(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func repositoryOptions(cfg Config) []decider.RepositoryOption[State] {
	return []decider.RepositoryOption[State]{
		decider.WithSnapshotStore[State](snap),
	}
}

func setup() {
	repo, _ := decider.NewRepository(store, bus, d, repositoryOptions(cfg)...)
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB025Detector(ctx))
	ruletest.AssertRule(t, findings, "B025", 1)
}

func TestB026_DetectsMissingCatalogRegistration(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func createEvents() {
	event.New("user.created", id1, "User", event.Version(1), payload1)
	event.New("user.updated", id2, "User", event.Version(1), payload2)
	event.New("user.deleted", id3, "User", event.Version(1), payload3)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB026Detector(ctx))
	ruletest.AssertRule(t, findings, "B026", 1)
}

func TestB026_NoFindingWithCatalogImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func createEvents() {
	event.New("user.created", id1, "User", event.Version(1), payload1)
	event.New("user.updated", id2, "User", event.Version(1), payload2)
	event.New("user.deleted", id3, "User", event.Version(1), payload3)
	catalog.NewBuilder()
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB026Detector(ctx))
	ruletest.AssertRule(t, findings, "B026", 0)
}
