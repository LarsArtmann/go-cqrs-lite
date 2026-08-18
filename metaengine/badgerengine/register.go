package badgerengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"badger",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			if err := metaengine.RejectDurabilityTier("badger", cfg); err != nil {
				return nil, err
			}

			return NewBadgerEngine(cfg.DSN)
		},
	)
}
