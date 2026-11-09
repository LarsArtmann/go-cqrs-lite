// Package integration provides cross-module tests that verify correct interaction
// between CQRS components (event, command, query, signing, simulation).
//
// These tests cannot live in individual modules because they exercise code paths
// that span multiple packages with different go.mod files.
package integration
