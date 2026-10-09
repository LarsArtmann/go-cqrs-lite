package main

import (
	"encoding/json/v2"
	"fmt"
)

// renderScorecardSARIF emits the scorecard as a SARIF 2.1.0 report for CI
// integration (GitHub Code Scanning, Azure DevOps). Missing modules appear as
// info-level results; the adoption summary lives in run.properties so CI
// scripts can extract coverage metrics without parsing human-readable output.
func renderScorecardSARIF(result ScorecardResult) (string, error) {
	props := &sarifProperties{
		CoveragePercent: result.Summary.CoveragePercent,
		Grade:           result.Summary.Grade,
		UsedCount:       result.Summary.UsedCount,
		RelevantTotal:   result.Summary.RelevantTotal,
		IrrelevantCount: result.Summary.IrrelevantCount,
		WaivedCount:     result.Summary.WaivedCount,
		ModernityGrade:  result.Summary.ModernityGrade,
	}
	if result.Deprecated != nil {
		props.RemovedAPIUses = result.Deprecated.RemovedAPIUses
		props.DeprecatedTransportUses = result.Deprecated.DeprecatedTransport
		props.StackPresetUses = result.Deprecated.StackPresetUses
	}
	if result.Metaengine != nil {
		detected := result.Metaengine.Detected
		props.MetaengineDetected = &detected
		if len(result.Metaengine.Engines) > 0 {
			props.MetaengineEngines = result.Metaengine.Engines
		}
		adopted := result.Metaengine.PushdownAdopted
		props.MetaenginePushdownAdopted = &adopted
	}

	report := sarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "cqrs-lint-scorecard",
					Version:        resolvedVersion(),
					InformationURI: "https://github.com/larsartmann/go-cqrs-lite/tree/main/cmd/cqrs-lint",
					Rules: []sarifRule{
						{
							ID:   "scorecard/missing-module",
							Name: "MissingModule",
							ShortDescription: sarifMessage{
								Text: "Relevant go-cqrs-lite module not adopted",
							},
							FullDescription: sarifMessage{
								Text: "This module is relevant to the project's feature profile but is not imported.",
							},
							DefaultConfig: sarifConfig{Level: "info"},
						},
					},
				},
			},
			Properties: props,
		}},
	}

	// Build logicalLocations from all scored modules (used + missing) so CI
	// tools can reference modules by logical name without parsing physical
	// paths. Each module gets an index; missing-module results reference
	// their index for programmatic consumption.
	moduleIndex := make(map[string]int)
	var logicalLocations []sarifLogicalLocation
	for _, m := range append(append([]ScorecardModule{}, result.Used...), result.Missing...) {
		if _, exists := moduleIndex[m.Key]; exists {
			continue
		}
		moduleIndex[m.Key] = len(logicalLocations)
		logicalLocations = append(logicalLocations, sarifLogicalLocation{
			Name:               m.DisplayName,
			FullyQualifiedName: m.Key,
			Kind:               "module",
		})
	}
	report.Runs[0].LogicalLocations = logicalLocations

	for _, m := range result.Missing {
		msg := fmt.Sprintf("Missing module: %s (%s)", m.DisplayName, m.Category)
		if m.Suggestion != "" {
			msg += " — " + m.Suggestion
		}

		loc := sarifLocation{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: "go.mod"},
			},
		}
		if idx, ok := moduleIndex[m.Key]; ok {
			loc.LogicalLocations = []sarifLogicalLocationRef{{Index: idx}}
		}

		report.Runs[0].Results = append(report.Runs[0].Results, sarifResult{
			RuleID:    "scorecard/missing-module",
			Level:     "info",
			Message:   sarifMessage{Text: msg},
			Locations: []sarifLocation{loc},
		})
	}

	data, err := json.Marshal(report, json.Deterministic(true))
	if err != nil {
		return "", fmt.Errorf("marshal scorecard SARIF: %w", err)
	}
	return string(data) + "\n", nil
}

// SARIF 2.1.0 structural types for scorecard output.

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	ShortDescription sarifMessage `json:"shortDescription"`
	FullDescription  sarifMessage `json:"fullDescription"`
	DefaultConfig    sarifConfig  `json:"defaultConfiguration"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation     `json:"physicalLocation"`
	LogicalLocations []sarifLogicalLocationRef `json:"logicalLocations,omitempty"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifRun struct {
	Tool             sarifTool              `json:"tool"`
	Properties       *sarifProperties       `json:"properties,omitempty"`
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
	Results          []sarifResult          `json:"results,omitempty"`
}

// sarifProperties is run.properties as a fixed-order struct, NOT a map:
// encoding/json/v2 emits map keys in iteration order (v1 sorted), so a map
// here made the SARIF byte-nondeterministic for CI consumers that diff
// reports (same class as the doctor severityOverrides fix, 2026-09-08).
type sarifProperties struct {
	CoveragePercent           int      `json:"coveragePercent"`
	Grade                     string   `json:"grade"`
	UsedCount                 int      `json:"usedCount"`
	RelevantTotal             int      `json:"relevantTotal"`
	IrrelevantCount           int      `json:"irrelevantCount"`
	WaivedCount               int      `json:"waivedCount,omitempty"`
	ModernityGrade            string   `json:"modernityGrade,omitempty"`
	RemovedAPIUses            int      `json:"removedApiUses,omitempty"`
	DeprecatedTransportUses   int      `json:"deprecatedTransportUses,omitempty"`
	StackPresetUses           int      `json:"stackPresetUses,omitempty"`
	MetaengineDetected        *bool    `json:"metaengineDetected,omitempty"`
	MetaengineEngines         []string `json:"metaengineEngines,omitempty"`
	MetaenginePushdownAdopted *bool    `json:"metaenginePushdownAdopted,omitempty"`
}

// sarifLogicalLocation describes a logical component of the analyzed codebase
// (module, package, namespace). Stored at the run level in
// run.logicalLocations[] and referenced by index from result locations.
type sarifLogicalLocation struct {
	Name               string `json:"name,omitempty"`
	FullyQualifiedName string `json:"fullyQualifiedName,omitempty"`
	Kind               string `json:"kind,omitempty"`
}

// sarifLogicalLocationRef references a run-level logicalLocation by its array
// index.
type sarifLogicalLocationRef struct {
	Index int `json:"index"`
}

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}
