package query_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func ExampleDispatcher() {
	d := query.NewDispatcher()

	type UserResult struct {
		Name string
	}

	err := query.RegisterTyped(
		d,
		"GetUser",
		func(_ context.Context, _ query.Query) (UserResult, error) {
			return UserResult{Name: "Alice"}, nil
		},
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	q, _ := query.New("GetUser")
	result, err := query.DispatchTyped[UserResult](context.Background(), d, q)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(result.Name)

	// Output:
	// Alice
}
