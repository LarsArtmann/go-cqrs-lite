package middleware

import (
	"sync"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

type testCommand struct {
	aggregateID id.AggregateID
}

func (c *testCommand) Type() command.Type          { return "test.cmd" }
func (c *testCommand) AggregateID() id.AggregateID { return c.aggregateID }
func (c *testCommand) IdempotencyKey() string       { return "" }

type testQuery struct{}

func (q *testQuery) Type() query.Type { return "test.query" }

type testLogger struct {
	mu     sync.Mutex
	Logs   []string
	Errors []string
}

func (l *testLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Logs = append(l.Logs, msg)
}

func (l *testLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Errors = append(l.Errors, msg)
}
