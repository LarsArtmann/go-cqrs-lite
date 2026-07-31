package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

func TestS010_DetectsBusEncryptionWithoutStoreWrap(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.Use(encryption.EncryptMiddleware(enc))
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS010Detector(ctx))
	ruletest.AssertRule(t, findings, "S010", 1)
}

func TestS010_NoFindingWhenStoreWrapped(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.Use(encryption.EncryptMiddleware(enc))
	store = signing.NewSignedStore(store, signer)
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS010Detector(ctx))
	ruletest.AssertRule(t, findings, "S010", 0)
}

func TestS010_NoFindingWithoutEncryption(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store = NewStore()
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS010Detector(ctx))
	ruletest.AssertRule(t, findings, "S010", 0)
}
