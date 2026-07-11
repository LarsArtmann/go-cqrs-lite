package idempotency

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// KVBackend is the contract a KV store must satisfy to back an idempotency
// Store. Every kv.Store implementation that also implements kv.ConditionalWriter
// satisfies this (e.g., kv.MemStore).
type KVBackend interface {
	kv.Reader
	kv.Writer
	kv.ConditionalWriter
	io.Closer
}

// KVStore adapts any KVBackend into an idempotency.Store.
// The expiry timestamp is stored as the value (Unix nano). Expired entries
// are lazily deleted on read. CheckAndRecord uses SetIfAbsent for atomicity.
type KVStore struct {
	backend KVBackend
}

// NewKVStore wraps a KVBackend as an idempotency.Store.
// The caller retains ownership of the backend's Close — calling Close on the
// returned store closes the underlying backend.
func NewKVStore(backend KVBackend) *KVStore {
	return &KVStore{backend: backend}
}

func (s *KVStore) Seen(ctx context.Context, key string) (bool, error) {
	val, err := s.backend.Get(ctx, []byte(key))
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return false, nil
		}

		return false, errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"idempotency.kv.seen",
			"key %q",
			key,
		)
	}

	expiry, err := strconv.ParseInt(string(val), 10, 64)
	if err != nil {
		return false, errorfamily.NewCorruption(
			"idempotency.kv.decode_failed",
			"failed to decode expiry timestamp from KV store",
		)
	}

	if time.Now().UnixNano() >= expiry {
		_ = s.backend.Delete(ctx, []byte(key))

		return false, nil
	}

	return true, nil
}

func (s *KVStore) Record(ctx context.Context, key string, ttl time.Duration) error {
	expiry := time.Now().Add(ttl).UnixNano()

	if err := s.backend.Set(
		ctx,
		[]byte(key),
		[]byte(strconv.FormatInt(expiry, 10)),
	); err != nil {
		return fmt.Errorf("idempotency: failed to record key %q: %w", key, err)
	}

	return nil
}

func (s *KVStore) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	expiry := time.Now().Add(ttl).UnixNano()
	val := []byte(strconv.FormatInt(expiry, 10))

	inserted, err := s.backend.SetIfAbsent(ctx, []byte(key), val)
	if err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"idempotency.kv.check_and_record",
			"key %q",
			key,
		)
	}

	if inserted {
		return nil
	}

	// Key exists — check if it's expired.
	existing, err := s.backend.Get(ctx, []byte(key))
	if err != nil { //nolint:nestif // retry-on-race logic
		if errors.Is(err, kv.ErrNotFound) {
			// Raced: another goroutine deleted it between SetIfAbsent and Get.
			// Retry once.
			inserted, err = s.backend.SetIfAbsent(ctx, []byte(key), val)
			if err != nil {
				return errorfamily.Wrapf(
					err,
					errorfamily.Transient,
					"idempotency.kv.retry",
					"key %q",
					key,
				)
			}

			if inserted {
				return nil
			}

			return ErrDuplicate
		}

		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"idempotency.kv.check_existing",
			"key %q",
			key,
		)
	}

	prevExpiry, err := strconv.ParseInt(string(existing), 10, 64)
	if err != nil {
		return errorfamily.NewCorruption(
			"idempotency.kv.decode_failed",
			"failed to decode expiry timestamp from KV store",
		)
	}

	if time.Now().UnixNano() >= prevExpiry {
		// Expired — overwrite and claim.
		if err := s.backend.Set(ctx, []byte(key), val); err != nil {
			return errorfamily.Wrapf(
				err,
				errorfamily.Transient,
				"idempotency.kv.overwrite_expired",
				"key %q",
				key,
			)
		}

		return nil
	}

	return ErrDuplicate
}

func (s *KVStore) Close() error {
	return s.backend.Close() //nolint:wrapcheck // passthrough
}
