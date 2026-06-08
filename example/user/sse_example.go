package main

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
)

// demonstrateSSE shows how to use the SSE broker for real-time event streaming.
// In a real application, you would serve this on an HTTP endpoint and connect
// from a JavaScript client using EventSource.
func demonstrateSSE(bus event.Bus) {
	broker := middleware.NewSSEBroker(bus)

	http.Handle("/events", middleware.SSEHandler(broker))

	fmt.Println("--- SSE Broker Demo ---")
	fmt.Printf("  SSE handler registered at /events\n")
	fmt.Printf("  Active SSE clients: %d\n", broker.ClientCount())

	fmt.Println("  JavaScript client example:")
	fmt.Println("    const source = new EventSource('/events');")
	fmt.Println("    source.onmessage = (e) => console.log('event:', JSON.parse(e.data));")

	broker.Close()
}
