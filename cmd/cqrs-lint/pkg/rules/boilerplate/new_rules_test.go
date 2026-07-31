package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
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
	findings := ruletest.RunDetector(t, boilerplate.NewB002Detector(ctx))
	ruletest.AssertRule(t, findings, "B002", 1)
}

func TestB002_NoFindingForSimpleFunc(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB002Detector(ctx))
	ruletest.AssertRule(t, findings, "B002", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB003Detector(ctx))
	ruletest.AssertRule(t, findings, "B003", 1)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB003Detector(ctx))
	ruletest.AssertRule(t, findings, "B003", 0)
}

// --- B006: Duplicate FK stub SQL ---

func TestB006_DetectsDuplicateFK(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"migrations.go": `package main

const userFK = "REFERENCES users(id)"
const orderFK = "REFERENCES users(id)"
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB006Detector(ctx))
	ruletest.AssertRule(t, findings, "B006", 1)
}

func TestB006_NoFindingForUniqueFK(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"migrations.go": `package main

const userFK = "REFERENCES users(id)"
const orderFK = "REFERENCES orders(id)"
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB006Detector(ctx))
	ruletest.AssertRule(t, findings, "B006", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB007Detector(ctx))
	ruletest.AssertRule(t, findings, "B007", 1)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB007Detector(ctx))
	ruletest.AssertRule(t, findings, "B007", 0)
}

// B007 must NOT fire on third-party Register calls whose method name collides
// with CQRS but serves a different purpose. huma.Register is a generic HTTP
// route registration; collecting routes into a table would erase Huma's type
// safety. Regression for the browser-history false positive (12 huma.Register
// calls flagged). The denylist also covers http, mux, chi, gin, echo, fiber,
// and grpc.

func TestB007_NoFindingForHumaRegister(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"routes.go": `package main

func registerRoutes(api huma.API) {
	huma.Register(api, op1, s.health)
	huma.Register(api, op2, s.extract)
	huma.Register(api, op3, s.create)
	huma.Register(api, op4, s.list)
	huma.Register(api, op5, s.get)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB007Detector(ctx))
	ruletest.AssertRule(t, findings, "B007", 0)
}

// B007 counts CQRS registrations qualified by a variable (the idiomatic
// pattern) even when third-party Register calls are interleaved in the same
// function body. The denylist filters per-call, not per-function.

func TestB007_CountsCQRSButSkipsHumaInSameFunction(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(d *Dispatcher, api huma.API) {
	d.Register("cmd1", h1)
	d.Register("cmd2", h2)
	d.Register("cmd3", h3)
	huma.Register(api, op1, s.health)
	huma.Register(api, op2, s.extract)
	huma.Register(api, op3, s.create)
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB007Detector(ctx))
	ruletest.AssertRule(t, findings, "B007", 1)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB008Detector(ctx))
	ruletest.AssertRule(t, findings, "B008", 1)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB008Detector(ctx))
	ruletest.AssertRule(t, findings, "B008", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB009Detector(ctx))
	ruletest.AssertRule(t, findings, "B009", 1)
}

func TestB009_NoFindingForRegularFunction(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(id string) {
	_ = id
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB009Detector(ctx))
	ruletest.AssertRule(t, findings, "B009", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB010Detector(ctx))
	ruletest.AssertRule(t, findings, "B010", 1)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB010Detector(ctx))
	ruletest.AssertRule(t, findings, "B010", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB011Detector(ctx))
	ruletest.AssertRule(t, findings, "B011", 1)
}

func TestB011_NoFindingForRegularFunc(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(v any) {
	_ = v
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB011Detector(ctx))
	ruletest.AssertRule(t, findings, "B011", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB012Detector(ctx))
	ruletest.AssertRule(t, findings, "B012", 1)
}

func TestB012_NoFindingForNonEventHelper(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"helper.go": `package main

func makeRequest(id string) string {
	return id
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB012Detector(ctx))
	ruletest.AssertRule(t, findings, "B012", 0)
}

// --- B013: Missing correlation enricher ---

func TestB013_NoFindingWithoutRepository(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB013Detector(ctx))
	ruletest.AssertRule(t, findings, "B013", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB013Detector(ctx))
	ruletest.AssertRule(t, findings, "B013", 1)
}

// --- B014: Missing OTel middleware ---

func TestB014_NoFindingWithoutMiddleware(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB014Detector(ctx))
	ruletest.AssertRule(t, findings, "B014", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB014Detector(ctx))
	ruletest.AssertRule(t, findings, "B014", 1)
}

// TestB014_SuppressedForNoServer: the same middleware fixture fires when
// HasServer=true (above); a local-only project is suppressed because
// distributed tracing is noise for CLI tools.
func TestB014_SuppressedForNoServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(bus *EventBus) {
	bus.Use(loggingMiddleware)
	bus.UsePublish(retryMiddleware)
}
`,
	})
	ctx.FeatureProfile.HasServer = false
	findings := ruletest.RunDetector(t, boilerplate.NewB014Detector(ctx))
	ruletest.AssertRule(t, findings, "B014", 0)
}

// --- B004: Command constructor boilerplate ---

func TestB004_NoFindingWithoutCommands(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB004Detector(ctx))
	ruletest.AssertRule(t, findings, "B004", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB004Detector(ctx))
	ruletest.AssertRule(t, findings, "B004", 1)
}

// --- B005: Fold switch boilerplate ---

func TestB005_NoFindingWithoutFolds(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB005Detector(ctx))
	ruletest.AssertRule(t, findings, "B005", 0)
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
	findings := ruletest.RunDetector(t, boilerplate.NewB005Detector(ctx))
	ruletest.AssertRule(t, findings, "B005", 1)
}

// B005 must NOT fire when the fold function is already wrapped in
// decider.StrictApply — the suggestion is already implemented. The detector
// matches the fold by the trailing identifier of its name, so both bare
// (foldCounter) and method-qualified ((d).foldCounter) names are suppressed.
// Regression for the browser-history latent gap (B005 fires after adopting
// StrictApply because the detector has no suppression logic).

func TestB005_NoFindingWhenFoldIsWrappedInStrictApply(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldCounter(state int, event event.Event) (int, error) {
	switch event.Type() {
	case "incremented":
		return state + 1, nil
	default:
		return state, nil
	}
}
`,
		"decider.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider"

var CounterDecider = decider.Decider[int]{
	Initial: 0,
	Apply:   decider.StrictApply(foldCounter, []event.Type{"incremented"}),
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB005Detector(ctx))
	ruletest.AssertRule(t, findings, "B005", 0)
}

// B005 must still fire for a fold that is NOT wrapped in StrictApply even when
// another fold in the same package IS wrapped. Guards against over-broad
// suppression.

func TestB005_FiresForUnwrappedFoldWhenAnotherFoldIsWrapped(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldCounter(state int, event event.Event) (int, error) {
	switch event.Type() {
	case "incremented":
		return state + 1, nil
	default:
		return state, nil
	}
}

func foldUnwrapped(state int, event event.Event) (int, error) {
	switch event.Type() {
	case "other":
		return state + 1, nil
	default:
		return state, nil
	}
}
`,
		"decider.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider"

var CounterDecider = decider.Decider[int]{
	Initial: 0,
	Apply:   decider.StrictApply(foldCounter, []event.Type{"incremented"}),
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB005Detector(ctx))
	ruletest.AssertRule(t, findings, "B005", 1)
}

// --- B015: Missing test utilities ---

func TestB015_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB015Detector(ctx))
	ruletest.AssertRule(t, findings, "B015", 0)
}

func TestB015_DetectsMissingTestUtils(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main

import "testing"

func TestSomething(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB015Detector(ctx))
	ruletest.AssertRule(t, findings, "B015", 1)
}
