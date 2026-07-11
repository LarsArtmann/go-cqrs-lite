package querytest_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
)

func TestNew_HappyPath(t *testing.T) {
	t.Parallel()

	q := querytest.New(t, "user.get")
	if q == nil {
		t.Fatal("expected non-nil query")
	}

	if q.Type() != query.Type("user.get") {
		t.Fatalf("expected type %q, got %q", "user.get", q.Type())
	}
}
