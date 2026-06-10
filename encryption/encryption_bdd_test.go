package encryption_test

import (
	"context"
	"crypto/rand"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	. "github.com/larsartmann/go-cqrs-lite/encryption/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var _ = Describe("Encryption", func() {
	makeEvent := func(payload string) *event.ImmutableEvent {
		aggID := id.NewAggregateID()
		evt, err := event.NewEvent("user.created", aggID, "User", 1, []byte(payload))
		Expect(err).NotTo(HaveOccurred())

		return evt
	}

	generateKey := func() []byte {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		Expect(err).NotTo(HaveOccurred())

		return key
	}

	Describe("AES-256-GCM encryption", func() {
		When("I encrypt and decrypt a payload with the same key", func() {
			It("should restore the original plaintext so I can store data securely", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				plaintext := []byte(`{"name":"Alice","ssn":"123-45-6789"}`)

				ct, err := enc.Encrypt(plaintext)
				Expect(err).NotTo(HaveOccurred())
				Expect(ct).NotTo(BeNil())

				decrypted, err := enc.Decrypt(ct)
				Expect(err).NotTo(HaveOccurred())
				Expect(decrypted).To(Equal(plaintext))
			})
		})

		When("I encrypt the same payload twice", func() {
			It("should produce different ciphertexts because nonces are random", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				payload := []byte(`{"name":"Alice"}`)

				ct1, err := enc.Encrypt(payload)
				Expect(err).NotTo(HaveOccurred())
				ct2, err := enc.Encrypt(payload)
				Expect(err).NotTo(HaveOccurred())

				Expect(ct1).NotTo(Equal(ct2))
			})
		})

		When("I try to decrypt with the wrong key", func() {
			It("should fail so unauthorized readers cannot access the data", func() {
				key1 := generateKey()
				key2 := generateKey()

				enc1, err := NewAES256GCM(key1)
				Expect(err).NotTo(HaveOccurred())
				enc2, err := NewAES256GCM(key2)
				Expect(err).NotTo(HaveOccurred())

				ct, err := enc1.Encrypt([]byte(`{"secret":"data"}`))
				Expect(err).NotTo(HaveOccurred())

				_, err = enc2.Decrypt(ct)
				Expect(err).To(HaveOccurred())
			})
		})

		When("I try to create an AES-256-GCM with a wrong-sized key", func() {
			It("should reject my key so I cannot accidentally use weak cryptography", func() {
				_, err := NewAES256GCM([]byte("too-short"))
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidKey)).To(BeTrue())
			})
		})

		When("I encrypt an empty payload", func() {
			It("should return nil ciphertext without error", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				ct, err := enc.Encrypt([]byte{})
				Expect(err).NotTo(HaveOccurred())
				Expect(ct.IsZero()).To(BeTrue())
			})
		})

		When("I decrypt an empty ciphertext", func() {
			It("should return nil plaintext without error", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				plain, err := enc.Decrypt(nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(plain).To(BeNil())
			})
		})

		When("I decrypt a truncated ciphertext", func() {
			It("should fail with a decryption error", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				_, err = enc.Decrypt(Ciphertext([]byte{1, 2, 3}))
				Expect(errors.Is(err, ErrDecryptionFailed)).To(BeTrue())
			})
		})
	})

	Describe("XChaCha20-Poly1305 encryption", func() {
		When("I encrypt and decrypt a payload with the same key", func() {
			It("should restore the original plaintext so I can store data securely", func() {
				key := generateKey()
				enc, err := NewXChaCha20Poly1305(key)
				Expect(err).NotTo(HaveOccurred())

				plaintext := []byte(`{"name":"Alice","ssn":"123-45-6789"}`)

				ct, err := enc.Encrypt(plaintext)
				Expect(err).NotTo(HaveOccurred())
				Expect(ct).NotTo(BeNil())

				decrypted, err := enc.Decrypt(ct)
				Expect(err).NotTo(HaveOccurred())
				Expect(decrypted).To(Equal(plaintext))
			})
		})

		When("I encrypt the same payload twice with XChaCha20", func() {
			It("should produce different ciphertexts because nonces are random", func() {
				key := generateKey()
				enc, err := NewXChaCha20Poly1305(key)
				Expect(err).NotTo(HaveOccurred())

				payload := []byte(`{"name":"Alice"}`)

				ct1, err := enc.Encrypt(payload)
				Expect(err).NotTo(HaveOccurred())
				ct2, err := enc.Encrypt(payload)
				Expect(err).NotTo(HaveOccurred())

				Expect(ct1).NotTo(Equal(ct2))
			})
		})

		When("I try to create XChaCha20 with a wrong-sized key", func() {
			It("should reject my key so I cannot accidentally use weak cryptography", func() {
				_, err := NewXChaCha20Poly1305([]byte("too-short"))
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidKey)).To(BeTrue())
			})
		})
	})

	Describe("publish pipeline middleware", func() {
		When("I use EncryptMiddleware on my publisher", func() {
			It("should encrypt the event payload so data at rest is protected", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				encryptMW := EncryptMiddleware(enc)

				original := makeEvent(`{"ssn":"123-45-6789"}`)

				var captured event.Event
				inner := event.PublisherFunc(
					func(_ context.Context, events ...event.Event) error {
						if len(events) > 0 {
							captured = events[0]
						}

						return nil
					},
				)

				wrapped := encryptMW(inner)
				err = wrapped.Publish(context.Background(), original)
				Expect(err).NotTo(HaveOccurred())

				Expect(HasEncryption(captured)).To(BeTrue())

				ct, err := ExtractCiphertext(captured)
				Expect(err).NotTo(HaveOccurred())

				decrypted, err := enc.Decrypt(ct)
				Expect(err).NotTo(HaveOccurred())
				Expect(decrypted).To(Equal([]byte(`{"ssn":"123-45-6789"}`)))
			})
		})

		When("EncryptMiddleware receives an event with empty payload", func() {
			It("should pass it through without encrypting", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				encryptMW := EncryptMiddleware(enc)

				aggID := id.NewAggregateID()
				emptyEvt, err := event.NewEvent("snapshot.taken", aggID, "User", 1, []byte{})
				Expect(err).NotTo(HaveOccurred())

				var captured event.Event
				inner := event.PublisherFunc(
					func(_ context.Context, events ...event.Event) error {
						if len(events) > 0 {
							captured = events[0]
						}

						return nil
					},
				)

				err = encryptMW(inner).Publish(context.Background(), emptyEvt)
				Expect(err).NotTo(HaveOccurred())
				Expect(HasEncryption(captured)).To(BeFalse())
			})
		})

		When("DecryptMiddleware receives an encrypted event", func() {
			It("should decrypt the payload so my handler sees plaintext", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				decryptMW := DecryptMiddleware(enc)

				original := makeEvent(`{"name":"Bob"}`)

				ct, err := enc.Encrypt(event.PayloadReadOnly(original))
				Expect(err).NotTo(HaveOccurred())

				encryptedEvt, err := AttachEncryption(original, ct)
				Expect(err).NotTo(HaveOccurred())

				var captured event.Event
				handler := func(_ context.Context, evt event.Event) error {
					captured = evt

					return nil
				}

				err = decryptMW(handler)(context.Background(), encryptedEvt)
				Expect(err).NotTo(HaveOccurred())
				Expect(captured.Payload()).To(Equal([]byte(`{"name":"Bob"}`)))
			})
		})

		When("DecryptMiddleware receives a plaintext event", func() {
			It("should pass it through without error for mixed streams", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				decryptMW := DecryptMiddleware(enc)

				plainEvt := makeEvent(`{"name":"Carol"}`)

				var captured event.Event
				handler := func(_ context.Context, evt event.Event) error {
					captured = evt

					return nil
				}

				err = decryptMW(handler)(context.Background(), plainEvt)
				Expect(err).NotTo(HaveOccurred())
				Expect(captured.Payload()).To(Equal([]byte(`{"name":"Carol"}`)))
			})
		})

		When("EncryptMiddleware is called with nil encrypter", func() {
			It("should reject all publishes so misconfiguration fails loudly", func() {
				mw := EncryptMiddleware(nil)

				inner := event.PublisherFunc(
					func(_ context.Context, _ ...event.Event) error { return nil },
				)

				err := mw(inner).Publish(context.Background(), makeEvent(`{}`))
				Expect(err).To(HaveOccurred())
			})
		})

		When("DecryptMiddleware is called with nil decrypter", func() {
			It("should reject all events so misconfiguration fails loudly", func() {
				mw := DecryptMiddleware(nil)

				err := mw(func(_ context.Context, _ event.Event) error { return nil })(
					context.Background(), makeEvent(`{}`),
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("round-trip through middleware pipeline", func() {
		When("I encrypt on publish and decrypt on handle", func() {
			It("should preserve the original payload end-to-end", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				encryptMW := EncryptMiddleware(enc)
				decryptMW := DecryptMiddleware(enc)

				original := makeEvent(`{"email":"alice@example.com","ssn":"999-88-7777"}`)

				var publishedEvent event.Event
				bus := event.PublisherFunc(
					func(_ context.Context, events ...event.Event) error {
						if len(events) > 0 {
							publishedEvent = events[0]
						}

						return nil
					},
				)

				encryptedBus := encryptMW(bus)
				err = encryptedBus.Publish(context.Background(), original)
				Expect(err).NotTo(HaveOccurred())

				Expect(publishedEvent).NotTo(BeNil())
				Expect(publishedEvent.Payload()).NotTo(Equal(original.Payload()))

				var handledEvent event.Event
				handler := func(_ context.Context, evt event.Event) error {
					handledEvent = evt

					return nil
				}

				err = decryptMW(handler)(context.Background(), publishedEvent)
				Expect(err).NotTo(HaveOccurred())

				Expect(handledEvent.Payload()).To(Equal(original.Payload()))
			})
		})
	})

	Describe("encrypting codec wrapper", func() {
		When("I use NewCodec with XChaCha20 and JSON", func() {
			It("should serialize, encrypt, decrypt, and deserialize transparently", func() {
				key := generateKey()
				enc, err := NewXChaCha20Poly1305(key)
				Expect(err).NotTo(HaveOccurred())

				c := NewCodec(codec.JSONCodec{}, enc)

				type Secret struct {
					SSN string `json:"ssn"`
				}

				original := Secret{SSN: "123-45-6789"}

				data, err := c.Encode(original)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeEmpty())

				var decoded Secret
				err = c.Decode(data, &decoded)
				Expect(err).NotTo(HaveOccurred())
				Expect(decoded.SSN).To(Equal("123-45-6789"))
			})
		})

		When("I encode with the encrypting codec", func() {
			It("should produce ciphertext, not plaintext JSON", func() {
				key := generateKey()
				enc, err := NewAES256GCM(key)
				Expect(err).NotTo(HaveOccurred())

				c := NewCodec(codec.JSONCodec{}, enc)

				data, err := c.Encode(map[string]string{"secret": "value"})
				Expect(err).NotTo(HaveOccurred())

				Expect(string(data)).NotTo(ContainSubstring("secret"))
				Expect(string(data)).NotTo(ContainSubstring("value"))
			})
		})

		When("I decode with the wrong key", func() {
			It("should fail so unauthorized readers cannot access the data", func() {
				key1 := generateKey()
				key2 := generateKey()

				enc1, _ := NewXChaCha20Poly1305(key1)
				enc2, _ := NewXChaCha20Poly1305(key2)

				encodeCodec := NewCodec(codec.JSONCodec{}, enc1)
				decodeCodec := NewCodec(codec.JSONCodec{}, enc2)

				data, err := encodeCodec.Encode(map[string]string{"secret": "data"})
				Expect(err).NotTo(HaveOccurred())

				var result map[string]string
				err = decodeCodec.Decode(data, &result)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
