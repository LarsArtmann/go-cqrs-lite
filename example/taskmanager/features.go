package main

import (
	"crypto/rand"

	"github.com/larsartmann/go-cqrs-lite/signing/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Features — signing helper.
//
// Event bus signing middleware is installed in setup.go via sys.Bus().
// OTel middleware is configured in setup.go via DomainConfig.Middleware.
// This file provides only the HMAC signer factory.
// ──────────────────────────────────────────────────────────────────────────

const hmacKeyBytes = 32

// newDemoSigner creates an HMAC-SHA256 signer-verifier with a random key.
// In production, load the key from a secret manager (vault, KMS, etc.).
//
//nolint:ireturn // factory returning interface for signing abstraction
func newDemoSigner() signing.SignerVerifier {
	key := make([]byte, hmacKeyBytes)
	if _, err := rand.Read(key); err != nil {
		//cqrs-lint:ignore(C009) library code or intentional pattern
		panic("failed to generate signing key: " + err.Error())
	}

	signer, err := signing.NewHMAC(key)
	if err != nil {
		//cqrs-lint:ignore(C009) library code or intentional pattern
		panic("failed to create HMAC signer: " + err.Error())
	}

	return signer
}
