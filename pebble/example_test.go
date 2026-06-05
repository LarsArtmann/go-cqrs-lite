package pebble_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/pebble/v2"
)

func ExampleNewConfig() {
	cfg := pebble.NewConfig()
	fmt.Println(cfg.Backend)
	fmt.Println(cfg.Provider == nil)

	// Output:
	// pebble
	// true
}
