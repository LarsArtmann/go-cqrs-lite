// Package dispatcher provides shared infrastructure for CQRS dispatchers.
package dispatcher

import "sync"

// Lifecycle manages the closed state of a dispatcher with thread-safe access.
type Lifecycle struct {
	mu     sync.RWMutex
	closed bool
}

// Close marks the lifecycle as closed. It is safe to call multiple times.
func (l *Lifecycle) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

// IsClosed returns true if the lifecycle has been closed.
func (l *Lifecycle) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

// CheckClosed returns an error if the lifecycle is closed, or nil otherwise.
func (l *Lifecycle) CheckClosed(closedErr error) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return closedErr
	}
	return nil
}
