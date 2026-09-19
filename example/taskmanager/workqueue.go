package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
	qsqlite "github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4"
)

// The engine-backed durable work queue (plan T22, ADR-0142): the deriver's
// follow-up commands no longer ride a fire-and-forget goroutine — they are
// durable tasks on queue/sqlite (which registers as the "queue-sqlite"
// metaengine driver and runs the ONE claimkit claim/dedup/facts runtime over
// the same database). Claims are lease-fenced (a crashed worker's task
// becomes claimable again after the lease lapses), failures retry with
// backoff, and exhausted tasks dead-letter instead of vanishing. The dedup
// key makes repeated task.created projections converge to one assignment —
// the old goroutine could double-dispatch.

const (
	assignmentQueueType = "task.assign"
	queueOwner          = "assign-worker-1"
	claimLease          = 30 * time.Second
	idlePollInterval    = 100 * time.Millisecond
	assignmentBackoff   = time.Second
	assignmentAttempts  = 3
)

// AssignmentJob is the queued payload: assign task TaskID to AssigneeID.
type AssignmentJob struct {
	TaskID     string `json:"task_id"`
	AssigneeID string `json:"assignee_id"`
}

// WorkQueue owns the durable assignment queue and its worker loop.
type WorkQueue struct {
	store  *qsqlite.Store[AssignmentJob]
	disp   *command.Dispatcher
	logger *slog.Logger
	cancel context.CancelFunc
	done   chan struct{}
}

// openWorkQueue opens (and migrates) the assignment queue on the same SQLite
// database the event journal lives in when DATABASE_PATH is a file; with the
// in-memory default it is a fresh private database.
func openWorkQueue(databasePath string, disp *command.Dispatcher, logger *slog.Logger) (*WorkQueue, error) {
	store, err := qsqlite.Open[AssignmentJob](databasePath)
	if err != nil {
		return nil, fmt.Errorf("open assignment queue: %w", err)
	}

	return &WorkQueue{store: store, disp: disp, logger: logger, done: make(chan struct{})}, nil
}

// enqueueAssignment persists the assignment as a due task. Idempotent by
// DedupKey: repeated task.created projections for the same task enqueue
// nothing new.
func (wq *WorkQueue) enqueueAssignment(ctx context.Context, job AssignmentJob) error {
	_, err := wq.store.Enqueue(ctx, task.New[AssignmentJob]{
		Type:        assignmentQueueType,
		Payload:     job,
		DedupKey:    "assign:" + job.TaskID,
		MaxAttempts: assignmentAttempts,
	})
	if err != nil {
		return fmt.Errorf("enqueue assignment %s: %w", job.TaskID, err)
	}

	return nil
}

// Start launches the worker loop in a background goroutine.
func (wq *WorkQueue) Start(ctx context.Context) {
	ctx, wq.cancel = context.WithCancel(ctx)

	go func() {
		defer close(wq.done)

		wq.run(ctx)
	}()
}

// Stop cancels the worker, waits for it, and releases the queue database.
func (wq *WorkQueue) Stop() error {
	if wq.cancel != nil {
		wq.cancel()
		<-wq.done
	}

	//cqrs-lint:ignore(C023) library code or intentional pattern
	return wq.store.Close()
}

// run is the claim-work-complete loop: lease-fenced ClaimDue, dispatch the
// command, Complete on success, Fail (backoff + attempt count) on error.
func (wq *WorkQueue) run(ctx context.Context) {
	for {
		claim, err := wq.store.ClaimDue(ctx, queueOwner, claimLease)
		if errors.Is(err, queue.ErrNoTaskDue) {
			if !sleepCtx(ctx, idlePollInterval) {
				return
			}

			continue
		}

		if err != nil {
			if ctx.Err() != nil {
				return
			}

			wq.logger.Error("assignment queue: claim", "error", err)

			if !sleepCtx(ctx, idlePollInterval) {
				return
			}

			continue
		}

		wq.process(ctx, claim)
	}
}

func (wq *WorkQueue) process(ctx context.Context, claim queue.Claim[AssignmentJob]) {
	job := claim.Task.Payload

	streamID, err := id.ParseStreamID(job.TaskID)
	if err != nil {
		// Malformed IDs never succeed: dead-letter immediately.
		if dlqErr := wq.store.FailPermanent(ctx, claim.Task.ID, claim.Token,
			fmt.Sprintf("parse task id: %v", err), nil); dlqErr != nil {
			wq.logger.Error("assignment queue: dead-letter", "task", job.TaskID, "error", dlqErr)
		}

		return
	}

	base, err := command.New(cmdAssignTask, streamID)
	if err != nil {
		// Malformed IDs never succeed: dead-letter immediately.
		if dlqErr := wq.store.FailPermanent(ctx, claim.Task.ID, claim.Token,
			fmt.Sprintf("build assign command: %v", err), nil); dlqErr != nil {
			wq.logger.Error("assignment queue: dead-letter", "task", job.TaskID, "error", dlqErr)
		}

		return
	}

	if err := wq.disp.Dispatch(ctx, AssignTaskCmd{
		BasicCommand: base,
		AssigneeID:   job.AssigneeID,
	}); err != nil {
		wq.logger.Warn("assignment dispatch failed, retrying with backoff",
			"task", job.TaskID, "attempt", claim.Task.Attempts+1, "error", err)

		if failErr := wq.store.Fail(ctx, claim.Task.ID, claim.Token,
			err.Error(), assignmentBackoff, nil); failErr != nil {
			wq.logger.Error("assignment queue: fail", "task", job.TaskID, "error", failErr)
		}

		return
	}

	if err := wq.store.Complete(ctx, claim.Task.ID, claim.Token, nil); err != nil {
		wq.logger.Error("assignment queue: complete", "task", job.TaskID, "error", err)
	}
}

// sleepCtx waits for d or ctx cancellation; reports whether the wait elapsed.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
