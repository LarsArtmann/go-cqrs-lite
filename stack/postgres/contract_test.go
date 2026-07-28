package postgres_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
)

func TestContract(t *testing.T) {
	dsn := postgresDSN(t)

	contracttest.RunSuite(t, func(_ *testing.T) (*stack.Bundle, error) {
		return postgres.New(dsn)
	})
}
