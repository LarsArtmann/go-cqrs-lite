package architecture_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- E012: Dual-write without completion ---

func TestE012_DetectsDualWriteWithoutFlag(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

type DualWriteBus struct {
	legacy Publisher
	newSystem Publisher
}

type Publisher interface{ Publish(any) error }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE012Detector(ctx))
	ruletest.AssertRule(t, findings, "E012", 1)
}

func TestE012_NoFindingWithFeatureFlag(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"bus.go": `package main

type DualWriteBus struct {
	legacy Publisher
	newSystem Publisher
}

type Config struct {
	DualWriteEnabled bool
}

type Publisher interface{ Publish(any) error }

func newDualWriteBus(cfg Config) *DualWriteBus {
	return &DualWriteBus{}
}

func setup() *DualWriteBus {
	return newDualWriteBus(Config{DualWriteEnabled: true})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE012Detector(ctx))
	ruletest.AssertRule(t, findings, "E012", 0)
}

func TestE012_NoFindingOnEmptyProject(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE012Detector(ctx))
	ruletest.AssertRule(t, findings, "E012", 0)
}

// --- E013: Signing disabled by default ---

func TestE013_DetectsSigningDisabled(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing"

type SigningConfig struct {
	Enabled bool
	Key     string
}

func DefaultSigningConfig() SigningConfig {
	_ = signing.NewHMAC
	return SigningConfig{Enabled: false, Key: ""}
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE013Detector(ctx))
	ruletest.AssertRule(t, findings, "E013", 1)
}

func TestE013_NoFindingWhenEnabled(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing"

type SigningConfig struct {
	Enabled bool
	Key     string
}

func DefaultSigningConfig() SigningConfig {
	_ = signing.NewHMAC
	return SigningConfig{Enabled: true, Key: "secret"}
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE013Detector(ctx))
	ruletest.AssertRule(t, findings, "E013", 0)
}

func TestE013_NoFindingWithoutSigningImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	Enabled bool
}

func defaultConfig() Config {
	return Config{Enabled: false}
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE013Detector(ctx))
	ruletest.AssertRule(t, findings, "E013", 0)
}

// --- E014: No read-your-writes ---

func TestE014_DetectsNoDrain(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projectionhost"

func setup(host *projectionhost.Host) {
	go host.Start(nil)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE014Detector(ctx))
	ruletest.AssertRule(t, findings, "E014", 1)
}

func TestE014_NoFindingWithStalenessGuard(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projectionhost"

func respond(host *projectionhost.Host) error {
	return host.CheckStaleness(0)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE014Detector(ctx))
	ruletest.AssertRule(t, findings, "E014", 0)
}

// TestE014_NoFindingWithLagRead: LagDuration is the other real freshness
// primitive on projectionhost.Host.
func TestE014_NoFindingWithLagRead(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projectionhost"

func lag(host *projectionhost.Host) string {
	return host.LagDuration().String()
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE014Detector(ctx))
	ruletest.AssertRule(t, findings, "E014", 0)
}

// TestE014_SuggestionNamesRealAPI: every method the suggestion tells the
// user to call must exist on projectionhost.Host (projectionhost/v4).
func TestE014_SuggestionNamesRealAPI(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projectionhost"

func setup(host *projectionhost.Host) {
	_ = host
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE014Detector(ctx))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	suggestion := findings[0].Suggestion
	for _, gone := range []string{"Drain", "Sync()", "Flush", "WaitFor"} {
		if strings.Contains(suggestion, gone) {
			t.Errorf("suggestion names %q, which projectionhost.Host does not have: %s", gone, suggestion)
		}
	}

	for _, real := range []string{"CheckStaleness", "CheckProjectionStaleness"} {
		if !strings.Contains(suggestion, real) {
			t.Errorf("suggestion must name the real %s: %s", real, suggestion)
		}
	}
}

func TestE014_NoFindingWithoutProjectionHost(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup() {
	println("hello")
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE014Detector(ctx))
	ruletest.AssertRule(t, findings, "E014", 0)
}

// --- E015: Watermill no ordered delivery ---

func TestE015_DetectsFalseBlockPublish(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/watermill"

func setup() {
	cfg := watermill.EventBusConfig{
		BlockPublishUntilSubscriberAck: false,
	}
	_ = cfg
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE015Detector(ctx))
	ruletest.AssertRule(t, findings, "E015", 1)
}

func TestE015_NoFindingWhenTrue(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/watermill"

func setup() {
	cfg := watermill.EventBusConfig{
		BlockPublishUntilSubscriberAck: true,
	}
	_ = cfg
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE015Detector(ctx))
	ruletest.AssertRule(t, findings, "E015", 0)
}

func TestE015_NoFindingWithoutWatermill(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type Config struct {
	BlockPublishUntilSubscriberAck bool
}

func setup() {
	_ = Config{BlockPublishUntilSubscriberAck: false}
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE015Detector(ctx))
	ruletest.AssertRule(t, findings, "E015", 0)
}
