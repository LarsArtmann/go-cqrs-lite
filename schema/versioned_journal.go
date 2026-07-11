package schema

import (
	"context"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

var _ event.SeekableJournal = (*VersionedSeekableJournal)(nil)

// VersionedSeekableJournal wraps an event.SeekableJournal with upcaster
// support. It applies registered upcasters to every event returned by ReadAll
// and ReadFrom, bridging schema evolution to the projection host pipeline.
//
// Use this when you need upcasters AND projectionhost.New (which takes
// event.SeekableJournal, not event.Store).
type VersionedSeekableJournal struct {
	inner    event.SeekableJournal
	registry *upcasterRegistry
}

// NewVersionedSeekableJournal wraps a SeekableJournal with upcaster support.
// Events read via ReadAll and ReadFrom are upcasted before being returned.
func NewVersionedSeekableJournal(
	journal event.SeekableJournal,
	upcasters ...Upcaster,
) (*VersionedSeekableJournal, error) {
	if journal == nil {
		return nil, ErrNilJournal
	}

	reg := newUpcasterRegistry()

	for _, u := range upcasters {
		if u != nil {
			reg.register(u)
		}
	}

	return &VersionedSeekableJournal{inner: journal, registry: reg}, nil
}

func (j *VersionedSeekableJournal) ReadAll(
	ctx context.Context,
) ([]event.Event, error) {
	events, err := j.inner.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "schema.versioned_journal_read_all",
			"versioned journal ReadAll",
		)
	}

	return j.registry.upcastAll(events)
}

func (j *VersionedSeekableJournal) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	events, err := j.inner.ReadFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "schema.versioned_journal_read_from",
			fmt.Sprintf("versioned journal ReadFrom after %s", afterEventID),
		)
	}

	return j.registry.upcastAll(events)
}

// Close closes the inner journal if it implements io.Closer.
func (j *VersionedSeekableJournal) Close() error {
	if c, ok := j.inner.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			return errorfamily.WrapInfrastructure(
				err, "schema.versioned_journal_close",
				"close versioned journal",
			)
		}
	}

	return nil
}
