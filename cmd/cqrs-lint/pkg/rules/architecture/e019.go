package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// E019: a declared data product serves outputs without a data contract.
// A data product's OUTPUT port is its contract with consumers — an output
// without an explicit DataContract renders in the catalog but gives
// downstream consumers nothing to code against (no schema file, no version
// pin, nothing for the federation hub to diff). Advisory by design:
// contractless outputs are legal, just incomplete — the same
// contract-completeness tier as E018's provider checks, docs-side twin of
// the DataProduct/DataContract declaration story (recipes §2.41).
//
//nolint:ireturn // factory returns public interface
func NewE019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E019-data-product-without-contract",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, dp := range ctx.Registry.DataProducts {
				var message string

				switch {
				case dp.OutputCount == 0:
					message = fmt.Sprintf(
						"data product %q declares no output ports — a product without outputs documents inputs only",
						dp.Name,
					)
				case dp.Contracted < dp.OutputCount:
					message = fmt.Sprintf(
						"data product %q serves %d output(s) but only %d carry a DataContract — consumers get no schema or version pin for the rest",
						dp.Name,
						dp.OutputCount,
						dp.Contracted,
					)
				default:
					continue
				}

				f, err := e019Finding(ctx, dp, message)
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

func e019Finding(
	ctx *analyzer.AnalysisContext,
	dp analyzer.DataProductInfo,
	message string,
) (finding.Finding, error) {
	file := dp.File
	if file == "" {
		file = ctx.ProjectRoot + "/go.mod"
	}

	return findingTemplate.Builder(
		"E019",
		message,
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(file), dp.Line, dp.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion(
			"Attach a DataContract (path + name) to every output, or drop the outputs field while the product is input-only; " +
				"recipes.md §2.41 walks the full data-product declaration",
		).
		WithSnippet(ctx.SourceLine(dp.File, dp.Line)).
		Build()
}
