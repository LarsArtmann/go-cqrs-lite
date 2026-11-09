// Package integration hosts CGo-gated integration tests that would pull
// heavy CGo dependencies (DuckDB, Arrow, FlatBuffers) into the main system
// module's go.mod if they lived there. By isolating them in a sub-module,
// consumers who import system/v4 never need a C compiler unless they also
// import a CGo engine directly.
package integration
