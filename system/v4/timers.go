package system

import (
	"context"
	"errors"
	"log/slog"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// errTimersStopped is the cancellation cause stopTimersLocked installs on the
// scheduler context. Any other cause means the context passed to
// System.Start was cancelled by its caller.
var errTimersStopped = errors.New("system: timers stopped")

// TimerEngine returns the engine deployments designate for timer storage:
// the engine named "timers" in DeploymentConfig.Engines when present, else
// the first configured engine. Consumers build engine-backed timer stores
// over it (ADR-0142):
//
//	store, _ := engine.NewTimerStore[CancelOrder](sys.TimerEngine())
//	sched := scheduling.New(store, dispatch)
//	sys.ManageTimers(sched)
//
// The returned engine implements metaengine.DueClaimer whenever its driver
// does (all first-party engines since the ADR-0142 wiring); NewTimerStore
// rejects it loudly otherwise.
func (s *System) TimerEngine() metaengine.Engine {
	for _, ne := range s.engines {
		if ne.name == "timers" {
			return ne.engine
		}
	}

	if len(s.engines) > 0 {
		return s.engines[0].engine
	}

	return nil
}

// TimerScheduler is the lifecycle surface scheduling.Scheduler[P] satisfies
// (Start blocks until ctx is cancelled); System runs it and stops it on
// GracefulClose/Close without needing the payload type parameter.
type TimerScheduler interface {
	Start(ctx context.Context) error
}

// ManageTimers hands a Scheduler's lifecycle to the composition root: it is
// started when the System starts and stopped — context cancelled and waited
// to completion — on both Close and GracefulClose. Timers become a
// system-owned concern, not a consumer-managed goroutine. Register before
// Start.
func (s *System) ManageTimers(sched TimerScheduler) {
	if sched == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.timers = append(s.timers, sched)
}

// startTimersLocked launches every managed scheduler under a context
// cancelled at shutdown. The caller must hold s.mu (Start does); a no-op
// when none are registered or already started.
func (s *System) startTimersLocked(parent context.Context) {
	if len(s.timers) == 0 || s.timerCancel != nil {
		return
	}

	ctx, cancel := context.WithCancelCause(parent)
	s.timerCancel = cancel

	for _, sched := range s.timers {
		s.timersAlive.Add(1)

		s.timersWG.Go(func() {
			defer s.timersAlive.Add(-1)

			// Start returns only on context cancellation (the documented
			// Scheduler contract); its error is terminal noise at shutdown.
			_ = sched.Start(ctx)
			s.warnIfStartContextCancelled(ctx)
		})
	}
}

// warnIfStartContextCancelled logs when a managed scheduler returned because
// the context passed to System.Start was cancelled while the system was not
// closed: timers then stay silent until Start is called again, and nothing
// else says so.
func (s *System) warnIfStartContextCancelled(ctx context.Context) {
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(cause, errTimersStopped) {
		return
	}

	slog.Warn(
		"system: managed timers stopped: the context passed to Start was cancelled while the system is not closed; "+
			"timers run only as long as that context, so due timers stay silent. "+
			"Pass a context that lives as long as the system (context.WithoutCancel) and stop with Close",
		"cause",
		cause,
	)
}

// stopTimers cancels the scheduler context and waits for every Start
// goroutine to return. Callers must NOT hold s.mu; use stopTimersLocked
// under Close's lock instead.
func (s *System) stopTimers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopTimersLocked()
}

// stopTimersLocked cancels the scheduler context and waits for every Start
// goroutine to return, so nothing dispatched by a managed scheduler can
// race the teardown that follows. The caller must hold s.mu; idempotent
// (a no-op when timers never started or are already stopped).
func (s *System) stopTimersLocked() {
	cancel := s.timerCancel
	s.timerCancel = nil

	if cancel != nil {
		cancel(errTimersStopped)
	}

	s.timersWG.Wait()
}
