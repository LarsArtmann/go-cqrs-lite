package security

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// S008: Asymmetric event signing setup.
// Detects signing.SignMiddleware on the publish side without a corresponding
// signing.VerifyMiddleware or signing.RequireSignatureMiddleware on the
// consume side (and vice versa). Signing is decorative if events are never
// verified on read.
//
//nolint:ireturn // factory returns public interface
func NewS008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S008-asymmetric-signing",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasSign := false
			var signPos ast.Node

			hasVerify := false
			var verifyPos ast.Node

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

					switch sel.Sel.Name {
					case "SignMiddleware":
						hasSign = true
						signPos = call
					case "VerifyMiddleware", "RequireSignatureMiddleware":
						hasVerify = true
						verifyPos = call
					}

					return true
				})
			}

			if hasSign && !hasVerify {
				pos := ctx.Fset.Position(signPos.Pos())
				f, err := finding.NewBuilder(
					"S008", toolName,
					"SignMiddleware configured but no VerifyMiddleware/RequireSignatureMiddleware "+
						"on consume side — signed events are never verified",
					finding.SeverityError,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Add signing.VerifyMiddleware(verifier) to bus.Use() "+
						"so consumers verify signatures on every event").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			if !hasSign && hasVerify && verifyPos != nil {
				pos := ctx.Fset.Position(verifyPos.Pos())
				f, err := finding.NewBuilder(
					"S008", toolName,
					"VerifyMiddleware configured but events are never signed — "+
						"verification is a no-op on unsigned events",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Add signing.SignMiddleware(signer) to bus.UsePublish() "+
						"so events carry signatures").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// S009: Asymmetric event encryption setup.
// Detects encryption.EncryptMiddleware on the publish side without a
// corresponding encryption.DecryptMiddleware on the consume side (and vice
// versa). Encrypted events silently break every consumer if never decrypted.
//
//nolint:ireturn // factory returns public interface
func NewS009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S009-asymmetric-encryption",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasEncrypt := false
			var encryptPos ast.Node

			hasDecrypt := false
			var decryptPos ast.Node

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

					switch sel.Sel.Name {
					case "EncryptMiddleware":
						hasEncrypt = true
						encryptPos = call
					case "DecryptMiddleware":
						hasDecrypt = true
						decryptPos = call
					}

					return true
				})
			}

			if hasEncrypt && !hasDecrypt {
				pos := ctx.Fset.Position(encryptPos.Pos())
				f, err := finding.NewBuilder(
					"S009", toolName,
					"EncryptMiddleware configured but no DecryptMiddleware on consume side — "+
						"encrypted events cannot be read by consumers",
					finding.SeverityError,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Add encryption.DecryptMiddleware(decrypter) to bus.Use() "+
						"so consumers can decrypt payloads").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			if !hasEncrypt && hasDecrypt && decryptPos != nil {
				pos := ctx.Fset.Position(decryptPos.Pos())
				f, err := finding.NewBuilder(
					"S009", toolName,
					"DecryptMiddleware configured but events are never encrypted — "+
						"decryption will fail on every event",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(finding.ConfidenceHigh).
					WithSuggestion("Add encryption.EncryptMiddleware(encrypter) to bus.UsePublish() "+
						"so payloads are encrypted before storage").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}
