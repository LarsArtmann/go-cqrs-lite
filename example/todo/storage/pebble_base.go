package storage

import (
	"log/slog"
	"sync"

	"github.com/cockroachdb/pebble"
)

// PebbleBase contains shared fields for Pebble-based storage.
type PebbleBase struct {
	db     *pebble.DB
	logger *slog.Logger
	mu     sync.RWMutex
	prefix string
}
