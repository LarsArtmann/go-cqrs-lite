package main

import (
	"os"

	atomicwrite "github.com/larsartmann/go-atomic-write"
)

// osReadFile isolates the os dependency for test seams.
func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// osWriteFile writes data to path atomically (staged temp file + fsync +
// rename), so an interrupted run can never leave a truncated file behind.
// The target's existing mode is preserved.
func osWriteFile(path string, data []byte) error {
	return atomicwrite.Write(path, data)
}
