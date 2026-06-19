package projection

import "github.com/larsartmann/go-cqrs-lite/event/v2"

var (
	// ErrNilHandler is returned when a nil projection is registered.
	ErrNilHandler = event.NewRejection(
		"projection.nil_handler",
		"projection: nil handler",
	)

	// ErrNilSubscriber is returned when a nil event subscriber is passed to NewRunner.
	ErrNilSubscriber = event.NewInfrastructure(
		"projection.nil_subscriber",
		"projection: nil subscriber",
	)

	// ErrNilBus is deprecated: use ErrNilSubscriber instead.
	ErrNilBus = ErrNilSubscriber

	// ErrNilCheckpoint is returned when a nil checkpoint store is passed to NewRunner.
	ErrNilCheckpoint = event.NewInfrastructure(
		"projection.nil_checkpoint",
		"projection: nil checkpoint store",
	)

	// ErrNoProjections is returned when Run is called without any registered projections.
	ErrNoProjections = event.NewRejection(
		"projection.no_projections",
		"projection: no projections registered",
	)

	// ErrDuplicateProjection is returned when a projection with the same name is registered twice.
	ErrDuplicateProjection = event.NewConflict(
		"projection.duplicate_projection",
		"projection: duplicate projection name",
	)

	// ErrAlreadyRunning is returned when Run is called while the runner is already running.
	ErrAlreadyRunning = event.NewConflict(
		"projection.already_running",
		"projection: runner is already running",
	)

	// ErrReplayRequired is returned when RunLive is called before RunReplay completed.
	ErrReplayRequired = event.NewRejection(
		"projection.replay_required",
		"projection: RunReplay must be called before RunLive",
	)

	// ErrNilLeaderElection is returned when a nil LeaderElection is passed to NewDistributedRunner.
	ErrNilLeaderElection = event.NewInfrastructure(
		"projection.nil_leader_election",
		"projection: nil leader election",
	)

	// ErrNilRunner is returned when a nil Runner is passed to NewDistributedRunner.
	ErrNilRunner = event.NewInfrastructure(
		"projection.nil_runner",
		"projection: nil runner",
	)

	// ErrLeadershipLost is returned when leadership is lost during live processing.
	ErrLeadershipLost = event.NewConflict(
		"projection.leadership_lost",
		"projection: leadership lost during live processing",
	)
)
