package storage

import (
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// ExpectOutboxInsert mocks a successful INSERT INTO outbox.
func ExpectOutboxInsert(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)")).
		WithArgs(evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// ExpectOutboxInsertError mocks a failing INSERT INTO outbox.
func ExpectOutboxInsertError(mock sqlmock.Sqlmock, evt *event.ImmutableEvent, err error) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)")).
		WithArgs(evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(err)
}
