# Security Policy

## Supported Versions

This is a library/SDK. Only the latest release line receives security fixes.

| Version | Supported          |
| ------- | ------------------ |
| v3.x    | :white_check_mark: |
| < v3    | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in go-cqrs-lite, please report it
responsibly:

1. **Do NOT open a public GitHub issue.**
2. Email the maintainer directly with details of the vulnerability.
3. Include a proof of concept or steps to reproduce if possible.
4. You will receive an acknowledgment within 48 hours.

## Security Features

The library provides opt-in security modules:

- **Event signing** (`signing/`): HMAC-SHA256 and Ed25519 signatures with
  COSE_Sign1 (RFC 9052) support. Tamper-evident event streams.
- **Event encryption** (`encryption/`): XChaCha20-Poly1305 and AES-256-GCM
  authenticated encryption for confidential payloads. HKDF key derivation
  for multi-tenant scenarios.
- **Input validation** (`schema/`): JSON Schema validation with typed
  registration. Prevents malformed payloads from entering the event stream.

These modules are **opt-in**. If you do not wire them into your middleware
chain, events are stored and transported in plaintext without integrity
protection.

## Dependency Policy

- Production dependencies are kept minimal and audited.
- Direct production dependencies: `oklog/ulid`, `go-branded-id`,
  `go-error-family`, `go-faster/yaml`, `fxamacker/cbor`, `golang.org/x/crypto`.
- Test-only dependencies (`ginkgo`, `gomega`, `rapid`) are excluded from
  production builds.
- Per-module dependency budgets are enforced by `nix run .#check-layers`.
