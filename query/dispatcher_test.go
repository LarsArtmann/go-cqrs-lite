package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query"
)

func TestDispatcher(t *testing.T) {
	d := query.NewDispatcher()
	ctx := context.Background()

	handler := func(q query.Query) (any, error) {
		return map[string]string{"result": "ok"}, nil
	}

	err := d.Register("GetUser", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := d.Dispatch(ctx, query.New("GetUser"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.(map[string]string)["result"] != "ok" {
		t.Errorf("expected result, got %v", result)
	}
}

func TestDispatcherClosed(t *testing.T) {
	d := query.NewDispatcher()
	_ = d.Close()

	err := d.Register("GetUser", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatchTyped(t *testing.T) {
	d := query.NewDispatcher()
	ctx := context.Background()

	user := User{Name: "John"}
	handler := func(q query.Query) (any, error) {
		return user, nil
	}

	err := d.Register("GetUser", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := query.DispatchTyped[User](ctx, d, query.New("GetUser"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("expected John, got %s", result.Name)
	}
}

type User struct {
	Name string
}
