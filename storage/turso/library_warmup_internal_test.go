package turso

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestTursoLibraryWarmupChild is the re-exec child leg of
// TestTursoLibraryWarmup_ConcurrentProcesses: a fresh process with a cold
// TURSO_GO_CACHE_DIR warms the library and proves the native .so landed in
// the cache before exiting 0.
func TestTursoLibraryWarmupChild(t *testing.T) {
	if os.Getenv("CQRS_TURSO_WARM_CHILD") != "1" {
		t.Skip("parent mode — spawned by TestTursoLibraryWarmup_ConcurrentProcesses")
	}

	if err := warmTursoLibrary(); err != nil {
		t.Fatalf("warmTursoLibrary: %v", err)
	}

	cacheRoot := os.Getenv("TURSO_GO_CACHE_DIR")
	found := false
	err := filepath.WalkDir(cacheRoot, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".so" && d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			found = info.Size() > 0
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk cache dir %s: %v", cacheRoot, err)
	}
	if !found {
		t.Fatalf("no native library extracted under %s", cacheRoot)
	}
}

// TestTursoLibraryWarmup_ConcurrentProcesses pins the upstream loader race
// (turso-go-platform-libs v0.8.2 writes the shared cache .so without locking
// or atomic rename): several processes cold-starting against the SAME fresh
// cache dir must all warm cleanly. Without the lockfile guard this is the
// "cached library file hash sum mismatch" panic class seen on CI (run 38548187646,
// Module storage/turso). The race is probabilistic per child pair — 6 children
// maximize the interleaving window.
func TestTursoLibraryWarmup_ConcurrentProcesses(t *testing.T) {
	cacheDir := t.TempDir()

	const children = 6
	type child struct {
		cmd *exec.Cmd
		out *bytes.Buffer
	}
	cmds := make([]child, 0, children)
	for range children {
		out := &bytes.Buffer{}
		cmd := exec.Command(os.Args[0], "-test.run=^TestTursoLibraryWarmupChild$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			"CQRS_TURSO_WARM_CHILD=1",
			"TURSO_GO_CACHE_DIR="+cacheDir,
		)
		cmd.Stdout = out
		cmd.Stderr = out
		cmds = append(cmds, child{cmd: cmd, out: out})
	}

	for _, c := range cmds {
		if err := c.cmd.Start(); err != nil {
			t.Fatalf("start child %s: %v", c.cmd.Path, err)
		}
	}

	for i, c := range cmds {
		if err := c.cmd.Wait(); err != nil {
			t.Fatalf("child %d failed (%v): %s", i, err, c.out)
		}
	}
}

// TestAcquireTursoLockFile_Blocking pins the cross-process serialization
// primitive: while the lockfile exists a second acquirer blocks, and it
// unblocks the moment the lock is removed.
func TestAcquireTursoLockFile_Blocking(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "turso-go", "cqrs-first-open.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}

	first, ok := acquireTursoLockFile(lockPath)
	if !ok {
		t.Fatal("first acquire did not get the lock")
	}

	acquired := make(chan bool, 1)
	go func() {
		_, second := acquireTursoLockFile(lockPath)
		acquired <- second
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire succeeded while the lock was held")
	case <-time.After(300 * time.Millisecond):
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}

	select {
	case second := <-acquired:
		if !second {
			t.Fatal("second acquire failed after the lock was released")
		}
	case <-time.After(tursoLibraryWarmupTimeout + 5*time.Second):
		t.Fatal("second acquire never returned after the lock was released")
	}
}
