package security

import (
	"context"
	"go/ast"
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

					if strings.Contains(callStr, "EncryptMiddleware") || strings.Contains(callStr, "encryption.New") {
						busEncrypted = true
					}

					if strings.Contains(callStr, "SignMiddleware") || strings.Contains(callStr, "signing.New") {
						busSigned = true
					}

					if strings.Contains(callStr, "NewSignedStore") || strings.Contains(callStr, "NewEncryptedStore") {
						storeWrapped = true
					}

					return true
				})
			}

			if (busEncrypted || busSigned) && !storeWrapped {
				pos := finding.Pos("project", 1, 1)

				f, err := finding.NewBuilder(
					"S010", toolName,
					"Bus has encryption/signing middleware but store is not wrapped — events stored in cleartext",
					finding.SeverityError,
					pos,
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Wrap the store with signing.NewSignedStore or encryption.NewEncryptedStore to match bus protection").
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}
