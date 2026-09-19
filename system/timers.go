package system

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

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
// started when the System starts and stopped (context cancelled, run to
// completion) on GracefulClose — timers become a system-owned concern, not
// a consumer-managed goroutine. Register before Start.
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

	ctx, cancel := context.WithCancel(parent)
	s.timerCancel = cancel

	for _, sched := range s.timers {
		go func() {
			// Start returns only on context cancellation (the documented
			// Scheduler contract); its error is terminal noise at shutdown.
			_ = sched.Start(ctx)
		}()
	}
}

// stopTimers cancels the scheduler context and waits for Start to return.
func (s *System) stopTimers() {
	s.mu.Lock()
	cancel := s.timerCancel
	s.timerCancel = nil
	s.mu.Unlock()

	if cancel == nil {
		return
	}

	cancel()
}
