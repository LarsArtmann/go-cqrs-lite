package security_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

func TestS005(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filename  string
		source    string
		wantCount int
	}{
		{"NoCrashOnEmptyInput", "main.go", `package main`, 0},
		{"DetectsSigningGuardedByDefaultFalseFlag", "config.go", `package main

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
`, 1},
		{"DetectsSigningMiddlewareGuarded", "server.go", `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	Enabled bool
}

func wireSigning(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.Enabled {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
}
`, 1},
		{"SuppressedWhenSigningNotImported", "config.go", `package main

type Config struct {
	SigningEnabled bool
}

func setup(cfg Config) {
	if cfg.SigningEnabled {
		doSomething()
	}
}
`, 0},
		{"SuppressedWhenSigningUsedUnconditionally", "server.go", `package main

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
`, 0},
		{"SuppressedWhenFlagExplicitlyTrue", "server.go", `package main

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
`, 0},
		{"NoFindingWhenSigningBehindErrorCheck", "server.go", `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

func setup(key []byte, bus EventBus) {
	signer, err := signing.NewHMAC(key)
	if err != nil {
		return
	}
	bus.UsePublish(signing.SignMiddleware(signer))
}
`, 0},
		{"NoFindingWhenNoSigningCalls", "server.go", `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	SigningEnabled bool
}

func setup(cfg Config) {
	if cfg.SigningEnabled {
		println("signing would go here")
	}
}
`, 0},
		{"DetectsVerifyMiddlewareGuarded", "server.go", `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	VerifyEnabled bool
}

func setup(cfg Config, verifier signing.Verifier, bus EventBus) {
	if cfg.VerifyEnabled {
		bus.Use(signing.VerifyMiddleware(verifier))
	}
}
`, 1},
		{"NoFindingForNonEnableBoolField", "server.go", `package main

import "github.com/larsartmann/go-cqrs-lite/signing/v4"

type Config struct {
	Debug bool
}

func setup(cfg Config, signer signing.Signer, bus EventBus) {
	if cfg.Debug {
		bus.UsePublish(signing.SignMiddleware(signer))
	}
}
`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := analyzer.BuildContextFromSource(t, map[string]string{
				tt.filename: tt.source,
			})
			findings := ruletest.RunDetector(t, security.NewS005Detector(ctx))
			ruletest.AssertRule(t, findings, "S005", tt.wantCount)
		})
	}
}
