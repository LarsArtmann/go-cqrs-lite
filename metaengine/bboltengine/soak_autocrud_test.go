package bboltengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

func TestSoak_AutoCRUD_Bbolt(t *testing.T) {
	eng := mustNewBboltEngine(t)

	enginetest.RunAutoCRUDSoak(t, eng)
}
