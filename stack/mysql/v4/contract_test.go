package mysql_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		return mysql.New(mysqlDSN(t))
	})
}
