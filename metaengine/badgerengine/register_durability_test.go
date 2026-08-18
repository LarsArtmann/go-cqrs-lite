package badgerengine_test

import (
	"context"
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept dep-isolated engine modules each need their own registration guard test
func TestRegisterDriver_RejectsDurabilityTier(t *testing.T) {
	t.Parallel()

	factory, err := metaengine.LookupDriver("badger")
	if err != nil {
		t.Fatalf("LookupDriver: %v", err)
	}

	_, err = factory(context.Background(), metaengine.DriverConfig{
		Durability: metaengine.DurabilityNormal,
	})
	if !errors.Is(err, metaengine.ErrUnsupportedDurability) {
		t.Fatalf("durability tier error = %v, want ErrUnsupportedDurability", err)
	}
}
