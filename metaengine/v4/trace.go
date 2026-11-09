package metaengine

import (
	"encoding/json/v2"
	"io"
	"sync"
	"time"
)

// Trace op kinds (TraceOp.Op).
const (
	// TraceOpApply records one applied event (name = event type).
	TraceOpApply = "apply"
	// TraceOpQuery records one executed query (name = query/collection name).
	TraceOpQuery = "query"
)

// traceVersion is the JSONL schema version stamped on every record.
const traceVersion = 1

// TraceOp is one recorded workload operation (METAENGINE-LAYOUT-ROLES.md §5).
// The JSON-lines format is one JSON object per line:
//
//	{"v":1,"ts":"...","op":"apply","name":"TaskCreated","dur_ms":0.42,"err":""}
//
// Payloads are intentionally not serialized — arbitrary Go values are not
// JSON-round-trip safe. Replay synthesizes payloads via a caller-supplied
// factory (StoreTraceSink); the trace carries the workload's shape and mix,
// which is what benchmark calibration needs.
type TraceOp struct {
	V     int       `json:"v"`
	TS    time.Time `json:"ts"`
	Op    string    `json:"op"`
	Name  string    `json:"name"`
	DurMS float64   `json:"dur_ms"`
	Err   string    `json:"err,omitempty"`
}

// TraceRecorder records a Store's applied events and executed queries as
// JSON-lines to an io.Writer. Attach with RecordTrace; detach with Close.
//
// The recorder chains any hooks already installed on the Store (WithHooks
// replaces hooks wholesale, so the recorder wraps and forwards). While
// recording, all queries invoke the execute hook regardless of any previous
// SlowQueryThreshold — a trace that silently drops fast queries is useless.
type TraceRecorder struct {
	mu    sync.Mutex
	w     io.Writer
	store *Store
	prev  *Hooks
	err   error
}

// RecordTrace attaches a TraceRecorder to the store. The returned recorder
// must be Closed when done to restore the previous hooks.
func RecordTrace(store *Store, w io.Writer) *TraceRecorder {
	tr := &TraceRecorder{w: w, store: store}

	if store.hooks != nil {
		prevCopy := *store.hooks
		tr.prev = &prevCopy
	}

	hooks := Hooks{
		OnApply: func(eventType string, d time.Duration, err error) {
			tr.record(TraceOpApply, eventType, d, err)
			if tr.prev != nil && tr.prev.OnApply != nil {
				tr.prev.OnApply(eventType, d, err)
			}
		},
		OnExecute: func(collection string, pattern ReadPattern, d time.Duration, err error) {
			tr.record(TraceOpQuery, collection, d, err)
			if tr.prev != nil && tr.prev.OnExecute != nil {
				tr.prev.OnExecute(collection, pattern, d, err)
			}
		},
	}

	if tr.prev != nil {
		hooks.OnFold = tr.prev.OnFold
		hooks.Logger = tr.prev.Logger
	}

	WithHooks(store, hooks)

	return tr
}

func (tr *TraceRecorder) record(op, name string, d time.Duration, err error) {
	opRecord := TraceOp{
		V:     traceVersion,
		TS:    time.Now(),
		Op:    op,
		Name:  name,
		DurMS: float64(d.Microseconds()) / 1e3,
	}

	if err != nil {
		opRecord.Err = err.Error()
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	if line, encErr := json.Marshal(opRecord); encErr != nil {
		if tr.err == nil {
			tr.err = encErr
		}
	} else if _, writeErr := tr.w.Write(append(line, '\n')); writeErr != nil && tr.err == nil {
		tr.err = writeErr
	}
}

// Err returns the first trace-encoding error (writer failure), if any. A nil
// result means every recorded op reached the writer.
func (tr *TraceRecorder) Err() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	return tr.err
}

// Close detaches the recorder and restores the hooks that were installed
// before RecordTrace. The underlying writer is not closed.
func (tr *TraceRecorder) Close() {
	if tr.prev != nil {
		WithHooks(tr.store, *tr.prev)
		return
	}

	WithHooks(tr.store, Hooks{})
}
