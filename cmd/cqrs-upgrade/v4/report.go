package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
// 2 = suggestions always-present (advisory report-only section).
const moduleSchemaVersion = 2

type moduleJSON struct {
	SchemaVersion        int           `json:"schemaVersion"`
	Dir                  string        `json:"dir"`
	NoPins               bool          `json:"noPins,omitempty"`
	Error                string        `json:"error,omitempty"`
	Bumps                []bumpJSON    `json:"bumps"`
	Deprecations         []findingJSON `json:"deprecations"`
	Suggestions          []findingJSON `json:"suggestions"`
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

// scanFindings is the pipeline-facing wrapper: deprecations, suggestions,
// and the scan error in one call.
func scanFindings(dir string) ([]findingJSON, []findingJSON, error) {
	deps, suggs, _, err := scanFindingsAnalyzed(dir)

	return deps, suggs, err
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

	return out, suggestionFindings(dir), analyzed, nil
}

// printSuggestions prints the advisory migration hints. Always rendered
// (an empty section is information too): unlike deprecations these never
// gate anything — they coach toward the systemscenario harness while a
// suite migrates.
func printSuggestions(w io.Writer, suggestions []findingJSON) {
	printFindings(w, suggestions,
		"suggestions: none (advisory; never auto-applied)",
		"suggestions: %d hint(s) — advisory migration hints, never auto-applied:\n")
}

// printDeprecations prints the v5-removal findings for one module.
func printDeprecations(w io.Writer, findings []findingJSON) {
	printFindings(w, findings,
		"deprecation report: no v5-removed API usage detected",
		"deprecation report: %d finding(s) — APIs removed at go-cqrs-lite v5:\n")
}

// printFindings renders one findings section: the emptySummary line when
// the slice is empty, else the count header (header, one %d) followed by
// one line per finding — the shared body of the two report sections.
func printFindings(w io.Writer, findings []findingJSON, emptySummary, header string) {
	if len(findings) == 0 {
		fmt.Fprintln(w, emptySummary)

		return
	}

	fmt.Fprintf(w, header, len(findings))

	for _, f := range findings {
		fmt.Fprintf(w, "  %s %s [%s]\n", f.Position, f.Message, f.Rule)
	}
}

// printReport renders one module's human-readable output.
func printReport(r moduleReport) {
	switch {
	case r.Error != "":
		fmt.Printf("error: %s\n", r.Error)
	case r.NoPins:
		fmt.Printf("no direct go-cqrs-lite pins found in %s\n", filepath.Join(r.Dir, "go.mod"))
		if r.ScanErr != nil {
			fmt.Printf("deprecation report: scan failed — %v (v5-readiness unknown)\n", r.ScanErr)
		} else {
			printDeprecations(os.Stdout, r.Deprecations)
		}
	default:
		printBumps(r.Bumps)
		if r.ScanErr != nil {
			fmt.Printf("deprecation report: scan failed — %v (v5-readiness unknown)\n", r.ScanErr)
		} else {
			printDeprecations(os.Stdout, r.Deprecations)
		}
	}

	printSuggestions(os.Stdout, r.Suggestions)
}

// printBumps renders the plan table for both dry-run and apply runs.
func printBumps(bumps []bump) {
	fmt.Println(strings.ReplaceAll(formatBumps(bumps), "\t", "  "))
}

// emitJSON writes the full report as one deterministic JSON document.
// Struct-field order (no maps) keeps the output byte-stable for CI diffs.
func emitJSON(w io.Writer, reports []moduleReport) error {
	wire := make([]moduleJSON, 0, len(reports))

	for _, r := range reports {
		wire = append(wire, r.toJSON())
	}

	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))

	return json.MarshalEncode(enc, wire)
}
