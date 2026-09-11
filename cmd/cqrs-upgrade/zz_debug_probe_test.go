package main

import (
	"path/filepath"
	"testing"
)

func TestDebugProbe(t *testing.T) {
	dir := filepath.Join("..", "..", "example", "getting-started")
	findings, err := deprecationFindings(dir)
	t.Logf("err=%v findings=%d", err, len(findings))
	for _, f := range findings {
		t.Logf("finding: %+v", f)
	}
}
