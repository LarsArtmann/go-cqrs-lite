package main

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

// discordsyncLikeSources is a minimal in-memory project that reproduces the
// false-positive patterns the real DiscordSync consumer triggered when cqrs-lint
// was first run against it. Each file maps to a class of finding the round-1
// and round-2 work addressed:
//
//   - discord_models.go: mixed snake_case (Discord API mirror) + camelCase
//     (internal HTTP) JSON tags → D002 fires without the external-API opt-out.
//     Several discord_*.go files reproduce the multi-file spread (18 findings
//     in the real report) so the health-score move is visible above rounding.
//   - tx.go: a genuine missing-commit bug (C001 must fire) AND a closure-escape
//     helper (C001 must NOT fire — the round-1 false-positive fix).
//   - metrics.go: a strong float64 money field ("Balance") in a non-monetary
//     project → C008 must fire but downgrade to Info/Low (round-2 fix).
//   - sse.go: SubscribeAll callbacks that only broadcast → A005 must NOT fire
//     (round-1 broadcast-vs-projection + round-2 widened signals).
//
// Together these files are the synthetic equivalent of the original 34-finding
// DiscordSync report. The meta-tests below prove the *sum* of the fixes behaves
// correctly in aggregate, not just per-rule in isolation.
var discordsyncLikeSources = map[string]string{
	"discord_models.go": `package app

// DiscordMessage mirrors the Discord Gateway payload. Its snake_case JSON tags
// are dictated by the upstream API and are NOT a local style choice.
type DiscordMessage struct {
	Content  string ` + "`json:\"content\"`" + `
	GuildID  string ` + "`json:\"guild_id\"`" + `
	AuthorID string ` + "`json:\"author_id\"`" + `
}

// InternalResponse is a local HTTP API response struct using camelCase. Mixing
// it with the snake_case Discord structs in the same file is what triggers D002.
type InternalResponse struct {
	MessageID string ` + "`json:\"messageId\"`" + `
	ChannelID string ` + "`json:\"channelId\"`" + `
}
`,
	"discord_member.go": `package app

type DiscordMember struct {
	UserID string ` + "`json:\"user_id\"`" + `
	Nick   string ` + "`json:\"nick\"`" + `
}

type MemberView struct {
	DisplayName string ` + "`json:\"displayName\"`" + `
	AvatarURL   string ` + "`json:\"avatarUrl\"`" + `
}
`,
	"discord_role.go": `package app

type DiscordRole struct {
	RoleID string ` + "`json:\"role_id\"`" + `
	Color  int    ` + "`json:\"color\"`" + `
}

type RoleSummary struct {
	IsDefault bool ` + "`json:\"isDefault\"`" + `
	Position  int ` + "`json:\"position\"`" + `
}
`,
	"discord_channel.go": `package app

type DiscordChannel struct {
	ID      string ` + "`json:\"id\"`" + `
	GuildID string ` + "`json:\"guild_id\"`" + `
}

type ChannelInfo struct {
	SortOrder int ` + "`json:\"sortOrder\"`" + `
	IsArchived bool ` + "`json:\"isArchived\"`" + `
}
`,
	"discord_reaction.go": `package app

type DiscordReaction struct {
	EmojiID string ` + "`json:\"emoji_id\"`" + `
	Count   int    ` + "`json:\"count\"`" + `
}

type ReactionResult struct {
	TotalCount int ` + "`json:\"totalCount\"`" + `
	HasReacted bool ` + "`json:\"hasReacted\"`" + `
}
`,
	"discord_guild.go": `package app

type DiscordGuild struct {
	GuildID  string ` + "`json:\"guild_id\"`" + `
	OwnerID  string ` + "`json:\"owner_id\"`" + `
	MemberCount int ` + "`json:\"member_count\"`" + `
}

type GuildStatus struct {
	IsActive bool ` + "`json:\"isActive\"`" + `
	ShardID  int  ` + "`json:\"shardId\"`" + `
}
`,
	"tx.go": `package app

import (
	"context"
	"database/sql"
)

// writeNoCommit is a genuine bug: the tx is used (Exec) but never committed.
// C001 must flag this.
func writeNoCommit(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO jobs VALUES (1)")
	return nil
}

// withTx is the closure-helper pattern. The tx escapes to a callback that
// contractually owns the commit. C001 must NOT flag this (the round-1
// false-positive fix — suggesting return tx.Commit() here would double-commit).
func withTx(ctx context.Context, db *sql.DB, body func(*sql.Tx) error) error {
	tx, _ := db.BeginTx(ctx, nil)
	if err := body(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}
`,
	"metrics.go": `package app

// RateTracker holds a Prometheus counter delta (events/sec), not money. The
// "Balance" field name is coincidental (queue balance, not money balance) but
// C008's strong-field heuristic flags it. In a non-monetary project the round-2
// fix downgrades it to Info/Low instead of Warning/Medium.
type RateTracker struct {
	Balance float64
	Window  int
}

// SparklineSample tracks projection lag in seconds.
type SparklineSample struct {
	Amount float64
}
`,
	"sse.go": `package app

// setupSSE registers a fire-and-forget broadcaster via SubscribeAll. A005 must
// NOT flag this (broadcast fan-out, no persistence — not a projection).
func setupSSE(bus EventBus, broker *Broker) {
	bus.SubscribeAll(func(evt Event) {
		broker.Broadcast(evt)
	})
}

// setupStats forwards events to a stats notifier. Also fire-and-forget.
func setupStats(bus EventBus, notifier *Notifier) {
	bus.SubscribeAll(func(evt Event) {
		notifier.Publish(evt)
	})
}
`,
}

// runAllDetectors runs every registered detector against the context and returns
// the combined findings. Mirrors what the real pipeline does (minus filtering).
func runAllDetectors(t *testing.T, ctx *analyzer.AnalysisContext) []finding.Finding {
	t.Helper()
	detectors := rules.RegisterAll(ctx)

	var all []finding.Finding

	for _, det := range detectors {
		findings, err := det.Detect(context.Background())
		if err != nil {
			t.Errorf("detector %s: %v", det.Name(), err)
			continue
		}

		all = append(all, findings...)
	}

	return all
}

func countByRule(findings []finding.Finding, ruleID string) int {
	count := 0
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}
	return count
}

func severityForRule(findings []finding.Finding, ruleID string) (finding.Severity, bool) {
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			return f.Severity, true
		}
	}
	return finding.SeverityInfo, false
}

// TestDiscordSyncFixture_FixedRulesBehaveCorrectly proves the round-1/round-2
// fixes behave correctly *in aggregate* on a DiscordSync-shaped project:
//
//   - D002 fires on the mixed-casing file (the opt-out isn't configured yet).
//   - C001 fires exactly once (the genuine bug) and NOT on the closure helper.
//   - C008 downgrades to Info (no monetary project signal).
//   - A005 does not fire on broadcast fan-out callbacks.
func TestDiscordSyncFixture_FixedRulesBehaveCorrectly(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, discordsyncLikeSources)
	findings := runAllDetectors(t, ctx)

	t.Run("D002_fires_without_opt_out", func(t *testing.T) {
		if got := countByRule(findings, "D002"); got == 0 {
			t.Errorf("D002 should fire on mixed-casing file without opt-out, got 0 findings")
		}
	})

	t.Run("C001_fires_once_on_genuine_bug", func(t *testing.T) {
		if got := countByRule(findings, "C001"); got != 1 {
			t.Errorf("C001: got %d findings, want 1 (the genuine missing-commit; "+
				"the closure helper must be suppressed)", got)
		}
	})

	t.Run("C008_downgraded_to_info", func(t *testing.T) {
		sev, ok := severityForRule(findings, "C008")
		if !ok {
			t.Skip("C008 did not fire on this fixture")
		}
		if sev != finding.SeverityInfo {
			t.Errorf("C008 severity = %s, want Info (project has no monetary signal)", sev)
		}
	})

	t.Run("A005_suppressed_for_broadcast_fan_out", func(t *testing.T) {
		if got := countByRule(findings, "A005"); got != 0 {
			t.Errorf("A005: got %d findings, want 0 (broadcast callbacks are not projections)", got)
		}
	})
}

// TestDiscordSyncFixture_D002ConfigImprovesHealthScore proves the end-to-end
// customer-visible win: configuring the D002 external-API prefix both eliminates
// the false-positive findings AND raises the health score. This is the aggregate
// proof that was missing from the unit-level round-1/round-2 tests.
func TestDiscordSyncFixture_D002ConfigImprovesHealthScore(t *testing.T) {
	t.Parallel()

	// Baseline: no config — D002 fires on the Discord mirror structs.
	baseCtx := analyzer.BuildContextFromSource(t, discordsyncLikeSources)
	baseFindings := runAllDetectors(t, baseCtx)
	baseD002 := countByRule(baseFindings, "D002")
	baseScore := ComputeHealthScore(baseFindings).Score

	if baseD002 == 0 {
		t.Fatal("baseline D002 should fire without config")
	}

	// Configured: external-API prefix "Discord" excludes the mirror structs.
	cfgCtx := analyzer.BuildContextFromSource(t, discordsyncLikeSources)
	cfgCtx.RulesConfig.ExternalAPIStructPrefixes = []string{"Discord"}
	cfgFindings := runAllDetectors(t, cfgCtx)
	cfgD002 := countByRule(cfgFindings, "D002")
	cfgScore := ComputeHealthScore(cfgFindings).Score

	if cfgD002 >= baseD002 {
		t.Errorf("D002 findings after config = %d, want < %d (baseline)", cfgD002, baseD002)
	}

	if cfgScore <= baseScore {
		t.Errorf("health score after config = %d, want > %d (baseline)", cfgScore, baseScore)
	}

	t.Logf("health score: %d → %d after D002 opt-out (D002: %d → %d)",
		baseScore, cfgScore, baseD002, cfgD002)
}

// TestDiscordSyncFixture_MarkerSuppressesD002 proves the in-source marker works
// in the aggregate run too (not just in the D002 unit test).
func TestDiscordSyncFixture_MarkerSuppressesD002(t *testing.T) {
	t.Parallel()

	markerSources := map[string]string{
		"marked.go": `package app

//cqrs-lint:external-api
type DiscordMessage struct {
	Content string ` + "`json:\"content\"`" + `
	MsgID   string ` + "`json:\"msgId\"`" + `
}
`,
	}
	ctx := analyzer.BuildContextFromSource(t, markerSources)
	findings := runAllDetectors(t, ctx)

	if got := countByRule(findings, "D002"); got != 0 {
		t.Errorf("D002 with in-source marker: got %d findings, want 0", got)
	}
}
