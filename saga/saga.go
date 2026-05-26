package saga

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Status represents the lifecycle state of a saga instance.
type Status string

const (
	StatusPending       Status = "pending"
	StatusRunning       Status = "running"
	StatusStepCompleted Status = "step_completed"
	StatusCompensating  Status = "compensating"
	StatusCompleted     Status = "completed"
	StatusFailed        Status = "failed"
)

// Step defines a single step in a saga process.
type Step struct {
	Name       string
	Action     func(ctx context.Context, instanceID id.AggregateID) command.Command
	Compensate func(ctx context.Context, instanceID id.AggregateID) command.Command
	Timeout    time.Duration
}

// Instance represents the persistent state of a running saga.
type Instance struct {
	ID          id.AggregateID
	SagaType    string
	Status      Status
	CurrentStep int
	Steps       []Step
	Err         error
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Definition describes a saga type that can be registered with a Runner.
type Definition interface {
	SagaType() string
	Steps() []Step
}
