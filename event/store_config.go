package event

import "fmt"

// Backend identifies a persistent storage backend for event stores.
type Backend string

const (
	// BackendMemory uses an in-memory store (testing, development).
	BackendMemory Backend = "memory"
)

// StoreConfig holds configuration for constructing an event Store.
// Only built-in backends are supported via NewStoreFromConfig.
// External backends should implement the Store interface directly.
type StoreConfig struct {
	Backend Backend
}

// StoreOption configures a StoreConfig.
type StoreOption func(*StoreConfig)

// WithBackend sets the storage backend.
func WithBackend(b Backend) StoreOption {
	return func(c *StoreConfig) { c.Backend = b }
}

// NewStoreConfig builds a StoreConfig from options.
func NewStoreConfig(opts ...StoreOption) StoreConfig {
	cfg := StoreConfig{Backend: BackendMemory}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// NewStoreFromConfig creates a Store based on the configuration.
// For external backends, implement the Store interface directly
// rather than registering through this factory.
func NewStoreFromConfig(cfg StoreConfig) (Store, error) {
	switch cfg.Backend {
	case BackendMemory:
		return NewMemoryStore(), nil
	default:
		//nolint:err113 // dynamic error required to include backend name
		return nil, fmt.Errorf("unknown event store backend: %q", cfg.Backend)
	}
}
