package dgraphengine

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxRetries     = 3
	retryBaseDelay = 50 * time.Millisecond
	retryMaxDelay  = 500 * time.Millisecond
)

// isTransientError returns true for errors that are safe to retry:
//   - gRPC Unavailable: server temporarily unreachable (leader election, network)
//   - gRPC Aborted: transaction conflict (RAFT proposal failed, common in Dgraph)
//   - "Please retry again": Dgraph-specific RAFT retry message
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.Aborted:
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "please retry again") ||
		strings.Contains(msg, "transaction conflict") ||
		strings.Contains(msg, "leader") && strings.Contains(msg, "not")
}

// withRetry wraps a Dgraph operation with transient-error retry.
// The function is retried up to maxRetries times with exponential backoff.
// Context cancellation is always respected.
func withRetry[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isTransientError(err) {
			return zero, err
		}

		if attempt < maxRetries {
			delay := min(retryBaseDelay<<attempt, retryMaxDelay)

			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return zero, lastErr
}

// withRetryVoid is the void variant of withRetry for operations with no return value.
func withRetryVoid(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := withRetry(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}
