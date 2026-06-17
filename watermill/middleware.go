package watermill

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// CorrelationIDMiddleware wraps Watermill's CorrelationID middleware for
// use in CQRS message routers. It automatically propagates the correlation
// ID from incoming message metadata to all outgoing messages produced by
// the handler.
//
//	router.AddMiddleware(watermill.CorrelationIDMiddleware())
func CorrelationIDMiddleware() message.HandlerMiddleware {
	return middleware.CorrelationID
}

// RetryConfig configures retry behavior for transient failures.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int
	// InitialInterval is the delay before the first retry.
	InitialInterval time.Duration
	// MaxInterval caps the exponential backoff growth.
	MaxInterval time.Duration
	// Multiplier is the backoff growth factor (e.g., 2.0 for doubling).
	Multiplier float64
	// Logger is an optional Watermill logger.
	Logger watermill.LoggerAdapter
}

// DefaultRetryConfig returns sensible defaults for CQRS retry behavior:
// 5 retries, 100ms initial interval, 10s max interval, 2.0x multiplier.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      5,                      //nolint:mnd // sensible default
		InitialInterval: 100 * time.Millisecond, //nolint:mnd // sensible default
		MaxInterval:     10 * time.Second,       //nolint:mnd // sensible default
		Multiplier:      2.0,                    //nolint:mnd // sensible default
	}
}

// NewRetryMiddleware creates a retry middleware with exponential backoff
// using the provided configuration. The middleware retries on handler errors
// with increasing delays between attempts.
//
//	router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultRetryConfig()))
//	// or with custom config:
//	router.AddMiddleware(watermill.NewRetryMiddleware(watermill.RetryConfig{
//	    MaxRetries: 10,
//	    InitialInterval: 50 * time.Millisecond,
//	    MaxInterval: 5 * time.Second,
//	    Multiplier: 1.5,
//	}))
func NewRetryMiddleware(cfg RetryConfig) message.HandlerMiddleware {
	retry := middleware.Retry{
		MaxRetries:      cfg.MaxRetries,
		InitialInterval: cfg.InitialInterval,
		MaxInterval:     cfg.MaxInterval,
		Multiplier:      cfg.Multiplier,
		Logger:          cfg.Logger,
	}

	return retry.Middleware
}
