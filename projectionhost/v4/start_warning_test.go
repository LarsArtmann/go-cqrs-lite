package projectionhost_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

const startContextWarning = "the Start context was cancelled while the host is not stopped"

// logBuffer is a log sink workers write to concurrently while the test polls.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.buf.Write(p)
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.buf.String()
}

// startWarningHost builds a two-projection host whose logs land in buf, so a
// Start-context cancellation is observed through the injected logger.
func startWarningHost(t *testing.T, buf *logBuffer) *projectionhost.Host {
	t.Helper()

	journal := &memoryJournal{}
	host, err := projectionhost.New(
		journal,
		newMemoryCheckpointStore(),
		projectionhost.WithLogger(slog.New(slog.NewTextHandler(buf, nil))),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, name := range []string{"alpha", "beta"} {
		if err := host.Register(&countingProjection{name: name}); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	return host
}

// stopAfterWorkersExit waits until no worker is running, then joins the
// goroutines via Stop; any pending warning has landed once this returns.
func stopAfterWorkersExit(t *testing.T, host *projectionhost.Host) {
	t.Helper()

	requireEventually(t, 3*time.Second, func() bool {
		for _, st := range host.Status() {
			switch st.Status {
			case projectionhost.WorkerRunning, projectionhost.WorkerLive,
				projectionhost.WorkerBackoff, projectionhost.WorkerDraining:
				return false
			}
		}

		return true
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func warningCount(buf *logBuffer) int {
	return strings.Count(buf.String(), startContextWarning)
}

// TestHost_StartContextCancelledWarnsOnce pins the #132 contract: cancelling
// the Start context without stopping the host is reported exactly once, with
// the fix hint, whichever exit path (stagger delay or drain) each worker took.
func TestHost_StartContextCancelledWarnsOnce(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		preCanc bool
	}{
		{name: "cancelled after Start"},
		{name: "cancelled before Start", preCanc: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := &logBuffer{}
			host := startWarningHost(t, buf)
			t.Cleanup(func() { _ = host.Stop() })

			ctx, cancelStart := context.WithCancel(context.Background())
			defer cancelStart()
			if tc.preCanc {
				cancelStart()
			}

			if err := host.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}

			if !tc.preCanc {
				cancelStart()
			}

			// The warning is logged from the worker goroutines, hence the poll.
			requireEventually(t, 3*time.Second, func() bool { return warningCount(buf) == 1 })

			stopAfterWorkersExit(t, host)

			if got := warningCount(buf); got != 1 {
				t.Fatalf("expected exactly one stale-projection warning, got %d; log:\n%s", got, buf)
			}

			if !strings.Contains(buf.String(), "context.WithoutCancel") ||
				!strings.Contains(buf.String(), "projections stay stale") {
				t.Fatalf("warning lacks the fix hint; log:\n%s", buf)
			}
		})
	}
}

// TestHost_StopDoesNotWarnStartContext pins the quiet path: a normal Stop
// cancels the worker context with the host's own cause and logs no warning.
func TestHost_StopDoesNotWarnStartContext(t *testing.T) {
	t.Parallel()

	buf := &logBuffer{}
	host := startWarningHost(t, buf)
	t.Cleanup(func() { _ = host.Stop() })

	if err := host.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 3*time.Second, func() bool {
		for _, st := range host.Status() {
			if st.Status != projectionhost.WorkerStopped {
				return false
			}
		}

		return true
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	stopAfterWorkersExit(t, host)

	if got := warningCount(buf); got != 0 {
		t.Fatalf("expected no stale-projection warning on Stop, got %d; log:\n%s", got, buf)
	}
}
