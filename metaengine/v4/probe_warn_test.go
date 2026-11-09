package metaengine_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// proberNoTrackerHost implements Engine + Prober but does NOT embed Calibration,
// so it cannot satisfy TrackerHost. This simulates the bug class where an engine
// uses a named cal field instead of embedding.
type proberNoTrackerHost struct {
	name     string
	probeRTT time.Duration
}

func (e *proberNoTrackerHost) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{
		Name:            e.name,
		RequiresNetwork: true,
		NetworkRTT:      10 * time.Millisecond,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
}

func (e *proberNoTrackerHost) Close() error { return nil }

func (e *proberNoTrackerHost) Probe(_ context.Context) (time.Duration, error) {
	return e.probeRTT, nil
}

// lockedBuffer makes the sink safe for concurrent use: while this test's
// handler is the process-wide slog default, PARALLEL tests in the package
// (e.g. the catchup stress writers) may log through it concurrently — a raw
// bytes.Buffer would be a data race (slog requires a concurrency-safe Writer).
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// TestProbeEngine_WarnOnMissingTrackerHost verifies that ProbeEngine emits a
// slog.Warn when an engine implements Prober but not TrackerHost — the exact
// symptom of a named-field-instead-of-embedded Calibration bug.
func TestProbeEngine_WarnOnMissingTrackerHost(t *testing.T) {
	t.Parallel()

	logs := &lockedBuffer{}
	orig := slog.Default()
	t.Cleanup(func() { slog.SetDefault(orig) })
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	eng := &proberNoTrackerHost{name: "fake-broken", probeRTT: 5 * time.Millisecond}
	ph := metaengine.ProbeEngine(eng, metaengine.WithProbeInterval(time.Second))
	t.Cleanup(ph.Stop)

	got := logs.String()
	if !strings.Contains(got, "not TrackerHost") {
		t.Fatalf("expected warning about missing TrackerHost, got: %s", got)
	}
	if !strings.Contains(got, "fake-broken") {
		t.Fatalf("expected engine name in warning, got: %s", got)
	}
}
