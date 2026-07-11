package signing_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	. "github.com/larsartmann/go-cqrs-lite/signing/v4"
)

var _ = Describe("Signing", func() {
	makeEvent := func() event.Event {
		aggID := id.NewAggregateID()
		evt, err := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"name":"Alice"}`))
		Expect(err).NotTo(HaveOccurred())

		return evt
	}

	Describe("HMAC-SHA256 signing", func() {
		secret := []byte("this-is-a-32-byte-secret-key-xxxx")

		When("I sign an event and verify it with the same key", func() {
			It(
				"should confirm the event has not been tampered with so I can trust its integrity",
				func() {
					signer, err := NewHMAC(secret)
					Expect(err).NotTo(HaveOccurred())

					evt := makeEvent()
					sig, err := signer.Sign(evt)
					Expect(err).NotTo(HaveOccurred())
					Expect(sig).NotTo(BeNil())

					err = signer.Verify(evt, sig)
					Expect(err).NotTo(HaveOccurred())
				},
			)
		})

		When("someone tampers with the event payload after signing", func() {
			It("should reject the signature so I know the event was modified in transit", func() {
				signer, err := NewHMAC(secret)
				Expect(err).NotTo(HaveOccurred())

				evt := makeEvent()
				sig, err := signer.Sign(evt)
				Expect(err).NotTo(HaveOccurred())

				tampered, err := event.NewEvent(
					"user.created",
					evt.AggregateID(),
					"User",
					1,
					[]byte(`{"name":"Eve"}`),
				)
				Expect(err).NotTo(HaveOccurred())

				err = signer.Verify(tampered, sig)
				Expect(err).To(MatchError(ErrInvalidSignature))
			})
		})

		When("I sign the same event twice with the same key", func() {
			It("should produce identical signatures so I can verify deterministically", func() {
				signer, err := NewHMAC(secret)
				Expect(err).NotTo(HaveOccurred())

				evt := makeEvent()
				sig1, err := signer.Sign(evt)
				Expect(err).NotTo(HaveOccurred())
				sig2, err := signer.Sign(evt)
				Expect(err).NotTo(HaveOccurred())

				Expect(sig1.Equal(sig2)).To(BeTrue())
			})
		})

		When("I try to create an HMAC signer with a key shorter than 32 bytes", func() {
			It("should reject my key so I cannot accidentally use weak cryptography", func() {
				_, err := NewHMAC([]byte("too-short"))
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidKey)).To(BeTrue())
			})
		})
	})

	Describe("Ed25519 signing", func() {
		When("I sign an event and verify it with the corresponding public key", func() {
			It(
				"should confirm the event came from the private key holder so I can authenticate the source",
				func() {
					pub, priv, err := GenerateEd25519KeyPair()
					Expect(err).NotTo(HaveOccurred())

					signer, err := NewEd25519(priv)
					Expect(err).NotTo(HaveOccurred())
					verifier, err := NewEd25519Verifier(pub)
					Expect(err).NotTo(HaveOccurred())

					evt := makeEvent()
					sig, err := signer.Sign(evt)
					Expect(err).NotTo(HaveOccurred())

					err = verifier.Verify(evt, sig)
					Expect(err).NotTo(HaveOccurred())
				},
			)
		})

		When("I verify with the wrong public key", func() {
			It(
				"should reject the signature so a different key pair cannot impersonate the signer",
				func() {
					_, priv, err := GenerateEd25519KeyPair()
					Expect(err).NotTo(HaveOccurred())

					wrongPub, _, err := GenerateEd25519KeyPair()
					Expect(err).NotTo(HaveOccurred())

					signer, err := NewEd25519(priv)
					Expect(err).NotTo(HaveOccurred())
					verifier, err := NewEd25519Verifier(wrongPub)
					Expect(err).NotTo(HaveOccurred())

					evt := makeEvent()
					sig, err := signer.Sign(evt)
					Expect(err).NotTo(HaveOccurred())

					err = verifier.Verify(evt, sig)
					Expect(err).To(MatchError(ErrInvalidSignature))
				},
			)
		})

		When("I create an Ed25519 signer with a nil private key", func() {
			It("should reject it so I fail fast instead of producing invalid signatures", func() {
				_, err := NewEd25519(nil)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidKey)).To(BeTrue())
			})
		})

		When("I create an Ed25519 verifier with a nil public key", func() {
			It("should reject it so I fail fast instead of accepting unverified events", func() {
				_, err := NewEd25519Verifier(nil)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidKey)).To(BeTrue())
			})
		})
	})

	Describe("publish pipeline middleware", func() {
		When("I use SignMiddleware on my publisher", func() {
			It(
				"should attach a signature to my event before it reaches the next publisher so downstream consumers can verify it",
				func() {
					signer, err := NewHMAC([]byte("this-is-a-32-byte-secret-key-xxxx"))
					Expect(err).NotTo(HaveOccurred())

					signMW := SignMiddleware(signer)

					evt := makeEvent()
					var captured event.Event
					inner := event.PublisherFunc(
						func(_ context.Context, events ...event.Event) error {
							if len(events) > 0 {
								captured = events[0]
							}

							return nil
						},
					)

					wrapped := signMW(inner)
					err = wrapped.Publish(context.Background(), evt)
					Expect(err).NotTo(HaveOccurred())

					Expect(HasSignature(captured)).To(BeTrue())
					sig, err := ExtractSignature(captured)
					Expect(err).NotTo(HaveOccurred())
					Expect(signer.Verify(captured, sig)).NotTo(HaveOccurred())
				},
			)
		})

		When("VerifyMiddleware receives a tampered event", func() {
			It("should block it so my handler never processes a corrupted event", func() {
				signer, err := NewHMAC([]byte("this-is-a-32-byte-secret-key-xxxx"))
				Expect(err).NotTo(HaveOccurred())

				evt := makeEvent()
				sig, err := signer.Sign(evt)
				Expect(err).NotTo(HaveOccurred())

				tampered, err := event.NewEvent(
					"user.created",
					evt.AggregateID(),
					"User",
					1,
					[]byte(`{"name":"Eve"}`),
				)
				Expect(err).NotTo(HaveOccurred())
				signed, err := AttachSignature(tampered, sig)
				Expect(err).NotTo(HaveOccurred())

				handlerCalled := false
				verifyMW := VerifyMiddleware(signer)
				wrapped := verifyMW(func(_ context.Context, _ event.Event) error {
					handlerCalled = true

					return nil
				})

				err = wrapped(context.Background(), signed)
				Expect(err).To(HaveOccurred())
				Expect(handlerCalled).To(BeFalse())
			})
		})

		When("RequireSignatureMiddleware receives an unsigned event", func() {
			It("should reject it so I can enforce that every event in my stream is signed", func() {
				signer, err := NewHMAC([]byte("this-is-a-32-byte-secret-key-xxxx"))
				Expect(err).NotTo(HaveOccurred())

				evt := makeEvent()

				handlerCalled := false
				verifyMW := RequireSignatureMiddleware(signer)
				wrapped := verifyMW(func(_ context.Context, _ event.Event) error {
					handlerCalled = true

					return nil
				})

				err = wrapped(context.Background(), evt)
				Expect(err).To(HaveOccurred())
				Expect(handlerCalled).To(BeFalse())
			})
		})
	})

	Describe("debug inspection", func() {
		When("I need to log or display a signature", func() {
			It(
				"should give me a base64 string so I can include it in debug output without binary garbage",
				func() {
					signer, err := NewHMAC([]byte("this-is-a-32-byte-secret-key-xxxx"))
					Expect(err).NotTo(HaveOccurred())

					evt := makeEvent()
					sig, err := signer.Sign(evt)
					Expect(err).NotTo(HaveOccurred())

					str := sig.String()
					Expect(str).NotTo(BeEmpty())
					Expect(str).NotTo(ContainSubstring("\x00"))
				},
			)
		})
	})
})
