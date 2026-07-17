package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
)

// --- B002: Manual repository wiring ---

func TestB002_DetectsManualWiring(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := storage.NewEventStore(db)
	bus := event.NewBus(store)
	repo := decider.NewRepository(store, bus, d)
	_ = repo
}
`,
	})
	findings := runDetector(t, boilerplate.NewB002Detector(ctx))
	assertRule(t, findings, "B002", 1)
}

func TestB002_NoFindingForSimpleFunc(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})
	findings := runDetector(t, boilerplate.NewB002Detector(ctx))
	assertRule(t, findings, "B002", 0)
}

// --- B003: SubscribeAll with large switch ---

func TestB003_DetectsLargeSwitch(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.SubscribeAll(func(evt Event) {
		switch evt.Type() {
		case "a":
		case "b":
		case "c":
		case "d":
		case "e":
		case "f":
		}
	})
}
`,
	})
	findings := runDetector(t, boilerplate.NewB003Detector(ctx))
	assertRule(t, findings, "B003", 1)
}

func TestB003_NoFindingForSmallSwitch(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func subscribe(bus *Bus) {
	bus.SubscribeAll(func(evt Event) {
		switch evt.Type() {
		case "a":
		case "b":
		}
	})
}
`,
	})
	findings := runDetector(t, boilerplate.NewB003Detector(ctx))
	assertRule(t, findings, "B003", 0)
}

// --- B006: Duplicate FK stub SQL ---

func TestB006_DetectsDuplicateFK(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"migrations.go": `package main

const userFK = "REFERENCES users(id)"
const orderFK = "REFERENCES users(id)"
`,
	})
	findings := runDetector(t, boilerplate.NewB006Detector(ctx))
	assertRule(t, findings, "B006", 1)
}

func TestB006_NoFindingForUniqueFK(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"migrations.go": `package main

const userFK = "REFERENCES users(id)"
const orderFK = "REFERENCES orders(id)"
`,
	})
	findings := runDetector(t, boilerplate.NewB006Detector(ctx))
	assertRule(t, findings, "B006", 0)
}

// --- B007: Repeated handler registration ---

func TestB007_DetectsRepeatedRegistrations(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(d *Dispatcher) {
	d.Register("cmd1", h1)
	d.Register("cmd2", h2)
	d.Register("cmd3", h3)
}
`,
	})
	findings := runDetector(t, boilerplate.NewB007Detector(ctx))
	assertRule(t, findings, "B007", 1)
}

func TestB007_NoFindingForTwoRegistrations(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(d *Dispatcher) {
	d.Register("cmd1", h1)
	d.Register("cmd2", h2)
}
`,
	})
	findings := runDetector(t, boilerplate.NewB007Detector(ctx))
	assertRule(t, findings, "B007", 0)
}

// --- B008: Manual retry ---

func TestB008_DetectsManualRetry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"retry.go": `package main

import "time"

func withRetry(fn func() error) error {
	for attempt := 0; attempt < 3; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return nil
}
`,
	})
	findings := runDetector(t, boilerplate.NewB008Detector(ctx))
	assertRule(t, findings, "B008", 1)
}

func TestB008_NoFindingForSimpleLoop(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"loop.go": `package main

func process(items []int) {
	for _, item := range items {
		_ = item
	}
}
`,
	})
	findings := runDetector(t, boilerplate.NewB008Detector(ctx))
	assertRule(t, findings, "B008", 0)
}

// --- B009: Emit function boilerplate ---

func TestB009_DetectsEmitFunction(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emitCreated(bus *Bus, id string) {
	evt := event.New("created", id, "User", 1, payload)
	bus.Publish(evt)
}
`,
	})
	findings := runDetector(t, boilerplate.NewB009Detector(ctx))
	assertRule(t, findings, "B009", 1)
}

func TestB009_NoFindingForRegularFunction(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(id string) {
	_ = id
}
`,
	})
	findings := runDetector(t, boilerplate.NewB009Detector(ctx))
	assertRule(t, findings, "B009", 0)
}

// --- B010: Catalog event list boilerplate ---

func TestB010_DetectsCatalogBoilerplate(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

func registerCatalog(r *Registry) {
	catalog.Event("user.created", UserCreated{})
	catalog.Event("user.updated", UserUpdated{})
	catalog.Event("user.deleted", UserDeleted{})
}
`,
	})
	findings := runDetector(t, boilerplate.NewB010Detector(ctx))
	assertRule(t, findings, "B010", 1)
}

func TestB010_NoFindingForTwoEvents(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

func registerCatalog(r *Registry) {
	catalog.Event("user.created", UserCreated{})
	catalog.Event("user.updated", UserUpdated{})
}
`,
	})
	findings := runDetector(t, boilerplate.NewB010Detector(ctx))
	assertRule(t, findings, "B010", 0)
}

// --- B011: Must-marshal helper ---

func TestB011_DetectsMustMarshal(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"helper.go": `package main

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
`,
	})
	findings := runDetector(t, boilerplate.NewB011Detector(ctx))
	assertRule(t, findings, "B011", 1)
}

func TestB011_NoFindingForRegularFunc(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(v any) {
	_ = v
}
`,
	})
	findings := runDetector(t, boilerplate.NewB011Detector(ctx))
	assertRule(t, findings, "B011", 0)
}

// --- B012: Make-event helper ---

func TestB012_DetectsMakeEvent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"helper.go": `package main

func makeEvent(t string, payload any) []byte {
	data, _ := json.Marshal(payload)
	return data
}
`,
	})
	findings := runDetector(t, boilerplate.NewB012Detector(ctx))
	assertRule(t, findings, "B012", 1)
}

func TestB012_NoFindingForNonEventHelper(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"helper.go": `package main

func makeRequest(id string) string {
	return id
}
`,
	})
	findings := runDetector(t, boilerplate.NewB012Detector(ctx))
	assertRule(t, findings, "B012", 0)
}

// --- B013: Missing correlation enricher ---

func TestB013_NoFindingWithoutRepository(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB013Detector(ctx))
	assertRule(t, findings, "B013", 0)
}

func TestB013_DetectsMissingCorrelation(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d)
	_ = repo
}
`,
	})
	findings := runDetector(t, boilerplate.NewB013Detector(ctx))
	assertRule(t, findings, "B013", 1)
}

// --- B014: Missing OTel middleware ---

func TestB014_NoFindingWithoutMiddleware(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB014Detector(ctx))
	assertRule(t, findings, "B014", 0)
}

func TestB014_DetectsMissingOTel(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(bus *EventBus) {
	bus.Use(loggingMiddleware)
	bus.UsePublish(retryMiddleware)
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, boilerplate.NewB014Detector(ctx))
	assertRule(t, findings, "B014", 1)
}

// --- B004: Command constructor boilerplate ---

func TestB004_NoFindingWithoutCommands(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB004Detector(ctx))
	assertRule(t, findings, "B004", 0)
}

func TestB004_DetectsManyFields(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CreateOrder struct {
	*BasicCommand
	CustomerID string
	ProductID  string
	Quantity   int
	Price      float64
	Address    string
	City       string
	Zip        string
	Country    string
}
`,
	})
	findings := runDetector(t, boilerplate.NewB004Detector(ctx))
	assertRule(t, findings, "B004", 1)
}

// --- B005: Fold switch boilerplate ---

func TestB005_NoFindingWithoutFolds(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB005Detector(ctx))
	assertRule(t, findings, "B005", 0)
}

func TestB005_DetectsFoldSwitch(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func fold(state int, event event.Event) (int, error) {
	switch event.Type() {
	case "created":
		return state + 1, nil
	default:
		return state, nil
	}
}
`,
	})
	findings := runDetector(t, boilerplate.NewB005Detector(ctx))
	assertRule(t, findings, "B005", 1)
}

// --- B015: Missing test utilities ---

func TestB015_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, boilerplate.NewB015Detector(ctx))
	assertRule(t, findings, "B015", 0)
}

func TestB015_DetectsMissingTestUtils(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main

import "testing"

func TestSomething(t *testing.T) {}
`,
	})
	findings := runDetector(t, boilerplate.NewB015Detector(ctx))
	assertRule(t, findings, "B015", 1)
}
