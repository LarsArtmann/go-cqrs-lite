package bbolt_test

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		return bbolt.New(filepath.Join(t.TempDir(), "contract.db"))
	})
}
