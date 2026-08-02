package security

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// piiFieldNames lists field names (case-insensitive) that indicate personally
// identifiable information. When these appear in event payload structs without
// encryption middleware on the bus, the data is persisted in cleartext.
//
//nolint:gochecknoglobals // read-only denylist
var piiFieldNames = map[string]bool{
	"email":       true,
	"password":    true,
	"passwd":      true,
	"secret":      true,
	"ssn":         true,
	"creditcard":  true,
	"cardnumber":  true,
	"cvv":         true,
	"phone":       true,
	"address":     true,
	"zipcode":     true,
	"firstname":   true,
	"lastname":    true,
	"fullname":    true,
	"birthdate":   true,
	"dateofbirth": true,
}

// S011: PII in event payloads without encryption.
//
// Detects event payload structs with PII-sensitive fields (email, password,
// ssn, credit card, etc.) when the event bus does NOT use encryption
// middleware. Without encryption, PII is persisted in cleartext to the event
// store — a compliance violation (GDPR, HIPAA, PCI-DSS).
//
//nolint:ireturn // factory returns public interface
func NewS011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S011-pii-without-encryption",
		func(_ context.Context) ([]finding.Finding, error) {
			busEncrypted := busHasEncryption(ctx)
			if busEncrypted {
				return nil, nil
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !lintutil.FileImportsCQRS(gf.AST) {
					continue
				}

				for _, decl := range gf.AST.Decls {
					gd, ok := decl.(*ast.GenDecl)
					if !ok || gd.Tok != token.TYPE {
						continue
					}

					for _, spec := range gd.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						structType, ok := ts.Type.(*ast.StructType)
						if !ok {
							continue
						}

						if !lintutil.IsEventPayloadName(ts.Name.Name) {
							continue
						}

						for _, field := range structType.Fields.List {
							for _, name := range field.Names {
								if isPIIField(name.Name) {
									pos := ctx.Fset.Position(name.Pos())

									f, err := finding.NewBuilder(
										"S011", toolName,
										fmt.Sprintf(
											"PII field %q in event payload %q without encryption — "+
												"data persisted in cleartext to event store",
											name.Name, ts.Name.Name,
										),
										finding.SeverityWarning,
										finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
									).
										WithCategory(finding.CategorySecurity).
										WithConfidence(finding.ConfidenceMedium).
										WithFixStrategy(finding.FixStrategySuggest).
										WithSuggestion(
											"Add encryption.EncryptMiddleware to the event bus, or remove PII " +
												"from event payloads (store it in a separate encrypted read model)",
										).
										WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
										Build()
									lintutil.AppendBuild(&findings, f, err)
								}
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// busHasEncryption returns true if any file in the project configures
// encryption middleware on the event bus.
func busHasEncryption(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		encrypted := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			callStr := analyzer.ExprString(call.Fun)
			if strings.Contains(callStr, "EncryptMiddleware") ||
				strings.Contains(callStr, "encryption.New") ||
				strings.Contains(callStr, "NewCodec") {
				encrypted = true
				return false
			}

			return true
		})

		if encrypted {
			return true
		}
	}

	return false
}

func isPIIField(name string) bool {
	return piiFieldNames[strings.ToLower(name)]
}
