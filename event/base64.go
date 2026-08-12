package event

import "github.com/larsartmann/go-cqrs-lite/codec/v4"

// Backward-compatibility re-exports. The canonical implementations live in
// [codec] — event re-exports them so existing callers that import event/
// for base64 JSON helpers don't break.
//
// New code should import codec/ directly.
var (
	DecodeBase64String  = codec.DecodeBase64String
	UnmarshalBase64JSON = codec.UnmarshalBase64JSON
)
