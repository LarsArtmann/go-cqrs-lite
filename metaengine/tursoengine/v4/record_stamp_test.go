package tursoengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestTurso_RecordStamping(t *testing.T) {
	t.Parallel()
	eng := mustNewTursoEngine(t)
	enginetest.RunRecordStampTest(t, eng)
}
