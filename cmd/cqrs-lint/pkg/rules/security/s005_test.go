package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

// --- S005: Signing available but disabled ---

func TestS005_NoCrashOnEmptyInput(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_DetectsSigningGuardedByDefaultFalseFlag(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type SignerConfig struct {
	SigningEnabled bool
}

func setupSigning(cfg SignerConfig, key []byte) {
	if cfg.SigningEnabled {
		signer, _ := signing.NewHMAC(key)
		_ = signer
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 1)
}

func TestS005_DetectsSigningMiddlewareGuarded(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	Enabled bool
}

func wireSigning(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.Enabled {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 1)
}

func TestS005_SuppressedWhenSigningNotImported(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	SigningEnabled bool
}

func setup(cfg Config) {
	if cfg.SigningEnabled {
		doSomething()
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_SuppressedWhenSigningUsedUnconditionally(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	SigningEnabled bool
}

func setup(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.SigningEnabled {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
	bus.UsePublish(signing.SignMiddleware(signer))
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_SuppressedWhenFlagExplicitlyTrue(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	SigningEnabled bool
}

func defaultConfig() Config {
	return Config{SigningEnabled: true}
}

func setup(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.SigningEnabled {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_NoFindingWhenSigningBehindErrorCheck(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

func setup(key []byte, bus EventBus) {
	signer, err := signing.NewHMAC(key)
	if err != nil {
		return
	}
	bus.UsePublish(signing.SignMiddleware(signer))
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_NoFindingWhenNoSigningCalls(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	SigningEnabled bool
}

func setup(cfg Config) {
	if cfg.SigningEnabled {
		println("signing would go here")
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}

func TestS005_DetectsVerifyMiddlewareGuarded(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	VerifyEnabled bool
}

func setup(cfg Config, verifier signing.Verifier, bus EventBus) {
	if cfg.VerifyEnabled {
		bus.Use(signing.VerifyMiddleware(verifier))
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 1)
}

func TestS005_NoFindingForNonEnableBoolField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	Debug bool
}

func setup(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.Debug {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
}
`,
	})
	findings := runDetector(t, security.NewS005Detector(ctx))
	assertRule(t, findings, "S005", 0)
}
