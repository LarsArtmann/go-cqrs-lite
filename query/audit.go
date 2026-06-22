package query

import (
	"context"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// AuditLevel controls how much detail the audit middleware records.
type AuditLevel int

const (
	// AuditOff disables auditing. The middleware is a pass-through.
	AuditOff AuditLevel = iota
	// AuditMetadata records query type, request ID, and timing, but not payload.
	AuditMetadata
	// AuditFull records query type, request ID, timing, and full payload.
	AuditFull
)

// AuditMiddleware creates a [Middleware] that persists every dispatched query
// to the given [QuerySink] for audit trail and replay.
//
// The level controls what is recorded:
//   - AuditOff:     no persistence (middleware is a no-op pass-through).
//   - AuditMetadata: type + request ID + timestamp (no payload).
//   - AuditFull:    type + request ID + timestamp + payload.
//
// If the sink returns an error, the error is logged but does not block the
// query — audit failure must not break the read path. The query still executes
// and returns its result normally.
//
// Usage:
//
//	d := query.NewDispatcher()
//	d.Use(query.AuditMiddleware(queryStore, query.AuditFull, logger))
func AuditMiddleware(sink QuerySink, level AuditLevel, logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	if level == AuditOff || sink == nil {
		return func(next Handler) Handler {
			return next
		}
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, q Query) (any, error) {
			start := time.Now()

			result, err := next(ctx, q)

			elapsed := time.Since(start)

			persisted, persistErr := buildAuditQuery(q, level, start)
			if persistErr != nil {
				logger.WarnContext(ctx, "query audit: build persisted query",
					"query_type", q.Type(),
					"error", persistErr)

				return result, err
			}

			persistErr = sink.SaveQuery(ctx, persisted)
			if persistErr != nil {
				logger.WarnContext(ctx, "query audit: save failed",
					"query_type", q.Type(),
					"duration", elapsed,
					"error", persistErr)

				return result, err
			}

			return result, err
		}
	}
}

func buildAuditQuery(q Query, level AuditLevel, receivedAt time.Time) (*PersistedQuery, error) {
	var payload []byte

	if level == AuditFull {
		if p, ok := q.(interface{ Payload() []byte }); ok {
			payload = p.Payload()
		}
	}

	return NewPersistedQuery(
		q.Type(),
		payload,
		WithQueryReceivedAt(receivedAt),
		WithQueryID(id.NewRequestID()),
	)
}
