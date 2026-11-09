package main

import "os"

// osReadFile isolates the os dependency for test seams.
func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// osWriteFile writes data to path.
func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
