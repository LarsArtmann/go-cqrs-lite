package postgres_test

import (
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
)

func TestContract(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping Postgres contract tests")
	}

	contracttest.RunSuite(t, func(_ *testing.T) (*stack.Bundle, error) {
		return postgres.New(dsn)
	})
}
