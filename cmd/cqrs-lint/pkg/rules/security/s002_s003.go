package security

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// S002: Missing encryption for sensitive payloads.
// Detects event payloads with PII fields (email, SSN, phone) without encryption middleware.
func NewS002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S002-missing-encryption-for-sensitive-payloads",
		func(_ context.Context) ([]finding.Finding, error) {
			piiFields := []string{
				"email",
				"phone",
				"ssn",
				"password",
				"address",
				"creditcard",
				"credit_card",
			}

			hasPII := false

			// Check event payload type names for PII keywords.
			for _, evt := range ctx.Registry.Events {
				name := strings.ToLower(evt.Name)
				for _, pii := range piiFields {
					if strings.Contains(name, pii) {
						hasPII = true

						break
					}
				}
			}

			// Also check struct field names for PII keywords.
			if !hasPII {
				for _, gf := range ctx.GoFiles {
					if gf.IsTest {
						continue
					}

					ast.Inspect(gf.AST, func(n ast.Node) bool {
						st, ok := n.(*ast.StructType)
						if !ok || st.Fields == nil {
							return true
						}

						for _, field := range st.Fields.List {
							for _, name := range field.Names {
								lname := strings.ToLower(name.Name)
								for _, pii := range piiFields {
									if strings.Contains(lname, pii) {
										hasPII = true

										return false
									}
								}
							}
						}

						return true
					})
					if hasPII {
						break
					}
				}
			}

			if !hasPII {
				return nil, nil
			}

			hasEncryption := false

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "go-cqrs-lite/encryption") {
						hasEncryption = true

						break
					}
				}
			}

			if hasEncryption {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"S002", toolName,
				"Event payloads contain PII fields but no encryption middleware — data is stored in plaintext",
				finding.SeverityError,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategorySecurity).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Add encryption.EncryptMiddleware(enc) to your bus.UsePublish chain").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// S003: Missing event signing.
// Detects event stores in production without signing middleware (tamper detection).
func NewS003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S003-missing-event-signing",
		func(_ context.Context) ([]finding.Finding, error) {
			hasSigning := false

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "go-cqrs-lite/signing") {
						hasSigning = true

						break
					}
				}
			}

			if hasSigning {
				return nil, nil
			}

			hasEventStore := false

			for _, fold := range ctx.Registry.Folds {
				_ = fold
				hasEventStore = true

				break
			}

			if !hasEventStore {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"S003", toolName,
				"Event store without signing middleware — events are vulnerable to tampering",
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategorySecurity).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add signing.SignMiddleware(signer) to bus.UsePublish and signing.VerifyMiddleware to bus.Use").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
