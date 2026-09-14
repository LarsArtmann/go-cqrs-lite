package snapshot

import (
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// NewSnapshot constructs a [Snapshot] that satisfies every invariant,
// stamping CreatedAt with the current UTC time and defensively cloning
// state so later caller mutations cannot alter the stored bytes.
//
// encoding declares the codec the state bytes were written with; pass
// [record.EncodingUnknown] only for legacy data that predates the stamp.
//
// It returns an error (family Rejection) when ref is invalid, version < 1,
// or state is empty. Use [Snapshot.Validate] to check values assembled by
// other means.
func NewSnapshot(
	ref id.StreamRef,
	version event.Version,
	state []byte,
	encoding record.Encoding,
) (Snapshot, error) {
	if err := ref.Validate(); err != nil {
		return Snapshot{}, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"snapshot.invalid_ref",
			"snapshot for %s: invalid stream ref", ref,
		)
	}

	snap := Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    version,
		State:      slices.Clone(state),
		Encoding:   encoding,
		CreatedAt:  time.Now().UTC(),
	}

	if err := snap.Validate(); err != nil {
		return Snapshot{}, err
	}

	return snap, nil
}

// Validate reports whether the snapshot satisfies the structural
// invariants documented on [Snapshot]. It returns an error of family
// Rejection with a distinct code per violation (snapshot.invalid_ref,
// snapshot.zero_version, snapshot.nil_state), or nil when valid.
func (s Snapshot) Validate() error {
	if err := s.Ref().Validate(); err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"snapshot.invalid_ref",
			"invalid snapshot stream identity",
		)
	}

	if s.Version < 1 {
		return errorfamily.Wrapf(
			ErrInvalidSnapshot,
			errorfamily.Rejection,
			"snapshot.zero_version",
			"version %d must be >= 1", s.Version,
		)
	}

	if len(s.State) == 0 {
		return errorfamily.Wrapf(
			ErrInvalidSnapshot,
			errorfamily.Rejection,
			"snapshot.nil_state",
			"state must not be empty",
		)
	}

	return nil
}

// Ref returns the snapshot's stream identity as a single [id.StreamRef],
// the pair-form counterpart of the StreamType/StreamID fields.
func (s Snapshot) Ref() id.StreamRef {
	return id.NewStreamRef(s.StreamType, s.StreamID)
}
