package store

import (
	"context"
	"errors"
	"testing"
)

// BackendFactory creates a fresh Backend for testing.
// Each call should return an independent instance.
type BackendFactory func(t *testing.T) Backend

// RunBackendTests runs the full conformance suite against a Backend.
func RunBackendTests(t *testing.T, factory BackendFactory) {
	t.Helper()

	t.Run("PutAndGet", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		key := []byte("test:key:1")
		val := []byte("hello")

		if err := b.Put(ctx, key, val); err != nil {
			t.Fatalf("Put: %v", err)
		}

		got, err := b.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if string(got) != string(val) {
			t.Fatalf("Get: got %q, want %q", got, val)
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		_, err := b.Get(ctx, []byte("nonexistent"))
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("Get nonexistent: got err=%v, want ErrNotFound", err)
		}
	})

	t.Run("PutOverwrites", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		key := []byte("test:overwrite")

		if err := b.Put(ctx, key, []byte("v1")); err != nil {
			t.Fatalf("Put v1: %v", err)
		}

		if err := b.Put(ctx, key, []byte("v2")); err != nil {
			t.Fatalf("Put v2: %v", err)
		}

		got, err := b.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if string(got) != "v2" {
			t.Fatalf("Get: got %q, want %q", got, "v2")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		key := []byte("test:delete")

		if err := b.Put(ctx, key, []byte("val")); err != nil {
			t.Fatalf("Put: %v", err)
		}

		if err := b.Delete(ctx, key); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err := b.Get(ctx, key)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("after Delete: got err=%v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteNonexistent", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		if err := b.Delete(ctx, []byte("nonexistent")); err != nil {
			t.Fatalf("Delete nonexistent: %v", err)
		}
	})

	t.Run("ScanEmpty", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		it, err := b.Scan(ctx, []byte("prefix:"))
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		defer it.Close()

		if it.Next() {
			t.Fatal("Scan on empty: unexpected item")
		}

		if err := it.Err(); err != nil {
			t.Fatalf("Iterator.Err: %v", err)
		}
	})

	t.Run("ScanPrefix", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		_ = b.Put(ctx, []byte("a:1"), []byte("v1"))
		_ = b.Put(ctx, []byte("b:1"), []byte("v2"))
		_ = b.Put(ctx, []byte("a:2"), []byte("v3"))
		_ = b.Put(ctx, []byte("a:10"), []byte("v4"))

		it, err := b.Scan(ctx, []byte("a:"))
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		defer it.Close()

		var keys []string

		for it.Next() {
			keys = append(keys, string(it.Key()))
		}

		if err := it.Err(); err != nil {
			t.Fatalf("Err: %v", err)
		}

		want := []string{"a:1", "a:10", "a:2"}

		if len(keys) != len(want) {
			t.Fatalf("Scan keys: got %v, want %v", keys, want)
		}

		for i, k := range keys {
			if k != want[i] {
				t.Fatalf("key[%d]: got %q, want %q", i, k, want[i])
			}
		}
	})

	t.Run("BatchAtomic", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		err := b.Batch(ctx, func(tx Transaction) error {
			_ = tx.Put([]byte("tx:1"), []byte("a"))
			_ = tx.Put([]byte("tx:2"), []byte("b"))

			return nil
		})
		if err != nil {
			t.Fatalf("Batch: %v", err)
		}

		v, err := b.Get(ctx, []byte("tx:1"))
		if err != nil {
			t.Fatalf("Get tx:1: %v", err)
		}

		if string(v) != "a" {
			t.Fatalf("tx:1: got %q, want %q", v, "a")
		}
	})

	t.Run("BatchRollback", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		err := b.Batch(ctx, func(tx Transaction) error {
			_ = tx.Put([]byte("rollback:1"), []byte("val"))

			return errors.New("intentional")
		})
		if err == nil {
			t.Fatal("Batch: expected error")
		}

		_, err = b.Get(ctx, []byte("rollback:1"))
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("after rollback: got err=%v, want ErrNotFound", err)
		}
	})

	t.Run("BatchReadModifyWrite", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := factory(t)
		defer b.Close()

		_ = b.Put(ctx, []byte("counter"), []byte("0"))

		err := b.Batch(ctx, func(tx Transaction) error {
			v, err := tx.Get([]byte("counter"))
			if err != nil {
				return err
			}

			if string(v) != "0" {
				return errors.New("unexpected counter value: " + string(v))
			}

			return tx.Put([]byte("counter"), []byte("1"))
		})
		if err != nil {
			t.Fatalf("Batch RMW: %v", err)
		}

		got, err := b.Get(ctx, []byte("counter"))
		if err != nil {
			t.Fatalf("Get: %v", err)
		}

		if string(got) != "1" {
			t.Fatalf("counter: got %q, want %q", got, "1")
		}
	})

	t.Run("Close", func(t *testing.T) {
		t.Parallel()
		b := factory(t)

		if err := b.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
}
