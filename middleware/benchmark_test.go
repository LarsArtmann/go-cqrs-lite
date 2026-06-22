package middleware

import (
	"context"
	"log/slog"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func benchCommandMiddleware(b *testing.B, mw command.Middleware) {
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandLogging(b *testing.B) {
	b.ReportAllocs()
	logger := slog.New(slog.DiscardHandler)
	benchCommandMiddleware(b, CommandLogging(logger))
}

func BenchmarkCommandRecovery(b *testing.B) {
	b.ReportAllocs()
	benchCommandMiddleware(b, CommandRecovery())
}

func BenchmarkCommandValidation(b *testing.B) {
	b.ReportAllocs()
	benchCommandMiddleware(b, CommandValidation(func(_ command.Command) error { return nil }))
}

func BenchmarkCommandRetry(b *testing.B) {
	b.ReportAllocs()
	benchCommandMiddleware(b, CommandRetry(DefaultRetryConfig()))
}

func BenchmarkCircuitBreaker_HappyPath(b *testing.B) {
	b.ReportAllocs()
	mw := CommandCircuitBreaker(DefaultCircuitBreakerConfig())
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCircuitBreaker_Concurrent(b *testing.B) {
	b.ReportAllocs()

	mw := CommandCircuitBreaker(DefaultCircuitBreakerConfig())
	handler := mw(NoopCommandHandler())
	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		cmd := &testCommand{aggregateID: id.NewAggregateID()}

		for pb.Next() {
			_ = handler(ctx, cmd)
		}
	})
}
