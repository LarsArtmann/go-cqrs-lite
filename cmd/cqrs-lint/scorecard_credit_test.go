package main

import (
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// absentUsage builds a usage map with every scored catalog module absent.
func absentUsage() map[analyzer.ModuleKey]analyzer.ModuleUsage {
	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage, 32)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		usage[e.Key] = analyzer.ModuleUsage{Key: e.Key, Status: analyzer.UsageAbsent}
	}
	return usage
}

func usedKeysOf(result ScorecardResult) map[string]bool {
	keys := make(map[string]bool, len(result.Used))
	for _, m := range result.Used {
		keys[m.Key] = true
	}
	return keys
}

// TestComputeScorecard_CompositionCreditSystemNewJournalShape reproduces the
// journal project's wiring (system.New + metaengine sqliteengine, no direct
// stack/storage imports) and pins that the persistence rows are credited
// instead of reading MISSING — the false-negative class that licensed the
// consumer's "do NOT chase the grade" moat.
func TestComputeScorecard_CompositionCreditSystemNewJournalShape(t *testing.T) {
	t.Parallel()

	fp := analyzer.FeatureProfile{
		HasMetaengine:        true,
		MetaengineEngines:    []string{"sqlite"},
		Store:                analyzer.StoreSQLite,
		Stores:               []analyzer.StoreKind{analyzer.StoreSQLite},
		HasSystemComposition: true,
	}

	result := ComputeScorecard(analyzer.DefaultCatalog, absentUsage(), fp, analyzer.PresetNone)

	used := usedKeysOf(result)
	evidence := make(map[string]string, len(result.Used))
	for _, m := range result.Used {
		evidence[m.Key] = m.Evidence
	}

	for _, key := range []string{"stack/sqlite", "storage", "stack/memory"} {
		if !used[key] {
			t.Errorf("module %s should be composition-credited as used, got MISSING", key)
		}
	}

	wantSQLite := "composed via metaengine sqlite engine import; composed via system.New (wired internally)"
	if evidence["stack/sqlite"] != wantSQLite {
		t.Errorf("stack/sqlite evidence = %q, want %q", evidence["stack/sqlite"], wantSQLite)
	}
	if evidence["storage"] != "composed via system.New (system wires storage/ internally)" {
		t.Errorf("storage evidence = %q", evidence["storage"])
	}
	if evidence["stack/memory"] != "composed via system.New (in-memory kernel embedded)" {
		t.Errorf("stack/memory evidence = %q", evidence["stack/memory"])
	}

	for _, key := range []string{"stack/sqlite", "storage", "stack/memory"} {
		if slices.ContainsFunc(
			result.Missing,
			func(m ScorecardModule) bool { return m.Key == key },
		) {
			t.Errorf("module %s must not remain MISSING after composition credit", key)
		}
	}
}

// TestComputeScorecard_CompositionCreditDirectImportWins pins that a direct
// import's evidence (the import path) is never overwritten by composition
// credit — the stronger signal stays.
func TestComputeScorecard_CompositionCreditDirectImportWins(t *testing.T) {
	t.Parallel()

	usage := absentUsage()
	usage["stack/sqlite"] = analyzer.ModuleUsage{
		Key:      "stack/sqlite",
		Status:   analyzer.UsageImported,
		Evidence: "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4",
	}
	fp := analyzer.FeatureProfile{
		Store:  analyzer.StoreSQLite,
		Stores: []analyzer.StoreKind{analyzer.StoreSQLite},
	}

	result := ComputeScorecard(analyzer.DefaultCatalog, usage, fp, analyzer.PresetNone)

	for _, m := range result.Used {
		if m.Key == "stack/sqlite" &&
			m.Evidence != "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4" {
			t.Errorf("direct import evidence overwritten: %q", m.Evidence)
		}
	}
}

// TestComputeScorecard_CompositionCreditNeverPullsIrrelevantRows pins the
// relevantSet guard: a detected backend whose row is irrelevant for the
// profile stays Irrelevant — credit must never inflate the denominator.
func TestComputeScorecard_CompositionCreditNeverPullsIrrelevantRows(t *testing.T) {
	t.Parallel()

	fp := analyzer.FeatureProfile{
		HasServer:   false,
		ServerLocal: false,
		Store:       analyzer.StorePostgres,
		Stores:      []analyzer.StoreKind{analyzer.StorePostgres},
	}

	result := ComputeScorecard(analyzer.DefaultCatalog, absentUsage(), fp, analyzer.PresetLocalCLI)

	if usedKeysOf(result)["stack/postgres"] {
		t.Error("stack/postgres is irrelevant for local-cli and must not be credited into Used")
	}
	if !slices.ContainsFunc(
		result.Irrelevant,
		func(m ScorecardModule) bool { return m.Key == "stack/postgres" },
	) {
		t.Error("stack/postgres should be Irrelevant for local-cli")
	}
}

// TestComputeScorecard_NoCompositionSignalsKeepsMissingRows pins that a
// profile without composition signals changes nothing: the credit path is a
// no-op for plain import-based consumers.
func TestComputeScorecard_NoCompositionSignalsKeepsMissingRows(t *testing.T) {
	t.Parallel()

	result := ComputeScorecard(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	if len(result.Used) != 0 {
		t.Errorf("expected 0 used without any signal, got %d", len(result.Used))
	}
}
