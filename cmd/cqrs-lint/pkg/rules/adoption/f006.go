package adoption

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// F006 detects event payloads with PII-like field names and no encryption import.
// Sensitive data in event streams should be encrypted at rest.
//
//nolint:ireturn // factory returns public interface
func NewF006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"F006-no-encryption-for-sensitive-data",
		func(_ context.Context) ([]finding.Finding, error) {
			if importsPath(ctx, "go-cqrs-lite/encryption") {
				return nil, nil
			}

			pos, ok := hasPIIInEventPayloads(ctx)
			if !ok {
				return nil, nil
			}

			return singleInfoFinding(ctx,
				"F006",
				"Event payload contains PII-like field but no encryption module — "+
					"sensitive data is stored in plaintext",
				"Import the encryption module and use encryption.EncryptMiddleware "+
					"on your event bus to protect sensitive payloads at rest. "+
					"Supports XChaCha20-Poly1305 and AES-256-GCM.",
				pos, finding.ConfidenceMedium,
			), nil
		},
	)
}
