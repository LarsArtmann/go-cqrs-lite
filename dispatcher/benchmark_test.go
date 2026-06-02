package dispatcher_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
)

type benchHandler func() error

func BenchmarkNewDispatcher(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d := dispatcher.NewDispatcher[benchHandler, func(benchHandler) benchHandler]()
		_ = d.Close()
	}
}

func BenchmarkDispatcher_Register(b *testing.B) {
	b.ReportAllocs()
	wrap := func(m func(benchHandler) benchHandler, h benchHandler) benchHandler { return h }
	noop := func() error { return nil }

	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		d := dispatcher.NewDispatcher[benchHandler, func(benchHandler) benchHandler]()
		b.StartTimer()

		err := d.Register("handler", noop, wrap)
		if err != nil {
			b.Fatalf("Register: %v", err)
		}

		b.StopTimer()
		_ = d.Close()
		b.StartTimer()
	}
}

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	b.ReportAllocs()
	d := dispatcher.NewDispatcher[benchHandler, func(benchHandler) benchHandler]()
	b.Cleanup(func() { _ = d.Close() })

	wrap := func(m func(benchHandler) benchHandler, h benchHandler) benchHandler { return h }
	noop := func() error { return nil }

	err := d.Register("bench", noop, wrap)
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := d.Dispatch("bench")
		if err != nil {
			b.Fatalf("Dispatch: %v", err)
		}
	}
}

func BenchmarkDispatcher_Close(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d := dispatcher.NewDispatcher[benchHandler, func(benchHandler) benchHandler]()
		_ = d.Close()
	}
}
