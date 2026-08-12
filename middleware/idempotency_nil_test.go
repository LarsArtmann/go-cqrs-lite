package middleware

import (
	"testing"
	"time"
)

func TestQueryIdempotency_NilKeyExtractorPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil keyExtractor, got none")
		}
	}()

	QueryIdempotency(newTestIdempotencyStore(t), time.Minute, nil)
}
