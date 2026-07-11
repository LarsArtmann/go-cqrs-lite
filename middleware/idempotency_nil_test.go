package middleware

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

func TestQueryIdempotency_NilKeyExtractorPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil keyExtractor, got none")
		}
	}()

	QueryIdempotency(idempotency.NewMemoryStore(0), time.Minute, nil)
}
