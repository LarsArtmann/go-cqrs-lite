package projectionhost

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Delete removes one entry, or — like List and Purge — every entry with that
// event ID across all projections when projectionName is empty.
func (s *SQLiteDeadLetterStore) Delete(
	ctx context.Context,
	projectionName, eventID string,
) error {
	const (
		oneEntry    = "DELETE FROM projection_dead_letters WHERE projection_name = ? AND event_id = ?"
		eventIDOnly = "DELETE FROM projection_dead_letters WHERE event_id = ?"
	)

	query, args := oneEntry, []any{projectionName, eventID}
	if projectionName == "" {
		query, args = eventIDOnly, []any{eventID}
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_delete",
			fmt.Sprintf("delete dead-letter entry %s/%s", projectionName, eventID),
		)
	}

	return nil
}
