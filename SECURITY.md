# Security Policy

## Supported Versions

This library is pre-v1. Security fixes are applied to the latest `master` branch only.

| Version | Supported          |
| ------- | ------------------ |
| master  | :white_check_mark: |
| < 1.0   | :x: (pre-release)  |

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities privately:

1. Open a GitHub Security Advisory via the **Security** tab → **Report a vulnerability**.
2. Or email the maintainer directly.

Please include:

- A description of the vulnerability and its potential impact.
- Steps to reproduce or a proof-of-concept.
- The affected module(s) (e.g., `signing/`, `encryption/`).
- Any suggested mitigations.

**Response timeline:**

- Acknowledgement within 48 hours.
- Initial assessment within 5 business days.
- Fix or mitigation published as soon as practicable, depending on severity.

## Scope

This library provides cryptographic building blocks for CQRS + Event Sourcing applications:

### `signing/` — Event Integrity

- **HMAC-SHA256** for symmetric event signing.
- **Ed25519** for asymmetric event signing (recommended for distributed systems).
- **Multisig** support for multi-party event integrity.
- Middleware for transparent sign-on-publish and verify-on-consume.

### `encryption/` — Event Confidentiality

- **XChaCha20-Poly1305** for authenticated encryption (recommended — nonce-misuse resistant, 192-bit nonce).
- **AES-256-GCM** for compatibility with systems requiring NIST-approved ciphers.
- HKDF-SHA256 key derivation for multi-tenant encryption.
- Codec wrapper for seamless encrypt/decrypt integration with the codec pipeline.
- Middleware for transparent encrypt-on-publish and decrypt-on-consume.

## Out of Scope

This library does **not** provide:

- Key management (rotation, storage, HSM integration) — bring your own KMS.
- Transport-layer security (TLS) — use your HTTP/gRPC server's TLS configuration.
- Access control / authorization — handled at the application layer.
- Side-channel resistance guarantees beyond what the underlying Go `crypto/*` packages provide.

## Security Recommendations for Consumers

1. **Never hardcode secrets** — load encryption keys and signing secrets from a secrets manager (Vault, AWS KMS, etc.).
2. **Rotate keys regularly** — use the `keyID` field on signed/encrypted events to support key rotation.
3. **Prefer Ed25519 for signing** — it is faster, has smaller signatures, and eliminates nonce-reuse risks.
4. **Prefer XChaCha20-Poly1305 for encryption** — the 192-bit nonce makes random-nonce collision practically impossible.
5. **Enable both signing AND encryption** — signing alone protects integrity but not confidentiality; encryption alone protects confidentiality but not integrity.
6. **Verify on every consume** — do not skip `signing.VerifyMiddleware` even in "trusted" internal services.
