package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	cqrsanalyzer "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	cqrsversion "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
)

// errPackageLoad marks a failed package load during the deprecation scan.
var errPackageLoad = errors.New("package load failed")

// moduleReport is the per-module result of one upgrade pipeline run. It
// feeds the human output directly; --json marshals it through moduleJSON.
type moduleReport struct {
	Dir          string
	NoPins       bool
	Error        string
	Bumps        []bump
	Deprecations []findingJSON
	// Suggestions are advisory migration hints (report-only, never
	// applied, ignored by --strict): today the eventually-loop →
	// systemscenario.ThenQuery matcher. Deprecations block v5; suggestions
	// coach toward the harness while migrating.
	Suggestions []findingJSON
	// ScanErr records a failed deprecation scan (load/detect failure). A
	// failed scan is NOT the same as a clean one: v5-readiness is unknown.
	ScanErr error
}

// moduleJSON is the stable wire shape of moduleReport; the field order
// below is the --json contract. `bumps` and `deprecations` are ALWAYS
// present (empty arrays when clean) so consumers never rely on
// key-absence folklore to tell "clean" from "old CLI" or a failed scan.
// `schemaVersion` (per module — the wire is a bare top-level array, so a
// wrapper object would break jq consumers) is the wire-format generation:
// 1 = bumps always-present + deprecations always-present + NoPins modules
// scanned. Old CLIs omit it; bump it on any contract change.
const moduleSchemaVersion = 1

type moduleJSON struct {
	SchemaVersion        int           `json:"schemaVersion"`
	Dir                  string        `json:"dir"`
	NoPins               bool          `json:"noPins,omitempty"`
	Error                string        `json:"error,omitempty"`
	Bumps                []bumpJSON    `json:"bumps"`
	Deprecations         []findingJSON `json:"deprecations"`
	DeprecationScanError string        `json:"deprecationScanError,omitempty"`
}

// toJSON converts the report for the wire, including every bump with its
// recomputed status.
func (r moduleReport) toJSON() moduleJSON {
	deprecations := r.Deprecations
	if deprecations == nil {
		deprecations = []findingJSON{} // emit [], never null
	}

	suggestions := r.Suggestions
	if suggestions == nil {
		suggestions = []findingJSON{} // emit [], never null
	}

	out := moduleJSON{
		SchemaVersion:        moduleSchemaVersion,
		Dir:                  r.Dir,
		NoPins:               r.NoPins,
		Error:                r.Error,
		Bumps:                make([]bumpJSON, 0, len(r.Bumps)), // emit [], never null
		Deprecations:         deprecations,
		Suggestions:          suggestions,
		DeprecationScanError: "",
	}

	if r.ScanErr != nil {
		out.DeprecationScanError = r.ScanErr.Error()
	}

	for _, b := range r.Bumps {
		out.Bumps = append(out.Bumps, b.toJSON())
	}

	return out
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
// usage) in-process over dir. A load or detect failure returns an error —
// callers must not confuse a failed scan with a clean one (a silent
// false-green is exactly how a strict gate rots). The upgrade pipeline
// records the error in the report instead of aborting; the strict gate
// fails on it.
func deprecationFindings(dir string) ([]findingJSON, error) {
	findings, _, _, err := scanFindingsAnalyzed(dir)

	return findings, err
}

// scanFindingsAnalyzed runs the v5-removal detector AND the advisory
// suggestion matcher over one BuildContext (one package load, two
// detectors) and also returns the number of Go files analyzed, so callers
// can assert the scan measured something (a zero-file scan proves nothing
// — the 02-47 lesson).
func scanFindingsAnalyzed(dir string) ([]findingJSON, []findingJSON, int, error) {
	ctx, err := cqrsanalyzer.BuildContext(dir)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("build context: %w", err)
	}

	analyzed := len(ctx.GoFiles)

	if len(ctx.LoadErrors) > 0 {
		first := ctx.LoadErrors[0]
		detail := ""
		if len(first.Errors) > 0 {
			detail = first.Errors[0]
		}

		return nil, nil, analyzed, fmt.Errorf("%w: %s: %s", errPackageLoad, first.Module, detail)
	}

	findings, detErr := cqrsversion.NewV007Detector(ctx).Detect(context.Background())
	if detErr != nil {
		return nil, nil, analyzed, fmt.Errorf("detect: %w", detErr)
	}

	out := make([]findingJSON, 0, len(findings))

	for _, f := range findings {
		out = append(out, findingJSON{
			Position: f.Position.String(),
			Rule:     string(f.Rule),
			Message:  strings.TrimSpace(f.Message),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}

		return out[i].Rule < out[j].Rule
	})

	return out, suggestionFindings(ctx), analyzed, nil
}

// printSuggestions prints the advisory migration hints. Always rendered
// (an empty section is information too): unlike deprecations these never
// gate anything — they coach toward the systemscenario harness while a
// suite migrates.
func printSuggestions(w io.Writer, suggestions []findingJSON) {
	if len(suggestions) == 0 {
		fmt.Fprintln(w, "suggestions: none (advisory; never auto-applied)")

		return
	}

	fmt.Fprintf(
		w,
		"suggestions: %d hint(s) — advisory migration hints, never auto-applied:\n",
		len(suggestions),
	)

	for _, f := range suggestions {
		fmt.Fprintf(w, "  %s %s [%s]\n", f.Position, f.Message, f.Rule)
	}
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
