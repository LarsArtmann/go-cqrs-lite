package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD002_DetectsMixedJSONCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type User struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 1)
}

func TestD002_NoFindingForConsistentCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 0)
}

// D002 reports per-struct, not per-file. Cross-struct mixing (struct A all
// camelCase, struct B all snake_case) is legitimate: different structs may
// follow different conventions (API types vs event payloads). Single-word tags
// like "content" and "nick" are NEUTRAL — they don't count as camelCase.
// This test proves cross-struct mixing no longer fires.

func TestD002_NoFindingForCrossStructMix(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type DiscordMessage struct {
	Content string ` + "`json:\"content\"`" + `
	GuildID string ` + "`json:\"guild_id\"`" + `
}

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 0)
}

func TestD002_NoFindingWhenExternalAPIMarkerPresent(t *testing.T) {
	// The //cqrs-lint:external-api doc-comment marker marks the DiscordMessage
	// struct as mirroring Discord's API, so its snake_case tags don't count
	// toward the file-level camel/snake mix.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

//cqrs-lint:external-api
type DiscordMessage struct {
	Content string ` + "`json:\"content\"`" + `
	GuildID string ` + "`json:\"guild_id\"`" + `
}

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 0)
}

func TestD002_NoFindingWhenMarkerOnGroupedTypeBlock(t *testing.T) {
	// For grouped `type ( ... )` blocks the doc comment lives on the TypeSpec,
	// not the GenDecl. The marker must be honored in both positions.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type (
	//cqrs-lint:external-api
	StripeCharge struct {
		Amount   string ` + "`json:\"amount\"`" + `
		Currency string ` + "`json:\"stripe_currency\"`" + `
	}
)

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 0)
}

func TestD002_NoFindingForConfiguredExternalAPIPrefix(t *testing.T) {
	// Config-driven suppression: .cqrs-lint.json →
	//   {"rules": {"external-api-struct-prefixes": ["Discord"]}}
	// marks any struct whose name starts with "Discord" as an external mirror.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type DiscordMessage struct {
	Content string ` + "`json:\"content\"`" + `
	GuildID string ` + "`json:\"guild_id\"`" + `
}

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
}
`,
	})
	ctx.RulesConfig.ExternalAPIStructPrefixes = []string{"Discord"}
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 0)
}

func TestD002_FiresWhenPrefixDoesNotMatch(t *testing.T) {
	// A configured prefix that does NOT match the struct name must not
	// suppress. DiscordWebhook genuinely mixes snake_case and camelCase
	// internally — the non-matching "Stripe" prefix must not exclude it.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type DiscordWebhook struct {
	GuildID   string ` + "`json:\"guild_id\"`" + `
	WebhookID string ` + "`json:\"webhookId\"`" + `
}
`,
	})
	ctx.RulesConfig.ExternalAPIStructPrefixes = []string{"Stripe"}
	findings := ruletest.RunDetector(t, consistency.NewD002Detector(ctx))
	ruletest.AssertRule(t, findings, "D002", 1)
}
