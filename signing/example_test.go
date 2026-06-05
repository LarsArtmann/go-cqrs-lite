package signing_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
)

func ExampleNewHMAC() {
	signer, err := signing.NewHMAC([]byte("my-super-secret-key-that-is-long-enough-32"))
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"Alice"}`))

	sig, err := signer.Sign(evt)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	err = signer.Verify(evt, sig)
	fmt.Println("verified:", err == nil)

	// Output:
	// verified: true
}

func ExampleNewHMAC_tamperDetection() {
	signer, err := signing.NewHMAC([]byte("my-super-secret-key-that-is-long-enough-32"))
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"Alice"}`))

	sig, err := signer.Sign(evt)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	tampered, _ := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"Bob"}`))

	err = signer.Verify(tampered, sig)
	fmt.Println("tamper detected:", err != nil)

	// Output:
	// tamper detected: true
}
