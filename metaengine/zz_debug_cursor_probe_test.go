package metaengine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDebugLegacyCursorProbe(t *testing.T) {
	reader := setupTieScan(t, 13)
	ctx := context.Background()

	items, cursor, err := reader.ScanPage(ctx,
		metaengine.WithSort("Priority", false), metaengine.WithLimit(5))
	if err != nil {
		t.Fatalf("page1: %v", err)
	}

	fmt.Printf("PAGE1 items=%v cursor.Value=%#v (%T)\n", tieIDs(items), cursor.Value, cursor.Value)

	items2, cursor2, err := reader.ScanPage(ctx,
		metaengine.WithSort("Priority", false), metaengine.WithLimit(5),
		metaengine.WithCursor(cursor.Value))
	if err != nil {
		t.Fatalf("page2: %v", err)
	}

	fmt.Printf("PAGE2 items=%v cursor2=%#v\n", tieIDs(items2), cursor2)

	items3, _, err := reader.ScanPage(ctx,
		metaengine.WithSort("Priority", false), metaengine.WithLimit(5),
		metaengine.WithCursor(0))
	if err != nil {
		t.Fatalf("page3 literal 0: %v", err)
	}

	fmt.Printf("PAGE3 (literal int 0 cursor) items=%v\n", tieIDs(items3))
}
