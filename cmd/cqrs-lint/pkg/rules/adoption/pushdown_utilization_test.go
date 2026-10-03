package adoption_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// Pushdown utilization coaching (nsfw-classifier feedback, 2026-10-03):
// metaengine importers with Go-side sort/filter over a registered Query's R
// type must be coached (F022/F023), gated on the declaration's Volume — the
// adoption-time blanket skip hid exactly this population.

func TestF022_UtilizationFiresForSortOverLargeVolumeQuery(t *testing.T) {
	// Not parallel: buildScanFixtureContext sets GOWORK via t.Setenv.
	ctx := buildScanFixtureContext(t)

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 1)

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d", len(findings))
	}
	suggestion := findings[0].Suggestion
	for _, want := range []string{"SortOnField[itemView]", "memory engine"} {
		if !strings.Contains(suggestion, want) {
			t.Errorf("F022 suggestion %q missing %q", suggestion, want)
		}
	}
}

func TestF023_UtilizationFiresNamingFilteredFields(t *testing.T) {
	ctx := buildScanFixtureContext(t)

	findings := ruletest.RunDetector(t, adoption.NewF023Detector(ctx))
	ruletest.AssertRule(t, findings, "F023", 1)

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding (small-volume and declarative queries are negative controls), got %d",
			len(findings))
	}
	message := findings[0].Message
	for _, want := range []string{"roomItemsCollection", "RoomID", "Score"} {
		if !strings.Contains(message, want) {
			t.Errorf("F023 message %q missing %q", message, want)
		}
	}
	suggestion := findings[0].Suggestion
	for _, want := range []string{
		"FilterOnField[itemView]",
		"WithFilter",    // read-time binding layer
		"runtime data",  // values never go in the declaration
		"memory engine", // declare-anyway framing
	} {
		if !strings.Contains(suggestion, want) {
			t.Errorf("F023 suggestion %q missing %q", suggestion, want)
		}
	}
}

func TestPushdown_ProfileLineIsActionable(t *testing.T) {
	ctx := buildScanFixtureContext(t)

	rendered := ctx.FeatureProfile.String()
	// The fixture's declarative query flips the global flag; the census makes
	// the line actionable either way.
	if !strings.Contains(rendered, "pushdown:    true (1/3 queries declarative)") {
		t.Errorf("profile pushdown line must name the query census, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "metaengine:    true") {
		t.Errorf("profile must detect metaengine, got:\n%s", rendered)
	}
}
