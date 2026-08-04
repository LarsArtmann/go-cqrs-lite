package security_test

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// TestS002_PerModuleDowngradesNonServerModule verifies that S002 evaluates
// HasServer per-module for severity modulation. PII payloads in a library
// module (no server) should get SeverityInfo even when the primary profile
// says HasServer=true (from an example sub-module).
func TestS002_PerModuleDowngradesNonServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go": `package main

type UserEmailChanged struct {
	Email string
}

func emit() {
	_ = event.New("user.email_changed", id, "User", 1, UserEmailChanged{Email: "x"})
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	findings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, findings, "S002", 1)

	if findings[0].Severity != finding.SeverityInfo {
		t.Errorf("non-server module PII should be INFO, got %s", findings[0].Severity)
	}
}

// TestS002_PerModuleFullSeverityForServerModule verifies that the same PII
// payload in a server module gets full Error severity.
func TestS002_PerModuleFullSeverityForServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/app/events.go": `package main

type UserEmailChanged struct {
	Email string
}

func emit() {
	_ = event.New("user.email_changed", id, "User", 1, UserEmailChanged{Email: "x"})
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib": {HasServer: false},
		"/repo/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: false}

	findings := ruletest.RunDetector(t, security.NewS002Detector(ctx))
	ruletest.AssertRule(t, findings, "S002", 1)

	if findings[0].Severity != finding.SeverityError {
		t.Errorf("server module PII should be ERROR, got %s", findings[0].Severity)
	}
}

// TestS003_PerModuleSkipsNonServerModule verifies that S003 skips event
// stores in non-server modules. A library defining test stores should not
// trigger signing coaching meant for server deployments.
func TestS003_PerModuleSkipsNonServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/store.go": `package main

func saveEvents(store Store, ref string, events []any) error {
	return store.Save(nil, ref, events)
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	findings := ruletest.RunDetector(t, security.NewS003Detector(ctx))
	ruletest.AssertRule(t, findings, "S003", 0)
}

// TestS003_PerModuleFiresForServerModule verifies that S003 still fires for
// event stores in server modules.
func TestS003_PerModuleFiresForServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/app/store.go": `package main

func saveEvents(store Store, ref string, events []any) error {
	return store.Save(nil, ref, events)
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib": {HasServer: false},
		"/repo/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: false}

	findings := ruletest.RunDetector(t, security.NewS003Detector(ctx))
	ruletest.AssertRule(t, findings, "S003", 1)
}
