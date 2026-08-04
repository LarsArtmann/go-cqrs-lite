package main

import (
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// aggregateMetadataKey is the Finding.Metadata key used to store the inferred
// aggregate/domain name for a finding. Detectors that know the aggregate at
// detection time can stamp this directly; enrichWithAggregate fills in the
// rest via file-level inference from the CQRSRegistry.
const aggregateMetadataKey = "aggregate"

// aggregateFromEventType extracts the aggregate name from a CQRS event type
// string. Event types follow the "aggregate.verb" convention
// (e.g. "user.created" → "User"). Event types without a dot use the full
// string. The result is capitalized to match Go type naming conventions.
func aggregateFromEventType(eventType string) string {
	prefix := eventType
	if idx := strings.Index(eventType, "."); idx > 0 {
		prefix = eventType[:idx]
	}

	return capitalizeFirst(prefix)
}

// aggregateFromStateType extracts the aggregate name from a decider/fold state
// type name. State types typically embed the aggregate name with a "State"
// suffix (e.g. "UserState" → "User"). Also strips "Aggregate" if present
// (e.g. "OrderAggregateState" → "Order").
func aggregateFromStateType(stateType string) string {
	name := stateType
	name = strings.TrimSuffix(name, "State")
	name = strings.TrimSuffix(name, "Aggregate")

	if name == "" {
		return ""
	}

	return capitalizeFirst(name)
}

// capitalizeFirst returns s with its first character uppercased.
// ASCII-only — aggregate names are Go identifiers.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

// buildFileAggregateMap builds a map from file path to the set of aggregate
// names associated with that file. It draws from three registry sources:
//   - EventTypesEmitted: "user.created" → "User" (strongest signal)
//   - Deciders: StateType "UserState" → "User"
//   - Folds: StateType "UserState" → "User"
//
// A single file may map to multiple aggregates (e.g. a shared events file).
// The returned slices are sorted alphabetically for deterministic output.
func buildFileAggregateMap(reg *analyzer.CQRSRegistry) map[string][]string {
	fileAggSets := make(map[string]map[string]bool)

	addAgg := func(file, agg string) {
		if file == "" || agg == "" {
			return
		}

		if fileAggSets[file] == nil {
			fileAggSets[file] = make(map[string]bool)
		}

		fileAggSets[file][agg] = true
	}

	for eventType, emission := range reg.EventTypesEmitted {
		addAgg(emission.File, aggregateFromEventType(eventType))
	}

	for _, d := range reg.Deciders {
		addAgg(d.File, aggregateFromStateType(d.StateType))
	}

	for _, f := range reg.Folds {
		addAgg(f.File, aggregateFromStateType(f.StateType))
	}

	result := make(map[string][]string, len(fileAggSets))

	for file, aggs := range fileAggSets {
		list := make([]string, 0, len(aggs))
		for agg := range aggs {
			list = append(list, agg)
		}

		sort.Strings(list)
		result[file] = list
	}

	return result
}

// enrichWithAggregate stamps Finding.Metadata["aggregate"] on each finding
// using a file-level aggregate map derived from the CQRSRegistry. Findings
// whose detectors already set the metadata key are left untouched.
//
// This is the enrichment counterpart to enrichWithDocURLs — it runs in
// filterFindings so all output formats (text, JSON, SARIF) carry the
// aggregate context.
func enrichWithAggregate(
	findings []finding.Finding,
	actx *analyzer.AnalysisContext,
) []finding.Finding {
	fileAggs := buildFileAggregateMap(actx.Registry)

	for i := range findings {
		if findings[i].Metadata == nil {
			findings[i].Metadata = make(map[string]string)
		}

		if _, ok := findings[i].Metadata[aggregateMetadataKey]; ok {
			continue
		}

		aggs, ok := fileAggs[string(findings[i].Position.File)]
		if !ok || len(aggs) == 0 {
			continue
		}

		findings[i].Metadata[aggregateMetadataKey] = strings.Join(aggs, ", ")
	}

	return findings
}
