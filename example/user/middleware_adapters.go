package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

type printMetricsRecorder struct{}

func (p *printMetricsRecorder) Observe(
	_ context.Context,
	name string,
	duration time.Duration,
	_ ...string,
) {
	fmt.Printf("  [metrics] %s → %s\n", name, duration.Round(time.Microsecond))
}

var _ middleware.MetricsRecorder = (*printMetricsRecorder)(nil)
