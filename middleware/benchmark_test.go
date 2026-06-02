package middleware

import (
	"context"
	"log/slog"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
