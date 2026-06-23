package postgres

import "io"

// multiCloser closes multiple io.Closers in order.
type multiCloser struct {
	closers []io.Closer
}

func (m *multiCloser) Close() error {
	var firstErr error

	for _, c := range m.closers {
		err := c.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

var _ io.Closer = (*multiCloser)(nil)

// funcCloser adapts a func() error into an io.Closer.
type funcCloser struct {
	fn func() error
}

func (c *funcCloser) Close() error { return c.fn() }

var _ io.Closer = (*funcCloser)(nil)
