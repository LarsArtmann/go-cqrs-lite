// Package lintutil holds helpers shared across the cqrs-lint rule
// subpackages (correctness, consistency, boilerplate, etc.).
package lintutil

import (
	"github.com/larsartmann/go-finding"
)

// AppendBuild appends a built [finding.Finding] to findings unless err is
// non-nil. Building a finding only fails when the builder was mis-configured
// at construction time (a programming bug in the rule itself), so the error
// is silently dropped — the rule's own unit tests catch builder bugs, and
// surfacing them in the lint output would pollute results for users.
//
// Collapses the repeated
//
//	f, err := finding.NewBuilder(...).Build()
//	if err != nil {
//	    return
//	}
//	*findings = append(*findings, f)
//
// boilerplate found in every rule's report* helper.
func AppendBuild(findings *[]finding.Finding, f finding.Finding, err error) {
	if err != nil {
		return
	}

	*findings = append(*findings, f)
}

// nonCQRSRegisterPackages lists package qualifiers whose Register/Handle
// method name collides with CQRS but serves a different purpose. Rules that
// look for CQRS handler registration must skip these to avoid false positives.
//
//nolint:gochecknoglobals // read-only denylist
var nonCQRSRegisterPackages = map[string]bool{
	"huma":  true, // Huma v2 HTTP framework: huma.Register[I,O,Body]
	"http":  true, // net/http
	"mux":   true, // gorilla/mux
	"chi":   true, // go-chi/chi
	"gin":   true, // gin-gonic/gin
	"echo":  true, // labstack/echo
	"fiber": true, // gofiber/fiber
	"grpc":  true, // grpc-go Server.Register (proto service registration)
}

// IsNonCQRSRegisterPackage reports whether pkgName is a third-party package
// qualifier whose Register/Handle method is unrelated to CQRS dispatching.
// Use to suppress false positives in rules that detect handler registration.
func IsNonCQRSRegisterPackage(pkgName string) bool {
	return nonCQRSRegisterPackages[pkgName]
}
