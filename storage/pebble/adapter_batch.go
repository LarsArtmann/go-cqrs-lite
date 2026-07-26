package pebble

import (
	"context"
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

type pebbleBatch struct {
	batch      *pebble.Batch
	commitOpts *pebble.WriteOptions
	committed  bool
}

var _ kv.Batch = (*pebbleBatch)(nil)

func (batch *pebbleBatch) Set(_ context.Context, key, value []byte) error {
	return wrapInfraOrOK(batch.batch.Set(key, value, nil), "pebble.batch.set",
		fmt.Sprintf("batch set %q", key))
}

func (batch *pebbleBatch) Delete(_ context.Context, key []byte) error {
	return wrapInfraOrOK(batch.batch.Delete(key, nil), "pebble.batch.delete",
		fmt.Sprintf("batch delete %q", key))
}

func (batch *pebbleBatch) Commit(_ context.Context) error {
	if batch.committed {
		return nil
	}

	batch.committed = true

	return wrapInfraOrOK(batch.batch.Commit(batch.commitOpts), "pebble.batch.commit",
		"commit batch")
}

func (batch *pebbleBatch) Close() error {
	if batch.committed {
		return nil
	}

	return wrapInfraOrOK(batch.batch.Close(), "pebble.batch.close",
		"close batch")
}
