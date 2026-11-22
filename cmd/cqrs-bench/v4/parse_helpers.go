package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func parseDurability(s string) (stack.DurabilityTier, bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return stack.DurabilityStrict, true, nil
	case "normal":
		return stack.DurabilityNormal, true, nil
	case "relaxed":
		return stack.DurabilityRelaxed, true, nil
	case "":
		return stack.DurabilityNormal, false, nil
	default:
		return "", false, fmt.Errorf(
			"unknown durability tier: %s (use strict, normal, or relaxed)", s,
		)
	}
}

func parseCodec(name string) codec.Codec {
	switch name {
	case "json":
		return codec.JSONCodec{}
	case "cbor":
		return codec.CBORCodec{}
	default:
		fatalf("unknown codec: %s (use json or cbor)", name)

		return nil
	}
}

// parsePayloadSizes parses a comma-separated list of payload sizes (e.g.
// "64,256,4096") into an int slice. Returns nil for an empty string (meaning:
// use the single --payload-size). Returns an error on malformed input.
//
// pflag blindly consumes the next token as a string flag's value, so
// "--payload-sizes --profile stress" sets payload-sizes to "--profile". This
// guard detects that pattern and returns an actionable error instead of a
// confusing strconv error.
func parsePayloadSizes(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	if looksLikeFlag(s) {
		return nil, fmt.Errorf(
			"value %q looks like a flag name, not a size list — "+
				"use --flag=VALUE syntax (e.g. --payload-sizes=64,256,4096) "+
				"or place the flag last in the command line",
			s,
		)
	}

	parts := strings.Split(s, ",")
	sizes := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", part, err)
		}

		if n <= 0 {
			return nil, errorfamily.Newf(errorfamily.Rejection,
				"cqrs-bench.invalid_size", "size must be > 0, got %d", n)
		}

		sizes = append(sizes, n)
	}

	if len(sizes) < 2 {
		return nil, errorfamily.Newf(errorfamily.Rejection,
			"cqrs-bench.too_few_sizes",
			"provide at least 2 sizes for a mixed workload, got %d", len(sizes))
	}

	return sizes, nil
}

// looksLikeFlag reports whether s appears to be a flag name that pflag
// consumed as a value (e.g. "--profile" or "-p"). Negative numbers like
// "-1" are NOT treated as flag-like.
func looksLikeFlag(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}

	// "--anything" is a long flag.
	if s[1] == '-' {
		return true
	}

	// "-x" where x is a letter is a short flag; "-1" or "-1.5" is not.
	return !strings.ContainsAny(s[1:2], "0123456789.")
}
