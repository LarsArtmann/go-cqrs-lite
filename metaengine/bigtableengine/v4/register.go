package bigtableengine

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"

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

			// BigTable versions cells natively, so cell_versioning never
			// gates construction here — a requested RetentionPolicy maps
			// onto the column-family GC policy (ADR-0141 §1's analog of
			// client-side retention trimming).
			var engineOpts []Option

			if cfg.CellVersioning != nil {
				if err := cfg.CellVersioning.Validate(); err != nil {
					return nil, fmt.Errorf("driver %q cell_versioning: %w", "bigtable", err)
				}

				if gc := gcPolicyFor(*cfg.CellVersioning); gc != nil {
					engineOpts = append(engineOpts, WithGCPolicy(gc))
				}
			}

			return dial(
				ctx,
				project,
				instance,
				table,
				engineOpts,
			)
		},
	)
}

// gcPolicyFor translates a RetentionPolicy into its BigTable GC-policy
// analog: MaxVersions → MaxVersionsPolicy, MaxAge → MaxAgePolicy, both →
// IntersectionPolicy (a version survives only when it satisfies BOTH —
// RetentionPolicy semantics). A zero policy returns nil: no GC policy,
// every version kept.
func gcPolicyFor(p metaengine.RetentionPolicy) bigtable.GCPolicy { //nolint:ireturn // vendor-typed
	switch {
	case p.MaxVersions > 0 && p.MaxAge > 0:
		return bigtable.IntersectionPolicy(
			bigtable.MaxVersionsPolicy(p.MaxVersions),
			bigtable.MaxAgePolicy(p.MaxAge),
		)
	case p.MaxVersions > 0:
		return bigtable.MaxVersionsPolicy(p.MaxVersions)
	case p.MaxAge > 0:
		return bigtable.MaxAgePolicy(p.MaxAge)
	default:
		return nil
	}
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
