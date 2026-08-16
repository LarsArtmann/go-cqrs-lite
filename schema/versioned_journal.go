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

// VersionedSeekableJournal is the pre-transform wrapper around an
// event.SeekableJournal with upcaster support.
//
// Deprecated: Use [UpcastSourceTransform] with [event.DecorateJournal]:
//
//	journal := event.DecorateJournal(raw, schema.UpcastSourceTransform(uc))
//
// The deprecated shell still works: it embeds the decorated journal, so
// ReadAll and ReadFrom keep applying upcasters. Unlike the shell, the
// decorated journal ALSO forwards StreamingJournal reads (ReadStream,
// ReadStreamFrom) with upcasting applied — the shell drops them.
type VersionedSeekableJournal struct {
	inner    event.Journal
	seekable event.SeekableJournal
}

// NewVersionedSeekableJournal wraps a SeekableJournal with upcaster support.
// Events read via ReadAll and ReadFrom are upcasted before being returned.
//
// Deprecated: Use [UpcastSourceTransform] with [event.DecorateJournal]. Kept
// so existing consumers keep compiling; returns the compatibility shell.
func NewVersionedSeekableJournal(
	journal event.SeekableJournal,
	upcasters ...Upcaster,
) (*VersionedSeekableJournal, error) {
	if journal == nil {
		return nil, ErrNilJournal
	}

	decorated := event.DecorateJournal(journal, UpcastSourceTransform(upcasters...))

	// Unreachable in practice: the decorated wrapper implements
	// SeekableJournal unconditionally; nil-transform pass-through keeps the
	// original SeekableJournal. Defensive guard for future changes.
	seekable, ok := decorated.(event.SeekableJournal)
	if !ok {
		return nil, errorfamily.Wrapf(event.ErrInnerStoreNotSeekable, errorfamily.Rejection,
			"schema.versioned_journal_seekable",
			"decorated journal %T lost SeekableJournal", decorated)
	}

	return &VersionedSeekableJournal{inner: decorated, seekable: seekable}, nil
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

	return events, nil
}

func (j *VersionedSeekableJournal) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	events, err := j.seekable.ReadFrom(ctx, afterEventID, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "schema.versioned_journal_read_from",
			fmt.Sprintf("versioned journal ReadFrom after %s", afterEventID),
		)
	}

	return events, nil
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
