package api

import (
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// a go-cqrs-lite module. Used by project-level rules to gate detection.
func projectImportsCQRS(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if lintutil.FileImportsCQRS(gf.AST) {
			return true
		}
	}

	return false
}
