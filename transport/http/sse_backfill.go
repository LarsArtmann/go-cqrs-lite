package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// maxBackfillLimit caps the number of events a single backfill response will
// return. Consumers asking for more get clamped to this cap to bound response
// time and memory. Replays use the same cap by default (see SSEBroker).
const maxBackfillLimit = 1000

// SSEAuthMiddleware wraps an http.Handler with token-based authentication.
// The tokenFunc extracts and validates a bearer token from the request,
// returning the authenticated client ID on success.
//
// This is a reference implementation — consumers can use it directly or as
// a template for JWT/OAuth/session-based auth. The tokenFunc should return
// an empty SSEClientID and false on authentication failure.
//
// Example:
//
//	handler := SSEAuthMiddleware(SSEHandler(broker), func(r *http.Request) (SSEClientID, bool) {
//	    token := r.Header.Get("Authorization")
//	    userID, err := validateJWT(token)
//	    return SSEClientID(userID), err == nil
//	})
//	http.Handle("/events", handler)
func SSEAuthMiddleware(
	next http.Handler,
	tokenFunc func(*http.Request) (SSEClientID, bool),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := tokenFunc(r)
		if !ok || clientID.IsZero() {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)

			return
		}

		// Inject the authenticated client ID as a query param so SSEHandler
		// picks it up. This keeps the existing SSEHandler contract unchanged.
		q := r.URL.Query()
		q.Set("client", clientID.String())
		r.URL.RawQuery = q.Encode()

		next.ServeHTTP(w, r)
	})
}

// BackfillHandler returns an HTTP handler that responds with missed events
// as a JSON array — a REST complement to SSE streaming. Clients that miss
// events while offline (or exceed the SSE replay window) can call this endpoint
// to catch up synchronously.
//
// The handler accepts these query parameters:
//   - after: the EventID to start from (exclusive). Required.
//   - limit: maximum events to return (default: 100, max: 1000).
//
// Response: 200 OK with `[]event.Event` JSON array, or 400/500 on error.
//
// Payload bytes are sent raw (same encoding as stored). For wire-format
// transcoding (e.g., CBOR→JSON), use BackfillHandlerWithTransform.
func BackfillHandler(journal event.SeekableJournal) http.Handler {
	return BackfillHandlerWithTransform(journal, nil)
}

// BackfillHandlerWithTransform is like BackfillHandler but applies the given
// transform to each event's payload before serializing. Pass nil for raw
// payload bytes (identical to BackfillHandler).
//
// This variant exists so consumers using CBOR (or any non-JSON codec) can
// transcode payloads to JSON for browser-compatible REST responses, matching
// the behavior of WithPayloadTransform on SSEBroker.
func BackfillHandlerWithTransform(
	journal event.SeekableJournal,
	transform func(event.Event) []byte,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		afterIDStr := r.URL.Query().Get("after")
		if afterIDStr == "" {
			writeBackfillError(w, http.StatusBadRequest, "missing 'after' query parameter")

			return
		}

		afterID, err := id.ParseEventID(afterIDStr)
		if err != nil {
			writeBackfillError(w, http.StatusBadRequest, "invalid event ID in 'after' parameter")

			return
		}

		limit := 100

		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		if limit > maxBackfillLimit {
			limit = maxBackfillLimit
		}

		events, err := journal.ReadFrom(r.Context(), afterID, limit)
		if err != nil {
			writeBackfillError(w, http.StatusInternalServerError, "failed to read journal")

			return
		}

		if events == nil {
			events = []event.Event{}
		}

		type backfillItem struct {
			ID      string         `json:"id"`
			Type    string         `json:"type"`
			Payload jsontext.Value `json:"payload"`
		}

		items := make([]backfillItem, 0, len(events))
		for _, evt := range events {
			var payload []byte
			if transform != nil {
				payload = transform(evt)
			} else {
				payload = event.PayloadReadOnly(evt)
			}

			items = append(items, backfillItem{
				ID:      evt.ID().String(),
				Type:    string(evt.Type()),
				Payload: payload,
			})
		}

		_ = json.MarshalWrite(w, items)
	})
}

func writeBackfillError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.MarshalWrite(w, map[string]string{"error": msg})
}
