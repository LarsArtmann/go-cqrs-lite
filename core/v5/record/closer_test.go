package record

import (
	"errors"
	"testing"
)

type fakeCloser struct {
	calls int
	err   error
}

func (f *fakeCloser) Close() error {
	f.calls++

	return f.err
}

func TestDeferClose_ClosesExactlyOnce(t *testing.T) {
	f := &fakeCloser{}

	DeferClose(f)

	if f.calls != 1 {
		t.Fatalf("Close called %d times, want 1", f.calls)
	}
}

func TestDeferClose_DiscardsError(t *testing.T) {
	f := &fakeCloser{err: errors.New("close failed")}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DeferClose panicked on close error: %v", r)
		}
	}()

	DeferClose(f)
}
