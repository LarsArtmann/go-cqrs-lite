package metaengine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// Diagnostic levels for plan output.
const (
	// DiagLevelScream indicates a configuration that will cause permanent data
	// loss. Source-of-truth data (event logs) on volatile engines with no
	// persistent alternative emit this level. The system's scream store treats
	// SCREAM-tier diagnostics as hard failures (New() refuses to start).
	DiagLevelScream   = "SCREAM"
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

// HasErrors returns true if any SCREAM-tier diagnostics exist. These indicate
// configurations that will cause permanent data loss (e.g., source-of-truth
// data on a volatile engine with no persistent alternative).
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Level == DiagLevelScream {
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

// String renders the assignment as a single line for EXPLAIN output.
func (a QueryAssignment) String() string {
	parts := []string{
		fmt.Sprintf("%s: %s/%s via %s (%s)",
			a.QueryName, a.ADT, a.ReadPattern, a.EngineName, a.Complexity),
	}

	if a.Cost.Volume > 0 {
		parts = append(parts, fmt.Sprintf("latency<%sms", humanize.Commaf(a.Cost.EstimatedLatencyMs)))
	}

	if a.IsPaginated {
		parts = append(parts, "[paginated]")
	}

	return strings.Join(parts, " ")
}

type PlanResult struct {
	Queries     []QueryAssignment
	Diagnostics Diagnostics
	// LayoutPlans holds the auto-generated LayoutPlans for collections that
	// used FilterOnField/SortOnField with a LayoutPlanner engine.
	// Populated during Plan(); empty when no auto-layout was applied.
	LayoutPlans []LayoutPlan
	// RuleTrace records which rules fired during planning and a brief reason.
	// Each rule that enriches the PlanResult appends an entry. This makes
	// EXPLAIN output debuggable — consumers can see WHY the planner made
	// each decision, not just WHAT it decided.
	RuleTrace []RuleTraceEntry

	// Version is the schema version of this plan. Incremented when the plan
	// is recomputed. Consumers can compare versions to detect drift without
	// a full re-plan.
	Version int

	// ComputedAt is the timestamp when this plan was created by Plan().
	ComputedAt time.Time
}

// RuleTraceEntry records a single rule's decision during planning.
type RuleTraceEntry struct {
	Rule   string // rule name (e.g., "schema-enforcement")
	Query  string // query name (or "*" for global rules)
	Reason string // brief human-readable explanation
	Layout StorageLayout
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

	if len(p.RuleTrace) > 0 {
		b.WriteString("\n--- Rule Trace ---\n")

		for _, rt := range p.RuleTrace {
			layout := ""
			if rt.Layout != "" {
				layout = fmt.Sprintf(" [layout=%s]", rt.Layout)
			}

			fmt.Fprintf(&b, "  %s: %s — %s%s\n", rt.Rule, rt.Query, rt.Reason, layout)
		}
	}

	return b.String()
}

func checkWriteAmplification(queries map[string]queryMeta, budget int) Diagnostics {
	eventCount := make(map[string]int)

	for _, q := range queries {
		seen := make(map[string]bool)
		for eventType := range q.QueryFoldByEvent() {
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
