package pebble

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
)

func openScratchDB(b *testing.B, base string) *pebble.DB {
	b.Helper()

	if err := os.MkdirAll(base, 0o755); err != nil {
		b.Skipf("scratch dir %s unavailable: %v", base, err)
	}

	dir, err := os.MkdirTemp(base, "scratch-*")
	if err != nil {
		b.Skipf("mkdir: %v", err)
	}

	b.Cleanup(func() { _ = os.RemoveAll(dir) })

	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("open: %v", err)
	}

	b.Cleanup(func() { _ = db.Close() })

	return db
}

func BenchmarkScratchSetAsync_Disk(b *testing.B) {
	db := openScratchDB(b, filepath.Join(os.Getenv("HOME"), ".cache", "pebble-scratch"))
	for i := 0; i < b.N; i++ {
		if err := db.Set([]byte{byte(i), byte(i >> 8)}, []byte("x"), nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScratchSetSync_Disk(b *testing.B) {
	db := openScratchDB(b, filepath.Join(os.Getenv("HOME"), ".cache", "pebble-scratch"))
	for i := 0; i < b.N; i++ {
		if err := db.Set([]byte{byte(i), byte(i >> 8)}, []byte("x"), pebble.Sync); err != nil {
			b.Fatal(err)
		}
	}
}
