package querytest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2/querytest"
)

func TestMustNew_HappyPath(t *testing.T) {
	t.Parallel()

	q := querytest.MustNew("user.get")
	if q == nil {
		t.Fatal("expected non-nil query")
	}

	if q.Type() != query.Type("user.get") {
		t.Fatalf("expected type %q, got %q", "user.get", q.Type())
	}
}

func TestMustNew_PanicsOnEmptyType(t *testing.T) {
	t.Parallel()

	if !didPanic(func() { _ = querytest.MustNew("") }) {
		t.Fatal("expected panic for empty query type")
	}
}

func didPanic(fn func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	fn()

	return panicked
}
