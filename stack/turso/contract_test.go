package turso_test

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		b, err := turso.New(filepath.Join(t.TempDir(), "contract.db"))
		if err != nil {
			return nil, err
		}

		return b.Bundle, nil
	})
}
