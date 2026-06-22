package dispatcher_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v3"
)

type handler func() error

type middleware func(handler) handler

func ExampleNewDispatcher() {
	d := dispatcher.NewDispatcher[handler, middleware]()

	_ = d.Register("greet", func() error {
		fmt.Println("hello")

		return nil
	}, func(m middleware, h handler) handler {
		return h
	})

	h, err := d.Dispatch("greet")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	_ = h()

	// Output:
	// hello
}
