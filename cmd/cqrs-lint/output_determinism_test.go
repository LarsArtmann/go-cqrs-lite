package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// renderSARIF runs cqrs-lint's actual SARIF output path (Report.WriteSARIF,
// as called from outputFindings) and returns the bytes.
func renderSARIF(t *testing.T, findings []finding.Finding) []byte {
	t.Helper()

	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: "test"})
	report.AddFindings(findings)

	var buf bytes.Buffer
	if err := report.WriteSARIF(context.Background(), &buf); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}

	return buf.Bytes()
}

// renderJSON runs cqrs-lint's actual JSON output path (Report.PrettyJSON,
// as called from outputFindings) and returns the string.
func renderJSON(t *testing.T, findings []finding.Finding) string {
	t.Helper()

	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: "test"})
	report.AddFindings(findings)

	out, err := report.PrettyJSON()
	if err != nil {
		t.Fatalf("PrettyJSON: %v", err)
	}

	return out
}

// TestSARIFGroupPropertyRoundTrip: a grouped finding rendered through
// cqrs-lint's SARIF path carries its GroupID as a SARIF property, and the
// upstream reader restores it — consumer tooling can regroup SARIF output.
func TestSARIFGroupPropertyRoundTrip(t *testing.T) {
	t.Parallel()

	sarif := renderSARIF(t, groupedFixture())

	if !strings.Contains(string(sarif), "c019:Order") {
		t.Fatalf("SARIF output lost the group ID:\n%s", sarif)
	}

	findings, err := finding.FindingsFromSARIF(context.Background(), sarif)
	if err != nil {
		t.Fatalf("FindingsFromSARIF: %v", err)
	}

	grouped := 0
	for _, f := range findings {
		if f.GroupID == "c019:Order" {
			grouped++
		}
	}

	if grouped != 2 {
		t.Fatalf("expected 2 findings with c019:Order after round trip, got %d", grouped)
	}
}

// TestJSONAndSARIFByteIdenticalAcrossRuns pins that cqrs-lint's JSON and
// SARIF output are byte-identical for identical findings, including grouped
// ones — machine consumers (CI annotation parsers, dashboards) rely on
// stable ordering and encoding; map iteration or time-stamped output would
// break them.
func TestJSONAndSARIFByteIdenticalAcrossRuns(t *testing.T) {
	t.Parallel()

	json1 := renderJSON(t, groupedFixture())
	json2 := renderJSON(t, groupedFixture())
	if json1 != json2 {
		t.Fatalf("JSON output is not byte-identical across runs:\n%s\n---\n%s", json1, json2)
	}

	if !strings.Contains(json1, "groupId") && !strings.Contains(json1, "c019:Order") {
		t.Fatalf("JSON output lost the group information:\n%s", json1)
	}

	sarif1 := renderSARIF(t, groupedFixture())
	sarif2 := renderSARIF(t, groupedFixture())
	if !bytes.Equal(sarif1, sarif2) {
		t.Fatalf("SARIF output is not byte-identical across runs")
	}
}
