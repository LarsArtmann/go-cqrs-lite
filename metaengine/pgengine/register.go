package pgengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"postgres",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			dsn, err := durabilityDSN(cfg.Durability, cfg.DSN)
			if err != nil {
				return nil, err
			}

			return New(
				dsn,
				withDurabilityTier(cfg.Durability),
			) //nolint:contextcheck // constructor takes no ctx
		},
	)
}
