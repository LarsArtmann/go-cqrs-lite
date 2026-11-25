package metaengine

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Plan rules run without the store lock, so they must not read Store.queries
// while RegisterQuery mutates it. (Rules also read each query's assigned
// engine without the lock, which a concurrent Replan phase 1 writes — that
// separate race is deliberately not exercised here.)
func TestRules_ConcurrentReplanAndRegistration(t *testing.T) {
	store, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	stop := make(chan struct{})

	var replanner sync.WaitGroup

	replanner.Add(1)

	go func() {
		defer replanner.Done()

		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}

			_ = store.Replan(ctx)
		}
	}()

	for i := range 12 {
		q := Query[roleFindItem, roleItem](
			fmt.Sprintf("late_items_%d", i),
			OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) (string, roleItem) {
				return e.ID, roleItem{Name: e.Name}
			}),
		)

		if err := store.RegisterQuery(q); err != nil {
			t.Errorf("RegisterQuery: %v", err)
		}
	}

	close(stop)
	replanner.Wait()
}
