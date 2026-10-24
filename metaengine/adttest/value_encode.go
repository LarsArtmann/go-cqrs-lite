package adttest

import (
	"context"
	"fmt"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertValueEncodeRejection pins the stored-value contract on one engine
// instance: a value that is not JSON-representable (channel, function, cyclic
// structure) must fail the WRITE with a Rejection-family error carrying the
// keycodec.encode_value code — never silently store its %v rendering into the
// read model (the legacy EncodeJSON fallback defect class). Run it from every
// engine module whose write path persists JSON values.
func AssertValueEncodeRejection(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Skipf("engine does not implement metaengine.MapBackend")
	}

	// Run-unique collection: a fixed name lets a previous run's rows make the
	// next run's asserts flaky on persistent database surfaces (the adttest
	// rerun-safety rule).
	col := fmt.Sprintf("valrej_%d", time.Now().UnixNano())
	ctx := context.Background()

	err := mb.MapSet(ctx, col, "k", map[string]any{"bad": make(chan int)})
	assertEncodeValueRejection(t, err, "MapSet")

	if updater, updOK := eng.(metaengine.MapUpdater); updOK {
		updErr := updater.MapUpdate(ctx, col, "k", func(_ any) any {
			return map[string]any{"bad": make(chan int)}
		})
		assertEncodeValueRejection(t, updErr, "MapUpdate")
	}
}

func assertEncodeValueRejection(t *testing.T, err error, surface string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s with a non-representable value must fail, got nil", surface)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("%s error %v classified as %v, want Rejection", surface, err, family)
	}

	if code := errorfamily.Code(err); code != "keycodec.encode_value" {
		t.Fatalf("%s error code = %q, want keycodec.encode_value (%v)", surface, code, err)
	}
}
