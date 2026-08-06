package bbolt

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// KVAdapter adapts a *bbolt.DB to the kv.Store interface using the cqrs_kv
// bucket. The adapter does NOT own the *bbolt.DB — Close is a no-op.
type KVAdapter struct {
	db     *bolt.DB
	closed atomic.Bool
}

func NewKVStore(database *bolt.DB) (kv.Store, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	return &KVAdapter{db: database}, nil
}

func (a *KVAdapter) checkClosed() error {
	if a.closed.Load() {
		return kv.ErrClosed
	}

	return nil
}

func (a *KVAdapter) Get(_ context.Context, key []byte) ([]byte, error) {
	if err := a.checkClosed(); err != nil {
		return nil, err
	}

	var result []byte

	err := a.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return kv.ErrNotFound
		}

		val := bucket.Get(key)
		if val == nil {
			return kv.ErrNotFound
		}

		result = slices.Clone(val)
		return nil
	})

	return result, wrapBucketErr(err, "bbolt.kv_get", "get key")
}

func (a *KVAdapter) Has(_ context.Context, key []byte) (bool, error) {
	if err := a.checkClosed(); err != nil {
		return false, err
	}

	var found bool

	err := a.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return nil
		}

		found = bucket.Get(key) != nil
		return nil
	})

	return found, wrapBucketErr(err, "bbolt.kv_has", "check key existence")
}

func (a *KVAdapter) Set(_ context.Context, key, value []byte) error {
	if err := a.checkClosed(); err != nil {
		return err
	}

	return a.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return errorfamily.NewInfrastructure("bbolt.bucket_missing", "kv bucket not found")
		}

		return wrapBucketErr(bucket.Put(key, value),
			"bbolt.kv_set", fmt.Sprintf("set %q", key))
	})
}

func (a *KVAdapter) Delete(_ context.Context, key []byte) error {
	if err := a.checkClosed(); err != nil {
		return err
	}

	return a.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return nil
		}

		return wrapBucketErr(bucket.Delete(key),
			"bbolt.kv_delete", fmt.Sprintf("delete %q", key))
	})
}

func (a *KVAdapter) Batch(_ context.Context) (kv.Batch, error) {
	if err := a.checkClosed(); err != nil {
		return nil, err
	}

	return &bboltBatch{db: a.db}, nil
}

func (a *KVAdapter) NewIterator(_ context.Context, prefix []byte) (kv.Iterator, error) {
	if err := a.checkClosed(); err != nil {
		return nil, err
	}

	tx, err := a.db.Begin(false)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "bbolt.kv_iterator",
			"begin read transaction")
	}

	bucket := tx.Bucket([]byte(bucketKV))
	if bucket == nil {
		_ = tx.Rollback()
		return &emptyIterator{}, nil
	}

	cursor := bucket.Cursor()

	var k, v []byte
	if len(prefix) > 0 {
		k, v = cursor.Seek(prefix)
	} else {
		k, v = cursor.First()
	}

	iter := &bboltIterator{tx: tx, cursor: cursor, prefix: prefix, k: k, v: v, started: false}

	if k == nil || (len(prefix) > 0 && !hasPrefix(k, prefix)) {
		iter.finished = true
	}

	return iter, nil
}

func (a *KVAdapter) Close() error {
	a.closed.Store(true)
	return nil
}

// SetIfAbsent implements kv.ConditionalWriter atomically — bbolt serializes
// all writers, so the check-then-set is safe inside a single Update.
func (a *KVAdapter) SetIfAbsent(_ context.Context, key, value []byte) (bool, error) {
	if err := a.checkClosed(); err != nil {
		return false, err
	}

	var inserted bool

	err := a.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return errorfamily.NewInfrastructure("bbolt.bucket_missing", "kv bucket not found")
		}

		if bucket.Get(key) != nil {
			inserted = false
			return nil
		}

		inserted = true
		return bucket.Put(key, value)
	})

	return inserted, wrapBucketErr(err, "bbolt.kv_set_if_absent", "set key if absent")
}
