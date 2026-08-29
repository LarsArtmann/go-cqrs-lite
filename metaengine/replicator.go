package metaengine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// replicationBufferJobs bounds a shadow engine's pending replication jobs.
// Overflow marks the engine stale (loud halt) rather than blocking writes.
const replicationBufferJobs = 1024

// replicationRetries is the per-job retry budget before a shadow goes stale.
const replicationRetries = 3

// replicationOpTimeout bounds ALL attempts for one job. A hung (not erroring)
// shadow write would otherwise hold the query fold lock — which primaries
// share — indefinitely. The timeout converts hangs into errors, which the
// retry budget then converts into stale. Total primary-path exposure to one
// hung shadow write is bounded by this timeout (METAENGINE-LAYOUT-ROLES §3.3).
const replicationOpTimeout = 3 * time.Second

// repJob is one accepted-but-not-yet-applied event destined for a shadow engine.
type repJob struct {
	eventType string
	rec       record.Record
	payload   any
}

// replicator mirrors every applied event into one shadow engine
// (METAENGINE-LAYOUT-ROLES.md §3). Failure-isolated: tryEnqueue never blocks,
// and every halt path (overflow, permanent failure, shutdown) is loud — a
// stale engine is visibly behind, never silently wrong.
type replicator struct {
	store  *Store
	name   string
	engine Engine

	ch     chan repJob
	stopCh chan struct{}
	doneCh chan struct{}

	baseCtx context.Context
	cancel  context.CancelFunc

	mu        sync.Mutex
	accepting bool
	stale     bool
	lastErr   string

	applied atomic.Int64
}

// newReplicator creates a shadow replicator. The caller must hold the store
// write lock and have registered the engine already.
func (s *Store) newReplicatorLocked(engine Engine) *replicator {
	ctx, cancel := context.WithCancel(context.Background())

	return &replicator{
		store: s, name: engine.Profile().Name, engine: engine,
		ch:     make(chan repJob, replicationBufferJobs),
		stopCh: make(chan struct{}), doneCh: make(chan struct{}),
		baseCtx: ctx, cancel: cancel, accepting: true,
	}
}

// run is the applier goroutine. It consumes accepted jobs in order; on stop it
// drains everything accepted unless halted (stale), in which case the backlog
// is abandoned — the engine is marked stale either way.
func (r *replicator) run() {
	defer close(r.doneCh)

	for {
		select {
		case job := <-r.ch:
			if err := r.applyWithRetry(r.baseCtx, job); err != nil {
				r.failHalt(err)
				return
			}

			r.applied.Add(1)
		case <-r.stopCh:
			r.drain()
			return
		}
	}
}

// drain applies every still-queued accepted job, then the caller exits. A
// stale replicator abandons its backlog; a drain-time failure halts it.
func (r *replicator) drain() {
	if r.isStale() {
		return
	}

	for {
		select {
		case job := <-r.ch:
			if err := r.applyWithRetry(r.baseCtx, job); err != nil {
				r.failHalt(err)
				return
			}

			r.applied.Add(1)
		default:
			return
		}
	}
}

func (r *replicator) applyWithRetry(ctx context.Context, job repJob) error {
	var err error

	opCtx, cancel := context.WithTimeout(ctx, replicationOpTimeout)
	defer cancel()

	for attempt := range replicationRetries {
		if err = r.applyJob(opCtx, job); err == nil {
			return nil
		}

		select {
		case <-time.After(time.Duration(attempt+1) * 50 * time.Millisecond):
		case <-opCtx.Done():
			return err
		}
	}

	return err
}

// applyJob applies one event's fold tasks to the shadow engine inside a single
// engine transaction when supported. Lock-free: fold tasks come from the
// immutable task snapshot (see task_snapshot.go).
func (r *replicator) applyJob(ctx context.Context, job repJob) error {
	return r.applyJobFilter(ctx, job, nil)
}

// applyJobFilter is applyJob restricted to the named queries (nil = all). The
// demotion catch-up replays only the collections the demoted engine never
// served; everything it already holds must not be re-applied.
func (r *replicator) applyJobFilter(
	ctx context.Context,
	job repJob,
	queryFilter map[string]bool,
) error {
	tasks := filterTasks(r.store.tasksFor(job.eventType), queryFilter)
	if len(tasks) == 0 {
		return nil
	}

	apply := func(tctx context.Context) error {
		for _, t := range tasks {
			sq := shadowQuery{queryMeta: t.q, eng: r.engine}

			l := r.store.foldLocks.get(t.q.QueryName())
			l.Lock()

			err := r.store.applyFold(tctx, sq, t.fold, job.rec, job.payload)

			l.Unlock()

			if err != nil {
				return fmt.Errorf("query %q fold for %s: %w", t.q.QueryName(), job.eventType, err)
			}
		}

		return nil
	}

	if tx, ok := r.engine.(Transactional); ok {
		if err := tx.RunInTx(ctx, apply); err != nil {
			return fmt.Errorf("replication batch on %s: %w", r.name, err)
		}

		return nil
	}

	return apply(ctx)
}

// tryEnqueue offers a job without blocking. Called under the store read lock,
// making enqueue atomic against PromoteEngine's write-locked transition.
// Overflow marks the engine stale and halts it.
func (r *replicator) tryEnqueue(job repJob) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.accepting {
		return
	}

	select {
	case r.ch <- job:
	default:
		r.accepting = false
		r.stale = true
		r.lastErr = fmt.Sprintf(
			"replication buffer full (%d jobs) — engine is stale; recover via remove+re-add+backfill",
			replicationBufferJobs,
		)
		r.cancel()
		close(r.stopCh)
	}
}

// halt stops the replicator immediately, abandoning any backlog. Used for
// shutdown and engine removal.
func (r *replicator) halt() {
	r.mu.Lock()

	if !r.accepting {
		r.mu.Unlock()
		return
	}

	r.accepting = false
	r.mu.Unlock()

	r.cancel()
	close(r.stopCh)
}

// stopAndDrain rejects new jobs and waits until every accepted job is applied.
// Fails when the engine is (or becomes) stale. Must NOT be called while the
// applier might need the store read lock (PromoteEngine holds the write lock).
func (r *replicator) stopAndDrain(ctx context.Context) error {
	r.mu.Lock()

	if r.stale {
		err := fmt.Errorf("engine is stale: %s", r.lastErr)
		r.mu.Unlock()

		return err
	}

	if r.accepting {
		r.accepting = false
		r.mu.Unlock()
		close(r.stopCh)
	} else {
		r.mu.Unlock()
	}

	select {
	case <-r.doneCh:
	case <-ctx.Done():
		return fmt.Errorf("drain timed out: %w", ctx.Err())
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stale {
		return fmt.Errorf("replication failed during drain: %s", r.lastErr)
	}

	return nil
}

func (r *replicator) failHalt(err error) {
	r.mu.Lock()
	r.accepting = false
	r.stale = true
	r.lastErr = err.Error()
	r.mu.Unlock()

	slog.Warn("metaengine: replication halted — engine marked stale",
		"engine", r.name, "error", err.Error())
}

func (r *replicator) isStale() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.stale
}

func (r *replicator) lastError() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lastErr
}

func (r *replicator) queued() int         { return len(r.ch) }
func (r *replicator) appliedCount() int64 { return r.applied.Load() }

// shadowQuery redirects one query's fold writes to a shadow engine. Everything
// else (name, folds, config) delegates to the planned query, so the shadow
// hosts the same collection names with the same fold semantics.
type shadowQuery struct {
	queryMeta

	eng Engine
}

func (s shadowQuery) QueryEngine() Engine { return s.eng }
func (s shadowQuery) isShadow() bool      { return true }
