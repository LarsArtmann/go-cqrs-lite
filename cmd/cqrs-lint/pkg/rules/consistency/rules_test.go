package consistency_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

func runDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

func assertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
	t.Helper()
	count := 0
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}
	if count != wantCount {
		t.Errorf("rule %s: got %d findings, want %d", ruleID, count, wantCount)
		for _, f := range findings {
			t.Logf("  finding: %s %s: %s", f.Rule, f.Severity, f.Message)
		}
	}
}

func TestD002_DetectsMixedJSONCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type User struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 1)
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
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 0)
}

// D002 must NOT fire when a file mixes camelCase consumer types with
// snake_case structs that mirror an external API (Discord/Stripe/GitHub).
// This is the biggest false-positive source in real consumer code (18 of 34
// DiscordSync findings). Two opt-outs are supported: an in-source marker and a
// config prefix list. These tests prove both, plus a regression that the same
// file still fires WITHOUT suppression.

func TestD002_FiresForCrossStructMixWithoutSuppression(t *testing.T) {
	// Same shape as the suppression tests below but with NO opt-out. Proves the
	// cross-struct mix (consumer camelCase + external mirror snake_case) still
	// triggers D002 so the suppression isn't silently swallowing real findings.
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
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 1)
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
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 0)
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
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 0)
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
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 0)
}

func TestD002_FiresWhenPrefixDoesNotMatch(t *testing.T) {
	// A configured prefix that does NOT match the struct name must not
	// suppress — guards against the suppression being too broad.
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
	ctx.RulesConfig.ExternalAPIStructPrefixes = []string{"Stripe"}
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 1)
}
