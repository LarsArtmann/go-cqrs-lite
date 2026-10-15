package main

// The runnable twin of recipes.md §2.47 ("Bind a system Query to an HTTP
// endpoint"): decode → typed dispatch → deterministic encode, with a strong
// content-hash ETag (If-None-Match → 304), a Server-Timing "query" phase, and
// errorfamily's family→status mapping doing the error responses. The behavior
// is pinned by query_http_test.go.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"
	etag "github.com/larsartmann/go-etag/server"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// bindQuery binds a registered system query to net/http. The result type is
// the explicit parameter (nothing at the call site infers it); Q infers from
// the decoder. Errors return to errorfamily.HTTPHandler, which maps
// family → status (Rejection→400, Conflict→409, Transient→503, …) without
// leaking the raw error.
func bindQuery[R any, Q query.Query](
	sys *system.System,
	decode func(r *http.Request) (Q, error),
	c codec.Codec,
) errorfamily.HandlerFunc {
	mediaType := "application/json"
	if c.Encoding() == codec.EncodingCBOR {
		mediaType = "application/cbor"
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		q, err := decode(r)
		if err != nil {
			return err
		}

		stop := servertiming.MeasureServerTiming(r.Context(), "query")
		data, tag, err := dispatchEncode[R](r.Context(), sys, q, c)
		stop() // stop BEFORE any write: the header is injected at first Write
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", mediaType)
		w.Header().Set("ETag", tag.String())
		if etag.MatchesIfNoneMatch(tag, r.Header.Get("If-None-Match")) {
			w.WriteHeader(http.StatusNotModified)

			return nil
		}
		_, err = w.Write(data)

		return err
	}
}

// dispatchEncode runs the typed dispatch and encodes the result with c. The
// ETag is the SHA-256 of the encoded bytes: any state change that alters the
// response also invalidates the cache, with no read-model cooperation needed.
func dispatchEncode[R any, Q query.Query](
	ctx context.Context,
	sys *system.System,
	q Q,
	c codec.Codec,
) ([]byte, etag.ETag, error) {
	result, err := system.DispatchQuery[Q, R](ctx, sys, q)
	if err != nil {
		return nil, etag.ETag{}, err
	}

	data, err := c.Encode(result)
	if err != nil {
		return nil, etag.ETag{}, err
	}

	sum := sha256.Sum256(data)

	return data, etag.NewETag(hex.EncodeToString(sum[:]), etag.Strong), nil
}
