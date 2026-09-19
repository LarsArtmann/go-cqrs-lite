package sqlstore

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	idempotency "github.com/larsartmann/go-idempotency"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DefaultEngineCollection is the claims-collection name engine-backed stores
// use for their dedup window.
const DefaultEngineCollection = "idempotency"

// NewFromEngine builds the engine-backed facade (ADR-0142): the SAME public
// Store API, backed by any metaengine.DedupStore — so dedup keys ride an
// engine (SQLite, Postgres, memory, KV) picked at deployment time instead of
// this module's own SQL. The one documented divergence: engine-mode Record
// RE-CLAIMS a lapsed window (fresh TTL) where the SQL store leaves a stale
// row untouched — lazier reads then behave identically either way.
func NewFromEngine(dedup metaengine.DedupStore, collection string) (*Store, error) {
	if dedup == nil {
		return nil, fmt.Errorf("idempotency/sqlstore.NewFromEngine: nil DedupStore")
	}

	if collection == "" {
		collection = DefaultEngineCollection
	}

	return &Store{engine: engineFacadeOps{dedup: dedup, collection: collection}}, nil
}

// engineFacadeOps holds the engine-backed implementation for the SQL-shaped
// Store (embedded via the Store's engine field; see store.go).
type engineFacadeOps struct {
	dedup      metaengine.DedupStore
	collection string
}

func (e engineFacadeOps) seen(ctx context.Context, key string) (bool, error) {
	seen, err := e.dedup.DedupSeen(ctx, e.collection, key, time.Now())
	if err != nil {
		return false, errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.engine.seen", "key %q", key,
		)
	}

	return seen, nil
}

func (e engineFacadeOps) record(ctx context.Context, key string, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}

	// Record is a NO-OP on an existing live window (the SQL store's
	// contract): the duplicate signal from the CAS is swallowed here.
	if _, err := e.dedup.DedupCheckAndRecord(ctx, e.collection, key, ttl, time.Now()); err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.engine.record", "key %q", key,
		)
	}

	return nil
}

func (e engineFacadeOps) checkAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}

	seen, err := e.dedup.DedupCheckAndRecord(ctx, e.collection, key, ttl, time.Now())
	if err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.engine.check_and_record", "key %q", key,
		)
	}

	if seen {
		return idempotency.ErrDuplicate
	}

	return nil
}

func (e engineFacadeOps) sweep(ctx context.Context) (int64, error) {
	removed, err := e.dedup.DedupSweep(ctx, e.collection, time.Now())
	if err != nil {
		return 0, errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.engine.sweep", "",
		)
	}

	return int64(removed), nil
}

func validateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return idempotency.ErrInvalidTTL
	}

	return nil
}
