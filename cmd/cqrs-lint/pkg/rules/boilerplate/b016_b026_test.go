package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
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
	findings := runDetector(t, boilerplate.NewB016Detector(ctx))
	assertRule(t, findings, "B016", 1)
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
	findings := runDetector(t, boilerplate.NewB016Detector(ctx))
	assertRule(t, findings, "B016", 0)
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
	findings := runDetector(t, boilerplate.NewB017Detector(ctx))
	assertRule(t, findings, "B017", 1)
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
	findings := runDetector(t, boilerplate.NewB017Detector(ctx))
	assertRule(t, findings, "B017", 0)
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
	findings := runDetector(t, boilerplate.NewB018Detector(ctx))
	assertRule(t, findings, "B018", 1)
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
	findings := runDetector(t, boilerplate.NewB018Detector(ctx))
	assertRule(t, findings, "B018", 0)
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
	findings := runDetector(t, boilerplate.NewB019Detector(ctx))
	assertRule(t, findings, "B019", 1)
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
	findings := runDetector(t, boilerplate.NewB019Detector(ctx))
	assertRule(t, findings, "B019", 0)
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
	findings := runDetector(t, boilerplate.NewB020Detector(ctx))
	assertRule(t, findings, "B020", 1)
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
	findings := runDetector(t, boilerplate.NewB020Detector(ctx))
	assertRule(t, findings, "B020", 0)
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
	findings := runDetector(t, boilerplate.NewB022Detector(ctx))
	assertRule(t, findings, "B022", 1)
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
	findings := runDetector(t, boilerplate.NewB022Detector(ctx))
	assertRule(t, findings, "B022", 0)
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
	findings := runDetector(t, boilerplate.NewB025Detector(ctx))
	assertRule(t, findings, "B025", 1)
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
	findings := runDetector(t, boilerplate.NewB025Detector(ctx))
	assertRule(t, findings, "B025", 0)
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
	findings := runDetector(t, boilerplate.NewB026Detector(ctx))
	assertRule(t, findings, "B026", 1)
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
	findings := runDetector(t, boilerplate.NewB026Detector(ctx))
	assertRule(t, findings, "B026", 0)
}
