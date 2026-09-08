package main

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// TestConsumerOnlyRulesAreRealRules pins consumerOnlyRules against the rule
// catalog: a dead ID in the map silently stops suppressing nothing while
// still documenting intent, and a typo'd new entry makes library self-lint
// noisier than designed. Every ID must resolve through rules.AllRules().
func TestConsumerOnlyRulesAreRealRules(t *testing.T) {
	t.Parallel()

	for id := range consumerOnlyRules {
		if _, ok := rules.LookupRule(id); !ok {
			t.Errorf("consumerOnlyRules contains %q, which is not a registered rule", id)
		}
	}
}

// TestPresetRuleIDsAreRealRules validates every rule ID referenced by any
// preset definition (Disable lists and SeverityOverrides): a typo in a preset
// silently disables nothing (or overrides nothing) and the preset lies.
func TestPresetRuleIDsAreRealRules(t *testing.T) {
	t.Parallel()

	for name, def := range analyzer.PresetDefinitions {
		for _, id := range def.Rules.Disable {
			if _, ok := rules.LookupRule(id); !ok {
				t.Errorf("preset %q disables unknown rule %q", name, id)
			}
		}
		for id := range def.Rules.SeverityOverrides {
			if _, ok := rules.LookupRule(id); !ok {
				t.Errorf("preset %q overrides severity of unknown rule %q", name, id)
			}
		}
	}
}

// TestPresetHelpTextListsAllPresets pins the --preset flag's help text
// against analyzer.ValidPresetNames(): the help string is hand-maintained
// prose, and a new preset that forgets to update it is undiscoverable
// (the S-family split-brain pattern — co-named enum members in docs must
// fail CI instead of drifting silently).
func TestPresetHelpTextListsAllPresets(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeOf(initPresetFlags{}).FieldByName("Preset")
	if !ok {
		t.Fatal("initPresetFlags has no Preset field")
	}
	help := field.Tag.Get("help")
	const prefix = "Config preset: "
	if !strings.HasPrefix(help, prefix) {
		t.Fatalf("preset help text does not start with %q: %q", prefix, help)
	}

	listed := strings.Split(strings.TrimPrefix(help, prefix), ", ")
	sorted := slices.Clone(listed)
	slices.Sort(sorted)
	want := analyzer.ValidPresetNames()

	if strings.Join(sorted, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("preset help text is out of sync with ValidPresetNames():\n  help:  %v\n  valid: %v", listed, want)
	}
}
