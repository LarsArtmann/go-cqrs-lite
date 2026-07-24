package pebble

import (
	"fmt"
	"strconv"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func (a *EventStore) checkVersion(
	ref id.StreamRef,
	expectedVersion event.Version,
) error {
	count, err := a.countEvents(ref)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.concurrency_check",
			"concurrency check")
	}

	err = event.CheckVersionConflict(count, expectedVersion)
	if err != nil {
		return errorfamily.WrapConflict(err, "pebble.version_conflict",
			"concurrency check")
	}

	return nil
}

func (a *EventStore) countEvents(ref id.StreamRef) (int, error) {
	prefix := a.streamPrefix(ref)
	upperBound := a.streamUpperBound(ref)

	iter, err := a.db.NewIter(
		&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(err, "pebble.create_iterator",
			"failed to create count iterator")
	}

	defer func() { _ = iter.Close() }()

	if !iter.Last() {
		err := iter.Error()
		if err != nil {
			return 0, errorfamily.WrapInfrastructure(err, "pebble.iterator_error",
				"last iterator error")
		}

		return 0, nil
	}

	key := iter.Key()

	version, err := parseVersionFromKey(key)
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(err, "pebble.parse_version",
			"failed to parse version from last key")
	}

	return version, nil
}

const versionDigits = 10

func parseVersionFromKey(key []byte) (int, error) {
	str := string(key)

	lastColon := len(str) - (versionDigits + 1)

	if lastColon < 0 || str[lastColon] != ':' {
		return 0, errorfamily.NewCorruption("pebble.invalid_key_format", "invalid key format: "+str)
	}

	n, err := strconv.Atoi(str[lastColon+1:])
	if err != nil {
		return 0, errorfamily.WrapCorruption(err, "pebble.parse_version", "parse version from key")
	}

	return n, nil
}

func (a *EventStore) writeEventsToBatch(
	batch *pebble.Batch,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	for i, evt := range events {
		err := validateEventOwnership(evt, ref)
		if err != nil {
			return errorfamily.WrapCorruption(err, "pebble.validate_event",
				fmt.Sprintf("validate event %d", i))
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return errorfamily.WrapConflict(ErrVersionMismatch, "pebble.version_mismatch",
				fmt.Sprintf("expected %d, got %d", expectedEventVersion, evt.Version()))
		}

		key := a.eventKey(ref, event.Version(expectedEventVersion))

		err = a.serializeAndAddToBatchWithJournal(batch, key, evt)
		if err != nil {
			return errorfamily.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %d for %s %s", i, ref.Type, ref.ID))
		}
	}

	return nil
}

func validateEventOwnership(
	evt event.Event,
	ref id.StreamRef,
) error {
	if evt.StreamType() != ref.Type {
		return errorfamily.WrapConflict(ErrStreamTypeMismatch, "pebble.aggregate_type_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.Type, evt.StreamType()))
	}

	if evt.StreamID() != ref.ID {
		return errorfamily.WrapConflict(ErrStreamIDMismatch, "pebble.aggregate_id_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.ID, evt.StreamID()))
	}

	return nil
}
