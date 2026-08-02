package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// --- D007: Inconsistent event creation API ---

func TestD007_DetectsMixedEventCreationAPI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

func makeA() {
	_ = event.New("a", sid, "A", 1, payload{})
}
`,
		"b.go": `package main

func makeB() {
	_ = event.NewEvent("b", sid, "B", 1, payload{})
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD007Detector(ctx))
	ruletest.AssertRule(t, findings, "D007", 1)
}

func TestD007_NoFindingForConsistentAPI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

func makeA() {
	_ = event.New("a", sid, "A", 1, payload{})
}

func makeB() {
	_ = event.New("b", sid, "B", 1, payload{})
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD007Detector(ctx))
	ruletest.AssertRule(t, findings, "D007", 0)
}

func TestD007_NoFindingWhenNoEventCreation(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

func doSomething() {}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD007Detector(ctx))
	ruletest.AssertRule(t, findings, "D007", 0)
}

func TestD007_MultipleNewEventCallsEmitPerSiteFindings(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

func makeA() {
	_ = event.New("a", sid, "A", 1, payload{})
}
`,
		"b.go": `package main

func makeB() {
	_ = event.NewEvent("b", sid, "B", 1, payload{})
	_ = event.NewEvent("c", sid, "C", 1, payload{})
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD007Detector(ctx))
	ruletest.AssertRule(t, findings, "D007", 2)
}

// --- D008: Inconsistent codec usage ---

func TestD008_DetectsMixedCodecUsage(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func fold() {
	_, _ = event.DecodePayloadAuto[Payload](evt)
}
`,
		"proj.go": `package main

func project() {
	_, _ = event.DecodePayload[Payload](evt, codec)
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD008Detector(ctx))
	ruletest.AssertRule(t, findings, "D008", 1)
}

func TestD008_NoFindingForConsistentCodec(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

func fold() {
	_, _ = event.DecodePayloadAuto[Payload](evt)
}

func project() {
	_, _ = event.DecodePayloadAuto[Other](evt)
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD008Detector(ctx))
	ruletest.AssertRule(t, findings, "D008", 0)
}

// --- D009: Inconsistent Close detection pattern ---

func TestD009_DetectsMixedClosePattern(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import "io"

func closeA(c io.Closer) error {
	return c.Close()
}
`,
		"b.go": `package main

func closeB(c interface{ Close() error }) error {
	return c.Close()
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD009Detector(ctx))
	ruletest.AssertRule(t, findings, "D009", 1)
}

func TestD009_NoFindingForConsistentClosePattern(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import "io"

func closeA(c io.Closer) error {
	return c.Close()
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD009Detector(ctx))
	ruletest.AssertRule(t, findings, "D009", 0)
}

// --- D010: Generic error code "internal" ---

func TestD010_DetectsGenericErrorCodeInternal(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func save() error {
	return errorfamily.WrapTransient(err, "internal", "failed to save")
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD010Detector(ctx))
	ruletest.AssertRule(t, findings, "D010", 1)
}

func TestD010_DetectsMultipleOccurrences(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func save() error {
	_ = errorfamily.WrapTransient(err1, "internal", "msg1")
	return errorfamily.NewRejection("internal", "msg2")
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD010Detector(ctx))
	ruletest.AssertRule(t, findings, "D010", 2)
}

func TestD010_NoFindingForDescriptiveCode(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func save() error {
	return errorfamily.WrapTransient(err, "user.save.transient", "failed to save")
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD010Detector(ctx))
	ruletest.AssertRule(t, findings, "D010", 0)
}

// --- D013: Schema version not stamped on events ---

func TestD013_DetectsMissingSchemaVersion(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func createEvent() {
	_ = event.New("user.created", sid, "User", 1, UserCreated{})
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD013Detector(ctx))
	ruletest.AssertRule(t, findings, "D013", 1)
}

func TestD013_NoFindingWhenSchemaVersionUsed(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func createEvent() {
	_ = event.New("user.created", sid, "User", 1, UserCreated{},
		event.WithSchemaVersion(1))
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD013Detector(ctx))
	ruletest.AssertRule(t, findings, "D013", 0)
}

func TestD013_NoFindingWhenNoEvents(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"svc.go": `package main

func doSomething() {}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD013Detector(ctx))
	ruletest.AssertRule(t, findings, "D013", 0)
}
