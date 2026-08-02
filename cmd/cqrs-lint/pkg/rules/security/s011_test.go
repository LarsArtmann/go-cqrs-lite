package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestS011_PIIWithoutEncryption(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type UserCreated struct {
	Email    string
	Password string
	Name     string
}
`,
	})

	findings := ruletest.RunDetector(t, security.NewS011Detector(ctx))
	ruletest.AssertRule(t, findings, "S011", 2) // Email + Password
}

func TestS011_PIIButWithEncryption(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Email    string
	Password string
}
`,
		"bus.go": `package main

func setupBus() {
	bus.UsePublish(encryption.EncryptMiddleware(enc))
}
`,
	})

	findings := ruletest.RunDetector(t, security.NewS011Detector(ctx))
	ruletest.AssertRule(t, findings, "S011", 0)
}

func TestS011_NonPayloadStructNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type UserView struct {
	Email string
}
`,
	})

	findings := ruletest.RunDetector(t, security.NewS011Detector(ctx))
	ruletest.AssertRule(t, findings, "S011", 0)
}
