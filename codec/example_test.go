package codec_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

func ExampleJSONCodec() {
	c := codec.JSONCodec{}

	type User struct {
		Name string `json:"name"`
	}

	data, err := c.Encode(User{Name: "Alice"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var user User

	err = c.Decode(data, &user)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(user.Name)

	// Output:
	// Alice
}

func ExampleCBORCodec() {
	c := codec.CBORCodec{}

	type User struct {
		Name string `json:"name"`
	}

	data, err := c.Encode(User{Name: "Alice"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var user User

	err = c.Decode(data, &user)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(user.Name)

	// Output:
	// Alice
}

func ExampleRawCodec() {
	c := codec.RawCodec{}

	raw := []byte(`{"name":"Bob"}`)

	data, err := c.Encode(raw)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var decoded []byte

	err = c.Decode(data, &decoded)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(string(decoded))

	// Output:
	// {"name":"Bob"}
}

// ExampleCBORCompactCodec demonstrates the strict CBOR codec that rejects
// unknown fields on decode — a schema drift detection mechanism.
func ExampleCBORCompactCodec() {
	c := codec.CBORCompactCodec{}

	type UserCreated struct {
		Name  string
		Email string
	}

	data, _ := c.Encode(UserCreated{Name: "Alice", Email: "alice@example.com"})

	var result UserCreated
	_ = c.Decode(data, &result)

	fmt.Println(result.Name, result.Email)

	// Output:
	// Alice alice@example.com
}

// ExampleCBORCodec_toarray demonstrates the toarray struct tag that encodes
// structs as positional CBOR arrays instead of keyed maps, reducing payload
// size by 30-40% by eliminating field-name string overhead.
func ExampleCBORCodec_toarray() {
	c := codec.CBORCodec{}

	// The _ field with `cbor:",toarray"` tag enables positional encoding.
	// Field ORDER becomes part of the wire format — add new fields only at end.
	type PaymentProcessed struct {
		_           struct{} `cbor:",toarray"`
		PaymentID   string
		AmountCents int64
		Currency    string
		OccurredAt  int64
	}

	payload := PaymentProcessed{
		PaymentID:   "pay_abc123",
		AmountCents: 4999,
		Currency:    "USD",
		OccurredAt:  1700000000,
	}

	mapCodec := codec.JSONCodec{}
	mapData, _ := mapCodec.Encode(payload)

	arrayData, _ := c.Encode(payload)

	fmt.Printf("JSON: %d bytes, CBOR+toarray: %d bytes (%.0f%% smaller)\n",
		len(mapData), len(arrayData),
		float64(len(mapData)-len(arrayData))/float64(len(mapData))*100)

	var decoded PaymentProcessed
	_ = c.Decode(arrayData, &decoded)
	fmt.Println(decoded.PaymentID, decoded.AmountCents)

	// Output:
	// JSON: 86 bytes, CBOR+toarray: 24 bytes (72% smaller)
	// pay_abc123 4999
}

// ExampleBufferEncoder demonstrates zero-allocation encoding by writing
// directly into a caller-provided buffer. Useful in hot paths where buffer
// reuse eliminates GC pressure.
func ExampleBufferEncoder() {
	type Metric struct {
		Name  string
		Value float64
	}

	c := codec.CBORCodec{}
	buf := &bytes.Buffer{}

	// Reuse the same buffer across multiple encode calls
	for _, m := range []Metric{
		{Name: "cpu", Value: 0.42},
		{Name: "mem", Value: 0.87},
	} {
		buf.Reset()

		if be, ok := any(c).(codec.BufferEncoder); ok {
			_ = be.EncodeToBuffer(m, buf)
		}

		fmt.Printf("%d bytes ", buf.Len())
	}

	fmt.Println("done")

	// Output:
	// 25 bytes 25 bytes done
}

// ExampleNewCBOREncoder demonstrates streaming CBOR encoding for large
// event batches without materializing the full byte slice in memory.
func ExampleNewCBOREncoder() {
	type Event struct {
		Type string
		Data string
	}

	var buf bytes.Buffer

	enc := codec.NewCBOREncoder(&buf)
	_ = enc.Encode(Event{Type: "user.created", Data: "alice"})
	_ = enc.Encode(Event{Type: "user.created", Data: "bob"})

	// Decode the stream
	dec := codec.NewCBORDecoder(&buf)

	var events []Event
	for {
		var evt Event
		if err := dec.Decode(&evt); err != nil {
			break
		}
		events = append(events, evt)
	}

	fmt.Printf("%d events decoded from stream\n", len(events))

	// Output:
	// 2 events decoded from stream
}

// ExampleDiagnose converts CBOR bytes to human-readable diagnostic notation.
// Useful for debugging corrupt events or inspecting raw CBOR payloads.
func ExampleDiagnose() {
	c := codec.CBORCodec{}

	type User struct {
		Name  string
		Email string
	}

	data, _ := c.Encode(User{Name: "Alice", Email: "alice@example.com"})

	diag, _ := codec.Diagnose(data)

	// Diagnostic notation is a map-like representation
	fmt.Println(strings.Contains(diag, "Alice"))

	// Output:
	// true
}

// ExampleCBOREncMode demonstrates using the exported canonical encoding mode
// directly. Storage backends should use this instead of creating their own
// CBOR mode, ensuring all modules share one deterministic encoding.
func ExampleCBOREncMode() {
	type Snapshot struct {
		State string `json:"state"`
		N     int    `json:"n"`
	}

	snap := Snapshot{State: "active", N: 42}

	// Same canonical EncMode used by CBORCodec internally
	data, err := codec.CBOREncMode().Marshal(snap)
	if err != nil {
		log.Fatal(err)
	}

	var decoded Snapshot
	_ = codec.CBORDecMode().Unmarshal(data, &decoded)

	fmt.Printf("%s/%d (%d bytes)\n", decoded.State, decoded.N, len(data))

	// Output:
	// active/42 (18 bytes)
}
