package pgengine

import (
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// synchronousCommit maps a durability tier to the Postgres synchronous_commit
// setting — the translation from the stack tier docs (proposal §5, 2026-08-17):
//
//	strict  → on  (WAL fsync per commit)
//	normal  → off (no per-commit WAL fsync — safe against app crash; small
//	          window of lost transactions on kernel crash)
//	relaxed → off (Postgres has no lower per-session knob; relaxed behaves
//	          as normal)
//
// The empty tier maps to no parameter: server defaults apply.
func synchronousCommit(tier metaengine.DurabilityTier) (string, error) {
	if err := metaengine.ValidateDurabilityTier(tier); err != nil {
		return "", fmt.Errorf("pgengine: %w", err)
	}

	switch tier {
	case metaengine.DurabilityStrict:
		return "on", nil
	case metaengine.DurabilityNormal, metaengine.DurabilityRelaxed:
		return "off", nil
	default:
		return "", nil
	}
}

// durabilityDSN appends the tier's synchronous_commit runtime parameter to a
// Postgres DSN. pgx passes unrecognised DSN parameters to the server as
// startup runtime parameters, so every pooled connection receives the
// setting — unlike a post-connect SET, which only reaches one connection.
//
// It fails when the DSN already sets synchronous_commit and a tier is also
// named — two sources of truth for one durability knob is a configuration
// error, not a last-writer-wins race.
func durabilityDSN(tier metaengine.DurabilityTier, dsn string) (string, error) {
	value, err := synchronousCommit(tier)
	if err != nil {
		return "", err
	}

	if value == "" {
		return dsn, nil
	}

	if strings.Contains(strings.ToLower(dsn), "synchronous_commit") {
		return "", fmt.Errorf(
			"pgengine: durability tier %q conflicts with synchronous_commit already set in the DSN — set one, not both",
			tier,
		)
	}

	param := "synchronous_commit=" + value
	if isURLDSN(dsn) {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}

		return dsn + separator + param, nil
	}

	return dsn + " " + param, nil
}

func isURLDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}
