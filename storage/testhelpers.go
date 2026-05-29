package storage

import (
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

const outboxInsertQuery = "INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)"

func expectOutboxInsert(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) *sqlmock.ExpectedExec {
	return mock.ExpectExec(regexp.QuoteMeta(outboxInsertQuery)).
		WithArgs(evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg())
}

// ExpectOutboxInsert mocks a successful INSERT INTO outbox.
func ExpectOutboxInsert(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	expectOutboxInsert(mock, evt).WillReturnResult(sqlmock.NewResult(1, 1))
}

// ExpectOutboxInsertError mocks a failing INSERT INTO outbox.
func ExpectOutboxInsertError(mock sqlmock.Sqlmock, evt *event.ImmutableEvent, err error) {
	expectOutboxInsert(mock, evt).WillReturnError(err)
}
