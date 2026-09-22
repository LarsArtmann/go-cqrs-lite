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

// HandlerRetryConfig configures handler retry behavior for transient
// failures in the Watermill router. It is Watermill's delivery-retry knob:
// the validated command/event/query-side retry with DLQ wiring is
// middleware.RetryConfig (different fields, different layer — the rename
// ends the E7 name collision).
type HandlerRetryConfig struct {
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

// RetryConfig is the pre-v5 spelling of [HandlerRetryConfig]. The name
// collided with middleware.RetryConfig (incompatible fields, E7); it is a
// type alias, so existing values keep compiling until the alias dies at the
// v6 marker.
//
// Deprecated: use [HandlerRetryConfig].
type RetryConfig = HandlerRetryConfig

// DefaultHandlerRetryConfig returns sensible defaults for CQRS retry
// behavior: 5 retries, 100ms initial interval, 10s max interval, 2.0x
// multiplier.
func DefaultHandlerRetryConfig() HandlerRetryConfig {
	return HandlerRetryConfig{
		MaxRetries:      5,                      //nolint:mnd // sensible default
		InitialInterval: 100 * time.Millisecond, //nolint:mnd // sensible default
		MaxInterval:     10 * time.Second,       //nolint:mnd // sensible default
		Multiplier:      2.0,                    //nolint:mnd // sensible default
	}
}

// DefaultRetryConfig is the pre-v5 spelling of [DefaultHandlerRetryConfig]
// (E7 rename).
//
// Deprecated: use [DefaultHandlerRetryConfig].
func DefaultRetryConfig() HandlerRetryConfig { return DefaultHandlerRetryConfig() }

// NewRetryMiddleware creates a retry middleware with exponential backoff
// using the provided configuration. The middleware retries on handler errors
// with increasing delays between attempts.
//
//	router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultHandlerRetryConfig()))
//	// or with custom config:
//	router.AddMiddleware(watermill.NewRetryMiddleware(watermill.HandlerRetryConfig{
//	    MaxRetries: 10,
//	    InitialInterval: 50 * time.Millisecond,
//	    MaxInterval: 5 * time.Second,
//	    Multiplier: 1.5,
//	}))
func NewRetryMiddleware(cfg HandlerRetryConfig) message.HandlerMiddleware {
	retry := middleware.Retry{
		MaxRetries:      cfg.MaxRetries,
		InitialInterval: cfg.InitialInterval,
		MaxInterval:     cfg.MaxInterval,
		Multiplier:      cfg.Multiplier,
		Logger:          cfg.Logger,
	}

	return retry.Middleware
}
