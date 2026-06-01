package pebble

import (
	"fmt"
	"log/slog"

	cqrsEvent "github.com/larsartmann/go-cqrs-lite/event"
)

// Backend identifies an event store backend.
type Backend string

// String returns the underlying string value.
func (b Backend) String() string { return string(b) }

const (
	// BackendPebble uses the embedded Pebble key-value store (default).
	BackendPebble Backend = "pebble"
	// BackendMemory uses an in-memory store (testing, development).
	BackendMemory Backend = "memory"
)

// EventStoreProvider creates an event store given a logger.
type EventStoreProvider func(logger *slog.Logger) (cqrsEvent.Store, error)

// Config holds configuration for constructing event store backends.
type Config struct {
	Backend  Backend
	Provider EventStoreProvider
}

// Option configures a Config.
type Option func(*Config)

// WithBackend sets the event store backend.
func WithBackend(b Backend) Option {
	return func(c *Config) { c.Backend = b }
}

// WithProvider sets a custom event store provider, overriding the backend.
func WithProvider(p EventStoreProvider) Option {
	return func(c *Config) { c.Provider = p }
}

// NewConfig builds a Config from options.
func NewConfig(opts ...Option) Config {
	cfg := Config{ //nolint:exhaustruct // Provider set via options below
		Backend: BackendPebble,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// NewEventStore creates an event store based on the configuration.
func NewEventStore(cfg Config, logger *slog.Logger) (cqrsEvent.Store, error) {
	if cfg.Provider != nil {
		return cfg.Provider(logger)
	}

	return nil, cqrsEvent.WrapInfrastructure(
		ErrPebbleProviderRequired,
		"storage.pebble_provider_required",
		fmt.Sprintf("backend %q requires a provider: use WithProvider", cfg.Backend),
	)
}

// Backward-compatible aliases.
type (
	PebbleBackend            = Backend
	PebbleEventStoreProvider = EventStoreProvider
	PebbleConfig             = Config
	PebbleOption             = Option
)

const (
	PebbleBackendPebble = BackendPebble
	PebbleBackendMemory = BackendMemory
)

//nolint:gochecknoglobals // backward-compatible aliases
var (
	WithPebbleBackend   = WithBackend
	WithPebbleProvider  = WithProvider
	NewPebbleConfig     = NewConfig
	NewPebbleEventStore = NewEventStore
)
