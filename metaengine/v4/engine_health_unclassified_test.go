package metaengine

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ErrKeyTypeMismatch is a caller mistake raised before any backend call. It
// is an unclassified error, which errorfamily fail-open-classifies Transient,
// yet it must never count toward quarantine.
func TestEngineHealth_KeyTypeMismatchNeverCounts(t *testing.T) {
	t.Parallel()

	q := Query[struct{ Count int }, roleItem](
		"keyless",
		OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) (string, roleItem) {
			return e.ID, roleItem{Name: e.Name}
		}),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, q)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	for i := range DefaultEngineFailureThreshold + 2 {
		if _, err := store.Execute(struct{ Count int }{Count: i}); !errors.Is(err, ErrKeyTypeMismatch) {
			t.Fatalf("execute %d: want ErrKeyTypeMismatch, got %v", i, err)
		}
	}

	if h, seen := store.HealthSnapshot()[store.engines[0].Profile().Name]; seen &&
		(h.State == EngineQuarantined || h.ConsecutiveFailures != 0) {
		t.Errorf("client mistakes must not count toward health: %+v", h)
	}
}
