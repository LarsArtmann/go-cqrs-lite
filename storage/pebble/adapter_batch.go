package pebble

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
)

type pebbleBatch struct {
	batch      *pebble.Batch
	commitOpts *pebble.WriteOptions
	committed  bool
}

var _ kv.Batch = (*pebbleBatch)(nil)

func (batch *pebbleBatch) Set(key, value []byte) error {
	err := batch.batch.Set(key, value, nil)
	if err != nil {
		return fmt.Errorf("pebble: batch set %q: %w", key, err)
	}

	return nil
}

func (batch *pebbleBatch) Delete(key []byte) error {
	err := batch.batch.Delete(key, nil)
	if err != nil {
		return fmt.Errorf("pebble: batch delete %q: %w", key, err)
	}

	return nil
}

func (batch *pebbleBatch) Commit() error {
	if batch.committed {
		return nil
	}

	batch.committed = true

	err := batch.batch.Commit(batch.commitOpts)
	if err != nil {
		return fmt.Errorf("pebble: batch commit: %w", err)
	}

	return nil
}

func (batch *pebbleBatch) Close() error {
	if batch.committed {
		return nil
	}

	err := batch.batch.Close()
	if err != nil {
		return fmt.Errorf("pebble: batch close: %w", err)
	}

	return nil
}
