package main

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestReadmePresetTableMatchesCode prevents documentation drift between the
// preset table in README.md and PresetDefinitions in code. The table is
// hand-maintained; without this test, adding a rule to a preset's disable list
// in code but forgetting to update the README (or vice versa) goes unnoticed.
//
// This test caught the session-1 drift where F015/F022 were added to the
// library preset in code but the README table was never updated.
func TestReadmePresetTableMatchesCode(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("cannot read README.md: %v", err)
	}

	// Parse the preset table: rows like "| `local-cli`  | ... | F004, F009 | ... |"
	type readmePresetRow struct {
		name        string
		ruleDisable []string
	}

	var rows []readmePresetRow

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `") {
			continue
		}

		cols := strings.Split(line, "|")
		if len(cols) < 5 {
			continue
		}

		name := strings.Trim(strings.TrimSpace(cols[1]), "`")
		rulesCol := strings.TrimSpace(cols[3])

		var rules []string
		if rulesCol != "(none)" && rulesCol != "" {
			for _, r := range strings.Split(rulesCol, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					rules = append(rules, r)
				}
			}
		}

		rows = append(rows, readmePresetRow{name: name, ruleDisable: rules})
	}

	if len(rows) == 0 {
		t.Fatal("no preset rows found in README.md — table format may have changed")
	}

	// Build a lookup of preset name → README row.
	readmeMap := make(map[string]readmePresetRow, len(rows))
	for _, r := range rows {
		readmeMap[r.name] = r
	}

	// Assert every PresetDefinition appears in the README and the disable
	// lists match as sets (order-independent).
	for presetName, def := range analyzer.PresetDefinitions {
		readmeRow, ok := readmeMap[string(presetName)]
		if !ok {
			t.Errorf("preset %q is missing from the README.md preset table", presetName)
			continue
		}

		codeDisable := def.Rules.Disable

		// Every README rule must be in the code disable list.
		for _, r := range readmeRow.ruleDisable {
			if !slices.Contains(codeDisable, r) {
				t.Errorf(
					"preset %q: README lists rule %q in disable column "+
						"but PresetDefinitions does not — update code or README",
					presetName, r,
				)
			}
		}

		// Every code disable rule must be in the README.
		for _, r := range codeDisable {
			if !slices.Contains(readmeRow.ruleDisable, r) {
				t.Errorf(
					"preset %q: PresetDefinitions disables rule %q "+
						"but README table does not list it — update README.md",
					presetName, r,
				)
			}
		}
	}
}
