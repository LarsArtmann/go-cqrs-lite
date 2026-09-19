package main

// race_enabled reports whether the test binary was built with -race. Go's
// build-tag mechanism is the only reliable detector (GOFLAGS sniffing misses
// plain `go test -race`).
const raceEnabled = true
