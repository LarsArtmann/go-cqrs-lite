package metaengine

import (
	"fmt"
	"sort"
	"strings"
)

// Diagnostic levels for plan output.
const (
	DiagLevelWarn     = "WARN"
	DiagLevelDegraded = "DEGRADED"
	DiagLevelInfo     = "INFO"
)

type Diagnostic struct {
	Level   string
	Query   string
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s", d.Level, d.Query, d.Message)
}

type Diagnostics []Diagnostic

func (d Diagnostics) HasWarnings() bool {
	for _, diag := range d {
		if diag.Level == DiagLevelWarn || diag.Level == DiagLevelDegraded {
			return true
		}
	}

	return false
}

// QueryAssignment shows the full plan for one query: engine, ADT, read pattern, cost.
type QueryAssignment struct {
	QueryName   string
	ADT         ADT
	EngineName  string
	Complexity  Complexity
	ReadPattern ReadPattern
	IsPaginated bool
	Cost        CostEstimate
	Diagnostics []Diagnostic
}

func (a QueryAssignment) String() string {
	parts := []string{
		fmt.Sprintf("%s: %s/%s via %s (%s)",
			a.QueryName, a.ADT, a.ReadPattern, a.EngineName, a.Complexity),
	}

	if a.Cost.Volume > 0 {
		parts = append(parts, fmt.Sprintf("latency<%.3fms", a.Cost.EstimatedLatencyMs))
	}

	if a.IsPaginated {
		parts = append(parts, "[paginated]")
	}

	return strings.Join(parts, " ")
}

type PlanResult struct {
	Queries     []QueryAssignment
	Diagnostics Diagnostics
}

func (p PlanResult) Report() string {
	var b strings.Builder
	b.WriteString("=== Meta-Engine Plan ===\n\n")

	b.WriteString("--- Queries ---\n")

	for _, a := range p.Queries {
		fmt.Fprintf(&b, "  %s\n", a)

		for _, d := range a.Diagnostics {
			fmt.Fprintf(&b, "    %s\n", d)
		}
	}

	if len(p.Diagnostics) > 0 {
		b.WriteString("\n--- Global Diagnostics ---\n")

		for _, d := range p.Diagnostics {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}

	return b.String()
}

func checkWriteAmplification(queries map[string]queryRuntime, budget int) Diagnostics {
	eventCount := make(map[string]int)

	for _, q := range queries {
		seen := make(map[string]bool)
		for eventType := range q.foldByEvent {
			if !seen[eventType] {
				seen[eventType] = true
				eventCount[eventType]++
			}
		}
	}

	var diags Diagnostics

	for evt, count := range eventCount {
		if count > budget {
			diags = append(diags, Diagnostic{
				Level: DiagLevelWarn,
				Query: "*",
				Message: fmt.Sprintf(
					"event %s updates %d projections — exceeds write amplification budget %d",
					evt, count, budget,
				),
			})
		}
	}

	sort.Slice(diags, func(i, j int) bool {
		return diags[i].Message < diags[j].Message
	})

	return diags
}
