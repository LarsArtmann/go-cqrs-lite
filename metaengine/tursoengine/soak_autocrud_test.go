package tursoengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_Turso runs the AutoCRUDByConvention soak against the
// Turso engine (libSQL backend) to verify CRUD lifecycle correctness under
// sustained write load. Turso delegates to sqliteengine, so this exercises
// the same SQL paths through the libSQL driver.
func TestSoak_AutoCRUD_Turso(t *testing.T) {
	t.Parallel()
	eng := mustNewTursoEngine(t)
	enginetest.RunAutoCRUDSoak(t, eng)
}
