package sqlite

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
)

//art-dupl:accept dialect twin of queue/mysql cancelRequestedSQL; identical wire shape
const cancelRequestedSQL = `SELECT EXISTS(
	SELECT 1 FROM facts WHERE task_id = ? AND type = 'task.cancel-requested')`

// cancelRequestedTx is the in-transaction variant of CancelRequested.
func cancelRequestedTx(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var requested bool

	err := tx.QueryRowContext(ctx, cancelRequestedSQL, id).Scan(&requested)

	return requested, err
}

// cancelRequestedReasonTx reads the reason a task's latest cancel
// request carried ("" when none). Best-effort: an unparsable detail
// yields "", never an error — the finalize must not fail on cosmetics.
func cancelRequestedReasonTx(ctx context.Context, tx *sql.Tx, id string) (string, error) {
	var detail string

	err := tx.QueryRowContext(ctx, `
		SELECT detail FROM facts
		WHERE task_id = ? AND type = 'task.cancel-requested'
		ORDER BY seq DESC LIMIT 1`, id).Scan(&detail)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	//art-dupl:accept dialect twin of queue/postgres cancelRequestedReasonTx; driver surface differs
	if err != nil {
		return "", err
	}

	var d struct {
		Reason string `json:"reason"`
	}

	if json.Unmarshal([]byte(detail), &d) != nil {
		return "", nil //nolint:nilerr // documented best-effort: unparsable detail yields "", never an error
	}

	return d.Reason, nil
}

// cancelReasonDetail builds the detail for a Cancel/CancelRunning fact:
// nil without a reason (no detail noise), {"reason": ...} with one.
func cancelReasonDetail(reason string) []byte {
	if reason == "" {
		return nil
	}

	return mustJSON(map[string]string{"reason": reason})
}

// dismissReasonDetail builds the cancelled detail for a DLQ dismiss: the
// reason plus who ruled. The reason is the point of the dismissal — an
// empty one still records the by.
func dismissReasonDetail(reason string, dismissedBy string) []byte {
	detail := map[string]string{"dismissed_by": dismissedBy}
	if reason != "" {
		detail["reason"] = reason
	}

	return mustJSON(detail)
}

// cooperativeCancelDetail builds the cancelled detail for a cooperative
// finalize: the cooperative marker, the finalize context ("after" key,
// when set) and the operator's reason, when one was given.
func cooperativeCancelDetail(reason string, after string) []byte {
	detail := map[string]string{"cooperative": "true"}
	if after != "" {
		detail["after"] = after
	}

	if reason != "" {
		detail["reason"] = reason
	}

	return mustJSON(detail)
}
