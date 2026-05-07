package storage

import (
	"fmt"
	"log/slog"

	cqrsEvent "github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

// PebbleBackend identifies an event store backend.
type PebbleBackend string

const (
	// PebbleBackendPebble uses the embedded Pebble key-value store (default).
	PebbleBackendPebble PebbleBackend = "pebble"
	// PebbleBackendMemory uses an in-memory store (testing, development).
	PebbleBackendMemory PebbleBackend = "memory"
)

// PebbleEventStoreProvider creates an event store given a logger.
type PebbleEventStoreProvider func(logger *slog.Logger) (cqrsEvent.Store, error)

// PebbleConfig holds configuration for constructing event store backends.
type PebbleConfig struct {
	Backend  PebbleBackend
	Provider PebbleEventStoreProvider
}

// PebbleOption configures a PebbleConfig.
type PebbleOption func(*PebbleConfig)

// WithPebbleBackend sets the event store backend.
func WithPebbleBackend(b PebbleBackend) PebbleOption {
	return func(c *PebbleConfig) { c.Backend = b }
}

// WithPebbleProvider sets a custom event store provider, overriding the backend.
func WithPebbleProvider(p PebbleEventStoreProvider) PebbleOption {
	return func(c *PebbleConfig) { c.Provider = p }
}

// NewPebbleConfig builds a PebbleConfig from options.
func NewPebbleConfig(opts ...PebbleOption) PebbleConfig {
	cfg := PebbleConfig{Backend: PebbleBackendPebble}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// NewPebbleEventStore creates an event store based on the configuration.
func NewPebbleEventStore(cfg PebbleConfig, logger *slog.Logger) (cqrsEvent.Store, error) {
	if cfg.Provider != nil {
		return cfg.Provider(logger)
	}

	switch cfg.Backend {
	case PebbleBackendPebble:
		return nil, fmt.Errorf(
			"%w: use WithPebbleProvider",
			ErrPebbleProviderRequired,
		)
	case PebbleBackendMemory:
		return memory.NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf(
			"%w: %q: use WithPebbleBackend or WithPebbleProvider",
			ErrUnknownBackend,
			cfg.Backend,
		)
	}
}
