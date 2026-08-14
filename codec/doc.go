// Package codec is DEPRECATED. Import github.com/larsartmann/go-codec directly.
//
// This module was the original home of the Codec interface and implementations
// (CBORCodec, JSONCodec, CBORCompactCodec, RawCodec). It has been extracted
// into the standalone repository github.com/larsartmann/go-codec.
//
// All types, functions, constants, and error sentinels are re-exported as
// type/variable aliases so existing imports continue to compile. New code
// should import go-codec directly:
//
//	import "github.com/larsartmann/go-codec"
//
// To migrate, change your import from:
//
//	import "github.com/larsartmann/go-codec"
//
// to:
//
//	import "github.com/larsartmann/go-codec"
//
// All symbols are identical (type aliases). No code changes are needed beyond
// the import path.
package codec
