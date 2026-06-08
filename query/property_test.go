package query_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestQueryCreation_ValidType(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := query.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))

		q, err := query.New(typ)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if q.Type() != typ {
			t.Fatalf("type mismatch: got %q, want %q", q.Type(), typ)
		}
	})
}

func TestQueryCreation_EmptyTypeRejected(t *testing.T) {
	t.Parallel()

	_, err := query.New("")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestQueryDispatch_UnknownTypeRejected(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := query.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))

		d := query.NewDispatcher()

		q, err := query.New(typ)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		_, err = d.Dispatch(context.Background(), q)
		if err == nil {
			t.Fatal("expected error for unregistered query")
		}
	})
}

func TestQueryDispatch_RegisterAndDispatch(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := query.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))

		d := query.NewDispatcher()

		err := d.Register(typ, func(_ context.Context, q query.Query) (any, error) {
			if q.Type() != typ {
				t.Fatalf("handler received wrong type: %q", q.Type())
			}

			return "result", nil
		})
		if err != nil {
			t.Fatalf("register: %v", err)
		}

		q, err := query.New(typ)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		result, err := d.Dispatch(context.Background(), q)
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if result != "result" {
			t.Fatalf("result mismatch: got %v", result)
		}
	})
}

func TestPagination_ValidBounds(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		page := uint(rapid.IntRange(1, 1000).Draw(t, "page"))
		size := uint(rapid.IntRange(1, 100).Draw(t, "size"))

		p := query.NewPagination(page, size)
		if p.Page != page {
			t.Fatalf("page mismatch: got %d, want %d", p.Page, page)
		}
		if p.PageSize != size {
			t.Fatalf("size mismatch: got %d, want %d", p.PageSize, size)
		}
		expectedOffset := int((page - 1) * size)
		if p.Offset() != expectedOffset {
			t.Fatalf("offset mismatch: got %d, want %d", p.Offset(), expectedOffset)
		}
	})
}
