package storage

import (
	"log/slog"
	"sync"

	"github.com/cockroachdb/pebble"
)

// PebbleHandle contains shared fields for Pebble-based storage.
type PebbleHandle struct {
	db     *pebble.DB
	logger *slog.Logger
	mu     sync.RWMutex
	prefix string
}
