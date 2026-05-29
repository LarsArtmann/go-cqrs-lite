package storage

import (
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

const outboxInsertQuery = "INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)"

func expectOutboxInsert(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) *sqlmock.ExpectedExec {
	return mock.ExpectExec(regexp.QuoteMeta(outboxInsertQuery)).
		WithArgs(evt.ID(), string(sqlpkg.OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg())
}

func ExpectOutboxInsert(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	expectOutboxInsert(mock, evt).WillReturnResult(sqlmock.NewResult(1, 1))
}

func ExpectOutboxInsertError(mock sqlmock.Sqlmock, evt *event.ImmutableEvent, err error) {
	expectOutboxInsert(mock, evt).WillReturnError(err)
}
