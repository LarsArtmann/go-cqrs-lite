package security

import (
	"context"
	"go/ast"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// S002: Missing encryption for sensitive payloads.
// Detects event payload structs with PII fields (email, SSN, phone) without encryption middleware.
// Downgrades to INFO when the project appears to be local-only (SQLite, no HTTP server).
func NewS002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S002-missing-encryption-for-sensitive-payloads",
		func(_ context.Context) ([]finding.Finding, error) {
			piiFields := []string{
				"email", "phone", "ssn", "password",
				"address", "creditcard", "credit_card",
			}

			var piiEventPayloads []analyzer.EventInfo

			for _, evt := range ctx.Registry.Events {
				name := strings.ToLower(evt.Name)
				for _, pii := range piiFields {
					if strings.Contains(name, pii) {
						piiEventPayloads = append(piiEventPayloads, evt)
						break
					}
				}
			}

			if len(piiEventPayloads) == 0 {
				piiEventPayloads = findPIIInPayloadStructs(ctx, piiFields)
			}

			if len(piiEventPayloads) == 0 {
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
				if hasEncryption {
					break
				}
			}

			if hasEncryption {
				return nil, nil
			}

			var findings []finding.Finding
			pos := piiEventPayloads[0].Pos
			if pos.Filename == "" {
				pos.Filename = filepath.Join(ctx.ProjectRoot, "go.mod")
			}

			severity := finding.SeverityError
			confidence := finding.ConfidenceMedium
			suggestion := "Add encryption.EncryptMiddleware(enc) to your bus.UsePublish chain"

			if !ctx.FeatureProfile.HasServer {
				severity = finding.SeverityInfo
				confidence = finding.ConfidenceLow
				suggestion = "This appears to be a local-only project (no HTTP/gRPC server). Consider adding encryption if the data may be exposed to networks"
			}

			f, err := finding.NewBuilder(
				"S002",
				toolName,
				"Event payloads contain PII fields but no encryption middleware — data is stored in plaintext",
				severity,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategorySecurity).
				WithConfidence(confidence).
				WithSuggestion(suggestion).
				WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

func findPIIInPayloadStructs(
	ctx *analyzer.AnalysisContext,
	piiFields []string,
) []analyzer.EventInfo {
	var matches []analyzer.EventInfo

	payloadSet := ctx.Registry.EventPayloadTypes
	if len(payloadSet) == 0 {
		return nil
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			if !payloadSet[ts.Name.Name] {
				return true
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}

			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					lname := strings.ToLower(name.Name)
					for _, pii := range piiFields {
						if strings.Contains(lname, pii) {
							pos := ctx.Fset.Position(ts.Pos())
							matches = append(matches, analyzer.EventInfo{
								Name: ts.Name.Name,
								File: gf.Path,
								Pos:  pos,
							})
							return false
						}
					}
				}
			}

			return true
		})
	}

	return matches
}

// S003: Missing event signing.
// Detects event stores in production without signing middleware (tamper detection).
// Suppressed for local-only systems (no server) — tamper detection matters for
// shared or multi-user stores where events cross trust boundaries.
func NewS003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S003-missing-event-signing",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

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
				if hasSigning {
					break
				}
			}

			if hasSigning {
				return nil, nil
			}

			var savePos finding.Position
			hasEventStore := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					method := sel.Sel.Name
					if method == "Save" || method == "AppendBatch" || method == "Publish" {
						pkgName := analyzer.SelectorPackage(sel)
						if strings.Contains(pkgName, "event") ||
							strings.Contains(pkgName, "store") ||
							strings.Contains(pkgName, "repo") {
							pos := ctx.Fset.Position(call.Pos())
							savePos = finding.Pos(
								finding.FilePath(pos.Filename), pos.Line, pos.Column,
							)
							hasEventStore = true
							return false
						}
					}

					return true
				})

				if hasEventStore {
					break
				}
			}

			if !hasEventStore {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"S003", toolName,
				"Event store without signing middleware — events are vulnerable to tampering",
				finding.SeverityWarning,
				savePos,
			).
				WithCategory(finding.CategorySecurity).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add signing.SignMiddleware(signer) to bus.UsePublish and signing.VerifyMiddleware to bus.Use").
				WithSnippet(ctx.SourceLine(string(savePos.File), savePos.Line)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
