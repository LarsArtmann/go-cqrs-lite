package encryption_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func ExampleNewAES256GCM() {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	plaintext := []byte(`{"ssn":"123-45-6789"}`)

	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	decrypted, err := enc.Decrypt(ct)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("round-trip:", string(decrypted) == string(plaintext))

	// Output:
	// round-trip: true
}

func ExampleEncryptMiddleware() {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, _ := encryption.NewAES256GCM(key)

	mw := encryption.EncryptMiddleware(enc)

	streamID := id.NewStreamID()
	evt, _ := event.NewEvent(
		"user.created",
		streamID,
		"User",
		1,
		[]byte(`{"email":"alice@example.com"}`),
	)

	var captured event.Event
	inner := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		if len(events) > 0 {
			captured = events[0]
		}

		return nil
	})

	_ = mw(inner).Publish(context.Background(), evt)
	fmt.Println("encrypted:", encryption.HasEncryption(captured))

	// Output:
	// encrypted: true
}

func ExampleNewXChaCha20Poly1305() {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := encryption.NewXChaCha20Poly1305(key)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	plaintext := []byte(`{"ssn":"123-45-6789"}`)

	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	decrypted, err := enc.Decrypt(ct)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("round-trip:", string(decrypted) == string(plaintext))

	// Output:
	// round-trip: true
}

func ExampleNewCodec() {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	enc, _ := encryption.NewXChaCha20Poly1305(key)
	c := encryption.NewCodec(codec.JSONCodec{}, enc)

	type Secret struct {
		SSN string `json:"ssn"`
	}

	data, err := c.Encode(Secret{SSN: "123-45-6789"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var decoded Secret
	err = c.Decode(data, &decoded)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("ssn:", decoded.SSN)

	// Output:
	// ssn: 123-45-6789
}
