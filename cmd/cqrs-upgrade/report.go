package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"

	cqrsanalyzer "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	cqrsversion "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
)

// moduleReport is the per-module result of one upgrade pipeline run. It
// feeds both the human-readable output and the --json document; the JSON
// field order below is the wire contract.
type moduleReport struct {
	Dir          string        `json:"dir"`
	NoPins       bool          `json:"noPins,omitempty"`
	Error        string        `json:"error,omitempty"`
	Bumps        []bumpJSON    `json:"bumps,omitempty"`
	Deprecations []findingJSON `json:"deprecations,omitempty"`
}

// bumpJSON is the stable wire shape of one planned version change.
type bumpJSON struct {
	Module  string `json:"module"`
	From    string `json:"from"`
	To      string `json:"to"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// findingJSON is the stable wire shape of one v5-removal finding.
type findingJSON struct {
	Position string `json:"position"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

// toJSON converts a bump for the wire, recomputing the status string.
func (b bump) toJSON() bumpJSON {
	out := bumpJSON{Module: b.Module, From: b.From, To: b.To}

	switch {
	case b.resolveErr != nil:
		out.Status = "resolve-error"
		out.Details = b.resolveErr.Error()
	case b.held:
		out.Status = "held"
	case b.upToDate:
		out.Status = "up-to-date"
	default:
		out.Status = "bump"
	}

	return out
}

// deprecationFindings runs the cqrs-lint V007 detector (v5-removed API
// usage) in-process over dir. Best-effort: a load failure yields no findings
// (the human output prints the reason), never an upgrade abort.
func deprecationFindings(dir string) []findingJSON {
	ctx, err := cqrsanalyzer.BuildContext(dir)
	if err != nil {
		return nil
	}

	findings, detErr := cqrsversion.NewV007Detector(ctx).Detect(context.Background())
	if detErr != nil {
		return nil
	}

	out := make([]findingJSON, 0, len(findings))

	for _, f := range findings {
		out = append(out, findingJSON{
			Position: f.Position.String(),
			Rule:     f.Rule,
			Message:  strings.TrimSpace(f.Message),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}

		return out[i].Rule < out[j].Rule
	})

	return out
}

// printDeprecations prints the v5-removal findings for one module.
func printDeprecations(w io.Writer, findings []findingJSON) {
	if len(findings) == 0 {
		fmt.Fprintln(w, "deprecation report: no v5-removed API usage detected")

		return
	}

	fmt.Fprintf(
		w,
		"deprecation report: %d finding(s) — APIs removed at go-cqrs-lite v5:\n",
		len(findings),
	)

	for _, f := range findings {
		fmt.Fprintf(w, "  %s %s [%s]\n", f.Position, f.Message, f.Rule)
	}
}
