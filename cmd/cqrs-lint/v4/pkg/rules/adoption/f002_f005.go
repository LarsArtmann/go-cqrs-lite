package adoption

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F002 detects projects with 3+ event types that do not call catalog.NewBuilder.
// The catalog module generates AsyncAPI, OpenAPI, D2, and EventCatalog docs.
//
//nolint:ireturn // factory returns public interface
func NewF002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F002-no-catalog-documentation",
		func(_ context.Context) ([]finding.Finding, error) {
			if eventCount(ctx) < 3 {
				return nil, nil
			}

			// catalog/simple.New is a thin, canonical wrapper around
			// catalog.NewBuilder; consumers declaring their whole vocabulary
			// through the builder are fully documented.
			if projectHasCall(ctx, "catalog", "NewBuilder") ||
				projectHasCallInPath(ctx, "go-cqrs-lite/catalog/simple", "New") {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(
				ctx,
				"F002",
				"Project has "+itoa(eventCount(ctx))+
					" event types but no catalog.NewBuilder — event documentation "+
					"is not generated",
				"Import catalog and call catalog.NewBuilder() to generate AsyncAPI, "+
					"OpenAPI, D2 diagrams, and EventCatalog documentation from your "+
					"event types.",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// F005 detects projects with evolving event schemas (a schema version beyond
// the first, or 5+ event types) that do not use schema.NewUpcaster for version
// migration.
//
//nolint:ireturn // factory returns public interface
func NewF005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F005-no-schema-upcasters",
		func(_ context.Context) ([]finding.Finding, error) {
			if projectHasCall(ctx, "schema", "NewUpcaster") {
				return nil, nil
			}

			// Signal 1: a schema version beyond the first (schemas are evolving).
			hasVersioning := hasSchemaVersionBeyondFirst(ctx)

			// Signal 2: many event types suggest a complex, evolving domain.
			if eventCount(ctx) < 5 && !hasVersioning {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			msg := "Project has evolving event schemas but no schema.NewUpcaster — " +
				"schema migrations on load are not handled"

			if hasVersioning {
				msg = "Project uses event.WithSchemaVersion but has no schema.NewUpcaster — " +
					"versioned events need upcasters for backward-compatible migration"
			}

			return singleInfoFinding(
				ctx, "F005", msg,
				"Use schema.NewUpcaster(eventType, fromVersion, migrateFn) to transform "+
					"old event payloads to the current schema on load. Register upcasters "+
					"via event.DecorateStore(store, nil, schema.UpcastSourceTransform(...)).",
				pos, finding.ConfidenceLow,
			), nil
		},
	)
}

// itoa is a thin wrapper to avoid importing strconv in every rule file.
func itoa(n int) string { return strconvItoa(n) }

// hasSchemaVersionBeyondFirst reports whether any non-test file stamps a
// schema version other than 1 through event.WithSchemaVersion. Stamping the
// first version is what D013 asks for and what cqrs-gen derives by default;
// until a second version exists there is nothing to upcast from.
func hasSchemaVersionBeyondFirst(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok || sel.Sel.Name != "WithSchemaVersion" {
				return true
			}

			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "event" {
				return true
			}

			if !stampsFirstVersion(call) {
				found = true

				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// stampsFirstVersion reports whether the call's only argument is the literal
// 1, bare or wrapped in a conversion such as event.SchemaVersion(1). Anything
// else, a variable included, may name a later version.
func stampsFirstVersion(call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}

	arg := call.Args[0]

	if conversion, ok := arg.(*ast.CallExpr); ok && len(conversion.Args) == 1 {
		arg = conversion.Args[0]
	}

	lit, ok := arg.(*ast.BasicLit)

	return ok && lit.Kind == token.INT && lit.Value == "1"
}
