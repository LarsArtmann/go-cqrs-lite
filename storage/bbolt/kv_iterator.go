package bbolt

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// bboltIterator holds a long-lived read transaction + cursor.
// Close rolls back the transaction, releasing the read lock.
type bboltIterator struct {
	tx       *bolt.Tx
	cursor   *bolt.Cursor
	prefix   []byte
	k, v     []byte
	started  bool
	finished bool
}

func (it *bboltIterator) Next() bool {
	if it.finished {
		return false
	}

	if !it.started {
		it.started = true
		// On first call, the cursor was already positioned by Seek/First
		// in the constructor. Use the k/v captured there.
	} else {
		it.k, it.v = it.cursor.Next()
	}

	if it.k == nil {
		it.finished = true
		return false
	}

	if len(it.prefix) > 0 && !hasPrefix(it.k, it.prefix) {
		it.finished = true
		return false
	}

	return true
}

func (it *bboltIterator) Key() []byte   { return it.k }
func (it *bboltIterator) Value() []byte { return it.v }
func (it *bboltIterator) Error() error  { return nil }

func (it *bboltIterator) Close() error {
	if it.tx != nil {
		return it.tx.Rollback()
	}

	return nil
}

type emptyIterator struct{}

func (*emptyIterator) Next() bool    { return false }
func (*emptyIterator) Key() []byte   { return nil }
func (*emptyIterator) Value() []byte { return nil }
func (*emptyIterator) Error() error  { return nil }
func (*emptyIterator) Close() error  { return nil }

type bboltBatch struct {
	db  *bolt.DB
	ops []batchOp
}

type batchOp struct {
	key    []byte
	value  []byte
	delete bool
}

func (b *bboltBatch) Set(_ context.Context, key, value []byte) error {
	b.ops = append(b.ops, batchOp{key: key, value: value})
	return nil
}

func (b *bboltBatch) Delete(_ context.Context, key []byte) error {
	b.ops = append(b.ops, batchOp{key: key, delete: true})
	return nil
}

func (b *bboltBatch) Commit(_ context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketKV))
		if bucket == nil {
			return errorfamily.NewInfrastructure(
				"bbolt.bucket_missing", "kv bucket not found")
		}

		for _, op := range b.ops {
			if op.delete {
				if err := bucket.Delete(op.key); err != nil {
					return err
				}
			} else {
				if err := bucket.Put(op.key, op.value); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (b *bboltBatch) Rollback() error {
	b.ops = nil
	return nil
}

func (b *bboltBatch) Close() error {
	b.ops = nil
	return nil
}

var (
	_ kv.Store    = (*KVAdapter)(nil)
	_ kv.Iterator = (*bboltIterator)(nil)
	_ kv.Batch    = (*bboltBatch)(nil)
)
