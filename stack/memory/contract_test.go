package memory_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3/contracttest"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(_ *testing.T) (*stack.Bundle, error) {
		return memory.New()
	})
}
