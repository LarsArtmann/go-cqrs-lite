package security_test

import (
	"fmt"
	"strings"
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

// TestS001_SelectorLHSMessageCarriesReceiver pins the receiver-context fix
// (03-44 #101 second half): for `cfg.Password = …` the message must name the
// full target ("cfg.Password"), not the bare field — the receiver is what a
// reader must locate to fix the finding.
func TestS001_SelectorLHSMessageCarriesReceiver(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	Password string
}

func configure(cfg *Config) {
	cfg.Password = "super-secret-password-123"
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 1)

	for _, f := range findings {
		if string(f.Rule) != "S001" {
			continue
		}

		if !strings.Contains(f.Message, "cfg.Password") {
			t.Errorf(
				"S001 selector-LHS message %q must carry the receiver path \"cfg.Password\"",
				f.Message,
			)
		}
	}
}

// TestS001_AllowsURLsAndPlaceholders pins the URL/placeholder value
// allowlist: doc links and env-var/insertion templates on secret-named
// fields are not credentials.
func TestS001_AllowsURLsAndPlaceholders(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

var apiKeyDocsURL = "https://docs.example.com/authentication/api-keys"

const webhookTokenURL = "http://localhost:8080/webhooks/register"

func templates() {
	passwordPlaceholder := "<your-password-here>"
	tokenTemplate := "${API_KEY}"
	_, _ = passwordPlaceholder, tokenTemplate
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 0)
}

// TestS001_StillDetectsHashesAndSecrets guards the allowlist against
// over-suppression: bcrypt hashes start with "$" and real secrets may
// mention URLs in a suffix — both must still fire.
func TestS001_StillDetectsHashesAndSecrets(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

var passwordHash = "$2b$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func keymaterial() {
	token := "ghp_super-secret-key-1234567890"
	_ = token
}
`,
	})
	findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
	ruletest.AssertRule(t, findings, "S001", 2)
}

// TestS001_AllowlistKeepsRealCredentials validates the URL/placeholder
// allowlist against a corpus of REAL credential shapes: every value below
// is a live-secret form seen in the wild, and each must still be flagged —
// proving the allowlist suppresses only placeholders/URLs, never true
// positives (the 03-44 S001 corpus-validation item).
func TestS001_AllowlistKeepsRealCredentials(t *testing.T) {
	t.Parallel()

	realCorpus := []string{
		"sk-live-9f4ac1b2e8d74310aa52",
		"ghp_R4nd0mHexStr1ng0fFortyCh",
		"AKIAIOSFODNN7EXAMPLEKEY99",
		"xoxb-123456789012-9876543210-AbCdEfGh123456",
		"-----BEGIN RSA PRIVATE KEY-----MIIB",
		"hunter2-do-not-ship-me",
	}

	for i, secret := range realCorpus {
		name := fmt.Sprintf("credential_%d", i)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			src := "package main\n\nvar apiKeyValue = \"" + secret + "\"\n"
			ctx := analyzer.BuildContextFromSource(t, map[string]string{"secrets.go": src})
			findings := ruletest.RunDetector(t, security.NewS001Detector(ctx))
			ruletest.AssertRule(t, findings, "S001", 1)
		})
	}
}
