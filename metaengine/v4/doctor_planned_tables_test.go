package metaengine_test

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestDoctor_PlannedTablesSection pins the "--- Planned tables ---" Doctor
// section: a store over an engine without the PlannedTablesReporter
// capability reports "none" instead of an empty or missing section.
func TestDoctor_PlannedTablesSection(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())

	if !strings.Contains(output, "--- Planned tables ---") {
		t.Errorf("Doctor output missing planned-tables section:\n%s", output)
	}

	// Extract the section and assert it reports none (memory engine does not
	// implement PlannedTablesReporter).
	start := strings.Index(output, "--- Planned tables ---")
	end := strings.Index(output[start:], "\n---")

	section := output[start:]
	if end >= 0 {
		section = output[start : start+end]
	}

	if !strings.Contains(section, "none") {
		t.Errorf("planned-tables section must report none without reporter engines:\n%s", section)
	}
}
