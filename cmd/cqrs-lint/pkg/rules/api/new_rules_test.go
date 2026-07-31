package api_test

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
)

// --- A009: Missing stack preset ---

func TestA009_FiresWithoutStackPreset(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA009Detector(ctx))
	// A009 fires when no package imports stack/ — the test context has no imports
	ruletest.AssertRule(t, findings, "A009", 1)
}

// A009 must NOT fire when the project uses the storage/ facade directly — this
// signals an intentional shared-DB architecture (one *sql.DB across CQRS +
// relational reads) that stack/ presets don't support. Covers item f-14/16.
func TestA009_NoFindingForStorageFacadeArchitecture(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	// Simulate a storage/ import by populating Packages.
	ctx.Packages = []*packages.Package{{
		PkgPath: "example.com/myapp",
		Imports: map[string]*packages.Package{
			"github.com/larsartmann/go-cqrs-lite/storage": {
				PkgPath: "github.com/larsartmann/go-cqrs-lite/storage",
			},
		},
	}}
	findings := ruletest.RunDetector(t, api.NewA009Detector(ctx))
	ruletest.AssertRule(t, findings, "A009", 0)
}

// --- A010: Custom error types ---

func TestA010_DetectsCustomErrorInterface(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"errors.go": `package main

type DomainError interface {
	Error() string
	Code() int
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA010Detector(ctx))
	ruletest.AssertRule(t, findings, "A010", 1)
}

func TestA010_NoFindingForNonErrorInterface(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type Handler interface {
	Handle() error
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA010Detector(ctx))
	ruletest.AssertRule(t, findings, "A010", 0)
}

// --- A011: Inconsistent JSON key casing in event payloads ---

func TestA011_DetectsMixedJSONCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA011Detector(ctx))
	ruletest.AssertRule(t, findings, "A011", 1)
}

func TestA011_NoFindingForConsistentCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"last_name\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA011Detector(ctx))
	ruletest.AssertRule(t, findings, "A011", 0)
}

// --- A012: Missing tombstone handling ---

func TestA012_NoFindingWithoutFolds(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA012Detector(ctx))
	ruletest.AssertRule(t, findings, "A012", 0)
}

// --- A013: Pointer vs value BasicCommand ---

func TestA013_DetectsPointerBasicCommand(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CreateCmd struct {
	*BasicCommand
	Name string
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA013Detector(ctx))
	ruletest.AssertRule(t, findings, "A013", 1)
}

// --- A014: Deprecated API usage ---

func TestA014_DetectsNewEventCall(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func createEvent() {
	event.NewEvent("user.created", "id", "User", 1, nil)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA014Detector(ctx))
	ruletest.AssertRule(t, findings, "A014", 1)
}

func TestA014_NoFindingForEventNew(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func createEvent() {
	event.New("user.created", "id", "User", 1, nil)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA014Detector(ctx))
	ruletest.AssertRule(t, findings, "A014", 0)
}

// --- A015: Global mutable state ---

func TestA015_DetectsGlobalCache(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

var globalCache = make(map[string]string)

func update(key, val string) {
	globalCache[key] = val
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := ruletest.RunDetector(t, api.NewA015Detector(ctx))
	ruletest.AssertRule(t, findings, "A015", 1)
}

func TestA015_NoFindingForErrPrefix(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

var ErrCacheMiss = errors.New("cache miss")
`,
	})
	findings := ruletest.RunDetector(t, api.NewA015Detector(ctx))
	ruletest.AssertRule(t, findings, "A015", 0)
}

func TestA015_NoFindingForNonMutableVar(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

var defaultTimeout = 30 * time.Second
`,
	})
	findings := ruletest.RunDetector(t, api.NewA015Detector(ctx))
	ruletest.AssertRule(t, findings, "A015", 0)
}

// --- A015: Read-only global registry (initialized at load, never written) ---

func TestA015_NoFindingForReadOnlyGlobal(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

var providerRegistry = map[string]string{
		"wise": "Wise API",
		"demo": "Demo Bank",
}

func lookup(key string) string {
	return providerRegistry[key]
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA015Detector(ctx))
	ruletest.AssertRule(t, findings, "A015", 0)
}

// --- A017: Missing snapshot strategy ---

func TestA017_DetectsRepoWithoutSnapshot(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA017Detector(ctx))
	ruletest.AssertRule(t, findings, "A017", 1)
}

func TestA017_NoFindingForRepoWithSnapshot(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d,
		decider.WithSnapshotStore(snap),
		decider.WithSnapshotStrategy(strategy),
	)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA017Detector(ctx))
	ruletest.AssertRule(t, findings, "A017", 0)
}

// A017: WithSnapshotStore without WithSnapshotStrategy should fire (warning).
func TestA017_SnapshotStoreWithoutStrategy(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d, decider.WithSnapshotStore(snap))
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA017Detector(ctx))
	ruletest.AssertRule(t, findings, "A017", 1)
}

// A017: WithStateCache alone is fine (no snapshot needed).
func TestA017_NoFindingWithStateCacheOnly(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d, decider.WithStateCache(cache))
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA017Detector(ctx))
	ruletest.AssertRule(t, findings, "A017", 0)
}

// --- A016, A018, A019: Package-import-based rules ---

func TestA016_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

func TestA018_FiresOnNoEventSourcing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA018Detector(ctx))
	// A018 fires when there are no Save/Publish calls and no folds
	// since the test context has none, it reports the anti-pattern
	ruletest.AssertRule(t, findings, "A018", 1)
}

func TestA019_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA019Detector(ctx))
	ruletest.AssertRule(t, findings, "A019", 0)
}

// --- Positive tests for previously untested rules ---

func TestA001_DetectsManualCommandInterface(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CreateUser struct {
	Name string
}

func (c *CreateUser) ID() string { return "" }
func (c *CreateUser) Type() string { return "createUser" }
func (c *CreateUser) StreamID() string { return "" }
`,
	})
	findings := ruletest.RunDetector(t, api.NewA001Detector(ctx))
	ruletest.AssertRule(t, findings, "A001", 1)
}

func TestA003_DetectsExplicitCodec(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decode.go": `package main

func handle(evt Event) {
	_ = event.DecodePayload(evt, codec.JSONCodec{})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA003Detector(ctx))
	ruletest.AssertRule(t, findings, "A003", 1)
}

func TestA004_DetectsTypeAssertionInHandler(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"register.go": `package main

func register(d Dispatcher) {
	d.Register("cmd", func(cmd any) error {
		c := cmd.(*CreateUser)
		_ = c
		return nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA004Detector(ctx))
	ruletest.AssertRule(t, findings, "A004", 1)
}

func TestA005_DetectsSubscribeAllWithoutProjectionHost(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func setup(bus EventBus, store Store) {
	bus.SubscribeAll(func(evt Event) {
		store.Save(evt.ID, evt)
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 1)
}

// A005 must NOT flag SubscribeAll callbacks that only broadcast/notify —
// SSE fan-out and stats notifiers are fire-and-forget, not projections.
// Regression test for the DiscordSync false positives (server.go stats
// notifier + sse.go broadcaster).
func TestA005_NoFindingForBroadcastFanOut(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"sse.go": `package main

func setupSSE(bus EventBus, broker *Broker) {
	bus.SubscribeAll(func(evt Event) {
		broker.Broadcast(evt)
	})
}

func setupStats(bus EventBus, notifier *Notifier) {
	bus.SubscribeAll(func(evt Event) {
		notifier.Notify()
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 0)
}

// A005 must NOT flag SubscribeAll callbacks that use the widened broadcast
// signal set (Publish, Emit, Forward, Dispatch, WriteTo, Flush). These are
// fire-and-forget distribution verbs, not projection persistence writes. A
// callback that republishes an event or forwards to another sink isn't a
// projection and shouldn't be told to use projectionhost.
func TestA005_NoFindingForWidenedBroadcastSignals(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fanout.go": `package main

func republish(bus EventBus, out EventBus) {
	bus.SubscribeAll(func(evt Event) {
		out.Publish(evt)
	})
}

func emitStats(bus EventBus, emitter *Emitter) {
	bus.SubscribeAll(func(evt Event) {
		emitter.Emit(evt)
	})
}

func forwardAll(bus EventBus, fwd *Forwarder) {
	bus.SubscribeAll(func(evt Event) {
		fwd.Forward(evt)
	})
}

func dispatchDerived(bus EventBus, cmdBus CommandBus) {
	bus.SubscribeAll(func(evt Event) {
		cmdBus.Dispatch(evt)
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 0)
}

// A005 must STILL flag a callback that both broadcasts AND persists — the
// persistence write is the defining trait of a projection. Guards against the
// widened broadcast list swallowing real manual projections.
func TestA005_FiresWhenCallbackBroadcastsAndPersists(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func setup(bus EventBus, store Store, out EventBus) {
	bus.SubscribeAll(func(evt Event) {
		store.Save(evt.ID, evt)
		out.Publish(evt)
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 1)
}

func TestA007_DetectsDualModel(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"oo.go": `package main

type Order struct {
	uncommittedEvents []Event
}

func (o *Order) Apply(evt Event) {
	o.uncommittedEvents = append(o.uncommittedEvents, evt)
}
`,
		"functional.go": `package main

var d = decider.Decider[OrderState]{
	Initial: OrderState{},
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA007Detector(ctx))
	ruletest.AssertRule(t, findings, "A007", 1)
}

// --- A012: Positive test — fold with switch but no tombstone handling ---

func TestA012_DetectsFoldWithoutTombstoneCheck(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "created":
		s.Count++
	}
	return s, nil
}

func emitDelete() {
	event.New("user.deleted", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA012Detector(ctx))
	ruletest.AssertRule(t, findings, "A012", 1)
}

// --- A016: Positive test — dispatcher with Use but no idempotency ---

func TestA016_DetectsMissingIdempotency(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(loggingMiddleware)
	d.Dispatch(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 1)
}

func TestA016_NoFindingWithIdempotency(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(middleware.CommandIdempotency(store, ttl, nil))
	d.Dispatch(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

// A016: idempotency.NewMemoryStore direct usage also satisfies idempotency check.
func TestA016_NoFindingWithDirectIdempotencyStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	store := idempotency.NewMemoryStore(5 * time.Minute)
	d.Use(middleware.CommandIdempotency(store, ttl, nil))
	d.Dispatch(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

// A016: QueryIdempotency also counts as idempotency.
func TestA016_NoFindingWithQueryIdempotency(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(middleware.QueryIdempotency(store, ttl, keyFn))
	d.Dispatch(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

// --- A016: Dispatcher that never dispatches is NOT flagged ---

func TestA016_NoFindingForReadOnlyDispatcher(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(loggingMiddleware)
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

// --- A019: Positive test — vendored cqrs import ---

func TestA019_DetectsVendoredCqrs(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app",
			Imports: map[string]*packages.Package{
				"vendor/github.com/larsartmann/go-cqrs-lite/event/v4": {
					PkgPath: "vendor/github.com/larsartmann/go-cqrs-lite/event/v4",
				},
			},
		},
	}
	findings := ruletest.RunDetector(t, api.NewA019Detector(ctx))
	ruletest.AssertRule(t, findings, "A019", 1)
}

// --- FeatureProfile suppression guards ---
// Each test reuses a fixture that WOULD fire, then proves the profile gate
// suppresses it. Together with the positive tests above, they pin the toggle.

// TestA015_SuppressedForNoServer: the globalCache fixture fires when
// HasServer=true (see TestA015_DetectsGlobalCache); a local-only project is
// suppressed because race conditions need concurrent access.
func TestA015_SuppressedForNoServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": `package main

var globalCache = make(map[string]string)

func update(key, val string) {
	globalCache[key] = val
}
`,
	})
	ctx.FeatureProfile.HasServer = false
	findings := ruletest.RunDetector(t, api.NewA015Detector(ctx))
	ruletest.AssertRule(t, findings, "A015", 0)
}

// TestA016_SuppressedForReadOnlyFlow: the dispatcher+Dispatch fixture fires
// when CommandFlow=Commands (see TestA016_DetectsMissingIdempotency); a
// read-only flow is suppressed because no commands are executed.
func TestA016_SuppressedForReadOnlyFlow(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(loggingMiddleware)
	d.Dispatch(ctx, cmd)
}
`,
	})
	ctx.FeatureProfile.CommandFlow = analyzer.CommandFlowReadOnly
	findings := ruletest.RunDetector(t, api.NewA016Detector(ctx))
	ruletest.AssertRule(t, findings, "A016", 0)
}

// TestA012_SuppressedForNoSoftDelete: the fold+deleted-event fixture fires
// when HasSoftDelete=true (see TestA012_DetectsFoldWithoutTombstoneCheck); a
// domain without soft-delete is suppressed.
func TestA012_SuppressedForNoSoftDelete(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	switch evt.Type() {
	case "created":
		s.Count++
	}
	return s, nil
}

func emitDelete() {
	event.New("user.deleted", id, "User", 1, payload)
}
`,
	})
	ctx.FeatureProfile.HasSoftDelete = false
	findings := ruletest.RunDetector(t, api.NewA012Detector(ctx))
	ruletest.AssertRule(t, findings, "A012", 0)
}

// TestA009_AdaptiveSuggestion proves the suggestion text adapts to the detected
// Store backend, pointing the user at the matching stack/ preset.
func TestA009_AdaptiveSuggestion(t *testing.T) {
	cases := []struct {
		name      string
		store     analyzer.StoreKind
		wantInSug string
	}{
		{"sqlite", analyzer.StoreSQLite, "stack/sqlite"},
		{"postgres", analyzer.StorePostgres, "stack/postgres"},
		{"pebble", analyzer.StorePebble, "stack/pebble"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				"main.go": `package main`,
			})
			ctx.FeatureProfile.Store = tc.store
			findings := ruletest.RunDetector(t, api.NewA009Detector(ctx))
			ruletest.AssertRule(t, findings, "A009", 1)
			if !strings.Contains(findings[0].Suggestion, tc.wantInSug) {
				t.Errorf("Store=%s: suggestion should mention %q, got %q",
					tc.store, tc.wantInSug, findings[0].Suggestion)
			}
		})
	}
}
