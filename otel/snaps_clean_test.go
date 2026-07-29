package otel_test

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestMain(m *testing.M) {
	code := m.Run()
	snaps.Clean(m) //art-dupl:accept go-snaps requires TestMain in-package; cannot be shared across Go modules
	os.Exit(code)
}
