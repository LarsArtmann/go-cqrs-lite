package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF010_TraversalPatternWithoutGraph(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func FindAncestors(id string) []string {
	return nil
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF010Detector(ctx))
	ruletest.AssertRule(t, findings, "F010", 1)
}

func TestF010_NoFindingWithoutTraversal(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func GetUser(id string) *User {
	return nil
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF010Detector(ctx))
	ruletest.AssertRule(t, findings, "F010", 0)
}

func TestF011_MultiExecWithoutRelationalProjection(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "database/sql"

func _() {
	event.New("msg.created", sid, st, v, p)
	db.Exec("INSERT INTO messages VALUES(1)")
	db.Exec("INSERT INTO channels VALUES(2)")
	db.Exec("INSERT INTO users VALUES(3)")
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF011Detector(ctx))
	ruletest.AssertRule(t, findings, "F011", 1)
}

func TestF012_SubscribeAllWithDispatchWithoutDeriver(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	bus.SubscribeAll(handler)
	disp.Dispatch(ctx, cmd)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF012Detector(ctx))
	ruletest.AssertRule(t, findings, "F012", 1)
}

func TestF013_ManualHTTPHandlerWithoutTransport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func main() {
	http.HandleFunc("/api", handler)
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF013Detector(ctx))
	ruletest.AssertRule(t, findings, "F013", 1)
}

func TestF013_NoFindingWhenTransportPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func main() {
	http.HandleFunc("/api", handler)
	http.ListenAndServe(":8080", nil)
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	ctx.FeatureProfile.HasTransport = true

	findings := ruletest.RunDetector(t, adoption.NewF013Detector(ctx))
	ruletest.AssertRule(t, findings, "F013", 0)
}

func TestF014_TypedStoreWithoutCache(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	kv.NewTypedStore[UserView, UserID](backend)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF014Detector(ctx))
	ruletest.AssertRule(t, findings, "F014", 1)
}

func TestF014_NoFindingWithCache(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	kv.NewTypedStore[UserView, UserID](backend)
	kv.NewCache[UserView, UserID](store)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF014Detector(ctx))
	ruletest.AssertRule(t, findings, "F014", 0)
}

func TestF015_ManyQueriesWithoutMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	query.RegisterTyped(d, t1, h1)
	query.RegisterTyped(d, t2, h2)
	query.RegisterTyped(d, t3, h3)
	query.RegisterTyped(d, t4, h4)
	query.RegisterTyped(d, t5, h5)
}
`,
	})
	ctx.FeatureProfile.HasServer = true

	findings := ruletest.RunDetector(t, adoption.NewF015Detector(ctx))
	ruletest.AssertRule(t, findings, "F015", 1)
}

func TestF016_ManyAggregatesWithoutListing(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func _() {
	event.New("user.created", sid, st, v, p)
	event.New("order.placed", sid, st, v, p)
	event.New("payment.processed", sid, st, v, p)
	event.New("invoice.sent", sid, st, v, p)
	event.New("shipment.dispatched", sid, st, v, p)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF016Detector(ctx))
	ruletest.AssertRule(t, findings, "F016", 1)
}

func TestF017_BusSubscriptionWithoutDedup(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	bus.Subscribe("user.created", handler)
}
`,
	})
	ctx.FeatureProfile.HasAsyncBus = true

	findings := ruletest.RunDetector(t, adoption.NewF017Detector(ctx))
	ruletest.AssertRule(t, findings, "F017", 1)
}
