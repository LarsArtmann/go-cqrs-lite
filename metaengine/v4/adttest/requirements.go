package adttest

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func requireMapBackend(t *testing.T, eng metaengine.Engine) metaengine.MapBackend {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Skipf("engine does not implement metaengine.MapBackend")
	}

	return mb
}
