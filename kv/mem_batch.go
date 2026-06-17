package kv

import "slices"

type batchOp struct {
	isDelete bool
	key      []byte
	value    []byte
}

type memBatch struct {
	store  *MemStore
	ops    []batchOp
	closed bool
}

var _ Batch = (*memBatch)(nil)

func (b *memBatch) Set(key, value []byte) error {
	if b.closed {
		return ErrClosed
	}

	b.ops = append(b.ops, batchOp{ //nolint:exhaustruct // set op: isDelete=false is correct
		key:   slices.Clone(key),
		value: slices.Clone(value),
	})

	return nil
}

func (b *memBatch) Delete(key []byte) error {
	if b.closed {
		return ErrClosed
	}

	b.ops = append(b.ops, batchOp{ //nolint:exhaustruct // delete op: value omitted is correct
		isDelete: true,
		key:      slices.Clone(key),
	})

	return nil
}

func (b *memBatch) Commit() error {
	if b.closed {
		return ErrClosed
	}

	defer func() { _ = b.Close() }()

	b.store.mu.Lock()
	defer b.store.mu.Unlock()

	err := b.store.checkClosed()
	if err != nil {
		return err
	}

	for _, op := range b.ops {
		if op.isDelete {
			delete(b.store.data, string(op.key))
		} else {
			b.store.data[string(op.key)] = op.value
		}
	}

	return nil
}

func (b *memBatch) Close() error {
	b.closed = true
	b.ops = nil

	return nil
}
