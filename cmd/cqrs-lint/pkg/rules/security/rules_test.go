package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/security"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestS001_DetectsHardcodedSecret(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	apiKey := "super-secret-key-1234567890"
	_ = apiKey
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 1)
}

func TestS001_NoFindingForShortString(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	token := "abc"
	_ = token
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 0)
}

// TestS001_DetectsPackageLevelVar pins the var/const coverage path: the
// AssignStmt-only walk silently skipped the most common secret placement.
func TestS001_DetectsPackageLevelVar(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

var apiKey = "super-secret-key-1234567890"

const privatePassword = "another-secret-1234567890"
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 2)
}

// TestS001_DetectsStructLiteralField pins the composite-literal coverage
// path (Config{Password: "…"}).
func TestS001_DetectsStructLiteralField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	Password string
	Host     string
}

func newConfig() Config {
	return Config{Password: "super-secret-key-12345", Host: "localhost"}
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 1)
}

// TestS001_DetectsMapKeyAssignment pins the IndexExpr coverage path
// (m["api_key"] = "…").
func TestS001_DetectsMapKeyAssignment(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func seed() map[string]string {
	m := map[string]string{}
	m["api_key"] = "super-secret-key-1234567890"
	return m
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 1)
}

// TestS001_NoFindingForLongNonSecretName pins the FP gate: a long string on
// a field that carries no secret keyword must stay silent.
func TestS001_NoFindingForLongNonSecretName(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func banner() {
	welcomeMessage := "welcome to the platform, enjoy your stay"
	_ = welcomeMessage
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 0)
}
