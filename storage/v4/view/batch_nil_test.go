package view

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// A nil item value is a Rejection like Set's, not a panic from the size
// estimate dereferencing it through the mapper.
func TestSQLViewStore_BatchSetNilItemIsRejection(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory(t)
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteViewStore[autoView, testKey](
		db,
		AutoMapperWithTombstone[autoView]("auto_views", "tombstoned"),
	)
	if err != nil {
		t.Fatalf("NewSQLiteViewStore: %v", err)
	}

	ctx := context.Background()

	items := []kv.ViewItem[autoView, testKey]{
		{Key: testKey("u1"), Value: &autoView{Name: "Alice", Age: 30}},
		{Key: testKey("u2")},
		{Key: testKey("u3"), Value: &autoView{Name: "Carol", Age: 31}},
	}

	err = store.BatchSet(ctx, items)
	if err == nil {
		t.Fatal("BatchSet with a nil item value must fail")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Errorf("BatchSet nil item: family = %v, want Rejection (err: %v)", family, err)
	}

	if !errors.Is(err, errNilViewValue) {
		t.Errorf("BatchSet nil item: err = %v, must wrap errNilViewValue", err)
	}

	if count, err := store.Count(ctx, kv.ViewQuery{}); err != nil || count != 0 {
		t.Errorf("Count after rejected BatchSet = %d (err %v), want 0", count, err)
	}
}
