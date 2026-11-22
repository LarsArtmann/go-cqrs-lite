package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// killChildEnv and killPathEnv carry the re-exec contract of
// TestOSWriteFileKillMidWrite: the child process re-runs this test binary
// with the child branch active, writing to the victim path until killed.
const (
	killChildEnv = "CQRS_UPGRADE_KILL_CHILD"
	killPathEnv  = "CQRS_UPGRADE_KILL_PATH"

	killPayloadSize = 64 << 20
)

// TestOSWriteFileKillMidWrite pins the crash-safety property of osWriteFile:
// a process SIGKILLed at ANY point during a write leaves the target file
// holding either the complete OLD content or the complete NEW content —
// never a truncated mix (the staged temp file absorbs the partial write).
// The child writes a 64 MiB payload in a loop; the parent kills it after a
// short, round-staggered delay so at least one round is likely to land
// mid-staging, but the invariant is valid for every kill timing.
func TestOSWriteFileKillMidWrite(t *testing.T) {
	newPayload := bytes.Repeat([]byte("n"), killPayloadSize)

	if os.Getenv(killChildEnv) == "1" {
		path := os.Getenv(killPathEnv)
		if path == "" {
			os.Exit(3)
		}
		for {
			if err := osWriteFile(path, newPayload); err != nil {
				os.Exit(3)
			}
		}
	}

	oldPayload := bytes.Repeat([]byte("o"), killPayloadSize)

	for round := range 3 {
		path := filepath.Join(t.TempDir(), "victim.txt")
		if err := os.WriteFile(path, oldPayload, 0o644); err != nil {
			t.Fatalf("round %d: seed old content: %v", round, err)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestOSWriteFileKillMidWrite")
		cmd.Env = append(os.Environ(), killChildEnv+"=1", killPathEnv+"="+path)
		if err := cmd.Start(); err != nil {
			t.Fatalf("round %d: start child: %v", round, err)
		}

		//go-sleep:accept kill-mid-write race window — the delay aims the SIGKILL into the child's staging write
		time.Sleep(time.Duration(20+round*15) * time.Millisecond)

		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("round %d: kill child: %v", round, err)
		}
		_ = cmd.Wait()

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("round %d: read after kill: %v", round, err)
		}
		if !bytes.Equal(got, oldPayload) && !bytes.Equal(got, newPayload) {
			t.Fatalf("round %d: victim file is neither old nor new content (%d bytes)", round, len(got))
		}
	}
}
