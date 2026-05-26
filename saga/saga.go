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
	StatusPending      Status = "pending"
	StatusRunning      Status = "running"
	StatusCompensating Status = "compensating"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
)

// Step defines a single step in a saga process.
type Step struct {
	Name       string
	Action     func(ctx context.Context, instanceID id.AggregateID) command.Command
	Compensate func(ctx context.Context, instanceID id.AggregateID) command.Command
	Timeout    time.Duration
}

// Instance represents the runtime view of a running saga, assembled by Runner.
type Instance struct {
	State
	Steps []Step
	Err   error
}

// Definition describes a saga type that can be registered with a Runner.
type Definition interface {
	SagaType() string
	Steps() []Step
}
