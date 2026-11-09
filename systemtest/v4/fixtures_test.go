package systemtest_test

// Task-domain fixtures shared by the real-engine suites. Twin of the block
// in system/system_test.go — test fixtures may not cross the module
// boundary created by the Feedback-#4 split.

import (
	"encoding/json/v2"
	"errors"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ── Domain types ──

// art-dupl:accept test-fixture twin of system/system_test.go domain block
type TaskCreated struct {
	Title string
	At    time.Time
}

type TaskCompleted struct {
	At time.Time
}

type TaskState struct {
	Title  string
	Status string
	Exists bool
}

func applyTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case "task.created":
		var p TaskCreated
		_ = json.Unmarshal(evt.Payload(), &p)
		state.Title = p.Title
		state.Status = "pending"
		state.Exists = true
	case "task.completed":
		state.Status = "completed"
	}

	return state, nil
}

var TaskDecider = decider.Decider[TaskState]{
	Initial: TaskState{},
	Apply:   applyTask,
}

func mustEvent(evt event.Event, err error) event.Event {
	if err != nil {
		panic(err)
	}

	return evt
}

func newCmd(cmdType command.Type, streamID id.StreamID) *command.BasicCommand {
	cmd, err := command.New(cmdType, streamID)
	if err != nil {
		panic(err)
	}

	return cmd
}

// ── Projection fixtures ──

type TaskView struct {
	Title  string
	Status string
}

type FindTask struct {
	ID string
}

func projectionDecoder(eventType string, payload []byte) (any, error) {
	switch eventType {
	case "task.created":
		var e TaskCreated
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}

		return e, nil
	}

	return nil, errors.New("unknown event type: " + eventType)
}
