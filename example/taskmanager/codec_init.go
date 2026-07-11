package main

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Use JSON encoding for events so payloads are human-readable in the database
// and SSE stream. The library default is CBOR (compact), but for a learning
// example JSON is more approachable. Set once at init to avoid data races.
func init() {
	event.DefaultCodec = codec.JSONCodec{}
}
