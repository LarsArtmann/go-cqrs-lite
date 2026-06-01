package dispatcher

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestDispatcherWithCatalog_Init(t *testing.T) {
	t.Parallel()

	var dwc DispatcherWithCatalog[string, HandlerMeta, func() string, func(func() string) func() string]
	dwc.Init()

	if dwc.Inner() == nil {
		t.Error("Init() should initialize inner Dispatcher")
	}
}

func TestDispatcherWithCatalog_Inner(t *testing.T) {
	t.Parallel()

	var dwc DispatcherWithCatalog[string, HandlerMeta, func() string, func(func() string) func() string]
	dwc.Init()

	inner := dwc.Inner()
	if inner == nil {
		t.Fatal("Inner() returned nil")
	}
}

func TestConcurrentDispatch(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[func() string, func(func() string) func() string]()

	for i := range 10 {
		key := string(rune('a' + i))
		handler := func() string { return key }

		err := d.Register(
			key,
			handler,
			func(m func(func() string) func() string, h func() string) func() string {
				return h
			},
		)
		if err != nil {
			t.Fatalf("Register(%q): %v", key, err)
		}
	}

	var (
		wg      sync.WaitGroup
		success atomic.Int32
	)

	for i := range 100 {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := string(rune('a' + (i % 10)))
			h, err := d.Dispatch(key)
			if err != nil {
				t.Errorf("Dispatch(%q): %v", key, err)

				return
			}

			result := h()
			if result != key {
				t.Errorf("handler() = %q, want %q", result, key)

				return
			}

			success.Add(1)
		}(i)
	}

	wg.Wait()

	if got := success.Load(); got != 100 {
		t.Errorf("successful dispatches = %d, want 100", got)
	}
}
