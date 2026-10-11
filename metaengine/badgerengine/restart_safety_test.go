package badgerengine_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestBadgerRestartSafety_StreamAndJournal verifies that reopening a persistent
// Badger DB does NOT reset seq counters to zero — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestBadgerRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return badgerengine.NewBadgerEngine(path)
	})
}

// TestBadgerRestartSafety_FromDB verifies seq seeding when using
// NewBadgerEngineFromDB (caller-owned DB path).
func TestBadgerRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyFromDBTest(t,
		func(dir string) (metaengine.Engine, error) {
			return badgerengine.NewBadgerEngine(filepath.Join(dir, "badger"))
		},
		func(dir string) (metaengine.Engine, error) {
			path := filepath.Join(dir, "badger")

			db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
			if err != nil {
				return nil, fmt.Errorf("raw badger open: %w", err)
			}

			return badgerengine.NewBadgerEngineFromDB(db)
		},
	)
}
