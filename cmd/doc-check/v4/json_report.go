package main

import (
	"encoding/json/v2"
	"fmt"
	"io"
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
	// ModPathIssues carries the dead-citation class (C27): docs citing
	// deleted module paths as live guidance. Same navIssue shape as the nav
	// and arity classes so CI annotations render uniformly.
	ModPathIssues []navIssue `json:"modPathIssues"`
	Warnings      []string   `json:"warnings"`
	Ambiguities   []string   `json:"ambiguities"`
}

// emitJSON prints the machine-readable summary. Human logs stay on stderr,
// so the JSON is the only stdout content.
func emitJSON(
	w io.Writer,
	files, totalRefs int,
	brokenRefs []brokenRef,
	warnings, ambiguities []string,
	navIssues, arityIssues, modPathIssues []navIssue,
	res *resolver,
) error {
	summary := jsonSummary{
		Valid: len(brokenRefs) == 0 && len(warnings) == 0 &&
			len(navIssues) == 0 && len(arityIssues) == 0 && len(modPathIssues) == 0,
		Files:         files,
		References:    totalRefs,
		Packages:      len(res.clauses),
		Broken:        brokenRefs,
		NavIssues:     navIssues,
		ArityIssues:   arityIssues,
		ModPathIssues: modPathIssues,
		Warnings:      warnings,
		Ambiguities:   ambiguities,
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

	if summary.ModPathIssues == nil {
		summary.ModPathIssues = []navIssue{}
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

	fmt.Fprintln(w, string(encoded))

	return nil
}
