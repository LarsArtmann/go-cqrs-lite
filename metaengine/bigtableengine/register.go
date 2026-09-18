package bigtableengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"bigtable",
		func(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			// A DriverConfig.DSN of "project/instance/table" is required; the
			// Bigtable emulator is picked up via BIGTABLE_EMULATOR_HOST by the
			// client when set.
			project, instance, table, err := parseDSN(cfg.DSN)
			if err != nil {
				return nil, err
			}

			if err := metaengine.RejectDurabilityTier("bigtable", cfg); err != nil {
				return nil, err
			}

			return New(
				ctx,
				project,
				instance,
				table,
			) //nolint:wrapcheck // factory errors flow to the caller verbatim
		},
	)
}

// parseDSN splits "project/instance/table" (3 non-empty components).
func parseDSN(dsn string) (project, instance, table string, err error) {
	parts := splitAll(dsn, '/')
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", errBadDSN(dsn)
	}

	return parts[0], parts[1], parts[2], nil
}

func splitAll(s string, sep byte) []string {
	var out []string

	start := 0

	for i := range len(s) {
		if s[i] == sep {
			out = append(out, s[start:i])
			start = i + 1
		}
	}

	return append(out, s[start:])
}

type dsnError struct{ dsn string }

func (e *dsnError) Error() string {
	return "bigtableengine: DSN must be \"project/instance/table\", got " + e.dsn
}

func errBadDSN(dsn string) error { return &dsnError{dsn: dsn} }
