package pebble_test

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		return pebble.New(filepath.Join(t.TempDir(), "contract"))
	})
}
