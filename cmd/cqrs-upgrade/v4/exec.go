package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// goOut runs a go command in dir with GOWORK=off (consumer modules must
// resolve published versions, never the local workspace) and returns stdout.
func goOut(dir string, args ...string) (string, error) {
	cmd := goCmd(dir, args...)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go %s: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

// goCmd builds a go command with the workspace isolated.
func goCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")

	return cmd
}
