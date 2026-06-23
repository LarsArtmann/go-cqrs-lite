package stack

import "io"

// MultiCloser closes multiple io.Closers in order. The first error encountered
// is returned; remaining closers are still called. Use [NewMultiCloser] to
// construct one, then pass it to [WithCloser].
type MultiCloser struct {
	closers []io.Closer
}

// NewMultiCloser creates a MultiCloser that closes all given closers in order.
func NewMultiCloser(closers ...io.Closer) *MultiCloser {
	return &MultiCloser{closers: closers}
}

func (m *MultiCloser) Close() error {
	var firstErr error

	for _, c := range m.closers {
		err := c.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

var _ io.Closer = (*MultiCloser)(nil)

// FuncCloser adapts a func() error into an io.Closer. It is a pointer-receiving
// struct (not a function type) so it is comparable and can be used as a map key
// for Bundle.Close deduplication. Use [NewFuncCloser] to construct one.
type FuncCloser struct {
	fn func() error
}

// NewFuncCloser wraps a func() error as an io.Closer.
func NewFuncCloser(fn func() error) *FuncCloser {
	return &FuncCloser{fn: fn}
}

func (c *FuncCloser) Close() error { return c.fn() }

var _ io.Closer = (*FuncCloser)(nil)
