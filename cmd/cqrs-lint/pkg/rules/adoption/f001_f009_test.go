package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF001_DeleteWithoutTombstone(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func DeleteUser(id string) {}

func _() {
	event.New("user.created", sid, st, v, p)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF001Detector(ctx))
	ruletest.AssertRule(t, findings, "F001", 1)
}

func TestF001_NoFindingWithMarkTombstone(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func DeleteUser(id string) {}

func _() {
	event.New("user.created", sid, st, v, p)
	event.MarkTombstone(evt)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF001Detector(ctx))
	ruletest.AssertRule(t, findings, "F001", 0)
}

func TestF002_NoCatalogWithThreeEventTypes(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func _() {
	event.New("user.created", sid, st, v, p)
	event.New("user.updated", sid, st, v, p)
	event.New("user.deleted", sid, st, v, p)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF002Detector(ctx))
	ruletest.AssertRule(t, findings, "F002", 1)
}

func TestF002_NoFindingWithCatalogBuilder(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func _() {
	event.New("user.created", sid, st, v, p)
	event.New("user.updated", sid, st, v, p)
	event.New("user.deleted", sid, st, v, p)
	catalog.NewBuilder()
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF002Detector(ctx))
	ruletest.AssertRule(t, findings, "F002", 0)
}

func TestF003_ServerWithoutOTel(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF003Detector(ctx))
	ruletest.AssertRule(t, findings, "F003", 1)
}

func TestF003_NoFindingWithoutServer(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF003Detector(ctx))
	ruletest.AssertRule(t, findings, "F003", 0)
}

func TestF004_ServerWithoutPrometheus(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF004Detector(ctx))
	ruletest.AssertRule(t, findings, "F004", 1)
}

func TestF005_WithSchemaVersionNoUpcaster(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.New("user.created", sid, st, v, p, event.WithSchemaVersion(2))
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF005Detector(ctx))
	ruletest.AssertRule(t, findings, "F005", 1)
}

func TestF005_NoFindingWithUpcaster(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	event.New("user.created", sid, st, v, p, event.WithSchemaVersion(2))
	schema.NewUpcaster("UserCreated", 1, fn)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF005Detector(ctx))
	ruletest.AssertRule(t, findings, "F005", 0)
}

func TestF006_PIIFieldWithoutEncryption(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreatedEvent struct {
	Email string
	Name  string
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF006Detector(ctx))
	ruletest.AssertRule(t, findings, "F006", 1)
}

func TestF006_NoFindingWithoutPII(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreatedEvent struct {
	Name string
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF006Detector(ctx))
	ruletest.AssertRule(t, findings, "F006", 0)
}

func TestF007_CommandDispatchWithoutIdempotency(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	disp.Dispatch(ctx, cmd)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF007Detector(ctx))
	ruletest.AssertRule(t, findings, "F007", 1)
}

func TestF008_JSONCodecWithManyEvents(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func _() {
	event.New("user.created", sid, st, v, p)
	event.New("user.updated", sid, st, v, p)
	event.New("user.deleted", sid, st, v, p)
	event.New("order.created", sid, st, v, p)
	event.New("order.shipped", sid, st, v, p)
}

var jsonCodec = codec.JSONCodec{}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF008Detector(ctx))
	ruletest.AssertRule(t, findings, "F008", 1)
}

func TestF008_NoFindingWithCBOR(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func _() {
	event.New("user.created", sid, st, v, p)
	event.New("user.updated", sid, st, v, p)
	event.New("user.deleted", sid, st, v, p)
	event.New("order.created", sid, st, v, p)
	event.New("order.shipped", sid, st, v, p)
}

var cborCodec = codec.CBORCodec{}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF008Detector(ctx))
	ruletest.AssertRule(t, findings, "F008", 0)
}

func TestF009_TimeAfterFuncWithoutScheduling(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "time"

func _() {
	time.AfterFunc(30e9, fn)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF009Detector(ctx))
	ruletest.AssertRule(t, findings, "F009", 1)
}
