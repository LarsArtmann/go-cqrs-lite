package main

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// TestModernityGrade pins the three modernity grades: Legacy dominates (any
// v5-removed surface), then Modern requires v5-clean plus a canonical-path
// signal, else Partial.
func TestModernityGrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		deprecated *ScorecardDeprecated
		fp         analyzer.FeatureProfile
		want       string
	}{
		{
			name:       "removed APIs force Legacy",
			deprecated: &ScorecardDeprecated{RemovedAPIUses: 1},
			fp:         analyzer.FeatureProfile{HasSystemComposition: true},
			want:       "Legacy",
		},
		{
			name:       "deprecated transport forces Legacy",
			deprecated: &ScorecardDeprecated{DeprecatedTransport: 2},
			want:       "Legacy",
		},
		{
			name:       "stack-preset import forces Legacy even with pushdown",
			deprecated: &ScorecardDeprecated{StackPresetUses: 1},
			fp: analyzer.FeatureProfile{
				HasMetaengine: true, MetaenginePushdown: true,
			},
			want: "Legacy",
		},
		{
			name:       "stack-preset import forces Legacy even with system composition",
			deprecated: &ScorecardDeprecated{StackPresetUses: 2},
			fp:         analyzer.FeatureProfile{HasSystemComposition: true},
			want:       "Legacy",
		},
		{
			name: "system composition is Modern",
			fp:   analyzer.FeatureProfile{HasSystemComposition: true},
			want: "Modern",
		},
		{
			name: "metaengine with pushdown is Modern",
			fp:   analyzer.FeatureProfile{HasMetaengine: true, MetaenginePushdown: true},
			want: "Modern",
		},
		{
			name: "metaengine without pushdown is Partial",
			fp:   analyzer.FeatureProfile{HasMetaengine: true},
			want: "Partial",
		},
		{
			name: "v5-clean without composition signals is Partial",
			want: "Partial",
		},
		{
			name:       "nil deprecated panel is treated as v5-clean",
			deprecated: nil,
			fp:         analyzer.FeatureProfile{HasSystemComposition: true},
			want:       "Modern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ModernityGrade(tt.deprecated, tt.fp); got != tt.want {
				t.Errorf("ModernityGrade = %q, want %q", got, tt.want)
			}
			if hint := ModernityHint(tt.want); hint == "" {
				t.Errorf("ModernityHint(%q) must not be empty", tt.want)
			}
		})
	}
}
