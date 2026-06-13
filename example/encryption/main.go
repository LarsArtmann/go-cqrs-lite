// Package main demonstrates event encryption with the go-cqrs-lite library.
//
// This example shows three patterns:
//  1. Bus-level encryption via middleware
//  2. Store-level encryption via NewEncryptedStore decorator
//  3. Key rotation with StaticKeyResolver
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/encryption/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Example 1: Bus-level encryption ===")
	busLevelEncryption(ctx)

	fmt.Println("\n=== Example 2: Store-level encryption ===")
	storeLevelEncryption(ctx)

	fmt.Println("\n=== Example 3: Key rotation ===")
	keyRotation(ctx)
}

func busLevelEncryption(ctx context.Context) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32-byte AES-256 key
	ed, err := encryption.NewAES256GCM(key)
	if err != nil {
		log.Fatal(err)
	}

	bus := memory.NewMemoryBus()
	bus.UsePublish(encryption.EncryptMiddleware(ed, encryption.WithMiddlewareKeyID("key-v1")))
	bus.Use(encryption.DecryptMiddleware(ed))

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("UserCreated", aggID, "User", 1,
		[]byte(`{"name":"Alice","email":"secret@example.com"}`))
	if err != nil {
		log.Fatal(err)
	}

	var captured event.Event
	bus.Use(func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			captured = evt
			return next(ctx, evt)
		}
	})

	if err := bus.Publish(ctx, evt); err != nil {
		log.Fatal(err)
	}

	payload := string(event.PayloadReadOnly(captured))
	fmt.Printf("Decrypted payload: %s\n", payload)
}

func storeLevelEncryption(ctx context.Context) {
	key := []byte("0123456789abcdef0123456789abcdef")
	ed, err := encryption.NewAES256GCM(key)
	if err != nil {
		log.Fatal(err)
	}

	inner := memory.NewMemoryStore()
	store, err := encryption.NewEncryptedStore(inner, ed, encryption.WithMiddlewareKeyID("key-v1"))
	if err != nil {
		log.Fatal(err)
	}

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1,
		[]byte(`{"name":"Bob","email":"bob@example.com"}`))
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		log.Fatal(err)
	}

	raw, err := inner.Load(ctx, ref)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Raw stored payload (encrypted): %s\n", truncate(string(event.PayloadReadOnly(raw[0])), 40))

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decrypted payload: %s\n", string(event.PayloadReadOnly(loaded[0])))
}

func keyRotation(_ context.Context) {
	oldKey := []byte("old-key-0123456789abcdef01234567")
	newKey := []byte("new-key-0123456789abcdef01234567")

	oldDec, _ := encryption.NewAES256GCM(oldKey)
	newDec, _ := encryption.NewAES256GCM(newKey)

	resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
		"key-v1": oldDec,
		"key-v2": newDec,
	})

	decV1, err := resolver.Resolve("key-v1")
	if err != nil {
		log.Fatal(err)
	}

	_ = decV1
	fmt.Printf("Resolver has keys: resolved key-v1 successfully\n")
	fmt.Printf("Available keys: key-v1, key-v2\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max] + "..."
}
