package aggregate

import (
	"time"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type TodoState struct {
	Title       domain.Title
	Description domain.Description
	Status      domain.TodoStatus
	Priority    domain.Priority
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	Deleted     bool
}

func (s TodoState) IsNew() bool {
	return s.Title == "" && s.CreatedAt.IsZero()
}

func NewTodoDecider() decider.Decider[TodoState] {
	return decider.Decider[TodoState]{
		Initial: TodoState{},
		Fold:    Fold,
	}
}

func Fold(state TodoState, evt event.Event) (TodoState, error) {
	if len(evt.Payload()) == 0 {
		return state, nil
	}

	var payload TodoPayload

	codec := codecpkg.JSONCodec{}
	if err := codec.Decode(evt.Payload(), &payload); err != nil {
		return state, event.Newf(
			event.Infrastructure,
			"todo.aggregate.decider.1",
			"decode payload for event %s: %v",
			evt.Type(),
			err,
		)
	}

	switch evt.Type() {
	case EventCreated:
		return foldCreated(payload), nil
	case EventUpdated:
		return foldUpdated(state, payload), nil
	case EventStatusChanged, EventCompleted:
		return foldStatusChanged(state, payload), nil
	case EventDeleted:
		state.Deleted = true
		state.UpdatedAt = payload.UpdatedAt

		return state, nil
	default:
		return state, event.Newf(
			event.Infrastructure,
			"todo.aggregate.decider.2",
			"%v: %s",
			ErrUnknownEventType,
			evt.Type(),
		)
	}
}

func foldCreated(p TodoPayload) TodoState {
	return TodoState{
		Title:       p.Title,
		Description: p.Description,
		Status:      p.Status,
		Priority:    p.Priority,
		Tags:        p.Tags,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		CompletedAt: p.CompletedAt,
	}
}

func foldUpdated(state TodoState, p TodoPayload) TodoState {
	state.Title = p.Title
	state.Description = p.Description
	state.UpdatedAt = p.UpdatedAt

	return state
}

func foldStatusChanged(state TodoState, p TodoPayload) TodoState {
	state.Status = p.Status
	state.UpdatedAt = p.UpdatedAt
	state.CompletedAt = p.CompletedAt

	return state
}

func (s TodoState) ToDomain(todoID domain.TodoID, version int64) *domain.Todo {
	return &domain.Todo{
		ID:          todoID,
		Title:       s.Title,
		Description: s.Description,
		Status:      s.Status,
		Priority:    s.Priority,
		Tags:        s.Tags,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		CompletedAt: s.CompletedAt,
		Version:     version,
	}
}

func DecideCreate(
	aggID id.AggregateID,
	title domain.Title,
	description domain.Description,
	priority domain.Priority,
	tags []string,
) decider.DecideFunc[TodoState] {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if !state.IsNew() {
			return nil, ErrTodoAlreadyExists
		}

		if title == "" {
			return nil, domain.ErrEmptyTitle
		}

		now := time.Now().UTC()
		payload := TodoPayload{
			Title: title, Description: description,
			Status: domain.StatusPending, Priority: priority,
			Tags: tags, CreatedAt: now, UpdatedAt: now,
		}

		evt, err := newEventFromPayload(EventCreated, aggID, int(version)+1, payload)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

func DecideUpdate(
	aggID id.AggregateID,
	title domain.Title,
	description domain.Description,
) decider.DecideFunc[TodoState] {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Deleted {
			return nil, domain.ErrNotFound
		}

		if title == "" {
			return nil, domain.ErrEmptyTitle
		}

		payload := TodoPayload{
			Title: title, Description: description,
			Status: state.Status, Priority: state.Priority,
			Tags: state.Tags, CreatedAt: state.CreatedAt,
			UpdatedAt: time.Now().UTC(), CompletedAt: state.CompletedAt,
		}

		evt, err := newEventFromPayload(EventUpdated, aggID, int(version)+1, payload)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

func DecideDelete(aggID id.AggregateID) decider.DecideFunc[TodoState] {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Deleted {
			return nil, domain.ErrNotFound
		}

		if state.IsNew() {
			return nil, domain.ErrNotFound
		}

		payload := TodoPayload{
			Title: state.Title, Description: state.Description,
			Status: state.Status, Priority: state.Priority,
			Tags: state.Tags, CreatedAt: state.CreatedAt,
			UpdatedAt: time.Now().UTC(), CompletedAt: state.CompletedAt,
		}

		evt, err := newEventFromPayload(EventDeleted, aggID, int(version)+1, payload)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

func DecideChangeStatus(
	aggID id.AggregateID,
	status domain.TodoStatus,
) decider.DecideFunc[TodoState] {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Deleted {
			return nil, domain.ErrNotFound
		}

		if state.IsNew() {
			return nil, domain.ErrNotFound
		}

		if !status.IsValid() {
			return nil, domain.ErrInvalidStatus
		}

		now := time.Now().UTC()

		var completedAt *time.Time
		if status == domain.StatusCompleted && state.CompletedAt == nil {
			completedAt = &now
		} else {
			completedAt = state.CompletedAt
		}

		payload := TodoPayload{
			Title: state.Title, Description: state.Description,
			Status: status, Priority: state.Priority,
			Tags: state.Tags, CreatedAt: state.CreatedAt,
			UpdatedAt: now, CompletedAt: completedAt,
		}

		eventType := EventStatusChanged
		if status == domain.StatusCompleted {
			eventType = EventCompleted
		}

		evt, err := newEventFromPayload(eventType, aggID, int(version)+1, payload)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

func DecideComplete(aggID id.AggregateID) decider.DecideFunc[TodoState] {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Deleted {
			return nil, domain.ErrNotFound
		}

		if state.IsNew() {
			return nil, domain.ErrNotFound
		}

		now := time.Now().UTC()
		payload := TodoPayload{
			Title: state.Title, Description: state.Description,
			Status: domain.StatusCompleted, Priority: state.Priority,
			Tags: state.Tags, CreatedAt: state.CreatedAt,
			UpdatedAt: now, CompletedAt: &now,
		}

		evt, err := newEventFromPayload(EventCompleted, aggID, int(version)+1, payload)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

func newEventFromPayload(
	eventType event.Type,
	aggID id.AggregateID,
	version int,
	payload TodoPayload,
) (event.Event, error) {
	data, err := codecpkg.JSONCodec{}.Encode(payload)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure, "todo.aggregate.decider.3",
			"marshal payload for event %s aggregate %s version %d: %v",
			eventType,
			aggID,
			version,
			err,
		)
	}

	return event.NewEvent(eventType, aggID, AggregateType, event.Version(version), data)
}
