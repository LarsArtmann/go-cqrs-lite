package encryption_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// TestDeprecatedErrorAliasesAreEventSentinels pins the v4 compatibility
// contract: the deprecated encryption.ErrInnerStoreNot* names must stay
// identical (errors.Is-equal) to the event.* sentinels they forward to, so
// external errors.Is checks keep matching until v5 removes the shells.
func TestDeprecatedErrorAliasesAreEventSentinels(t *testing.T) {
	t.Parallel()

	pairs := []struct {
		deprecated error
		canonical  error
	}{
		{encryption.ErrInnerStoreNotJournal, event.ErrInnerStoreNotJournal},
		{encryption.ErrInnerStoreNotSeekable, event.ErrInnerStoreNotSeekable},
		{encryption.ErrInnerStoreNotBackwards, event.ErrInnerStoreNotBackwards},
	}

	for _, pair := range pairs {
		if !errors.Is(pair.deprecated, pair.canonical) {
			t.Errorf(
				"deprecated sentinel %v is not errors.Is-equal to canonical %v",
				pair.deprecated,
				pair.canonical,
			)
		}
	}
}
