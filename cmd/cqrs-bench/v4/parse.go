package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
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
			"%w: unknown durability tier %q (use strict, normal, or relaxed)", errFlagValue, s,
		)
	}
}

// codecByName resolves --codec into the payload encoder.
func codecByName(name string) (codec.Codec, error) {
	switch name {
	case "json":
		return codec.JSONCodec{}, nil
	case "cbor":
		return codec.CBORCodec{}, nil
	default:
		return nil, fmt.Errorf("%w %q (use: json, cbor)", errCodec, name)
	}
}

// profileByName resolves --profile into a workload profile.
func profileByName(name string) (benchkit.Profile, error) {
	profile, ok := benchkit.ProfileByName(name)
	if !ok {
		return benchkit.Profile{}, fmt.Errorf("%w %q (use: %s)", errProfile, name, strings.Join(profileNames(), ", "))
	}

	return profile, nil
}

// profileNames lists the named workload profiles in help order. benchkit
// exposes no list of its own, so this one is pinned against
// benchkit.ProfileByName and the --profile flag by TestProfileNames.
func profileNames() []string {
	return []string{
		"dev", "small", "medium", "large", "stress", "write-heavy", "read-heavy", "analytical",
	}
}

// parseSizeList parses a comma-separated list of positive integers such as
// "64,256,4096". An empty value yields nil. flagName names the flag in error
// messages.
//
// pflag blindly consumes the next token as a string flag's value, so
// "--payload-sizes --profile stress" sets payload-sizes to "--profile". That
// pattern gets an actionable error instead of a confusing strconv one.
func parseSizeList(flagName, value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if looksLikeFlag(value) {
		return nil, fmt.Errorf(
			"%w: --%s %q looks like a flag name, not a number list; "+
				"use --flag=VALUE (e.g. --%s=64,256,4096) or place the flag last",
			errFlagValue, flagName, value, flagName,
		)
	}

	parts := strings.Split(value, ",")
	numbers := make([]int, 0, len(parts))

	for _, part := range parts {
		number, err := parseSize(flagName, strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}

		numbers = append(numbers, number)
	}

	return numbers, nil
}

func parseSize(flagName, text string) (int, error) {
	number, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%w: --%s has an invalid number %q", errFlagValue, flagName, text)
	}

	if number <= 0 {
		return 0, fmt.Errorf("%w: --%s entries must be greater than 0, got %d", errFlagValue, flagName, number)
	}

	return number, nil
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
