package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// E012: Dual-write migration bus without completion criteria.
// Detects projects with a dual-write bus type (DualWrite, dual_write) that
// publishes to both legacy and new systems, but have no feature flag or
// completion check to detect when the migration is done and disable the
// dual-write. Without a completion mechanism, the dual-write runs forever,
// adding latency and coupling to both systems.
//
//nolint:ireturn // factory returns public interface
func NewE012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E012-dual-write-no-completion",
		func(_ context.Context) ([]finding.Finding, error) {
			if !typeExists(ctx, "DualWrite") && !typeExists(ctx, "dualWrite") {
				return nil, nil
			}

			// Check for feature-flag patterns: a config struct with an
			// Enabled/Active/DualWrite bool field, or flag.BoolVar usage.
			hasFeatureFlag := findKeyBoolLit(ctx, "DualWriteEnabled", true) ||
				findKeyBoolLit(ctx, "DualWriteActive", true) ||
				findKeyBoolLit(ctx, "MigrationEnabled", true) ||
				projectCallsImportPathBool(ctx, "flag", "BoolVar")

			if hasFeatureFlag {
				return nil, nil
			}

			pos, ok := firstFilePos(ctx)
			if !ok {
				return nil, nil
			}

			return singleFinding(
				ctx,
				"E012",
				"Dual-write bus detected without a feature flag or completion check — "+
					"the dual-write will run forever, adding latency and coupling to both systems",
				"Add a feature flag (config struct with DualWriteEnabled bool) or a "+
					"completion check (count-based or time-based) to disable the dual-write "+
					"once the migration is verified",
				pos,
				finding.SeverityWarning,
				finding.ConfidenceLow,
			), nil
		},
	)
}

// E013: Signing/encryption configured but disabled by default.
// Detects projects that import signing or encryption modules but set their
// config Enabled field to false IN a signing/encryption config struct. The
// security infrastructure is present but inert — events are not actually
// signed or encrypted. This is especially dangerous when the code looks
// production-ready but silently ships without security guarantees.
//
// The composite literal type is verified to prevent false positives from
// unrelated structs that also have an Enabled: false field (e.g., feature
// flags, debug configs).
//
//nolint:ireturn // factory returns public interface
func NewE013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E013-signing-disabled-by-default",
		func(_ context.Context) ([]finding.Finding, error) {
			hasSigning := importsPathSuffix(ctx, "go-cqrs-lite/signing")
			hasEncryption := importsPathSuffix(ctx, "go-cqrs-lite/encryption")
			if !hasSigning && !hasEncryption {
				return nil, nil
			}

			typeSubs := []string{"signing", "encryption"}
			if !findKeyBoolLitInTypedComposite(ctx, "Enabled", false, typeSubs...) {
				return nil, nil
			}

			pos, ok := firstKeyBoolPosInTypedComposite(ctx, "Enabled", false, typeSubs...)
			if !ok {
				pos, _ = firstFilePos(ctx)
			}

			module := "signing"
			if hasEncryption {
				module = "encryption"
			}

			if hasSigning && hasEncryption {
				module = "signing/encryption"
			}

			return singleFinding(
				ctx,
				"E013",
				fmt.Sprintf(
					"Project imports %s module but config has Enabled: false — "+
						"security infrastructure is present but inert, events are not protected",
					module,
				),
				"Set Enabled: true in production configs, or remove the signing/encryption "+
					"imports if not ready to enforce. Document the security implications of "+
					"shipping with Enabled: false",
				pos,
				finding.SeverityWarning,
				finding.ConfidenceMedium,
			), nil
		},
	)
}
