package rules_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// BenchmarkDetectorC001 benchmarks the C001 detector on a simple fixture.
func BenchmarkDetectorC001(b *testing.B) {
	ctx := analyzer.BuildContextFromSource(&testing.T{}, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
)

func withTx(ctx context.Context, db *sql.DB, body func(*sql.Tx) error) error {
	tx, _ := db.BeginTx(ctx, nil)
	if err := body(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}
`,
	})
	det := correctness.NewC001Detector(ctx)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = det.Detect(context.Background())
	}
}

// BenchmarkRegisterAll benchmarks registering all detectors.
func BenchmarkRegisterAll(b *testing.B) {
	ctx := analyzer.BuildContextFromSource(&testing.T{}, map[string]string{
		"main.go": `package main`,
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rules.RegisterAll(ctx)
	}
}

// BenchmarkFilterByRuleIDs benchmarks individual rule filtering.
func BenchmarkFilterByRuleIDs(b *testing.B) {
	ctx := analyzer.BuildContextFromSource(&testing.T{}, map[string]string{
		"main.go": `package main`,
	})
	all := rules.RegisterAll(ctx)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rules.FilterByRuleIDs(all, []string{"C001", "C002", "C003"})
	}
}

// TestGoldenFile_JSONOutput verifies JSON output format stability.
func TestGoldenFile_JSONOutput(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"C001", "cqrs-lint",
		"test finding for golden file",
		finding.SeverityError,
		finding.Pos(finding.FilePath("example.go"), 10, 5),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion("Fix this issue").
		Build()
	if err != nil {
		t.Fatal(err)
	}

	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: "0.1.0"})
	report.AddFindings([]finding.Finding{f})

	jsonOut, err := report.PrettyJSON()
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join("testdata", "json_output.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(goldenPath, []byte(jsonOut+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Skipf("golden file not found (run with UPDATE_GOLDEN=1 to create): %v", err)
	}

	if strings.TrimRight(string(expected), "\n") != jsonOut {
		t.Errorf("JSON output changed from golden file. Run with UPDATE_GOLDEN=1 to update.")
	}
}

// TestGoldenFile_SARIFOutput verifies SARIF output format stability.
// Uses JSON structural comparison because SARIF properties are serialized
// from Go maps with non-deterministic key ordering.
func TestGoldenFile_SARIFOutput(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"C001", "cqrs-lint",
		"test finding for golden file",
		finding.SeverityError,
		finding.Pos(finding.FilePath("example.go"), 10, 5),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion("Fix this issue").
		Build()
	if err != nil {
		t.Fatal(err)
	}

	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: "0.1.0"})
	report.AddFindings([]finding.Finding{f})

	sarifOut, err := report.ToSARIF()
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join("testdata", "sarif_output.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(goldenPath, append(sarifOut, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Skipf("golden file not found (run with UPDATE_GOLDEN=1 to create): %v", err)
	}

	var expectedJSON, actualJSON any
	if err := json.Unmarshal(expected, &expectedJSON); err != nil {
		t.Fatalf("failed to parse golden SARIF: %v", err)
	}

	if err := json.Unmarshal(sarifOut, &actualJSON); err != nil {
		t.Fatalf("failed to parse actual SARIF: %v", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Errorf(
			"SARIF output structure changed from golden file. Run with UPDATE_GOLDEN=1 to update.",
		)
	}
}
