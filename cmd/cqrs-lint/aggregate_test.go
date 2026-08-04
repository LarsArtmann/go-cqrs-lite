package main

import (
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestAggregateFromEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"user.created", "User"},
		{"order.shipped", "Order"},
		{"payment.processed", "Payment"},
		{"inventory.adjusted", "Inventory"},
		{"payment", "Payment"},
		{"noprefix", "Noprefix"},
		{"", ""},
		{"single", "Single"},
		{"a.b.c", "A"},
	}

	for _, tt := range tests {
		got := aggregateFromEventType(tt.input)
		if got != tt.want {
			t.Errorf("aggregateFromEventType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAggregateFromStateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"UserState", "User"},
		{"OrderState", "Order"},
		{"CounterState", "Counter"},
		{"OrderAggregateState", "Order"},
		{"UserAggregateState", "User"},
		{"PlainState", "Plain"},
		{"", ""},
		{"State", ""},
		{"AggregateState", ""},
	}

	for _, tt := range tests {
		got := aggregateFromStateType(tt.input)
		if got != tt.want {
			t.Errorf("aggregateFromStateType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCapitalizeFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"user", "User"},
		{"order", "Order"},
		{"", ""},
		{"Already", "Already"},
		{"UPPER", "UPPER"},
	}

	for _, tt := range tests {
		got := capitalizeFirst(tt.input)
		if got != tt.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildFileAggregateMap(t *testing.T) {
	t.Parallel()

	reg := &analyzer.CQRSRegistry{
		EventTypesEmitted: map[string]analyzer.EventEmission{
			"user.created":  {File: "user/events.go", Line: 10},
			"user.updated":  {File: "user/events.go", Line: 20},
			"order.placed":  {File: "order/events.go", Line: 5},
			"payment.paid":  {File: "payment/events.go", Line: 15},
		},
		Deciders: []analyzer.DeciderInfo{
			{StateType: "UserState", File: "user/decider.go"},
			{StateType: "OrderState", File: "order/decider.go"},
		},
		Folds: []analyzer.FoldInfo{
			{StateType: "UserState", File: "user/fold.go"},
			{StateType: "CounterAggregateState", File: "counter/fold.go"},
		},
	}

	m := buildFileAggregateMap(reg)

	// user/events.go should have only "User" (deduplicated)
	userAggs := m["user/events.go"]
	if len(userAggs) != 1 || userAggs[0] != "User" {
		t.Errorf("user/events.go aggregates = %v, want [User]", userAggs)
	}

	// order/events.go should have "Order"
	orderAggs := m["order/events.go"]
	if len(orderAggs) != 1 || orderAggs[0] != "Order" {
		t.Errorf("order/events.go aggregates = %v, want [Order]", orderAggs)
	}

	// user/decider.go should have "User" from decider
	userDeciderAggs := m["user/decider.go"]
	if len(userDeciderAggs) != 1 || userDeciderAggs[0] != "User" {
		t.Errorf("user/decider.go aggregates = %v, want [User]", userDeciderAggs)
	}

	// user/fold.go should have "User" from fold
	userFoldAggs := m["user/fold.go"]
	if len(userFoldAggs) != 1 || userFoldAggs[0] != "User" {
		t.Errorf("user/fold.go aggregates = %v, want [User]", userFoldAggs)
	}

	// counter/fold.go should have "Counter" from CounterAggregateState
	counterAggs := m["counter/fold.go"]
	if len(counterAggs) != 1 || counterAggs[0] != "Counter" {
		t.Errorf("counter/fold.go aggregates = %v, want [Counter]", counterAggs)
	}
}

func TestBuildFileAggregateMap_Empty(t *testing.T) {
	t.Parallel()

	reg := &analyzer.CQRSRegistry{
		EventTypesEmitted: map[string]analyzer.EventEmission{},
	}

	m := buildFileAggregateMap(reg)
	if len(m) != 0 {
		t.Errorf("expected empty map for empty registry, got %d entries", len(m))
	}
}

func TestBuildFileAggregateMap_MultipleAggregatesPerFile(t *testing.T) {
	t.Parallel()

	reg := &analyzer.CQRSRegistry{
		EventTypesEmitted: map[string]analyzer.EventEmission{
			"user.created":  {File: "shared/events.go"},
			"order.placed":  {File: "shared/events.go"},
		},
	}

	m := buildFileAggregateMap(reg)
	aggs := m["shared/events.go"]
	if len(aggs) != 2 {
		t.Fatalf("expected 2 aggregates, got %d: %v", len(aggs), aggs)
	}

	// Should be sorted alphabetically
	if aggs[0] != "Order" || aggs[1] != "User" {
		t.Errorf("expected [Order, User], got %v", aggs)
	}
}

func TestEnrichWithAggregate(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("user/events.go")}},
		{Position: finding.Position{File: finding.FilePath("order/events.go")}},
		{Position: finding.Position{File: finding.FilePath("utils/helper.go")}}, // no aggregate
		{
			Position: finding.Position{File: finding.FilePath("user/events.go")},
			Metadata: map[string]string{"aggregate": "AlreadySet"}, // detector-stamped
		},
	}

	actx := &analyzer.AnalysisContext{
		Registry: &analyzer.CQRSRegistry{
			EventTypesEmitted: map[string]analyzer.EventEmission{
				"user.created":  {File: "user/events.go"},
				"order.placed":  {File: "order/events.go"},
			},
		},
	}

	result := enrichWithAggregate(findings, actx)

	// user/events.go → "User"
	if result[0].Metadata["aggregate"] != "User" {
		t.Errorf("finding 0 aggregate = %q, want User", result[0].Metadata["aggregate"])
	}

	// order/events.go → "Order"
	if result[1].Metadata["aggregate"] != "Order" {
		t.Errorf("finding 1 aggregate = %q, want Order", result[1].Metadata["aggregate"])
	}

	// utils/helper.go → no aggregate (empty metadata key)
	if _, ok := result[2].Metadata["aggregate"]; ok {
		t.Errorf("finding 2 should have no aggregate, got %q", result[2].Metadata["aggregate"])
	}

	// Already-set aggregate should be preserved
	if result[3].Metadata["aggregate"] != "AlreadySet" {
		t.Errorf("finding 3 aggregate = %q, want AlreadySet (detector-stamped)", result[3].Metadata["aggregate"])
	}
}

func TestGroupFindingsByAggregate(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Metadata: map[string]string{"aggregate": "User"}},
		{Metadata: map[string]string{"aggregate": "User"}},
		{Metadata: map[string]string{"aggregate": "User"}},
		{Metadata: map[string]string{"aggregate": "Order"}},
		{Metadata: map[string]string{"aggregate": "Order"}},
		{}, // Uncategorized
	}

	groups := groupFindingsByAggregate(findings)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// User should be first (3 findings, most issues first)
	if groups[0].name != "User" {
		t.Errorf("expected first group to be User (3 findings), got %s", groups[0].name)
	}

	if len(groups[0].findings) != 3 {
		t.Errorf("expected 3 User findings, got %d", len(groups[0].findings))
	}

	// Order should be second (2 findings)
	if groups[1].name != "Order" {
		t.Errorf("expected second group to be Order (2 findings), got %s", groups[1].name)
	}

	// Uncategorized should be last
	if groups[2].name != "Uncategorized" {
		t.Errorf("expected last group to be Uncategorized, got %s", groups[2].name)
	}
}

func TestGroupFindingsByAggregate_TieBreakAlphabetical(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Metadata: map[string]string{"aggregate": "Zebra"}},
		{Metadata: map[string]string{"aggregate": "Alpha"}},
	}

	groups := groupFindingsByAggregate(findings)

	// Same count → alphabetical: Alpha before Zebra
	if groups[0].name != "Alpha" {
		t.Errorf("expected Alpha first (alphabetical tie-break), got %s", groups[0].name)
	}
}

func TestGroupFindingsByAggregate_AllUncategorized(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("a.go")}},
		{Position: finding.Position{File: finding.FilePath("b.go")}},
	}

	groups := groupFindingsByAggregate(findings)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].name != "Uncategorized" {
		t.Errorf("expected Uncategorized, got %s", groups[0].name)
	}

	if len(groups[0].findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(groups[0].findings))
	}
}

func TestResolveGroupMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		groupBy string
		verbose bool
		want    string
	}{
		{"explicit aggregate", "aggregate", false, "aggregate"},
		{"explicit module", "module", false, "module"},
		{"explicit none", "none", false, "none"},
		{"verbose maps to module", "", true, "module"},
		{"default is none", "", false, "none"},
		{"group-by overrides verbose", "aggregate", true, "aggregate"},
		{"uppercase normalized", "Aggregate", false, "aggregate"},
	}

	for _, tt := range tests {
		cfg := &AppConfig{GroupBy: tt.groupBy, Verbose: tt.verbose}
		got := resolveGroupMode(cfg)
		if got != tt.want {
			t.Errorf("%s: resolveGroupMode() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
