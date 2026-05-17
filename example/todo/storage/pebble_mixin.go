package storage

import (
	"log/slog"
	"sync"

	"github.com/cockroachdb/pebble"
)

// PebbleMixin contains shared fields for Pebble-based storage.
type PebbleMixin struct {
	db     *pebble.DB
	logger *slog.Logger
	mu     sync.RWMutex
	prefix string
}
