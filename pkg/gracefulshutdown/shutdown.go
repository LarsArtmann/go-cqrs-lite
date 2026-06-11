package gracefulshutdown

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// defaultShutdownTimeout is the default timeout for graceful shutdown.
const defaultShutdownTimeout = 10 * time.Second

// ErrHookPanicked is returned when a shutdown hook panics.
var ErrHookPanicked = errors.New("shutdown hook panicked")

// Hook is a function that is called during shutdown.
type Hook func(ctx context.Context) error

// Config configures the graceful shutdown behavior.
type Config struct {
	Timeout time.Duration
	Signals []os.Signal
	Logger  *slog.Logger
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: defaultShutdownTimeout,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		Logger:  slog.Default(),
	}
}

// Shutdown waits for an OS signal and then runs all hooks with a timeout.
func Shutdown(cfg Config, hooks ...Hook) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, cfg.Signals...)

	sig := <-sigCh
	cfg.Logger.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var wg sync.WaitGroup

	errCh := make(chan error, len(hooks))

	for _, hook := range hooks {
		wg.Add(1)
		go func(h Hook) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case errCh <- ErrHookPanicked:
					default:
					}
				}
			}()

			err := h(ctx)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(hook)
	}

	wg.Wait()
	close(errCh)

	var hadError bool
	for err := range errCh {
		hadError = true

		cfg.Logger.Error("shutdown hook failed", "error", err)
	}

	if hadError {
		cfg.Logger.Warn("shutdown completed with errors")
	} else {
		cfg.Logger.Info("shutdown completed successfully")
	}
}
