package security_test

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

// TestS006_PerModuleProfileDowngradesNonServerModule verifies that S006
// evaluates HasServer per-module for severity modulation. A financial struct
// in a library module (no server) should get SeverityInfo even when the
// primary profile says HasServer=true (from an example sub-module).
func TestS006_PerModuleProfileDowngradesNonServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/payroll.go": `package main

type EmployeePayroll struct {
	Salary float64 ` + "`json:\"salary\"`" + `
}
`,
	})

	// Multi-module: lib has no server, app has server.
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	// Primary profile = server (from app module).
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)

	// The struct is in /repo/lib (no server). Severity should be downgraded
	// to Info, not the full Warning/Error it would get in a server module.
	if findings[0].Severity != finding.SeverityInfo {
		t.Errorf(
			"non-server module financial data should be INFO, got %s",
			findings[0].Severity,
		)
	}
}

// TestS006_PerModuleProfileFullSeverityForServerModule verifies that the same
// struct in a server module gets full severity.
func TestS006_PerModuleProfileFullSeverityForServerModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/app/payroll.go": `package main

type EmployeePayroll struct {
	Salary float64 ` + "`json:\"salary\"`" + `
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib": {HasServer: false},
		"/repo/app": {HasServer: true},
	}
	// Primary profile = no server (from lib module).
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: false}

	findings := ruletest.RunDetector(t, security.NewS006Detector(ctx))
	ruletest.AssertRule(t, findings, "S006", 1)

	// The struct is in /repo/app (server). Salary is tierMedium → default
	// SeverityWarning when HasServer is true. Should NOT be downgraded.
	if findings[0].Severity == finding.SeverityInfo {
		t.Error(
			"server module financial data should NOT be downgraded to INFO",
		)
	}
}
