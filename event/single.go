package event

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Single creates a single event and returns it as a one-element slice.
// This is a convenience wrapper around New for the common case where a
// decide function emits exactly one event.
//
// Example:
//
//	func decideCreate(s State, cmd CreateCmd) ([]event.Event, error) {
//	    return event.Single("user.created", cmd.StreamID(), "User", s.Version.Increment(), UserCreated{Name: cmd.Name})
//	}
//
// This replaces the singleEvent/makeEvent/mustEvent helper functions
// that every consumer project reimplements.
func Single(
	eventType Type,
	streamID id.StreamID,
	streamType id.StreamType,
	version Version,
	payload any,
	opts ...Option,
) ([]Event, error) {
	evt, err := New(eventType, streamID, streamType, version, payload, opts...)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "event.single", "event.Single")
	}

	return []Event{evt}, nil
}
