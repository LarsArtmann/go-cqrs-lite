package middleware

import "log/slog"

type middlewareConfig struct {
	logger *slog.Logger

	// metricsEnabled is only consumed by [NewOTelBundle], which seeds it to
	// true before applying options (plain middleware paths ignore it).
	metricsEnabled bool
}

// Option configures middleware behavior.
type Option func(*middlewareConfig)

// WithLogger adds structured logging to middleware operations.
// When set, retry logs each attempt, recovery logs panics, and validation logs failures.
func WithLogger(logger *slog.Logger) Option {
	return func(c *middlewareConfig) {
		c.logger = logger
	}
}

func applyOptions(opts []Option) middlewareConfig {
	var c middlewareConfig

	for _, opt := range opts {
		opt(&c)
	}

	return c
}
