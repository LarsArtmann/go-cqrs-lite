//go:build wasm

package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("go-cqrs-lite WASM build: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("Core modules compiled successfully:")
	fmt.Println("  - id: branded IDs, ULID generation")
	fmt.Println("  - codec: JSON, CBOR encoding/decoding")
	fmt.Println("  - dispatcher: generic handler dispatch")
	fmt.Println("  - event: event creation, metadata, tombstone detection")
	fmt.Println("  - command: command creation, typed handlers")
	fmt.Println("  - query: query dispatch, pagination")
	fmt.Println("  - decider: pure-function aggregate decision logic")
}
