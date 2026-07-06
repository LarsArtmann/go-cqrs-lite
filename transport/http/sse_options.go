package http

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// WithReconnectJournal enables Last-Event-ID reconnection.
// When a client sends the Last-Event-ID header, the broker replays missed
// events from the journal before starting live delivery.
// Events replayed from the journal are tracked by EventID and suppressed if
// they also arrive via the live bus (same dedup strategy as
// watermill.CatchUpSubscriber).
//
// replayLimit controls the maximum number of replayed events:
//   - replayLimit > 0: bounded replay capped at that many events.
//   - replayLimit <= 0: unlimited replay — events are streamed in batches
//     from the journal so memory stays bounded regardless of journal size.
//
// For a sensible bounded default, pass DefaultSSEReplayLimit (1000).
func WithReconnectJournal(journal event.SeekableJournal, replayLimit int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.journal = journal
		b.replayLimit = replayLimit
	}
}

// DefaultSSEReplayLimit is the suggested bounded replay cap for callers who
// want a finite replay window. Pass it to WithReconnectJournal, or pass <= 0
// for unlimited streaming replay.
const DefaultSSEReplayLimit = 1000

// SSEReplayIncompleteEvent is the SSE event type sent when journal replay is
// cut short by a timeout (see WithReplayTimeout). The event carries no id
// field so it does not advance the client's Last-Event-ID. Clients receiving
// this event know they are behind and should reconnect with their latest
// received EventID (or use a backfill endpoint) to catch up.
const SSEReplayIncompleteEvent = "cqrs.replay.incomplete"

// WithReplayByteBudget caps unlimited replay by total payload bytes instead
// of event count. When the cumulative size of replayed event payloads exceeds
// the budget, replay stops and an SSEReplayIncompleteEvent advisory is sent.
//
// This is safer than count-based batching (sseReplayBatchSize) for journals
// containing very large payloads (e.g. 1MB+ blob events): a fixed count of 500
// such events would consume 500MB. The default budget
// (sseDefaultReplayByteBudget = 8MB) is applied automatically for unlimited
// replay; pass an explicit value to override it.
//
// Applies only when replayLimit <= 0 (unlimited replay). Bounded replay
// (replayLimit > 0) is capped by event count and ignores this setting.
func WithReplayByteBudget(bytes int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayByteBudget = bytes
	}
}

// WithReplayTimeout sets the maximum duration for journal replay before
// switching to live delivery. If replay is not complete when the timeout
// fires, the broker sends an advisory SSEReplayIncompleteEvent and begins
// live streaming.
//
// A timeout of zero (the default) means no limit — replay runs until the
// journal is exhausted. Use a non-zero timeout for browser-facing SSE where
// handler starvation must be avoided (e.g. a client reconnecting after a long
// offline period with a very large journal).
func WithReplayTimeout(d time.Duration) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayTimeout = d
	}
}

// WithReplayMetrics installs OpenTelemetry instruments for SSE replay
// observability (duration histogram, events counter, incomplete counter).
// Pass a *ReplayMetrics from NewReplayMetrics; nil disables metrics (no-op).
//
// Without this option, replay records only span attributes — useful in traces
// but invisible to dashboards. This option promotes replay telemetry to
// first-class OTel instruments scrapeable by Prometheus.
func WithReplayMetrics(metrics *ReplayMetrics) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayMetrics = metrics
	}
}

// WithDedupRingCapacity overrides the default SSE dedup ring capacity
// (sseDedupRingCapacity = 1024). The ring bounds replay→live deduplication
// memory at ~capacity × 90 bytes.
//
// Increase if your live channel buffer (sseChannelBufSize) is raised above
// the default 100. Decrease for memory-constrained deployments with small
// journals. Values <= 0 fall back to the default.
func WithDedupRingCapacity(capacity int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.dedupRingCap = capacity
	}
}

// DefaultSSERetryInterval is the default reconnection interval sent via the
// SSE retry field when WithRetryInterval is not set.
const DefaultSSERetryInterval = 5 * time.Second

// WithRetryInterval sets the SSE retry field (milliseconds) the broker sends
// to browsers on connect, controlling how long they wait before reconnecting
// after a dropped connection. Default: 5s.
func WithRetryInterval(d time.Duration) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.retryInterval = d
	}
}

// sseReplayBatchSize is the default batch size for unlimited streaming replay.
// Each batch is fetched from the journal, written to the client, and flushed
// before the next batch is loaded — keeping memory bounded. For very large
// payloads (1MB+), prefer WithReplayByteBudget which bounds by total bytes
// rather than event count.
const sseReplayBatchSize = 500

// sseDefaultReplayByteBudget is the default byte budget applied when unlimited
// replay (replayLimit <= 0) is used without an explicit WithReplayByteBudget.
// 8MB accommodates ~5000 typical 1.5KB events while keeping per-client memory
// bounded. Callers can override via WithReplayByteBudget.
const sseDefaultReplayByteBudget = 8 * 1024 * 1024

// sseDedupRingCapacity is the maximum number of event IDs retained for
// replay→live deduplication. Only the tail of the replay stream can overlap
// with the live channel (events published during the replay window), and the
// live channel buffer is bounded at sseChannelBufSize. A ring of 1024 entries
// gives a 10x safety margin while bounding memory to ~90KB regardless of
// journal size.
const sseDedupRingCapacity = 1024

// WithEventFilter sets a predicate that controls which event types the broker
// forwards to connected clients. Events for which the predicate returns false
// are silently dropped (never written to any client channel). This is a
// broker-level filter — for per-client filtering, create multiple brokers.
//
// Example: only forward user-related events:
//
//	broker, _ := NewSSEBroker(bus, WithEventFilter(func(t event.Type) bool {
//	    return strings.HasPrefix(string(t), "user.")
//	}))
func WithEventFilter(fn func(event.Type) bool) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.eventFilter = fn
	}
}
