package saga

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// Runner manages the lifecycle of saga instances.
type Runner struct {
	store      Store
	dispatcher CommandDispatcher
	registry   map[string]Definition
	mu         sync.RWMutex
	config     runnerConfig
}

// NewRunner creates a new saga runner.
func NewRunner(store Store, dispatcher CommandDispatcher, opts ...RunnerOption) *Runner {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Runner{
		store:      store,
		dispatcher: dispatcher,
		registry:   make(map[string]Definition),
		config:     cfg,
	}
}

// Register adds a saga definition to the runner.
func (r *Runner) Register(def Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def == nil {
		return event.WrapRejection(ErrSagaNotFound, "saga.invalid_definition", "definition is nil")
	}

	sagaType := def.SagaType()
	if sagaType == "" {
		return event.WrapRejection(ErrSagaNotFound, "saga.invalid_type", "saga type is empty")
	}

	if _, exists := r.registry[sagaType]; exists {
		return event.WrapConflict(
			ErrSagaAlreadyExists,
			"saga.already_exists",
			"saga "+sagaType+" already exists",
		)
	}

	r.registry[sagaType] = def
	return nil
}

// Start begins a new saga instance with the given initial command.
func (r *Runner) Start(
	ctx context.Context,
	sagaType string,
	initialCommand command.Command,
) (*Instance, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "saga.start",
		trace.SpanKindClient,
		trace.WithAttributes(sagaAttrs(sagaType)...),
	)
	defer span.End()

	r.mu.RLock()
	def, ok := r.registry[sagaType]
	r.mu.RUnlock()

	if !ok {
		err := event.WrapRejection(
			ErrSagaNotRegistered,
			"saga.not_registered",
			"saga "+sagaType+" not registered",
		)
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	state := State{
		ID:          id.NewAggregateID(),
		SagaType:    sagaType,
		Status:      StatusPending,
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.saveSagaState(
		ctx,
		span,
		&state,
		"saga.save_failed",
		"save saga state",
	); err != nil {
		return nil, err
	}

	state.Status = StatusRunning
	if err := r.saveSagaState(
		ctx,
		span,
		&state,
		"saga.update_failed",
		"update saga status",
	); err != nil {
		return nil, err
	}

	instance := &Instance{
		State: state,
		Steps: def.Steps(),
	}

	r.logInfo("saga started", "type", sagaType, "id", instance.ID)

	if initialCommand != nil {
		if err := r.dispatcher.Dispatch(ctx, initialCommand); err != nil {
			cqrsotel.RecordError(span, err)

			instance.Status = StatusFailed
			instance.Err = err
			instance.ErrMsg = err.Error()
			instance.UpdatedAt = time.Now()
			_ = r.store.Save(ctx, &instance.State)
			r.logError("initial command failed", "type", sagaType, "id", instance.ID, "error", err)

			return instance, event.WrapInfrastructure(
				err,
				"saga.dispatch_failed",
				"dispatch initial command",
			)
		}
	}

	return instance, nil
}

func (r *Runner) logInfo(msg string, attrs ...any) {
	if r.config.logger != nil {
		r.config.logger.Info(msg, attrs...)
	}
}

func (r *Runner) logError(msg string, attrs ...any) {
	if r.config.logger != nil {
		r.config.logger.Error(msg, attrs...)
	}
}

func (r *Runner) saveSagaState(
	ctx context.Context,
	span trace.Span,
	state *State,
	code, msg string,
) error {
	if err := r.store.Save(ctx, state); err != nil {
		cqrsotel.RecordError(span, err)
		return event.WrapInfrastructure(err, code, msg)
	}
	return nil
}
