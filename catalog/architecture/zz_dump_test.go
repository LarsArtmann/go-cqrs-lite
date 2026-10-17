package architecture_test

import (
	"os"
	"testing"
)

func TestZZDump2(t *testing.T) {
	os.WriteFile("/tmp/archref.html", []byte(renderFixture(t)), 0o600)
}
