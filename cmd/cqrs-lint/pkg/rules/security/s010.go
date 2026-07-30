package security

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// S010: Encryption/signing mismatch.
// If the event bus has encryption middleware but the store doesn't (or vice
// versa), events are stored in cleartext but transmitted encrypted (or vice
// versa). This is security theater: the data is exposed at the storage layer.
//
//nolint:ireturn // factory returns public interface
func NewS010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S010-encryption-signing-mismatch",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			busEncrypted := false
			busSigned := false
			storeWrapped := false
			var triggerPos token.Position

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					callStr := analyzer.ExprString(call.Fun)

					if strings.Contains(callStr, "EncryptMiddleware") ||
						strings.Contains(callStr, "encryption.New") {
						if !busEncrypted && !busSigned {
							triggerPos = ctx.Fset.Position(call.Pos())
						}
						busEncrypted = true
					}

					if strings.Contains(callStr, "SignMiddleware") ||
						strings.Contains(callStr, "signing.New") {
						if !busEncrypted && !busSigned {
							triggerPos = ctx.Fset.Position(call.Pos())
						}
						busSigned = true
					}

					if strings.Contains(callStr, "NewSignedStore") ||
						strings.Contains(callStr, "NewEncryptedStore") {
						storeWrapped = true
					}

					return true
				})
			}

			if (busEncrypted || busSigned) && !storeWrapped {
				f, err := finding.NewBuilder(
					"S010", toolName,
					"Bus has encryption/signing middleware but store is not wrapped — events stored in cleartext",
					finding.SeverityError,
					finding.Pos(finding.FilePath(triggerPos.Filename), triggerPos.Line, triggerPos.Column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Wrap the store with signing.NewSignedStore or encryption.NewEncryptedStore to match bus protection").
					WithSnippet(ctx.SourceLine(triggerPos.Filename, triggerPos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}
