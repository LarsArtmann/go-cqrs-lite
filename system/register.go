package system

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// RegisterDeciderOption tunes decider registration.
type RegisterDeciderOption func(*registerDeciderConfig)

type registerDeciderConfig struct {
	snapshotStrategy snapshot.SnapshotStrategy
}

// WithSnapshotStrategy sets the snapshot strategy for the decider. When the
// engine implements SnapshotBackend, this enables automatic snapshot creation.
// Without a strategy, the snapshot store is wired for reads but snapshots are
// never written automatically.
func WithSnapshotStrategy(s snapshot.SnapshotStrategy) RegisterDeciderOption {
	return func(c *registerDeciderConfig) { c.snapshotStrategy = s }
}

// ─── D10: Generic registration functions ───

// RegisterDecider registers a decider for a stream type. The System creates
// a [decider.Repository] for it, backed by the EventAdapter.
//
// Multiple commands targeting the same stream type share the same repository
// automatically — just call RegisterCommand with the same streamType in the
// system.Execute call.
//
// When the engine implements SnapshotBackend, the snapshot store and a default
// JSON codec are wired automatically. Pass [WithSnapshotStrategy] to enable
// automatic snapshot creation on writes.
func RegisterDecider[State any](
	sys *System,
	streamType string,
	d decider.Decider[State],
	opts ...RegisterDeciderOption,
) error {
	if sys.eventStore == nil {
		return ErrEventStoreMissing
	}

	cfg := registerDeciderConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var repoOpts []decider.RepositoryOption[State]

	// Wire snapshot store + codec if available (D12: snapshot support through System).
	if sys.snapStore != nil {
		repoOpts = append(
			repoOpts,
			decider.WithSnapshotStore[State](sys.snapStore),
			decider.WithCodec[State](codec.JSONCodec{}),
		)

		if cfg.snapshotStrategy != nil {
			repoOpts = append(repoOpts, decider.WithSnapshotStrategy[State](cfg.snapshotStrategy))
		}
	}

	repo, err := decider.NewRepository(sys.eventStore, sys.pubBus, d, repoOpts...)
	if err != nil {
		return fmt.Errorf("system: create repository for %q: %w", streamType, err)
	}

	sys.mu.Lock()
	sys.repos[streamType] = repo
	sys.deciders[streamType] = d
	sys.mu.Unlock()

	return nil
}

// RegisterCommand registers a typed command handler that returns an [Op].
// The System executes the Op via the decider repository registered for the
// Op's stream type (D10: declarative routing).
//
// The command type Cmd must implement [command.Command] (Type, StreamID, ID).
// The State type must match the decider registered for the stream type
// referenced in the handler's [Execute] call.
func RegisterCommand[Cmd command.Command, State any](
	sys *System,
	name command.Type,
	handler func(ctx context.Context, cmd Cmd) Op[State],
) error {
	sys.mu.Lock()
	sys.cmdHandlerCount++
	sys.mu.Unlock()

	return sys.cmdDisp.Register(name, func(ctx context.Context, cmd command.Command) error {
		typed, ok := any(cmd).(Cmd)
		if !ok {
			return fmt.Errorf("%w: got %T", ErrCommandTypeMismatch, cmd)
		}

		op := handler(ctx, typed)

		sys.mu.RLock()
		repoAny, exists := sys.repos[string(op.streamType)]
		sys.mu.RUnlock()

		if !exists {
			return fmt.Errorf("%w: stream type %q", ErrNoDecider, op.streamType)
		}

		repo, ok := repoAny.(*decider.Repository[State])
		if !ok {
			return fmt.Errorf("%w: stream type %q", ErrDeciderTypeMismatch, op.streamType)
		}

		return repo.Execute(ctx, op.streamID, op.streamType, op.decide)
	})
}

// RegisterQuery registers a typed query handler.
func RegisterQuery[Q any, R any](
	sys *System,
	name string,
	handler func(ctx context.Context, q Q) (R, error),
) error {
	return sys.qryDisp.Register(
		query.Type(name),
		func(ctx context.Context, q query.Query) (any, error) {
			typed, ok := q.(Q)
			if !ok {
				return nil, fmt.Errorf("%w: got %T", ErrQueryTypeMismatch, q)
			}

			return handler(ctx, typed)
		},
	)
}

// DispatchQuery dispatches a typed query and returns the result.
func DispatchQuery[Q query.Query, R any](ctx context.Context, sys *System, q Q) (R, error) {
	result, err := sys.qryDisp.Dispatch(ctx, q)
	if err != nil {
		var zero R

		return zero, err
	}

	typed, ok := result.(R)
	if !ok {
		var zero R

		return zero, fmt.Errorf("%w: got %T", ErrQueryResultMismatch, result)
	}

	return typed, nil
}
