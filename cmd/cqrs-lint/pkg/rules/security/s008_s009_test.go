package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

func TestS008_DetectsSignWithoutVerify(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.UsePublish(signing.SignMiddleware(signer))
}
`,
	})
	findings := runDetector(t, security.NewS008Detector(ctx))
	assertRule(t, findings, "S008", 1)
}

func TestS008_DetectsVerifyWithoutSign(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.Use(signing.VerifyMiddleware(verifier))
}
`,
	})
	findings := runDetector(t, security.NewS008Detector(ctx))
	assertRule(t, findings, "S008", 1)
}

func TestS008_NoFindingWhenBothPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.UsePublish(signing.SignMiddleware(signer))
	bus.Use(signing.VerifyMiddleware(verifier))
}
`,
	})
	findings := runDetector(t, security.NewS008Detector(ctx))
	assertRule(t, findings, "S008", 0)
}

func TestS008_NoFindingWhenNeitherPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})
	findings := runDetector(t, security.NewS008Detector(ctx))
	assertRule(t, findings, "S008", 0)
}

func TestS008_NoFindingForRequireSignatureWithSign(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.UsePublish(signing.SignMiddleware(signer))
	bus.Use(signing.RequireSignatureMiddleware(verifier))
}
`,
	})
	findings := runDetector(t, security.NewS008Detector(ctx))
	assertRule(t, findings, "S008", 0)
}

func TestS009_DetectsEncryptWithoutDecrypt(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.UsePublish(encryption.EncryptMiddleware(enc))
}
`,
	})
	findings := runDetector(t, security.NewS009Detector(ctx))
	assertRule(t, findings, "S009", 1)
}

func TestS009_DetectsDecryptWithoutEncrypt(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.Use(encryption.DecryptMiddleware(dec))
}
`,
	})
	findings := runDetector(t, security.NewS009Detector(ctx))
	assertRule(t, findings, "S009", 1)
}

func TestS009_NoFindingWhenBothPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus.UsePublish(encryption.EncryptMiddleware(enc))
	bus.Use(encryption.DecryptMiddleware(dec))
}
`,
	})
	findings := runDetector(t, security.NewS009Detector(ctx))
	assertRule(t, findings, "S009", 0)
}

func TestS009_NoFindingWhenNeitherPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})
	findings := runDetector(t, security.NewS009Detector(ctx))
	assertRule(t, findings, "S009", 0)
}
