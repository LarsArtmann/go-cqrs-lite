package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMain_RunsAndPrintsUser(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	func() {
		defer func() { os.Stdout = oldStdout }()
		main()
	}()

	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "User: Alice") {
		t.Fatalf("expected output to contain 'User: Alice', got %q", out)
	}
}
