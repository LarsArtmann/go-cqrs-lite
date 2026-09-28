package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// F006 detects event payloads with PII-like field names and no encryption import.
// Sensitive data in event streams should be encrypted at rest.
//
//nolint:ireturn // factory returns public interface
func NewF006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F006-no-encryption-for-sensitive-data",
		func(_ context.Context) ([]finding.Finding, error) {
			var out []finding.Finding

			for _, sc := range coachingScopes(ctx) {
				if importsPathIn(sc.files, "go-cqrs-lite/encryption") {
					continue
				}

				pos, ok := hasPIIInEventPayloadsIn(ctx.Fset, sc.files)
				if !ok {
					continue
				}

				out = append(out, singleInfoFinding(
					ctx,
					"F006",
					"Event payload contains PII-like field but no encryption module — "+
						"sensitive data is stored in plaintext",
					"Import the encryption module and use encryption.EncryptMiddleware "+
						"on your event bus to protect sensitive payloads at rest. "+
						"Supports XChaCha20-Poly1305 and AES-256-GCM.",
					pos, finding.ConfidenceMedium,
				)...)
			}

			return out, nil
		},
	)
}
