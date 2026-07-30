package security_test

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

// --- S001: Hardcoded secrets ---

func TestS001_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, security.NewS001Detector(ctx))
	assertRule(t, findings, "S001", 0)
}

// --- S002: Missing encryption for sensitive payloads ---

func TestS002_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, security.NewS002Detector(ctx))
	assertRule(t, findings, "S002", 0)
}

// --- S003: Missing event signing ---

func TestS003_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, security.NewS003Detector(ctx))
	assertRule(t, findings, "S003", 0)
}

// --- S001: Positive test — hardcoded API key ---

func TestS001_DetectsHardcodedKey(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	apiKey := "supersecretvalue123"
	_ = apiKey
}
`,
	})
	findings := runDetector(t, security.NewS001Detector(ctx))
	assertRule(t, findings, "S001", 1)
}

// --- S002: Positive test — PII event without encryption ---

func TestS002_DetectsPIIWithoutEncryption(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserEmailChanged struct {
	Email string
}
`,
	})
	findings := runDetector(t, security.NewS002Detector(ctx))
	assertRule(t, findings, "S002", 1)
}

// --- S003: Positive test — event store without signing ---

func TestS003_DetectsMissingSigning(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}

func saveEvents(store event.Store, ref event.StreamRef, events []event.Event) error {
	return store.Save(nil, ref, events)
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS003Detector(ctx))
	assertRule(t, findings, "S003", 1)
}

// --- FeatureProfile suppression guards ---

// TestS002_DowngradedForLocalCLI proves the HasServer toggle changes severity:
// a server project gets Error (production PII risk) while a local-only project
// is downgraded to Info (no network exposure). This guards the FeatureProfile
// rewiring of S002 against silent regression.
func TestS002_DowngradedForLocalCLI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserEmailChanged struct {
	Email string
}
`,
	})

	ctx.FeatureProfile.HasServer = true
	serverFindings := runDetector(t, security.NewS002Detector(ctx))
	assertRule(t, serverFindings, "S002", 1)
	if serverFindings[0].Severity != finding.SeverityError {
		t.Fatalf("server project PII should be ERROR, got %s", serverFindings[0].Severity)
	}

	ctx.FeatureProfile.HasServer = false
	localFindings := runDetector(t, security.NewS002Detector(ctx))
	assertRule(t, localFindings, "S002", 1)
	if localFindings[0].Severity != finding.SeverityInfo {
		t.Errorf("local-only PII should be downgraded to INFO, got %s", localFindings[0].Severity)
	}
}

// TestS003_SuppressedForNoServer proves signing is fully suppressed when there
// is no server: a fixture that would normally fire S003 yields zero findings.
func TestS003_SuppressedForNoServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}

func saveEvents(store event.Store, ref event.StreamRef, events []event.Event) error {
	return store.Save(nil, ref, events)
}
`,
	})

	ctx.FeatureProfile.HasServer = false
	findings := runDetector(t, security.NewS003Detector(ctx))
	assertRule(t, findings, "S003", 0)
}

// --- S007: In-memory session/token store ---

func TestS007_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 0)
}

func TestS007_DetectsInMemorySessionStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 1)
}

func TestS007_SuppressedWithoutServer(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cli.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = false
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 0)
}

func TestS007_DetectsMemoryTokenStoreComposite(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"auth.go": `package main

type MemoryTokenStore struct{}

func setup() {
	s := MemoryTokenStore{}
	_ = s
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 1)
}

func TestS007_IgnoresCQRSEventMemoryStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

func setup() {
	s := memory.NewStore()
	_ = s
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 0)
}

func TestS007_IgnoresInMemoryTokenBucket(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"ratelimit.go": `package main

func setup() {
	b := NewInMemoryTokenBucket()
	_ = b
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 0)
}

func TestS007_IgnoresTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"session_test.go": `package main

func setup() {
	store := NewInMemorySessionStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.HasServer = true
	findings := runDetector(t, security.NewS007Detector(ctx))
	assertRule(t, findings, "S007", 0)
}
