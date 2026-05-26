package saga

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// State is the fully serializable persistent state of a saga instance.
type State struct {
	ID          id.AggregateID
	SagaType    string
	Status      Status
	CurrentStep int
	ErrMsg      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
