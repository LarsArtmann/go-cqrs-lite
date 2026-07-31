package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
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
