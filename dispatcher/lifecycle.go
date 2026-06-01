package dispatcher

import "sync"

// Lifecycle provides thread-safe closed state management for composable types.
type Lifecycle struct {
	mu     sync.RWMutex
	closed bool
}

// Close marks the lifecycle as closed. It is safe to call multiple times.
func (m *Lifecycle) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true

	return nil
}

// IsClosed returns true if the lifecycle has been closed.
func (m *Lifecycle) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.closed
}

// CheckClosed returns an error if the lifecycle is closed, or nil otherwise.
func (m *Lifecycle) CheckClosed(closedErr error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return closedErr
	}

	return nil
}
