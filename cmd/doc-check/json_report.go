package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
)

// jsonSummary is the --json wire shape: deterministic field order, arrays
// instead of counts, so CI consumers can annotate per finding.
type jsonSummary struct {
	Valid       bool        `json:"valid"`
	Files       int         `json:"files"`
	References  int         `json:"references"`
	Packages    int         `json:"packages"`
	Broken      []brokenRef `json:"broken"`
	NavIssues   []navIssue  `json:"navIssues"`
	ArityIssues []navIssue  `json:"arityIssues"`
	Warnings    []string    `json:"warnings"`
	Ambiguities []string    `json:"ambiguities"`
}

// emitJSON prints the machine-readable summary to stdout. Human logs stay on
// stderr, so the JSON is the only stdout content.
func emitJSON(
	files, totalRefs int,
	brokenRefs []brokenRef,
	warnings, ambiguities []string,
	navIssues, arityIssues []navIssue,
	res *resolver,
) error {
	summary := jsonSummary{
		Valid: len(brokenRefs) == 0 && len(warnings) == 0 &&
			len(navIssues) == 0 && len(arityIssues) == 0,
		Files:       files,
		References:  totalRefs,
		Packages:    len(res.clauses),
		Broken:      brokenRefs,
		NavIssues:   navIssues,
		ArityIssues: arityIssues,
		Warnings:    warnings,
		Ambiguities: ambiguities,
	}

	if summary.Broken == nil {
		summary.Broken = []brokenRef{}
	}

	if summary.NavIssues == nil {
		summary.NavIssues = []navIssue{}
	}

	if summary.ArityIssues == nil {
		summary.ArityIssues = []navIssue{}
	}

	if summary.Warnings == nil {
		summary.Warnings = []string{}
	}

	if summary.Ambiguities == nil {
		summary.Ambiguities = []string{}
	}

	encoded, err := json.Marshal(summary, json.Deterministic(true))
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(encoded))

	return nil
}
