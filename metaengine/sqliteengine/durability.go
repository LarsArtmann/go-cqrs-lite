package sqliteengine

import (
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// synchronousPragma maps a durability tier to the SQLite PRAGMA synchronous
// setting — the translation table lifted from the stack presets (proposal
// §5, 2026-08-17) so system deployments get the same semantics:
//
//	strict  → synchronous=FULL   (fsync at every commit)
//	normal  → synchronous=NORMAL (WAL default — fsync only at checkpoint)
//	relaxed → synchronous=OFF    (no fsync ever)
//
// The empty tier maps to no pragma: engine defaults apply.
func synchronousPragma(tier metaengine.DurabilityTier) (string, error) {
	if err := metaengine.ValidateDurabilityTier(tier); err != nil {
		return "", err
	}

	switch tier {
	case metaengine.DurabilityStrict:
		return "synchronous=FULL", nil
	case metaengine.DurabilityNormal:
		return "synchronous=NORMAL", nil
	case metaengine.DurabilityRelaxed:
		return "synchronous=OFF", nil
	default:
		return "", nil
	}
}

// durabilityPragmas composes the pragma list for an engine construction:
// the tier's synchronous pragma first, then the operator's own pragmas.
//
// It fails when the operator already set PRAGMA synchronous themselves and
// also named a durability tier — two sources of truth for one durability
// knob is a configuration error, not a last-writer-wins race.
func durabilityPragmas(
	tier metaengine.DurabilityTier, user []string,
) ([]string, error) {
	if tier != "" {
		for _, pragma := range user {
			name, _, _ := strings.Cut(pragma, "=")
			if strings.EqualFold(strings.TrimSpace(name), "synchronous") {
				return nil, fmt.Errorf(
					"sqliteengine: durability tier %q conflicts with explicit pragma %q — set one, not both",
					tier,
					pragma,
				)
			}
		}
	}

	tierPragma, err := synchronousPragma(tier)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine: %w", err)
	}

	if tierPragma == "" {
		return user, nil
	}

	return append([]string{tierPragma}, user...), nil
}
