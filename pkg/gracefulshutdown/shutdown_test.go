package gracefulshutdown

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestShutdown_RunsHooks(t *testing.T) {
	called := make(chan bool, 1)

	hook := func(_ context.Context) error {
		called <- true

		return nil
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	}()

	cfg := DefaultConfig()
	cfg.Signals = []os.Signal{syscall.SIGUSR1}
	cfg.Timeout = 1 * time.Second

	Shutdown(cfg, hook)

	select {
	case <-called:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("hook was not called")
	}
}

func TestShutdown_HookError(t *testing.T) {
	hook := func(_ context.Context) error {
		return errors.New("test error")
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
	}()

	cfg := DefaultConfig()
	cfg.Signals = []os.Signal{syscall.SIGUSR2}
	cfg.Timeout = 1 * time.Second

	Shutdown(cfg, hook)
	// Should complete without panic even with error
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timeout != 10*time.Second {
		t.Fatalf("expected 10s timeout, got %v", cfg.Timeout)
	}
	if len(cfg.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(cfg.Signals))
	}
}
